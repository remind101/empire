// Package cloudformation provides a server for the CloudFormation interface to
// Empire.
package cloudformation

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/remind101/empire/pkg/base62"
	"github.com/remind101/empire/scheduler/ecs/lb"
	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/reporter"
)

var (
	// Allow custom resource provisioners this amount of time do their
	// thing.
	ProvisioningTimeout = time.Duration(20 * time.Minute)

	// And this amount of time to cleanup when they're canceled.
	ProvisioningGraceTimeout = time.Duration(1 * time.Minute)
)

// Provisioner is something that can provision custom resources.
type Provisioner interface {
	// Provision should do the appropriate provisioning, then return:
	//
	// 1. The physical id that was created, if any.
	// 2. The data to return.
	Provision(context.Context, Request) (string, interface{}, error)

	// Properties should return an instance of a type that the properties
	// can be json.Unmarshalled into.
	Properties() interface{}
}

type ProvisionerFunc func(context.Context, Request) (string, interface{}, error)

func (fn ProvisionerFunc) Provision(ctx context.Context, r Request) (string, interface{}, error) {
	return fn(ctx, r)
}

// withTimeout wraps a Provisioner with a context.WithTimeout.
func withTimeout(p Provisioner, timeout time.Duration, grace time.Duration) Provisioner {
	return &timeoutProvisioner{
		Provisioner: p,
		timeout:     timeout,
		grace:       grace,
	}
}

type result struct {
	id   string
	data interface{}
	err  error
}

type timeoutProvisioner struct {
	Provisioner
	timeout time.Duration
	grace   time.Duration
}

func (p *timeoutProvisioner) Provision(ctx context.Context, r Request) (string, interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	done := make(chan result)
	go func() {
		id, data, err := p.Provisioner.Provision(ctx, r)
		done <- result{id, data, err}
	}()

	select {
	case r := <-done:
		return r.id, r.data, r.err
	case <-ctx.Done():
		// When the context is canceled, give the provisioner
		// some extra time to cleanup.
		<-time.After(p.grace)
		select {
		case r := <-done:
			return r.id, r.data, r.err
		default:
			return "", nil, ctx.Err()
		}
	}
}

// Possible request types.
const (
	Create = "Create"
	Update = "Update"
	Delete = "Delete"
)

// Request represents a Custom Resource request.
//
// See http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/crpg-ref-requests.html
type Request struct {
	// The request type is set by the AWS CloudFormation stack operation
	// (create-stack, update-stack, or delete-stack) that was initiated by
	// the template developer for the stack that contains the custom
	// resource.
	//
	// Must be one of: Create, Update, or Delete.
	RequestType string `json:"RequestType"`

	// The response URL identifies a pre-signed Amazon S3 bucket that
	// receives responses from the custom resource provider to AWS
	// CloudFormation.
	ResponseURL string `json:"ResponseURL"`

	// The Amazon Resource Name (ARN) that identifies the stack containing
	// the custom resource.
	//
	// Combining the StackId with the RequestId forms a value that can be
	// used to uniquely identify a request on a particular custom resource.
	StackId string `json:"StackId"`

	// A unique ID for the request.
	//
	// Combining the StackId with the RequestId forms a value that can be
	// used to uniquely identify a request on a particular custom resource.
	RequestId string `json:"RequestId"`

	// The template developer-chosen resource type of the custom resource in
	// the AWS CloudFormation template. Custom resource type names can be up
	// to 60 characters long and can include alphanumeric and the following
	// characters: _@-.
	ResourceType string `json:"ResourceType"`

	// The template developer-chosen name (logical ID) of the custom
	// resource in the AWS CloudFormation template. This is provided to
	// facilitate communication between the custom resource provider and the
	// template developer.
	LogicalResourceId string `json:"LogicalResourceId"`

	// A required custom resource provider-defined physical ID that is
	// unique for that provider.
	//
	// Always sent with Update and Delete requests; never sent with Create.
	PhysicalResourceId string `json:"PhysicalResourceId"`

	// This field contains the contents of the Properties object sent by the
	// template developer. Its contents are defined by the custom resource
	// provider.
	ResourceProperties interface{} `json:"ResourceProperties"`

	// Used only for Update requests. Contains the resource properties that
	// were declared previous to the update request.
	OldResourceProperties interface{} `json:"OldResourceProperties"`
}

// Possible response statuses.
const (
	StatusSuccess = "SUCCESS"
	StatusFailed  = "FAILED"
)

// Response represents the response body we send back to CloudFormation when
// provisioning is complete.
//
// See http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/crpg-ref-responses.html
type Response struct {
	// The status value sent by the custom resource provider in response to
	// an AWS CloudFormation-generated request.
	//
	// Must be either SUCCESS or FAILED.
	Status string `json:"Status"`

	// Describes the reason for a failure response.
	//
	// Required if Status is FAILED; optional otherwise.
	Reason string `json:"Reason"`

	// This value should be an identifier unique to the custom resource
	// vendor, and can be up to 1Kb in size. The value must be a non-empty
	// string.
	PhysicalResourceId string `json:"PhysicalResourceId"`

	// The Amazon Resource Name (ARN) that identifies the stack containing
	// the custom resource. This response value should be copied verbatim
	// from the request.
	StackId string `json:"StackId"`

	// A unique ID for the request. This response value should be copied
	// verbatim from the request.
	RequestId string `json:"RequestId"`

	// The template developer-chosen name (logical ID) of the custom
	// resource in the AWS CloudFormation template. This response value
	// should be copied verbatim from the request.
	LogicalResourceId string `json:"LogicalResourceId"`

	// Optional, custom resource provider-defined name-value pairs to send
	// with the response. The values provided here can be accessed by name
	// in the template with Fn::GetAtt.
	Data interface{} `json:"Data"`
}

// Represents the body of the SQS message, which would have been received from
// SNS.
type Message struct {
	Message string `json:"Message"`
}

// NewResponseFromRequest initializes a new Response from a Request, filling in
// the required verbatim fields.
func NewResponseFromRequest(req Request) Response {
	return Response{
		StackId:           req.StackId,
		RequestId:         req.RequestId,
		LogicalResourceId: req.LogicalResourceId,
	}
}

// sqsClient duck types the sqs.SQS interface.
type sqsClient interface {
	ReceiveMessage(*sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(*sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error)
}

// CustomResourceProvisioner polls for CloudFormation Custom Resource requests
// from an sqs queue, provisions them, then responds back.
type CustomResourceProvisioner struct {
	// Root context.Context to use. If a reporter.Reporter is embedded,
	// errors generated will be reporter there. If a logger.Logger is
	// embedded, logging will be logged there.
	Context context.Context

	// The SQS queue url to listen for CloudFormation Custom Resource
	// requests.
	QueueURL string

	// Provisioners routes a custom resource to the thing that should do the
	// provisioning.
	Provisioners map[string]Provisioner

	client interface {
		Do(*http.Request) (*http.Response, error)
	}
	sqs sqsClient
}

// NewCustomResourceProvisioner returns a new CustomResourceProvisioner with an
// sqs client configured from config.
func NewCustomResourceProvisioner(db *sql.DB, config client.ConfigProvider) *CustomResourceProvisioner {
	p := &CustomResourceProvisioner{
		Provisioners: make(map[string]Provisioner),
		client:       http.DefaultClient,
		sqs:          sqs.New(config),
	}

	p.add("Custom::InstancePort", &InstancePortsProvisioner{
		ports: lb.NewDBPortAllocator(db),
	})

	p.add("Custom::ECSService", &ECSServiceResource{
		ecs: ecs.New(config),
	})

	return p
}

// add adds a custom resource provisioner.
func (c *CustomResourceProvisioner) add(resourceName string, p Provisioner) {
	// Wrap the provisioner with timeouts.
	p = withTimeout(p, ProvisioningTimeout, ProvisioningGraceTimeout)
	c.Provisioners[resourceName] = p
}

// Start starts pulling requests from the queue and provisioning them.
func (c *CustomResourceProvisioner) Start() {
	t := time.Tick(10 * time.Second)

	for range t {
		ctx := c.Context

		resp, err := c.sqs.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl: aws.String(c.QueueURL),
		})
		if err != nil {
			reporter.Report(ctx, err)
			continue
		}

		for _, m := range resp.Messages {
			go func(m *sqs.Message) {
				if err := c.handle(ctx, m); err != nil {
					reporter.Report(ctx, err)
				}
			}(m)
		}
	}
}

func (c *CustomResourceProvisioner) handle(ctx context.Context, message *sqs.Message) error {
	err := c.Handle(ctx, message)
	if err == nil {
		_, err = c.sqs.DeleteMessage(&sqs.DeleteMessageInput{
			QueueUrl:      aws.String(c.QueueURL),
			ReceiptHandle: message.ReceiptHandle,
		})
	}

	return err
}

// Handle handles a single sqs.Message to perform the provisioning.
func (c *CustomResourceProvisioner) Handle(ctx context.Context, message *sqs.Message) error {
	var m Message
	err := json.Unmarshal([]byte(*message.Body), &m)
	if err != nil {
		return fmt.Errorf("error unmarshalling sqs message body: %v", err)
	}

	var req Request
	err = json.Unmarshal([]byte(m.Message), &req)
	if err != nil {
		return fmt.Errorf("error unmarshalling to cloudformation request: %v", err)
	}

	resp := NewResponseFromRequest(req)
	resp.PhysicalResourceId, resp.Data, err = c.provision(ctx, m, req)
	switch err {
	case nil:
		resp.Status = StatusSuccess
		logger.Info(ctx, "cloudformation.provision",
			"request", req,
			"response", resp,
		)
	default:
		resp.Status = StatusFailed
		resp.Reason = err.Error()
		logger.Error(ctx, "cloudformation.provision.error",
			"request", req,
			"response", resp,
			"err", err.Error(),
		)
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	r, err := http.NewRequest("PUT", req.ResponseURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}

	httpResp, err := c.client.Do(r)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()
	body, _ := ioutil.ReadAll(httpResp.Body)

	if code := httpResp.StatusCode; code/100 != 2 {
		return fmt.Errorf("unexpected response from pre-signed url: %v: %v", code, string(body))
	}

	return nil
}

func (c *CustomResourceProvisioner) provision(ctx context.Context, m Message, req Request) (string, interface{}, error) {
	p, ok := c.Provisioners[req.ResourceType]
	if !ok {
		return "", nil, fmt.Errorf("no provisioner for %v", req.ResourceType)
	}

	// If the provisioner defines a type for the properties, let's unmarhsal
	// into that Go type.
	req.ResourceProperties = p.Properties()
	req.OldResourceProperties = p.Properties()
	err := json.Unmarshal([]byte(m.Message), &req)
	if err != nil {
		return "", nil, fmt.Errorf("error unmarshalling to cloudformation request: %v", err)
	}

	return p.Provision(ctx, req)
}

// IntValue defines an int64 type that can parse integers as strings from json.
// It's common to use `Ref`'s inside templates, which means the value of some
// properties could be a string or an integer.
type IntValue int64

func intValue(v int64) *IntValue {
	i := IntValue(v)
	return &i
}

func (i *IntValue) UnmarshalJSON(b []byte) error {
	var si int64
	if err := json.Unmarshal(b, &si); err == nil {
		*i = IntValue(si)
		return nil
	}

	v, err := strconv.Atoi(string(b[1 : len(b)-1]))
	if err != nil {
		return fmt.Errorf("error parsing int from string: %v", err)
	}

	*i = IntValue(v)
	return nil
}

func (i *IntValue) Value() *int64 {
	if i == nil {
		return nil
	}
	p := int64(*i)
	return &p
}

// hashRequest returns a compact unique identifier for the request.
func hashRequest(r Request) string {
	h := fnv.New64()
	h.Write([]byte(fmt.Sprintf("%s.%s", r.StackId, r.RequestId)))
	return base62.Encode(h.Sum64())
}

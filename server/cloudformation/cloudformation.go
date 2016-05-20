package cloudformation

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/remind101/empire/scheduler/ecs/lb"
	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/reporter"
)

// Provisioner is something that can provision custom resources.
type Provisioner interface {
	// Provision should do the appropriate provisioning, then return:
	//
	// 1. The physical id that was created, if any.
	// 2. The data to return.
	Provision(Request) (string, interface{}, error)
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
	ResourceProperties json.RawMessage `json:"ResourceProperties"`

	// Used only for Update requests. Contains the resource properties that
	// were declared previous to the update request.
	OldResourceProperties json.RawMessage `json:"OldResourceProperties"`
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
	// Logger to use to perform logging.
	Logger logger.Logger

	// The SQS queue url to listen for CloudFormation Custom Resource
	// requests.
	QueueURL string

	// Provisioners routes a custom resource to the thing that should do the
	// provisioning.
	Provisioners map[string]Provisioner

	// Reporter is called when an error occurs during provisioning.
	Reporter reporter.Reporter

	client interface {
		Do(*http.Request) (*http.Response, error)
	}
	sqs sqsClient
}

// NewCustomResourceProvisioner returns a new CustomResourceProvisioner with an
// sqs client configured from config.
func NewCustomResourceProvisioner(db *sql.DB, config client.ConfigProvider) *CustomResourceProvisioner {
	return &CustomResourceProvisioner{
		Provisioners: map[string]Provisioner{
			"Custom::InstancePort": &InstancePortsProvisioner{
				ports: lb.NewDBPortAllocator(db),
			},
			"Custom::ECSService": &ECSServiceResource{
				ecs: ecs.New(config),
			},
		},
		client: http.DefaultClient,
		sqs:    sqs.New(config),
	}
}

// Start starts pulling requests from the queue and provisioning them.
func (c *CustomResourceProvisioner) Start() {
	t := time.Tick(10 * time.Second)

	for range t {
		ctx := context.Background()

		resp, err := c.sqs.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl: aws.String(c.QueueURL),
		})
		if err != nil {
			c.Reporter.Report(ctx, err)
			continue
		}

		for _, m := range resp.Messages {
			if err := c.handle(m); err != nil {
				c.Reporter.Report(ctx, err)
				continue
			}
		}
	}
}

func (c *CustomResourceProvisioner) handle(message *sqs.Message) error {
	err := c.Handle(message)
	if err == nil {
		_, err = c.sqs.DeleteMessage(&sqs.DeleteMessageInput{
			QueueUrl:      aws.String(c.QueueURL),
			ReceiptHandle: message.ReceiptHandle,
		})
	}

	return err
}

// Handle handles a single sqs.Message to perform the provisioning.
func (c *CustomResourceProvisioner) Handle(message *sqs.Message) error {
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

	p, ok := c.Provisioners[req.ResourceType]
	if !ok {
		return fmt.Errorf("no provisioner for %v", req.ResourceType)
	}

	resp := NewResponseFromRequest(req)
	resp.PhysicalResourceId, resp.Data, err = p.Provision(req)
	switch err {
	case nil:
		resp.Status = StatusSuccess
		c.Logger.Info("cloudformation.provision",
			"request", req,
			"response", resp,
		)
	default:
		resp.Status = StatusFailed
		resp.Reason = err.Error()
		c.Logger.Error("cloudformation.provision.error",
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

// IntValue defines an int64 type that can parse integers as strings from json.
// It's common to use `Ref`'s inside templates, which means the value of some
// properties could be a string or an integer.
type IntValue int64

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

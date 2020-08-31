// Package customresources provides a Go library for building CloudFormation
// custom resource handlers.
package customresources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/remind101/empire/pkg/base62"
	"golang.org/x/net/context"
)

// Possible request types.
const (
	Create = "Create"
	Update = "Update"
	Delete = "Delete"
)

// Possible response statuses.
const (
	StatusSuccess = "SUCCESS"
	StatusFailed  = "FAILED"
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

// Hash returns a compact unique identifier for the request.
func (r *Request) Hash() string {
	h := fnv.New64()
	h.Write([]byte(fmt.Sprintf("%s.%s", r.StackId, r.RequestId)))
	return base62.Encode(h.Sum64())
}

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
	Reason string `json:"Reason,omitempty"`

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
	Data interface{} `json:"Data,omitempty"`
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

// SendResponse sends the response the Response to the requests response url
func SendResponse(req Request, response Response) error {
	return SendResponseWithClient(http.DefaultClient, req, response)
}

// SendResponseWithClient uploads the response to the requests signed
// ResponseURL.
func SendResponseWithClient(client interface {
	Do(*http.Request) (*http.Response, error)
}, req Request, response Response) error {
	c := ResponseClient{client}
	return c.SendResponse(req, response)
}

// ResponseClient is a client that can send responses to a requests ResponseURL.
type ResponseClient struct {
	client interface {
		Do(*http.Request) (*http.Response, error)
	}
}

// SendResponse sends the response to the request's ResponseURL.
func (c *ResponseClient) SendResponse(req Request, response Response) error {
	raw, err := json.Marshal(response)
	if err != nil {
		return err
	}

	r, err := http.NewRequest("PUT", req.ResponseURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}

	resp, err := c.client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	if code := resp.StatusCode; code/100 != 2 {
		return fmt.Errorf("unexpected response from pre-signed url: %v: %v", code, string(body))
	}

	return nil
}

// WithTimeout wraps a Provisioner with a context.WithTimeout.
func WithTimeout(p Provisioner, timeout time.Duration, grace time.Duration) Provisioner {
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
	// TODO: how to make this a debug log level?
	log.Println(r)
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

// Package cloudformation provides a server for the CloudFormation interface to
// Empire.
package cloudformation

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/remind101/empire/pkg/cloudformation/customresources"
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

// Represents the body of the SQS message, which would have been received from
// SNS.
type Message struct {
	Message string `json:"Message"`
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
	Provisioners map[string]customresources.Provisioner

	sendResponse func(customresources.Request, customresources.Response) error
	sqs          sqsClient
}

// NewCustomResourceProvisioner returns a new CustomResourceProvisioner with an
// sqs client configured from config.
func NewCustomResourceProvisioner(db *sql.DB, config client.ConfigProvider) *CustomResourceProvisioner {
	p := &CustomResourceProvisioner{
		Provisioners: make(map[string]customresources.Provisioner),
		sendResponse: customresources.SendResponse,
		sqs:          sqs.New(config),
	}

	p.add("Custom::InstancePort", &InstancePortsProvisioner{
		ports: lb.NewDBPortAllocator(db),
	})

	p.add("Custom::ECSService", &ECSServiceResource{
		ecs: ecs.New(config),
	})

	store := &dbEnvironmentStore{db}
	p.add("Custom::ECSEnvironment", &ECSEnvironmentResource{
		environmentStore: store,
	})
	p.add("Custom::ECSTaskDefinition", &ECSTaskDefinitionResource{
		ecs:              ecs.New(config),
		environmentStore: store,
	})

	return p
}

// add adds a custom resource provisioner.
func (c *CustomResourceProvisioner) add(resourceName string, p customresources.Provisioner) {
	// Wrap the provisioner with timeouts.
	p = customresources.WithTimeout(p, ProvisioningTimeout, ProvisioningGraceTimeout)
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

	var req customresources.Request
	err = json.Unmarshal([]byte(m.Message), &req)
	if err != nil {
		return fmt.Errorf("error unmarshalling to cloudformation request: %v", err)
	}

	logger.Info(ctx, "cloudformation.provision.request",
		"request_id", req.RequestId,
		"stack_id", req.StackId,
		"request_type", req.RequestType,
		"resource_type", req.ResourceType,
		"logical_resource_id", req.LogicalResourceId,
		"physical_resource_id", req.PhysicalResourceId,
	)

	resp := customresources.NewResponseFromRequest(req)

	// CloudFormation is weird. PhysicalResourceId is required when creating
	// a resource, but if the creation fails, how would we have a physical
	// resource id? In cases where a Create request fails, we set the
	// physical resource id to `failed/Create`. When a delete request comes
	// in to delete that resource, we just send back a SUCCESS response so
	// CloudFormation is happy.
	if req.RequestType == customresources.Delete && req.PhysicalResourceId == fmt.Sprintf("failed/%s", customresources.Create) {
		resp.PhysicalResourceId = req.PhysicalResourceId
	} else {
		resp.PhysicalResourceId, resp.Data, err = c.provision(ctx, m, req)
	}

	// Allow provisioners to just return "" to indicate that the physical
	// resource id did not change.
	if resp.PhysicalResourceId == "" && req.PhysicalResourceId != "" {
		resp.PhysicalResourceId = req.PhysicalResourceId
	}

	switch err {
	case nil:
		resp.Status = customresources.StatusSuccess
		logger.Info(ctx, "cloudformation.provision.success",
			"request_id", req.RequestId,
			"stack_id", req.StackId,
			"physical_resource_id", resp.PhysicalResourceId,
		)
	default:
		// A physical resource id is required, so if a Create request
		// fails, and there's no physical resource id, CloudFormation
		// will only say `Invalid PhysicalResourceId` in the status
		// Reason instead of the actual error that caused the Create to
		// fail.
		if req.RequestType == customresources.Create && resp.PhysicalResourceId == "" {
			resp.PhysicalResourceId = fmt.Sprintf("failed/%s", req.RequestType)
		}

		resp.Status = customresources.StatusFailed
		resp.Reason = err.Error()
		logger.Error(ctx, "cloudformation.provision.error",
			"request_id", req.RequestId,
			"stack_id", req.StackId,
			"err", err.Error(),
		)
	}

	return c.sendResponse(req, resp)
}

func (c *CustomResourceProvisioner) provision(ctx context.Context, m Message, req customresources.Request) (string, interface{}, error) {
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

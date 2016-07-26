// Package cloudformation provides a server for the CloudFormation interface to
// Empire.
package cloudformation

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/remind101/empire/pkg/cloudformation/customresources"
	"github.com/remind101/empire/scheduler/ecs/lb"
	"github.com/remind101/pkg/logger"
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

// CustomResourceProvisioner polls for CloudFormation Custom Resource requests
// from an sqs queue, provisions them, then responds back.
type CustomResourceProvisioner struct {
	*SQSDispatcher

	// Provisioners routes a custom resource to the thing that should do the
	// provisioning.
	Provisioners map[string]customresources.Provisioner

	sendResponse func(customresources.Request, customresources.Response) error
}

// NewCustomResourceProvisioner returns a new CustomResourceProvisioner with an
// sqs client configured from config.
func NewCustomResourceProvisioner(db *sql.DB, config client.ConfigProvider) *CustomResourceProvisioner {
	p := &CustomResourceProvisioner{
		SQSDispatcher: newSQSDispatcher(config),
		Provisioners:  make(map[string]customresources.Provisioner),
		sendResponse:  customresources.SendResponse,
	}

	p.add("Custom::InstancePort", &InstancePortsProvisioner{
		ports: lb.NewDBPortAllocator(db),
	})

	ecs := newECSClient(config)
	p.add("Custom::ECSService", &ECSServiceResource{
		ecs: ecs,
	})

	store := &dbEnvironmentStore{db}
	p.add("Custom::ECSEnvironment", newECSEnvironmentProvisioner(&ECSEnvironmentResource{
		environmentStore: store,
	}))
	p.add("Custom::ECSTaskDefinition", newECSTaskDefinitionProvisioner(&ECSTaskDefinitionResource{
		ecs:              ecs,
		environmentStore: store,
	}))

	return p
}

// add adds a custom resource provisioner.
func (c *CustomResourceProvisioner) add(resourceName string, p customresources.Provisioner) {
	// Wrap the provisioner with timeouts.
	p = customresources.WithTimeout(p, ProvisioningTimeout, ProvisioningGraceTimeout)
	c.Provisioners[resourceName] = p
}

func (c *CustomResourceProvisioner) Start() {
	c.SQSDispatcher.Start(c.Handle)
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

type properties interface {
	ReplacementHash() (uint64, error)
}

// provisioner provides convenience over the customresources.Provisioner
// interface.
type provisioner struct {
	properties func() properties

	Create func(context.Context, customresources.Request) (string, interface{}, error)
	Update func(context.Context, customresources.Request) (interface{}, error)
	Delete func(context.Context, customresources.Request) error
}

func (p *provisioner) Properties() interface{} {
	return p.properties()
}

func (p *provisioner) Provision(ctx context.Context, req customresources.Request) (string, interface{}, error) {
	switch req.RequestType {
	case customresources.Create:
		return p.Create(ctx, req)
	case customresources.Update:
		n := req.ResourceProperties.(properties)
		o := req.OldResourceProperties.(properties)

		replace, err := requiresReplacement(n, o)
		if err != nil {
			return req.PhysicalResourceId, nil, err
		}

		// If the new properties require a replacement of the resource,
		// perform a Create. CloudFormation will send us a request to
		// delete the old resource later.
		if replace {
			return p.Create(ctx, req)
		}

		id := req.PhysicalResourceId
		data, err := p.Update(ctx, req)
		return id, data, err
	case customresources.Delete:
		return req.PhysicalResourceId, nil, p.Delete(ctx, req)
	default:
		panic(fmt.Sprintf("unable to handle %s request", req.RequestType))
	}
}

// requiresReplacement returns true if the new properties require a replacement
// of the old properties.
func requiresReplacement(n, o properties) (bool, error) {
	a, err := n.ReplacementHash()
	if err != nil {
		return false, err
	}

	b, err := o.ReplacementHash()
	if err != nil {
		return false, err
	}

	return a != b, nil
}

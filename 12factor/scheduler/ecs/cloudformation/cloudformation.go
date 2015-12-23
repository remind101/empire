// Package cloudformation provides a StackBuilder for provisioning the AWS
// resources for an App using a CloudFormation stack.
package cloudformation

import (
	"encoding/json"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/remind101/empire/12factor"
)

const ecsServiceType = "AWS::ECS::Service"

type cloudformationClient interface {
	CreateStack(input *cloudformation.CreateStackInput) (*cloudformation.CreateStackOutput, error)
	DeleteStack(input *cloudformation.DeleteStackInput) (*cloudformation.DeleteStackOutput, error)
	ListStackResourcesPages(*cloudformation.ListStackResourcesInput, func(*cloudformation.ListStackResourcesOutput, bool) bool) error
	DescribeStackResource(input *cloudformation.DescribeStackResourceInput) (*cloudformation.DescribeStackResourceOutput, error)
}

type serviceMetadata struct {
	Name string `json:"name"`
}

// StackBuilder is an implementation of the ecs.StackBuilder interface that
// builds the stack using CloudFormation.
type StackBuilder struct {
	// Template is a text/template that will be executed using the
	// twelvefactor.Manifest as data. This template should return a valid
	// CloudFormation JSON manifest.
	Template *template.Template

	// stackName returns the name of the stack for the app.
	stackName func(app string) string

	cloudformation cloudformationClient
}

// NewStackBuilder returns a new StackBuilder instance.
func NewStackBuilder(config client.ConfigProvider) *StackBuilder {
	return &StackBuilder{
		cloudformation: cloudformation.New(config),
		stackName:      stackName,
	}
}

// Build builds the CloudFormation stack for the App.
func (b *StackBuilder) Build(app twelvefactor.App) error {
	return nil
}

// Services returns a mapping of process -> ecs service. It assumes the ECS
// service resources in the cloudformation template have metadata that includes
// a "Name" key that specifies the process name.
func (b *StackBuilder) Services(app string) (map[string]string, error) {
	stack := b.stackName(app)

	// Get a summary of all of the stacks resources.
	var summaries []*cloudformation.StackResourceSummary
	if err := b.cloudformation.ListStackResourcesPages(&cloudformation.ListStackResourcesInput{
		StackName: aws.String(stack),
	}, func(p *cloudformation.ListStackResourcesOutput, lastPage bool) bool {
		summaries = append(summaries, p.StackResourceSummaries...)
		return true
	}); err != nil {
		return nil, err
	}

	services := make(map[string]string)
	for _, summary := range summaries {
		if *summary.ResourceType == ecsServiceType {
			resp, err := b.cloudformation.DescribeStackResource(&cloudformation.DescribeStackResourceInput{
				StackName:         aws.String(stack),
				LogicalResourceId: summary.LogicalResourceId,
			})
			if err != nil {
				return services, err
			}

			var meta serviceMetadata
			if err := json.Unmarshal([]byte(*resp.StackResourceDetail.Metadata), &meta); err != nil {
				return services, err
			}

			services[meta.Name] = *resp.StackResourceDetail.PhysicalResourceId
		}
	}

	return services, nil
}

// stackName returns a stack name for the app id.
func stackName(app string) string {
	return app
}

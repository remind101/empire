// Package cloudformation implements the Scheduler interface for ECS by using
// CloudFormation to provision and update resources.
package cloudformation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/remind101/empire/pkg/arn"
	"github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/scheduler"
	"golang.org/x/net/context"
)

const ecsServiceType = "AWS::ECS::Service"

type cloudformationClient interface {
	CreateStack(*cloudformation.CreateStackInput) (*cloudformation.CreateStackOutput, error)
	UpdateStack(*cloudformation.UpdateStackInput) (*cloudformation.UpdateStackOutput, error)
	DeleteStack(*cloudformation.DeleteStackInput) (*cloudformation.DeleteStackOutput, error)
	ListStackResourcesPages(*cloudformation.ListStackResourcesInput, func(*cloudformation.ListStackResourcesOutput, bool) bool) error
	DescribeStackResource(*cloudformation.DescribeStackResourceInput) (*cloudformation.DescribeStackResourceOutput, error)
	DescribeStacks(*cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error)
	WaitUntilStackCreateComplete(*cloudformation.DescribeStacksInput) error
	WaitUntilStackUpdateComplete(*cloudformation.DescribeStacksInput) error
}

type ecsClient interface {
	ListTasksPages(input *ecs.ListTasksInput, fn func(p *ecs.ListTasksOutput, lastPage bool) (shouldContinue bool)) error
	DescribeTaskDefinition(*ecs.DescribeTaskDefinitionInput) (*ecs.DescribeTaskDefinitionOutput, error)
	DescribeTasks(*ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error)
	StopTask(*ecs.StopTaskInput) (*ecs.StopTaskOutput, error)
	UpdateService(*ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error)
}

// Template represents something that can generate a stack body. Conveniently
// the same interface as text/template.Template.
type Template interface {
	Execute(wr io.Writer, data interface{}) error
}

// Templates should add metadata to the ECS resources with the following
// structure.
type serviceMetadata struct {
	// This is the name of the process that this ECS service is for.
	Name string `json:"name"`
}

// Scheduler implements the scheduler.Scheduler interface using CloudFormation
// to provision resources.
type Scheduler struct {
	// Template is a text/template that will be executed using the
	// twelvefactor.Manifest as data. This template should return a valid
	// CloudFormation JSON manifest.
	Template Template

	// The ECS cluster to run tasks in.
	Cluster string

	// If true, wait for stack updates and creates to complete.
	Wait bool

	// stackName returns the name of the stack for the app.
	stackName func(app string) string

	// CloudFormation client for creating stacks.
	cloudformation cloudformationClient

	// ECS client for performing ECS API calls.
	ecs ecsClient
}

// NewScheduler returns a new Scheduler instance.
func NewScheduler(config client.ConfigProvider) *Scheduler {
	return &Scheduler{
		cloudformation: cloudformation.New(config),
		ecs:            ecs.New(config),
		stackName:      stackName,
	}
}

// Submit creates (or updates) the CloudFormation stack for the app.
func (s *Scheduler) Submit(ctx context.Context, app *scheduler.App) error {
	stackName := s.stackName(app.ID)

	buf := new(bytes.Buffer)
	if err := s.Template.Execute(buf, app); err != nil {
		return err
	}

	desc, err := s.cloudformation.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})

	if err, ok := err.(awserr.Error); ok && err.Message() == fmt.Sprintf("Stack with id %s does not exist", stackName) {
		if _, err := s.cloudformation.CreateStack(&cloudformation.CreateStackInput{
			StackName:    aws.String(stackName),
			TemplateBody: aws.String(buf.String()),
		}); err != nil {
			return err
		}

		if s.Wait {
			if err := s.cloudformation.WaitUntilStackCreateComplete(&cloudformation.DescribeStacksInput{
				StackName: aws.String(stackName),
			}); err != nil {
				return err
			}
		}
	} else if err == nil {
		stack := desc.Stacks[0]
		status := *stack.StackStatus

		// If there's currently an update happening, wait for it to
		// complete.
		if strings.Contains(status, "IN_PROGRESS") {
			if strings.Contains(status, "CREATE") {
				if err := s.cloudformation.WaitUntilStackCreateComplete(&cloudformation.DescribeStacksInput{
					StackName: aws.String(stackName),
				}); err != nil {
					return err
				}
			} else if strings.Contains(status, "UPDATE") {
				if err := s.cloudformation.WaitUntilStackUpdateComplete(&cloudformation.DescribeStacksInput{
					StackName: aws.String(stackName),
				}); err != nil {
					return err
				}
			}
		}

		if _, err := s.cloudformation.UpdateStack(&cloudformation.UpdateStackInput{
			StackName:    aws.String(stackName),
			TemplateBody: aws.String(buf.String()),
		}); err != nil {
			return err
		}

		if s.Wait {
			if err := s.cloudformation.WaitUntilStackUpdateComplete(&cloudformation.DescribeStacksInput{
				StackName: aws.String(stackName),
			}); err != nil {
				return err
			}
		}
	} else {
		return err
	}

	return nil
}

// Remove removes the CloudFormation stack for the given app.
func (s *Scheduler) Remove(_ context.Context, appID string) error {
	stackName := s.stackName(appID)

	if _, err := s.cloudformation.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(stackName),
	}); err != nil {
		return err
	}

	return nil
}

// Instances returns all of the running tasks for this application.
func (s *Scheduler) Instances(ctx context.Context, app string) ([]*scheduler.Instance, error) {
	var instances []*scheduler.Instance

	tasks, err := s.tasks(app)
	if err != nil {
		return nil, err
	}

	taskDefinitions := make(map[string]*ecs.TaskDefinition)
	for _, t := range tasks {
		k := *t.TaskDefinitionArn

		if _, ok := taskDefinitions[k]; !ok {
			resp, err := s.ecs.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
				TaskDefinition: t.TaskDefinitionArn,
			})
			if err != nil {
				return nil, err
			}
			taskDefinitions[k] = resp.TaskDefinition
		}
	}

	for _, t := range tasks {
		taskDefinition := taskDefinitions[*t.TaskDefinitionArn]

		id, err := arn.ResourceID(*t.TaskArn)
		if err != nil {
			return instances, err
		}

		p, err := taskDefinitionToProcess(taskDefinition)
		if err != nil {
			return instances, err
		}

		state := safeString(t.LastStatus)
		var updatedAt time.Time
		switch state {
		case "PENDING":
			updatedAt = *t.CreatedAt
		case "RUNNING":
			updatedAt = *t.StartedAt
		case "STOPPED":
			updatedAt = *t.StoppedAt
		}

		instances = append(instances, &scheduler.Instance{
			Process:   p,
			State:     state,
			ID:        id,
			UpdatedAt: updatedAt,
		})
	}

	return instances, nil
}

// tasks returns all of the ECS tasks for this app.
func (s *Scheduler) tasks(app string) ([]*ecs.Task, error) {
	services, err := s.Services(app)
	if err != nil {
		return nil, err
	}

	var arns []*string
	for _, serviceArn := range services {
		id, err := arn.ResourceID(serviceArn)
		if err != nil {
			return nil, err
		}

		var taskArns []*string
		if err := s.ecs.ListTasksPages(&ecs.ListTasksInput{
			Cluster:     aws.String(s.Cluster),
			ServiceName: aws.String(id),
		}, func(resp *ecs.ListTasksOutput, lastPage bool) bool {
			taskArns = append(taskArns, resp.TaskArns...)
			return true
		}); err != nil {
			return nil, err
		}

		if len(taskArns) == 0 {
			continue
		}

		arns = append(arns, taskArns...)
	}

	resp, err := s.ecs.DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: aws.String(s.Cluster),
		Tasks:   arns,
	})

	return resp.Tasks, err
}

// Services returns the names of the map that maps the name of a process to the
// ARN of the ECS service.
func (s *Scheduler) Services(app string) (map[string]string, error) {
	stackName := s.stackName(app)

	// Get a summary of all of the stacks resources.
	var summaries []*cloudformation.StackResourceSummary
	if err := s.cloudformation.ListStackResourcesPages(&cloudformation.ListStackResourcesInput{
		StackName: aws.String(stackName),
	}, func(p *cloudformation.ListStackResourcesOutput, lastPage bool) bool {
		summaries = append(summaries, p.StackResourceSummaries...)
		return true
	}); err != nil {
		return nil, err
	}

	services := make(map[string]string)
	for _, summary := range summaries {
		if *summary.ResourceType == ecsServiceType {
			resp, err := s.cloudformation.DescribeStackResource(&cloudformation.DescribeStackResourceInput{
				StackName:         aws.String(stackName),
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

// Stop stops the given ECS task.
func (s *Scheduler) Stop(ctx context.Context, instanceID string) error {
	_, err := s.ecs.StopTask(&ecs.StopTaskInput{
		Cluster: aws.String(s.Cluster),
		Task:    aws.String(instanceID),
	})
	return err
}

// Scale scales the ECS service for the given process to the desired number of
// instances.
func (s *Scheduler) Scale(ctx context.Context, app string, process string, instances uint) error {
	services, err := s.Services(app)
	if err != nil {
		return err
	}

	serviceArn, ok := services[process]
	if !ok {
		return fmt.Errorf("no %s process found", process)
	}

	// TODO: Should we just update a parameter in the stack instead?
	_, err = s.ecs.UpdateService(&ecs.UpdateServiceInput{
		Cluster:      aws.String(s.Cluster),
		DesiredCount: aws.Int64(int64(instances)),
		Service:      aws.String(serviceArn),
	})
	return err
}

func (s *Scheduler) Run(ctx context.Context, app *scheduler.App, process *scheduler.Process, in io.Reader, out io.Writer) error {
	// Do the same thing as the ECS scheduler.
	return nil
}

// stackName returns a stack name for the app id.
func stackName(appID string) string {
	return fmt.Sprintf("app-%s", appID)
}

// taskDefinitionToProcess takes an ECS Task Definition and converts it to a
// Process.
func taskDefinitionToProcess(td *ecs.TaskDefinition) (*scheduler.Process, error) {
	// If this task definition has no container definitions, then something
	// funky is up.
	if len(td.ContainerDefinitions) == 0 {
		return nil, errors.New("task definition had no container definitions")
	}

	container := td.ContainerDefinitions[0]

	var command []string
	for _, s := range container.Command {
		command = append(command, *s)
	}

	env := make(map[string]string)
	for _, kvp := range container.Environment {
		if kvp != nil {
			env[safeString(kvp.Name)] = safeString(kvp.Value)
		}
	}

	return &scheduler.Process{
		Type:        safeString(container.Name),
		Command:     command,
		Env:         env,
		CPUShares:   uint(*container.Cpu),
		MemoryLimit: uint(*container.Memory) * bytesize.MB,
		Nproc:       uint(softLimit(container.Ulimits, "nproc")),
	}, nil
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func softLimit(ulimits []*ecs.Ulimit, name string) int64 {
	if ulimits == nil {
		return 0
	}

	for _, u := range ulimits {
		if *u.Name == name {
			return *u.SoftLimit
		}
	}

	return 0
}

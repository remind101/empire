// Package cloudformation implements the Scheduler interface for ECS by using
// CloudFormation to provision and update resources.
package cloudformation

import (
	"bytes"
	"crypto/sha1"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/remind101/empire/pkg/arn"
	"github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/scheduler"
	"golang.org/x/net/context"
)

// The identifier of the ECS Service resource in CloudFormation.
const ecsServiceType = "AWS::ECS::Service"

// DefaultStackNameTemplate is the default text/template for generating a
// CloudFormation stack name for an app.
var DefaultStackNameTemplate = template.Must(template.New("stack_name").Parse("{{.Name}}"))

// errNoStack can be returned when there's no CloudFormation stack for a given
// app.
var errNoStack = errors.New("no stack for app found")

// cloudformationClient duck types the cloudformation.CloudFormation interface
// that we use.
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

// ecsClient duck types the ecs.ECS interface that we use.
type ecsClient interface {
	ListTasksPages(input *ecs.ListTasksInput, fn func(p *ecs.ListTasksOutput, lastPage bool) (shouldContinue bool)) error
	DescribeTaskDefinition(*ecs.DescribeTaskDefinitionInput) (*ecs.DescribeTaskDefinitionOutput, error)
	DescribeTasks(*ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error)
	RunTask(*ecs.RunTaskInput) (*ecs.RunTaskOutput, error)
	RegisterTaskDefinition(*ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error)
	StopTask(*ecs.StopTaskInput) (*ecs.StopTaskOutput, error)
	UpdateService(*ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error)
}

// s3Client duck types the s3.S3 interface that we use.
type s3Client interface {
	PutObject(*s3.PutObjectInput) (*s3.PutObjectOutput, error)
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

	// The name of the bucket to store templates in.
	Bucket string

	// A text/template that will generate the stack name for the app. This
	// template will be executed with a scheduler.App as it's data.
	StackNameTemplate *template.Template

	// CloudFormation client for creating stacks.
	cloudformation cloudformationClient

	// ECS client for performing ECS API calls.
	ecs ecsClient

	// S3 client to upload templates to s3.
	s3 s3Client

	db *sql.DB
}

// NewScheduler returns a new Scheduler instance.
func NewScheduler(db *sql.DB, config client.ConfigProvider) *Scheduler {
	return &Scheduler{
		cloudformation: cloudformation.New(config),
		ecs:            ecs.New(config),
		s3:             s3.New(config),
		db:             db,
	}
}

// Submit creates (or updates) the CloudFormation stack for the app.
func (s *Scheduler) Submit(ctx context.Context, app *scheduler.App) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	err = s.submit(ctx, tx, app)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// Submit creates (or updates) the CloudFormation stack for the app.
func (s *Scheduler) submit(ctx context.Context, tx *sql.Tx, app *scheduler.App) error {
	stackName, err := s.stackName(app.ID)
	if err == errNoStack {
		t := s.StackNameTemplate
		if t == nil {
			t = DefaultStackNameTemplate
		}
		buf := new(bytes.Buffer)
		if err := t.Execute(buf, app); err != nil {
			return fmt.Errorf("error generating stack name: %v", err)
		}
		stackName = buf.String()
		if _, err := tx.Exec(`INSERT INTO stacks (app_id, stack_name) VALUES ($1, $2)`, app.ID, stackName); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err := s.Template.Execute(buf, app); err != nil {
		return err
	}

	key := fmt.Sprintf("%x", sha1.Sum(buf.Bytes()))

	_, err = s.s3.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(s.Bucket),
		Key:         aws.String(fmt.Sprintf("/%s", key)),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.Bucket, key)

	desc, err := s.cloudformation.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})

	tags := []*cloudformation.Tag{
		{Key: aws.String("empire.app.id"), Value: aws.String(app.ID)},
		{Key: aws.String("empire.app.name"), Value: aws.String(app.Name)},
	}

	if err, ok := err.(awserr.Error); ok && err.Message() == fmt.Sprintf("Stack with id %s does not exist", stackName) {
		if _, err := s.cloudformation.CreateStack(&cloudformation.CreateStackInput{
			StackName:   aws.String(stackName),
			TemplateURL: aws.String(url),
			Tags:        tags,
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
			StackName:   aws.String(stackName),
			TemplateURL: aws.String(url),
			// TODO: Update Go client
			// Tags:         tags,
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
func (s *Scheduler) Remove(ctx context.Context, appID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	err = s.remove(ctx, tx, appID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// Remove removes the CloudFormation stack for the given app.
func (s *Scheduler) remove(_ context.Context, tx *sql.Tx, appID string) error {
	stackName, err := s.stackName(appID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM stacks WHERE app_id = $1`, appID)
	if err != nil {
		return err
	}

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

	// Find all of the tasks started by the ECS services.
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

	// Find all of the tasks started by Run.
	if err := s.ecs.ListTasksPages(&ecs.ListTasksInput{
		Cluster:   aws.String(s.Cluster),
		StartedBy: aws.String(app),
	}, func(resp *ecs.ListTasksOutput, lastPage bool) bool {
		arns = append(arns, resp.TaskArns...)
		return true
	}); err != nil {
		return nil, err
	}

	resp, err := s.ecs.DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: aws.String(s.Cluster),
		Tasks:   arns,
	})

	return resp.Tasks, err
}

// Services returns a map that maps the name of the process (e.g. web) to the
// ARN of the associated ECS service.
func (s *Scheduler) Services(appID string) (map[string]string, error) {
	stackName, err := s.stackName(appID)
	if err != nil {
		return nil, err
	}

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
func (s *Scheduler) Scale(ctx context.Context, appID string, process string, instances uint) error {
	services, err := s.Services(appID)
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

// Run registers a TaskDefinition for the process, and calls RunTask.
func (m *Scheduler) Run(ctx context.Context, app *scheduler.App, process *scheduler.Process, in io.Reader, out io.Writer) error {
	if out != nil {
		return errors.New("running an attached process is not implemented by the ECS manager.")
	}

	t, ok := m.Template.(interface {
		ContainerDefinition(*scheduler.Process) *ecs.ContainerDefinition
	})
	if !ok {
		return errors.New("provided template can't generate a container definition for this process")
	}

	resp, err := m.ecs.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
		Family: aws.String(fmt.Sprintf("%s--%s", app.ID, process.Type)),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			t.ContainerDefinition(process),
		},
	})
	if err != nil {
		return fmt.Errorf("error registering TaskDefinition: %v", err)
	}

	_, err = m.ecs.RunTask(&ecs.RunTaskInput{
		TaskDefinition: resp.TaskDefinition.TaskDefinitionArn,
		Cluster:        aws.String(m.Cluster),
		Count:          aws.Int64(1),
		StartedBy:      aws.String(app.ID),
	})
	if err != nil {
		return fmt.Errorf("error calling RunTask: %v", err)
	}

	return nil
}

// stackName returns the name of the CloudFormation stack for the app id.
func (s *Scheduler) stackName(appID string) (string, error) {
	var stackName string
	err := s.db.QueryRow(`SELECT stack_name FROM stacks WHERE app_id = $1`, appID).Scan(&stackName)
	if err == sql.ErrNoRows {
		return "", errNoStack
	}
	return stackName, err
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

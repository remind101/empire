// Package cloudformation implements the Scheduler interface for ECS by using
// CloudFormation to provision and update resources.
package cloudformation

import (
	"bytes"
	"crypto/sha1"
	"database/sql"
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

// The name of the output key where process names are mapped to ECS services.
// This output is expected to be a comma delimited list of `process=servicearn`
// values.
const servicesOutput = "Services"

// CloudFormation limits
//
// See http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cloudformation-limits.html
const (
	MaxTemplateSize = 460800 // bytes
)

// ECS limits
const (
	MaxDescribeTasks = 100
)

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

// Scheduler implements the scheduler.Scheduler interface using CloudFormation
// to provision resources.
type Scheduler struct {
	// Template is a text/template that will be executed using the
	// twelvefactor.Manifest as data. This template should return a valid
	// CloudFormation JSON template.
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

	// Any additional tags to add to stacks.
	Tags []*cloudformation.Tag

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
	return s.SubmitWithOptions(ctx, app, SubmitOptions{
		Wait: s.Wait,
	})
}

// SubmitOptions are options provided to SubmitWithOptions.
type SubmitOptions struct {
	// If true, waits to for the stack to complete the create/update
	// successfully.
	Wait bool

	// When true, does not make any changes to DNS. This is only used when
	// migrating to this scheduler
	NoDNS bool
}

// SubmitWithOptions submits (or updates) the CloudFormation stack for the app.
func (s *Scheduler) SubmitWithOptions(ctx context.Context, app *scheduler.App, opts SubmitOptions) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	err = s.submit(ctx, tx, app, opts)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// Submit creates (or updates) the CloudFormation stack for the app.
func (s *Scheduler) submit(ctx context.Context, tx *sql.Tx, app *scheduler.App, opts SubmitOptions) error {
	wait := opts.Wait

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

	key := fmt.Sprintf("%s/%s/%x", app.Name, app.ID, sha1.Sum(buf.Bytes()))

	_, err = s.s3.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(s.Bucket),
		Key:         aws.String(fmt.Sprintf("/%s", key)),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("error uploading stack template to s3: %v", err)
	}

	url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.Bucket, key)

	tags := append(s.Tags,
		&cloudformation.Tag{Key: aws.String("empire.app.id"), Value: aws.String(app.ID)},
		&cloudformation.Tag{Key: aws.String("empire.app.name"), Value: aws.String(app.Name)},
	)

	// Build parameters for the stack.
	var parameters []*cloudformation.Parameter
	for _, p := range app.Processes {
		parameters = append(parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(scaleParameter(p.Type)),
			ParameterValue: aws.String(fmt.Sprintf("%d", p.Instances)),
		})
	}
	parameters = append(parameters, &cloudformation.Parameter{
		ParameterKey:   aws.String("DNS"),
		ParameterValue: aws.String(fmt.Sprintf("%t", !opts.NoDNS)),
	})

	desc, err := s.cloudformation.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err, ok := err.(awserr.Error); ok && err.Message() == fmt.Sprintf("Stack with id %s does not exist", stackName) {
		if _, err := s.cloudformation.CreateStack(&cloudformation.CreateStackInput{
			StackName:   aws.String(stackName),
			TemplateURL: aws.String(url),
			Tags:        tags,
			Parameters:  parameters,
		}); err != nil {
			return fmt.Errorf("error creating stack: %v", err)
		}

		if wait {
			if err := s.cloudformation.WaitUntilStackCreateComplete(&cloudformation.DescribeStacksInput{
				StackName: aws.String(stackName),
			}); err != nil {
				return err
			}
		}
	} else if err == nil {
		if _, err := s.updateStack(desc.Stacks[0], &cloudformation.UpdateStackInput{
			StackName:   aws.String(stackName),
			TemplateURL: aws.String(url),
			Parameters:  parameters,
			// TODO: Update Go client
			// Tags:         tags,
		}, wait); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("error describing stack: %v", err)
	}

	return nil
}

// updateStack performs a stack update, but waits for the stack to enter a
// stable state before starting.
//
// TODO: Timeout?
func (s *Scheduler) updateStack(stack *cloudformation.Stack, input *cloudformation.UpdateStackInput, wait bool) (*cloudformation.UpdateStackOutput, error) {
	status := *stack.StackStatus
	stackName := input.StackName

	// The parameters that the stack defines. We need to make sure that we
	// provide all parameters in the update (lame).
	definedParams := make(map[string]bool)
	for _, p := range stack.Parameters {
		definedParams[*p.ParameterKey] = true
	}

	// The parameters that are provided in this update.
	providedParams := make(map[string]bool)
	for _, p := range input.Parameters {
		providedParams[*p.ParameterKey] = true
	}

	// Fill in any parameters that weren't provided with their default
	// value.
	for k := range definedParams {
		if !providedParams[k] {
			input.Parameters = append(input.Parameters, &cloudformation.Parameter{
				ParameterKey:     aws.String(k),
				UsePreviousValue: aws.Bool(true),
			})
		}
	}

	// If there's currently an update happening, wait for it to
	// complete.
	if strings.Contains(status, "IN_PROGRESS") {
		if strings.Contains(status, "CREATE") {
			if err := s.cloudformation.WaitUntilStackCreateComplete(&cloudformation.DescribeStacksInput{
				StackName: stackName,
			}); err != nil {
				return nil, err
			}
		} else if strings.Contains(status, "UPDATE") {
			if err := s.cloudformation.WaitUntilStackUpdateComplete(&cloudformation.DescribeStacksInput{
				StackName: stackName,
			}); err != nil {
				return nil, err
			}
		}
	}

	resp, err := s.cloudformation.UpdateStack(input)
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			if err.Code() == "ValidationError" && err.Message() == "No updates are to be performed." {
				return resp, nil
			}
		}

		return resp, fmt.Errorf("error updating stack: %v", err)
	}

	if wait {
		if err := s.cloudformation.WaitUntilStackUpdateComplete(&cloudformation.DescribeStacksInput{
			StackName: stackName,
		}); err != nil {
			return resp, err
		}
	}

	return resp, nil
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

// Remove removes the CloudFormation stack for the given app, if it exists.
func (s *Scheduler) remove(_ context.Context, tx *sql.Tx, appID string) error {
	stackName, err := s.stackName(appID)

	// if there's no stack entry in the db for this app, nothing to remove
	if err == errNoStack {
		return nil
	}
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM stacks WHERE app_id = $1`, appID)
	if err != nil {
		return err
	}

	_, err = s.cloudformation.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err, ok := err.(awserr.Error); ok && err.Message() == fmt.Sprintf("Stack with id %s does not exist", stackName) {
		return nil
	}

	if _, err := s.cloudformation.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(stackName),
	}); err != nil {
		return fmt.Errorf("error deleting stack: %v", err)
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
	for process, serviceArn := range services {
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
			return nil, fmt.Errorf("error listing tasks for %s: %v", process, err)
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
		return nil, fmt.Errorf("error listing tasks started by %s: %v", app, err)
	}

	var tasks []*ecs.Task
	for _, chunk := range chunkStrings(arns, MaxDescribeTasks) {
		resp, err := s.ecs.DescribeTasks(&ecs.DescribeTasksInput{
			Cluster: aws.String(s.Cluster),
			Tasks:   chunk,
		})
		if err != nil {
			return nil, fmt.Errorf("error describing %d tasks: %v", len(chunk), err)
		}

		tasks = append(tasks, resp.Tasks...)
	}

	return tasks, nil
}

// Services returns a map that maps the name of the process (e.g. web) to the
// ARN of the associated ECS service.
func (s *Scheduler) Services(appID string) (map[string]string, error) {
	stackName, err := s.stackName(appID)
	if err != nil {
		return nil, err
	}

	resp, err := s.cloudformation.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return nil, fmt.Errorf("error describing stack: %v", err)
	}

	stack := resp.Stacks[0]

	var output *cloudformation.Output
	for _, o := range stack.Outputs {
		if *o.OutputKey == servicesOutput {
			output = o
		}
	}
	if output == nil {
		// Nothing to do but wait until the outputs are set.
		if *stack.StackStatus == cloudformation.StackStatusCreateInProgress {
			return nil, nil
		}

		return nil, fmt.Errorf("stack didn't provide a \"%s\" output key", servicesOutput)
	}

	return extractServices(*output.OutputValue), nil
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
	stackName, err := s.stackName(appID)
	if err != nil {
		return err
	}

	desc, err := s.cloudformation.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return err
	}

	_, err = s.updateStack(desc.Stacks[0], &cloudformation.UpdateStackInput{
		StackName:           aws.String(stackName),
		UsePreviousTemplate: aws.Bool(true),
		Parameters: []*cloudformation.Parameter{
			{
				ParameterKey:   aws.String(scaleParameter(process)),
				ParameterValue: aws.String(fmt.Sprintf("%d", instances)),
			},
		},
	}, s.Wait)

	return err
}

// Run registers a TaskDefinition for the process, and calls RunTask.
func (m *Scheduler) Run(ctx context.Context, app *scheduler.App, process *scheduler.Process, in io.Reader, out io.Writer) error {
	if out != nil {
		return errors.New("running an attached process is not implemented by the ECS manager.")
	}

	t, ok := m.Template.(interface {
		ContainerDefinition(*scheduler.App, *scheduler.Process) *ecs.ContainerDefinition
	})
	if !ok {
		return errors.New("provided template can't generate a container definition for this process")
	}

	resp, err := m.ecs.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
		Family: aws.String(fmt.Sprintf("%s--%s", app.ID, process.Type)),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			t.ContainerDefinition(app, process),
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

// extractServices extracts a map that maps the process name to the ARN of the
// associated ECS service.
func extractServices(value string) map[string]string {
	services := make(map[string]string)
	pairs := strings.Split(value, ",")

	for _, p := range pairs {
		parts := strings.Split(p, "=")
		services[parts[0]] = parts[1]
	}

	return services
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

// chunkStrings slices a slice of string pointers in equal length chunks, with
// the last slice being the leftovers.
func chunkStrings(s []*string, size int) [][]*string {
	var chunks [][]*string
	for len(s) > 0 {
		end := size
		if len(s) < size {
			end = len(s)
		}

		chunks = append(chunks, s[0:end])
		s = s[end:]
	}
	return chunks
}

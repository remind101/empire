// Package cloudformation implements the Scheduler interface for ECS by using
// CloudFormation to provision and update resources.
package cloudformation

import (
	"bytes"
	"crypto/sha1"
	"database/sql"
	"errors"
	"fmt"
	"hash/crc32"
	"html/template"
	"io"
	"strings"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/remind101/empire/pkg/arn"
	"github.com/remind101/empire/pkg/bytesize"
	pglock "github.com/remind101/empire/pkg/pg/lock"
	"github.com/remind101/empire/scheduler"
	"golang.org/x/net/context"
)

// newUUID returns a new UUID. Set to a var so we can stub it out in tests.
var newUUID = uuid.New

// The name of the output key where process names are mapped to ECS services.
// This output is expected to be a comma delimited list of `process=servicearn`
// values.
const servicesOutput = "Services"

const deploymentsOutput = "Deployments"

// Parameter used to trigger a restart of the application.
const restartParameter = "RestartKey"

// Variables to control stack update locking.
var (
	// This controls how long a pending stack update has to wait in the queue before
	// it gives up.
	lockTimeout = 10 * time.Minute

	// Controls how long we'll wait to obtain the stack update lock before we
	// consider the update to be asynchronous.
	lockWait = 2 * time.Second

	// Controls the maximum amount of time we'll wait for a stack operation to
	// complete before releasing the lock.
	stackOperationTimeout = 10 * time.Minute
)

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
	ValidateTemplate(*cloudformation.ValidateTemplateInput) (*cloudformation.ValidateTemplateOutput, error)
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
	DescribeServices(*ecs.DescribeServicesInput) (*ecs.DescribeServicesOutput, error)
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

	after func(time.Duration) <-chan time.Time
}

// NewScheduler returns a new Scheduler instance.
func NewScheduler(db *sql.DB, config client.ConfigProvider) *Scheduler {
	return &Scheduler{
		cloudformation: cloudformation.New(config),
		ecs:            ecs.New(config),
		s3:             s3.New(config),
		db:             db,
		after:          time.After,
	}
}

// Submit creates (or updates) the CloudFormation stack for the app.
func (s *Scheduler) Submit(ctx context.Context, app *scheduler.App, ss scheduler.StatusStream) error {
	return s.SubmitWithOptions(ctx, app, ss, SubmitOptions{})
}

// SubmitOptions are options provided to SubmitWithOptions.
type SubmitOptions struct {
	// When true, does not make any changes to DNS. This is only used when
	// migrating to this scheduler
	NoDNS *bool
}

// SubmitWithOptions submits (or updates) the CloudFormation stack for the app.
func (s *Scheduler) SubmitWithOptions(ctx context.Context, app *scheduler.App, ss scheduler.StatusStream, opts SubmitOptions) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	err = s.submit(ctx, tx, app, ss, opts)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Submit creates (or updates) the CloudFormation stack for the app.
func (s *Scheduler) submit(ctx context.Context, tx *sql.Tx, app *scheduler.App, ss scheduler.StatusStream, opts SubmitOptions) error {
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

	t, err := s.createTemplate(ctx, app)
	if err != nil {
		return err
	}

	scheduler.Publish(ctx, ss, fmt.Sprintf("Created cloudformation template: %v", *t.URL))
	tags := append(s.Tags,
		&cloudformation.Tag{Key: aws.String("empire.app.id"), Value: aws.String(app.ID)},
		&cloudformation.Tag{Key: aws.String("empire.app.name"), Value: aws.String(app.Name)},
	)

	// Build parameters for the stack.
	parameters := []*cloudformation.Parameter{
		// FIXME: Remove this in favor of a Restart method.
		{
			ParameterKey:   aws.String(restartParameter),
			ParameterValue: aws.String(newUUID()),
		},
	}

	if opts.NoDNS != nil {
		parameters = append(parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String("DNS"),
			ParameterValue: aws.String(fmt.Sprintf("%t", *opts.NoDNS)),
		})
	}

	for _, p := range app.Processes {
		parameters = append(parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(scaleParameter(p.Type)),
			ParameterValue: aws.String(fmt.Sprintf("%d", p.Instances)),
		})
	}

	output := make(chan stackOperationOutput, 1)
	_, err = s.cloudformation.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err, ok := err.(awserr.Error); ok && err.Message() == fmt.Sprintf("Stack with id %s does not exist", stackName) {
		if err := s.createStack(ctx, &createStackInput{
			StackName:  aws.String(stackName),
			Template:   t,
			Tags:       tags,
			Parameters: parameters,
		}, output, ss); err != nil {
			return fmt.Errorf("error creating stack: %v", err)
		}
	} else if err == nil {
		if err := s.updateStack(ctx, &updateStackInput{
			StackName:  aws.String(stackName),
			Template:   t,
			Parameters: parameters,
			// TODO: Update Go client
			// Tags:         tags,
		}, output, ss); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("error describing stack: %v", err)
	}

	if ss != nil {
		o := <-output
		if o.Err != nil {
			return o.Err
		}
		// indicates that the stack operation was cancelled, not an error, but
		// we don't need to wait.
		if o.Stack == nil {
			return nil
		}
		done := make(chan error, 1)
		s.waitForDeploymentToStabilize(ctx, o.Stack, ss, done)
		if err = <-done; err != nil {
			return err
		}
	}
	return nil
}

func (s *Scheduler) waitForDeploymentToStabilize(ctx context.Context, stack *cloudformation.Stack, ss scheduler.StatusStream, done chan error) {
	deployments := s.output(stack, deploymentsOutput)
	services := s.output(stack, servicesOutput)
	if deployments == nil {
		done <- fmt.Errorf("deployments output missing from stack")
		return
	}
	if services == nil {
		done <- fmt.Errorf("services output missing from stack")
		return
	}

	arns := extractProcessData(*services.OutputValue)
	deploymentIDs := extractProcessData(*deployments.OutputValue)

	if len(arns) == 0 {
		done <- fmt.Errorf("no services found in output")
		return
	}
	if len(deploymentIDs) == 0 {
		done <- fmt.Errorf("no deploymentIDs found in output")
		return
	}

	var serviceIDs []*string
	arnToDeploymentID := make(map[string]string)
	arnToProcessName := make(map[string]string)
	for p, a := range arns {
		id, err := arn.ResourceID(a)
		if err != nil {
			done <- fmt.Errorf("error retrieving id from arn: %v", err)
			return
		}
		deploymentID, ok := deploymentIDs[p]
		if !ok {
			done <- fmt.Errorf("deployment id not found for process: %v", p)
			return
		}

		arnToDeploymentID[a] = deploymentID
		arnToProcessName[a] = p
		serviceIDs = append(serviceIDs, aws.String(id))
	}

	dedupe := make(map[string]bool)
	check := func() bool {
		status, err := s.ecs.DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  aws.String(s.Cluster),
			Services: serviceIDs,
		})
		if err != nil {
			done <- fmt.Errorf("error describing services: %v", err)
			return true
		}

		var completed []bool
		for _, service := range status.Services {
			deploymentID, ok := arnToDeploymentID[*service.ServiceArn]
			if !ok {
				done <- fmt.Errorf("missing arn for deploymentID: %s", deploymentID)
				return true
			}

			name, ok := arnToProcessName[*service.ServiceArn]
			if !ok {
				done <- fmt.Errorf("missing process name for arn: %s", *service.ServiceArn)
				return true
			}

			primary := false
			stable := len(service.Deployments) == 1
			for _, deployment := range service.Deployments {
				if *deployment.Id == deploymentID {
					primary = *deployment.Status == "PRIMARY"
				}
			}

			var status string
			if primary && stable {
				status = "stable"
			} else if primary {
				status = "in progress"
			} else {
				status = "superseded by newer release"
			}
			msg := fmt.Sprintf("Deployment %s for %s", status, name)
			d := fmt.Sprintf("%s:%s", *service.ServiceArn, msg)
			if _, ok := dedupe[d]; !ok {
				scheduler.Publish(ctx, ss, msg)
				dedupe[d] = true
			}
			complete := !primary || primary && stable
			if complete {
				completed = append(completed, complete)
			}
		}
		return len(completed) == len(status.Services)
	}

	completed := false
	for !completed {
		completed = check()
		if !completed {
			<-s.after(2 * time.Second)
		}
	}
	done <- nil
}

// createTemplate takes a scheduler.App, and returns a validated cloudformation
// template.
func (s *Scheduler) createTemplate(ctx context.Context, app *scheduler.App) (*cloudformationTemplate, error) {
	buf := new(bytes.Buffer)
	if err := s.Template.Execute(buf, app); err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s/%s/%x", app.Name, app.ID, sha1.Sum(buf.Bytes()))
	url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.Bucket, key)

	if _, err := s.s3.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(s.Bucket),
		Key:         aws.String(fmt.Sprintf("/%s", key)),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("application/json"),
	}); err != nil {
		return nil, fmt.Errorf("error uploading stack template to s3: %v", err)
	}

	resp, err := s.cloudformation.ValidateTemplate(&cloudformation.ValidateTemplateInput{
		TemplateURL: aws.String(url),
	})
	if err != nil {
		return nil, &templateValidationError{templateURL: url, err: err, templateBody: buf}
	}

	return &cloudformationTemplate{
		URL:        aws.String(url),
		Parameters: resp.Parameters,
	}, nil
}

// cloudformationTemplate represents a validated CloudFormation template.
type cloudformationTemplate struct {
	URL        *string
	Parameters []*cloudformation.TemplateParameter
}

// createStackInput are options provided to createStack.
type createStackInput struct {
	StackName  *string
	Parameters []*cloudformation.Parameter
	Tags       []*cloudformation.Tag
	Template   *cloudformationTemplate
}

// createStack creates a new CloudFormation stack with the given input. This
// function returns as soon as the stack creation has been submitted. It does
// not wait for the stack creation to complete.
func (s *Scheduler) createStack(ctx context.Context, input *createStackInput, output chan stackOperationOutput, ss scheduler.StatusStream) error {
	waiter := func(input *cloudformation.DescribeStacksInput) error {
		scheduler.Publish(ctx, ss, "Creating stack")
		err := s.cloudformation.WaitUntilStackCreateComplete(input)
		if err == nil {
			scheduler.Publish(ctx, ss, "Stack created")
		}
		return err
	}

	submitted := make(chan error)
	fn := func() error {
		_, err := s.cloudformation.CreateStack(&cloudformation.CreateStackInput{
			StackName:   input.StackName,
			TemplateURL: input.Template.URL,
			Tags:        input.Tags,
			Parameters:  input.Parameters,
		})
		submitted <- err
		return err
	}
	go func() {
		output <- s.performStackOperation(ctx, *input.StackName, fn, waiter, ss)
	}()

	return <-submitted
}

// updateStackInput are options provided to update a stack.
type updateStackInput struct {
	StackName  *string
	Parameters []*cloudformation.Parameter
	Template   *cloudformationTemplate
}

// updateStack updates an existing CloudFormation stack with the given input.
// If there are no other active updates, this function returns as soon as the
// stack update has been submitted. If there are other updates, the function
// returns after `lockTimeout` and the update continues in the background.
func (s *Scheduler) updateStack(ctx context.Context, input *updateStackInput, output chan stackOperationOutput, ss scheduler.StatusStream) error {
	waiter := func(input *cloudformation.DescribeStacksInput) error {
		scheduler.Publish(ctx, ss, "Waiting for stack update to complete")
		err := s.cloudformation.WaitUntilStackUpdateComplete(input)
		if err == nil {
			scheduler.Publish(ctx, ss, "Stack update complete")
		}
		return err
	}

	locked := make(chan struct{})
	submitted := make(chan error, 1)
	fn := func() error {
		close(locked)
		err := s.executeStackUpdate(input)
		if err == nil {
			scheduler.Publish(ctx, ss, "Stack update submitted")
		}
		submitted <- err
		return err
	}

	go func() {
		output <- s.performStackOperation(ctx, *input.StackName, fn, waiter, ss)
	}()

	var err error
	select {
	case <-s.after(lockWait):
		scheduler.Publish(ctx, ss, "Waiting for existing stack operation to complete")
		// FIXME: At this point, we don't want to affect UX by waiting
		// around, so we return. But, if the stack update times out, or
		// there's an error, that information is essentially silenced.
		return nil
	case <-locked:
		// if a lock is obtained within the time frame, we might as well
		// just wait for the update to get submitted.
		err = <-submitted
	}

	return err
}

type stackOperationOutput struct {
	Err   error
	Stack *cloudformation.Stack
}

// performStackOperation encapsulates the process of obtaining the stack
// operation lock, performing the stack operation, waiting for it to complete,
// then unlocking the stack operation lock.
//
// * If there are no operations currently in progress, the stack operation will execute.
// * If there is a currently active stack operation, this operation will wait
//   until the other stack operation has completed.
// * If there is another pending stack operation, it will be replaced by the new
//   update.
func (s *Scheduler) performStackOperation(ctx context.Context, stackName string, fn func() error, waiter func(*cloudformation.DescribeStacksInput) error, ss scheduler.StatusStream) stackOperationOutput {
	var output stackOperationOutput
	l, err := newAdvisoryLock(s.db, stackName)
	if err != nil {
		output.Err = err
		return output
	}

	// Cancel any pending stack operation, since this one obsoletes older
	// operations.
	if err := l.CancelPending(); err != nil {
		output.Err = fmt.Errorf("error canceling pending stack operation: %v", err)
		return output
	}

	if err := l.Lock(); err != nil {
		// This will happen when a newer stack update obsoletes
		// this one. We simply return nil.
		//
		// TODO: Should we return an error here?
		if err == pglock.Canceled {
			scheduler.Publish(ctx, ss, "Operation superseded by newer release")
			return output
		}
		output.Err = fmt.Errorf("error obtaining stack operation lock %s: %v", stackName, err)
		return output
	}
	defer l.Unlock()

	// Once the lock has been obtained, let's perform the stack operation.
	if err := fn(); err != nil {
		output.Err = err
		return output
	}

	wait := func() error {
		return waiter(&cloudformation.DescribeStacksInput{
			StackName: aws.String(stackName),
		})
	}

	// Wait until this stack operation has completed. The lock will be
	// unlocked when this function returns.
	if err := s.waitUntilStackOperationComplete(l, wait); err != nil {
		output.Err = err
		return output
	}

	output.Stack, output.Err = s.stack(&stackName)
	return output
}

// waitUntilStackOperationComplete waits until wait returns, or it times out.
func (s *Scheduler) waitUntilStackOperationComplete(lock *pglock.AdvisoryLock, wait func() error) error {
	errCh := make(chan error)
	go func() { errCh <- wait() }()

	var err error
	select {
	case <-s.after(stackOperationTimeout):
		err = errors.New("timed out waiting for stack operation to complete")
	case err = <-errCh:
	}

	return err
}

// executeStackUpdate performs a stack update.
func (s *Scheduler) executeStackUpdate(input *updateStackInput) error {
	stack, err := s.stack(input.StackName)
	if err != nil {
		return err
	}

	i := &cloudformation.UpdateStackInput{
		StackName:  input.StackName,
		Parameters: updateParameters(input.Parameters, stack, input.Template),
	}
	if input.Template != nil {
		i.TemplateURL = input.Template.URL
	} else {
		i.UsePreviousTemplate = aws.Bool(true)
	}

	_, err = s.cloudformation.UpdateStack(i)
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			if err.Code() == "ValidationError" && err.Message() == "No updates are to be performed." {
				return nil
			}
		}

		return fmt.Errorf("error updating stack: %v", err)
	}

	return nil
}

// stack returns the cloudformation.Stack for the given stack name.
func (s *Scheduler) stack(stackName *string) (*cloudformation.Stack, error) {
	resp, err := s.cloudformation.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: stackName,
	})
	if err != nil {
		return nil, err
	}
	return resp.Stacks[0], nil
}

// output returns the cloudformation.Output that matches the given key.
func (s *Scheduler) output(stack *cloudformation.Stack, key string) (output *cloudformation.Output) {
	for _, o := range stack.Outputs {
		if *o.OutputKey == key {
			output = o
		}
	}
	return
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

	stack, err := s.stack(aws.String(stackName))
	if err != nil {
		return nil, fmt.Errorf("error describing stack: %v", err)
	}

	output := s.output(stack, servicesOutput)
	if output == nil {
		// Nothing to do but wait until the outputs are set.
		if *stack.StackStatus == cloudformation.StackStatusCreateInProgress {
			return nil, nil
		}

		return nil, fmt.Errorf("stack didn't provide a \"%s\" output key", servicesOutput)
	}

	return extractProcessData(*output.OutputValue), nil
}

// Stop stops the given ECS task.
func (s *Scheduler) Stop(ctx context.Context, taskID string) error {
	_, err := s.ecs.StopTask(&ecs.StopTaskInput{
		Cluster: aws.String(s.Cluster),
		Task:    aws.String(taskID),
	})
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

// extractProcessData extracts a map that maps the process name to some
// corresponding value.
func extractProcessData(value string) map[string]string {
	data := make(map[string]string)
	pairs := strings.Split(value, ",")

	for _, p := range pairs {
		parts := strings.Split(p, "=")
		data[parts[0]] = parts[1]
	}

	return data
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

// updateParameters returns the parameters that should be provided in an
// UpdateStack operation.
func updateParameters(provided []*cloudformation.Parameter, stack *cloudformation.Stack, template *cloudformationTemplate) []*cloudformation.Parameter {
	parameters := provided[:]

	// This tracks the names of the parameters that have pre-existing values
	// on the stack.
	existingParams := make(map[string]bool)
	for _, p := range stack.Parameters {
		existingParams[*p.ParameterKey] = true
	}

	// These are the parameters that can be set for the stack. If a template
	// is provided, then these are the parameters defined in the template.
	// If no template is provided, then these are the parameters that the
	// stack provides.
	settableParams := make(map[string]bool)
	if template != nil {
		for _, p := range template.Parameters {
			settableParams[*p.ParameterKey] = true
		}
	} else {
		settableParams = existingParams
	}

	// The parameters that are provided in this update.
	providedParams := make(map[string]bool)
	for _, p := range parameters {
		providedParams[*p.ParameterKey] = true
	}

	// Fill in any parameters that weren't provided with their previous
	// value, if available
	for k := range settableParams {
		notProvided := !providedParams[k]
		hasExistingValue := existingParams[k]

		// If the parameter hasn't been provided with an explicit value,
		// and the stack has this parameter set, we'll use the previous
		// value. Not doing this would result in the parameters
		// `Default` getting used.
		if notProvided && hasExistingValue {
			parameters = append(parameters, &cloudformation.Parameter{
				ParameterKey:     aws.String(k),
				UsePreviousValue: aws.Bool(true),
			})
		}
	}

	return parameters
}

// stackLackKey returns the key to use when obtaining an advisory lock for a
// CloudFormation stack.
func stackLockKey(stackName string) uint32 {
	return crc32.ChecksumIEEE([]byte(fmt.Sprintf("stack_%s", stackName)))
}

// newAdvsiroyLock returns a new AdvisoryLock suitable for obtaining a lock to
// perform the stack update.
func newAdvisoryLock(db *sql.DB, stackName string) (*pglock.AdvisoryLock, error) {
	l, err := pglock.NewAdvisoryLock(db, stackLockKey(stackName))
	if err != nil {
		return l, err
	}
	l.LockTimeout = lockTimeout
	l.Context = fmt.Sprintf("stack %s", stackName)
	return l, nil
}

// templateValidationError wraps an error from ValidateTemplate to provide more
// information.
type templateValidationError struct {
	templateURL  string
	templateBody *bytes.Buffer
	err          error
}

func (e *templateValidationError) Error() string {
	t := `TemplateValidationError:
  Template URL: %s
  Template Size: %d bytes
  Error: %v`
	return fmt.Sprintf(t, e.templateURL, e.templateBody.Len(), e.err)
}

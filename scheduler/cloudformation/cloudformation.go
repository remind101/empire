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
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/s3"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/pkg/arn"
	"github.com/remind101/empire/pkg/bytesize"
	pglock "github.com/remind101/empire/pkg/pg/lock"
	"github.com/remind101/empire/scheduler"
	"github.com/remind101/empire/stats"
	"github.com/remind101/pkg/logger"
	"golang.org/x/net/context"
)

// newTimestamp returns the current time in seconds since epoch. Set to a var
// so we can stub it out in tests.
var newTimestamp = func() string { return strconv.FormatInt(time.Now().Unix(), 10) }

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
	stackOperationTimeout = 1 * time.Hour

	// Controls how long we'll wait between requests to describe services when
	// waiting for a deployment to stabilize
	pollServicesWait = 20 * time.Second
)

// CloudFormation limits
//
// See http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cloudformation-limits.html
const (
	MaxTemplateSize = 460800 // bytes
)

// ECS limits
const (
	MaxDescribeTasks              = 100
	MaxDescribeServices           = 10
	MaxDescribeContainerInstances = 100
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
	DescribeContainerInstances(*ecs.DescribeContainerInstancesInput) (*ecs.DescribeContainerInstancesOutput, error)
	WaitUntilTasksNotPending(*ecs.DescribeTasksInput) error
}

// s3Client duck types the s3.S3 interface that we use.
type s3Client interface {
	PutObject(*s3.PutObjectInput) (*s3.PutObjectOutput, error)
}

// ec2Client duck types the ec2.EC2 interface that we use.
type ec2Client interface {
	DescribeInstances(*ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error)
}

// DockerClient defines the interface we use when using Docker to connect to a
// running ECS task.
type DockerClient interface {
	ListContainers(docker.ListContainersOptions) ([]docker.APIContainers, error)
	AttachToContainer(docker.AttachToContainerOptions) error
}

// Data handed to template generators.
type TemplateData struct {
	*scheduler.App
	StackTags []*cloudformation.Tag
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

	// NewDockerClient is used to open a new Docker connection to an ec2
	// instance.
	NewDockerClient func(*ec2.Instance) (DockerClient, error)

	// CloudFormation client for creating stacks.
	cloudformation cloudformationClient

	// ECS client for performing ECS API calls.
	ecs ecsClient

	// S3 client to upload templates to s3.
	s3 s3Client

	// EC2 client to interact with EC2.
	ec2 ec2Client

	db *sql.DB

	after func(time.Duration) <-chan time.Time
}

// NewScheduler returns a new Scheduler instance.
func NewScheduler(db *sql.DB, config client.ConfigProvider) *Scheduler {
	return &Scheduler{
		cloudformation: cloudformation.New(config),
		ecs:            ecsWithCaching(&ECS{ecs.New(config)}),
		s3:             s3.New(config),
		ec2:            ec2.New(config),
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

func (s *Scheduler) Restart(ctx context.Context, app *scheduler.App, ss scheduler.StatusStream) error {
	stackName, err := s.stackName(app.ID)
	if err != nil {
		return err
	}
	output := make(chan stackOperationOutput, 1)
	if err := s.updateStack(ctx, &updateStackInput{
		StackName: aws.String(stackName),
		Parameters: []*cloudformation.Parameter{
			{
				ParameterKey:   aws.String(restartParameter),
				ParameterValue: aws.String(newTimestamp()),
			},
		},
	}, output, ss); err != nil {
		return err
	}

	if ss != nil {
		o := <-output
		if o.err != nil || o.stack == nil {
			return o.err
		}
		// TODO: Wait for services to stabilize?
	}

	return nil
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

	stackTags := append(s.Tags, tagsFromLabels(app.Labels)...)

	t, err := s.createTemplate(ctx, app, stackTags)
	if err != nil {
		return err
	}

	stats.Histogram(ctx, "scheduler.cloudformation.template_size", float32(t.Size), 1.0, []string{
		fmt.Sprintf("stack:%s", stackName),
	})

	scheduler.Publish(ctx, ss, fmt.Sprintf("Created cloudformation template: %v (%d/%d bytes)", *t.URL, t.Size, MaxTemplateSize))

	var parameters []*cloudformation.Parameter
	if opts.NoDNS != nil {
		parameters = append(parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String("DNS"),
			ParameterValue: aws.String(fmt.Sprintf("%t", !*opts.NoDNS)),
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
			Tags:       stackTags,
			Parameters: parameters,
		}, output, ss); err != nil {
			return fmt.Errorf("error creating stack: %v", err)
		}
	} else if err == nil {
		if err := s.updateStack(ctx, &updateStackInput{
			StackName:  aws.String(stackName),
			Template:   t,
			Parameters: parameters,
			Tags:       stackTags,
		}, output, ss); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("error describing stack: %v", err)
	}

	if ss != nil {
		o := <-output
		if o.err != nil || o.stack == nil {
			return o.err
		}
		if err := s.waitUntilStable(ctx, o.stack, ss); err != nil {
			logger.Warn(ctx, fmt.Sprintf("error waiting for submit to stabilize: %v", err))
		}
	}
	return nil
}

func (s *Scheduler) waitUntilStable(ctx context.Context, stack *cloudformation.Stack, ss scheduler.StatusStream) error {
	deployments, err := deploymentsToWatch(stack)
	if err != nil {
		return err
	}
	deploymentStatuses := s.waitForDeploymentsToStabilize(ctx, deployments)
	for status := range deploymentStatuses {
		scheduler.Publish(ctx, ss, fmt.Sprintf("Service %s became %s", status.deployment.process, status))
	}
	// TODO publish notification to empire
	return nil
}

type deploymentStatus struct {
	deployment *ecsDeployment
	status     string
}

func (d *deploymentStatus) String() string {
	return d.status
}

func (s *Scheduler) waitForDeploymentsToStabilize(ctx context.Context, deployments map[string]*ecsDeployment) <-chan *deploymentStatus {
	ch := make(chan *deploymentStatus)

	wait := func(deployments map[string]*ecsDeployment) (bool, error) {
		arns := make([]*string, 0, len(deployments))
		for arn := range deployments {
			arns = append(arns, aws.String(arn))
		}

		services, err := s.services(arns)
		if err != nil {
			return false, err
		}

		for _, service := range services {
			d, ok := deployments[*service.ServiceArn]
			if !ok {
				return false, fmt.Errorf("missing deployment for: %s", service.ServiceArn)
			}
			primary := false
			stable := len(service.Deployments) == 1
			for _, deployment := range service.Deployments {
				if *deployment.Id == d.ID {
					primary = *deployment.Status == "PRIMARY"
				}
			}

			if primary && stable {
				ch <- &deploymentStatus{d, "stable"}
				delete(deployments, *service.ServiceArn)
			} else if primary {
				// do nothing
			} else {
				ch <- &deploymentStatus{d, "inactive"}
				return false, nil
			}
		}
		return true, nil
	}

	go func(deployments map[string]*ecsDeployment) {
		keepWaiting := true
		var err error
		for keepWaiting && len(deployments) > 0 {
			keepWaiting, err = wait(deployments)
			if err != nil {
				logger.Warn(ctx, fmt.Sprintf("error waiting for services to stabilize: %v", err))
				break
			}
			if keepWaiting {
				<-s.after(pollServicesWait)
			}
		}
		close(ch)
	}(deployments)

	return ch
}

// createTemplate takes a scheduler.App, and returns a validated cloudformation
// template.
func (s *Scheduler) createTemplate(ctx context.Context, app *scheduler.App, stackTags []*cloudformation.Tag) (*cloudformationTemplate, error) {
	data := &TemplateData{
		App:       app,
		StackTags: stackTags,
	}

	buf := new(bytes.Buffer)
	if err := s.Template.Execute(buf, data); err != nil {
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

	t := &cloudformationTemplate{
		URL:  aws.String(url),
		Size: buf.Len(),
	}

	resp, err := s.cloudformation.ValidateTemplate(&cloudformation.ValidateTemplateInput{
		TemplateURL: aws.String(url),
	})
	if err != nil {
		return t, &templateValidationError{template: t, err: err}
	}

	t.Parameters = resp.Parameters

	return t, nil
}

// cloudformationTemplate represents a validated CloudFormation template.
type cloudformationTemplate struct {
	URL        *string
	Size       int
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
	waiter := s.waitFor(ctx, createStack, ss)

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
		stack, err := s.performStackOperation(ctx, *input.StackName, fn, waiter, ss)
		output <- stackOperationOutput{stack, err}
	}()

	return <-submitted
}

// updateStackInput are options provided to update a stack.
type updateStackInput struct {
	StackName  *string
	Parameters []*cloudformation.Parameter
	Template   *cloudformationTemplate
	Tags       []*cloudformation.Tag
}

// updateStack updates an existing CloudFormation stack with the given input.
// If there are no other active updates, this function returns as soon as the
// stack update has been submitted. If there are other updates, the function
// returns after `lockTimeout` and the update continues in the background.
func (s *Scheduler) updateStack(ctx context.Context, input *updateStackInput, output chan stackOperationOutput, ss scheduler.StatusStream) error {
	waiter := s.waitFor(ctx, updateStack, ss)

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
		stack, err := s.performStackOperation(ctx, *input.StackName, fn, waiter, ss)
		output <- stackOperationOutput{stack, err}
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
	stack *cloudformation.Stack
	err   error
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
func (s *Scheduler) performStackOperation(ctx context.Context, stackName string, fn func() error, waiter waitFunc, ss scheduler.StatusStream) (*cloudformation.Stack, error) {
	l, err := newAdvisoryLock(s.db, stackName)
	if err != nil {
		return nil, err
	}

	// Cancel any pending stack operation, since this one obsoletes older
	// operations.
	if err := l.CancelPending(); err != nil {
		return nil, fmt.Errorf("error canceling pending stack operation: %v", err)
	}

	if err := l.Lock(); err != nil {
		// This will happen when a newer stack update obsoletes
		// this one. We simply return nil.
		//
		// TODO: Should we return an error here?
		if err == pglock.Canceled {
			scheduler.Publish(ctx, ss, "Operation superseded by newer release")
			return nil, nil
		}
		return nil, fmt.Errorf("error obtaining stack operation lock %s: %v", stackName, err)
	}
	defer l.Unlock()

	// Once the lock has been obtained, let's perform the stack operation.
	if err := fn(); err != nil {
		return nil, err
	}

	wait := func() error {
		return waiter(&cloudformation.DescribeStacksInput{
			StackName: aws.String(stackName),
		})
	}

	// Wait until this stack operation has completed. The lock will be
	// unlocked when this function returns.
	if err := s.waitUntilStackOperationComplete(l, wait); err != nil {
		return nil, err
	}

	return s.stack(&stackName)
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
		Tags:       input.Tags,
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

	// Map from clusterARN to containerInstanceARN pointers to batch tasks from the same cluster
	clusterMap := make(map[string][]*string)

	for _, t := range tasks {
		k := *t.ClusterArn
		clusterMap[k] = append(clusterMap[k], t.ContainerInstanceArn)
	}

	// Map from containerInstanceARN to ec2-instance-id
	hostMap := make(map[string]string)

	for clusterArn, containerArnPtrs := range clusterMap {
		for _, chunk := range chunkStrings(containerArnPtrs, MaxDescribeContainerInstances) {
			resp, err := s.ecs.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
				Cluster:            aws.String(clusterArn),
				ContainerInstances: chunk,
			})
			if err != nil {
				return nil, err
			}
			for _, f := range resp.Failures {
				return nil, fmt.Errorf("error describing container instance %s: %s", aws.StringValue(f.Arn), aws.StringValue(f.Reason))
			}

			for _, ci := range resp.ContainerInstances {
				hostMap[aws.StringValue(ci.ContainerInstanceArn)] = aws.StringValue(ci.Ec2InstanceId)
			}
		}
	}

	for _, t := range tasks {
		taskDefinition := taskDefinitions[*t.TaskDefinitionArn]

		id, err := arn.ResourceID(*t.TaskArn)
		if err != nil {
			return instances, err
		}

		hostId := hostMap[*t.ContainerInstanceArn]

		p, err := taskDefinitionToProcess(taskDefinition)
		if err != nil {
			return instances, err
		}

		state := aws.StringValue(t.LastStatus)
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
			Host:      scheduler.Host{ID: hostId},
			UpdatedAt: updatedAt,
		})
	}

	return instances, nil
}

func (s *Scheduler) services(arns []*string) ([]*ecs.Service, error) {
	var services []*ecs.Service
	for _, chunk := range chunkStrings(arns, MaxDescribeServices) {
		resp, err := s.ecs.DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  aws.String(s.Cluster),
			Services: chunk,
		})
		if err != nil {
			return nil, fmt.Errorf("error describing %d services: %v", len(chunk), err)
		}
		services = append(services, resp.Services...)
	}
	return services, nil
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

	o := output(stack, servicesOutput)
	if o == nil {
		// Nothing to do but wait until the outputs are set.
		if *stack.StackStatus == cloudformation.StackStatusCreateInProgress {
			return nil, nil
		}

		return nil, fmt.Errorf("stack didn't provide a \"%s\" output key", servicesOutput)
	}

	return extractProcessData(*o.OutputValue), nil
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
	var attached bool
	if out != nil {
		attached = true
	}

	t, ok := m.Template.(interface {
		ContainerDefinition(*scheduler.App, *scheduler.Process) *ecs.ContainerDefinition
	})
	if !ok {
		return errors.New("provided template can't generate a container definition for this process")
	}

	containerDefinition := t.ContainerDefinition(app, process)
	if attached {
		if containerDefinition.DockerLabels == nil {
			containerDefinition.DockerLabels = make(map[string]*string)
		}
		// NOTE: Currently, this depends on a patched version of the
		// Amazon ECS Container Agent, since the official agent doesn't
		// provide a method to pass these down to the `CreateContainer`
		// call.
		containerDefinition.DockerLabels["docker.config.Tty"] = aws.String("true")
		containerDefinition.DockerLabels["docker.config.OpenStdin"] = aws.String("true")
	}

	resp, err := m.ecs.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
		Family:      aws.String(fmt.Sprintf("%s--%s", app.ID, process.Type)),
		TaskRoleArn: taskRoleArn(app),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			containerDefinition,
		},
	})
	if err != nil {
		return fmt.Errorf("error registering TaskDefinition: %v", err)
	}

	runResp, err := m.ecs.RunTask(&ecs.RunTaskInput{
		TaskDefinition: resp.TaskDefinition.TaskDefinitionArn,
		Cluster:        aws.String(m.Cluster),
		Count:          aws.Int64(1),
		StartedBy:      aws.String(app.ID),
	})
	if err != nil {
		return fmt.Errorf("error calling RunTask: %v", err)
	}

	for _, f := range runResp.Failures {
		return fmt.Errorf("error running task %s: %s", aws.StringValue(f.Arn), aws.StringValue(f.Reason))
	}

	task := runResp.Tasks[0]

	if attached {
		// Ensure that we atleast try to stop the task, after we detach
		// from the process. This ensures that we don't have zombie
		// one-off processes lying around.
		defer m.ecs.StopTask(&ecs.StopTaskInput{
			Cluster: task.ClusterArn,
			Task:    task.TaskArn,
		})

		if err := m.attach(ctx, task, in, out); err != nil {
			return err
		}
	}

	return nil
}

// attach attaches to the given ECS task.
func (m *Scheduler) attach(ctx context.Context, task *ecs.Task, in io.Reader, out io.Writer) error {
	defer tryClose(out)

	if a, _ := arn.Parse(aws.StringValue(task.TaskArn)); a != nil {
		// TODO: This should really go to a STDERR stream instead. Putting this
		// on stdout breaks redirection, but not including it leads to
		// confusion.
		fmt.Fprintf(out, "Attaching to %s...\r\n", a.Resource)
	}

	descContainerInstanceResp, err := m.ecs.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            task.ClusterArn,
		ContainerInstances: []*string{task.ContainerInstanceArn},
	})
	if err != nil {
		return fmt.Errorf("error describing container instance (%s): %v", aws.StringValue(task.ContainerInstanceArn), err)
	}

	containerInstance := descContainerInstanceResp.ContainerInstances[0]
	descInstanceResp, err := m.ec2.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{containerInstance.Ec2InstanceId},
	})
	if err != nil {
		return fmt.Errorf("error describing ec2 instance (%s): %v", aws.StringValue(containerInstance.Ec2InstanceId), err)
	}

	ec2Instance := descInstanceResp.Reservations[0].Instances[0]

	// Wait for the task to start running. It will stay in the
	// PENDING state while the container is being pulled.
	if err := m.ecs.WaitUntilTasksNotPending(&ecs.DescribeTasksInput{
		Cluster: task.ClusterArn,
		Tasks:   []*string{task.TaskArn},
	}); err != nil {
		return fmt.Errorf("error waiting for %s to transition from PENDING state: %s", aws.StringValue(task.TaskArn), err)
	}

	// Open a new connection to the Docker daemon on the EC2
	// instance where the task is running.
	d, err := m.NewDockerClient(ec2Instance)
	if err != nil {
		return fmt.Errorf("error connecting to docker daemon on %s: %v", aws.StringValue(ec2Instance.InstanceId), err)
	}

	// Find the container id for the ECS task.
	containers, err := d.ListContainers(docker.ListContainersOptions{
		All: true,
		Filters: map[string][]string{
			"label": []string{
				fmt.Sprintf("com.amazonaws.ecs.task-arn=%s", aws.StringValue(task.TaskArn)),
				//fmt.Sprintf("com.amazonaws.ecs.container-name", p)
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error listing containers for task: %v", err)
	}

	if len(containers) != 1 {
		return fmt.Errorf("unable to find container for %s running on %s", aws.StringValue(task.TaskArn), aws.StringValue(ec2Instance.InstanceId))
	}

	containerID := containers[0].ID

	if err := d.AttachToContainer(docker.AttachToContainerOptions{
		Container:    containerID,
		InputStream:  in,
		OutputStream: out,
		ErrorStream:  out,
		Logs:         true,
		Stream:       true,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  true,
	}); err != nil {
		return fmt.Errorf("error attaching to container (%s): %v", containerID, err)
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

type stackOperation string

const (
	createStack = "CreateStack"
	updateStack = "UpdateStack"
)

// waitFunc represents a function that can wait for a CloudFormation stack to
// reach a certain state.
type waitFunc func(*cloudformation.DescribeStacksInput) error

// waiter holds information about a waitFunc.
type waiter struct {
	startMessage, successMessage string

	wait func(cloudformationClient) waitFunc
}

// waiters maps a stack operation to the waiter that should be used to wait for
// it to complete.
var waiters = map[stackOperation]waiter{
	createStack: {
		"Creating stack",
		"Stack created",
		func(c cloudformationClient) waitFunc { return c.WaitUntilStackCreateComplete },
	},
	updateStack: {
		"Updating stack",
		"Stack updated",
		func(c cloudformationClient) waitFunc { return c.WaitUntilStackUpdateComplete },
	},
}

// waitFor returns a wait function that will wait for the given stack operation
// to complete, and sends status messages to the status stream, and also records
// metrics for how long the operation took.
func (s *Scheduler) waitFor(ctx context.Context, op stackOperation, ss scheduler.StatusStream) func(*cloudformation.DescribeStacksInput) error {
	waiter := waiters[op]
	wait := waiter.wait(s.cloudformation)

	return func(input *cloudformation.DescribeStacksInput) error {
		tags := []string{
			fmt.Sprintf("stack:%s", *input.StackName),
		}
		scheduler.Publish(ctx, ss, waiter.startMessage)
		start := time.Now()
		err := wait(input)
		stats.Timing(ctx, fmt.Sprintf("scheduler.cloudformation.%s", op), time.Since(start), 1.0, tags)
		if err == nil {
			scheduler.Publish(ctx, ss, waiter.successMessage)
		}
		return err
	}
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
			env[aws.StringValue(kvp.Name)] = aws.StringValue(kvp.Value)
		}
	}

	return &scheduler.Process{
		Type:        aws.StringValue(container.Name),
		Command:     command,
		Env:         env,
		CPUShares:   uint(*container.Cpu),
		MemoryLimit: uint(*container.Memory) * bytesize.MB,
		Nproc:       uint(softLimit(container.Ulimits, "nproc")),
	}, nil
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
	template *cloudformationTemplate
	err      error
}

func (e *templateValidationError) Error() string {
	t := `TemplateValidationError:
  Template URL: %s
  Template Size: %d bytes
  Error: %v`
	return fmt.Sprintf(t, *e.template.URL, e.template.Size, e.err)
}

// output returns the cloudformation.Output that matches the given key.
func output(stack *cloudformation.Stack, key string) (output *cloudformation.Output) {
	for _, o := range stack.Outputs {
		if *o.OutputKey == key {
			output = o
		}
	}
	return
}

type ecsDeployment struct {
	process string
	ID      string
}

// deploymentsToWatch returns an array of ecsDeployments for the given cloudformation stack
func deploymentsToWatch(stack *cloudformation.Stack) (map[string]*ecsDeployment, error) {
	deployments := output(stack, deploymentsOutput)
	services := output(stack, servicesOutput)
	if deployments == nil {
		return nil, fmt.Errorf("deployments output missing from stack")
	}
	if services == nil {
		return nil, fmt.Errorf("services output missing from stack")
	}

	arns := extractProcessData(*services.OutputValue)
	deploymentIDs := extractProcessData(*deployments.OutputValue)

	if len(arns) == 0 {
		return nil, fmt.Errorf("no services found in output")
	}
	if len(deploymentIDs) == 0 {
		return nil, fmt.Errorf("no deploymentIDs found in output")
	}

	ecsDeployments := make(map[string]*ecsDeployment)
	for p, a := range arns {
		deploymentID, ok := deploymentIDs[p]
		if !ok {
			return nil, fmt.Errorf("deployment id not found for process: %v", p)
		}
		ecsDeployments[a] = &ecsDeployment{
			process: p,
			ID:      deploymentID,
		}
	}
	return ecsDeployments, nil
}

func tryClose(w io.Writer) error {
	if w, ok := w.(io.Closer); ok {
		return w.Close()
	}

	return nil
}

// Package ecs implements an empire.TaskEngine that uses CloudFormation + ECS to
// run and list tasks.
package ecs

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/arn"
	"github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/pkg/constraints"
)

// DefaultStackNameTemplate is the default text/template for generating a
// CloudFormation stack name for an app.
var DefaultStackNameTemplate = template.Must(template.New("stack_name").Parse("{{.Name}}"))

const (
	// The name of the output key where process names are mapped to ECS services.
	// This output is expected to be a comma delimited list of `process=servicearn`
	// values.
	servicesOutput        = "Services"
	taskDefinitionsOutput = "TaskDefinitions"
)

// ECS limits
const (
	MaxDescribeTasks              = 100
	MaxDescribeServices           = 10
	MaxDescribeContainerInstances = 100
)

var ECSAttachedContainerEnvironmentVariables = []*ecs.KeyValuePair{
	{Name: aws.String("ECS_DOCKER_CONFIG_TTY"), Value: aws.String("true")},
	{Name: aws.String("ECS_DOCKER_CONFIG_OPEN_STDIN"), Value: aws.String("true")},
}

// cloudformationClient duck types the cloudformation.CloudFormation interface
// that we use.
type cloudformationClient interface {
	DescribeStacks(*cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error)
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

type TaskEngine struct {
	Cluster string

	// NewDockerClient is used to open a new Docker connection to an ec2
	// instance.
	NewDockerClient func(*ec2.Instance) (DockerClient, error)

	// A text/template that will generate the stack name for the app. This
	// template will be executed with a scheduler.App as it's data.
	StackNameTemplate *template.Template

	cloudformation cloudformationClient
	ecs            ecsClient
	ec2            ec2Client
}

func NewTaskEngine(config client.ConfigProvider) *TaskEngine {
	return &TaskEngine{
		cloudformation: cloudformation.New(config),
		ecs:            ecsWithCaching(&ECS{ecs.New(config)}),
		ec2:            ec2.New(config),
	}
}

func (s *TaskEngine) stackName(app *empire.App) (string, error) {
	t := s.StackNameTemplate
	if t == nil {
		t = DefaultStackNameTemplate
	}
	buf := new(bytes.Buffer)
	if err := t.Execute(buf, app); err != nil {
		return "", fmt.Errorf("error generating stack name: %v", err)
	}
	return buf.String(), nil
}

// Run registers a TaskDefinition for the process, and calls RunTask.
func (s *TaskEngine) Run(ctx context.Context, app *empire.App, stdio *empire.IO) error {
	stackName, err := s.stackName(app)
	if err != nil {
		return err
	}

	taskDefinitions, err := s.TaskDefinitions(stackName)
	if err != nil {
		return err
	}

	for name, p := range app.Formation {
		td, ok := taskDefinitions[name]
		if !ok {
			return fmt.Errorf("no task definition found for %q process", name)
		}

		var command []*string
		for _, e := range p.Command {
			command = append(command, aws.String(e))
		}

		var env []*ecs.KeyValuePair
		for k, v := range p.Environment {
			env = append(env, &ecs.KeyValuePair{
				Name:  aws.String(k),
				Value: aws.String(v),
			})
		}

		if stdio != nil {
			env = append(env, ECSAttachedContainerEnvironmentVariables...)
		}

		input := &ecs.RunTaskInput{
			TaskDefinition: aws.String(td),
			Cluster:        aws.String(s.Cluster),
			Count:          aws.Int64(1),
			StartedBy:      aws.String(stackName),
			Overrides: &ecs.TaskOverride{
				ContainerOverrides: []*ecs.ContainerOverride{
					{
						Name:        aws.String(name),
						Command:     command,
						Environment: env,
					},
				},
			},
		}

		runResp, err := s.ecs.RunTask(input)
		if err != nil {
			return fmt.Errorf("error calling RunTask: %v", err)
		}

		for _, f := range runResp.Failures {
			return fmt.Errorf("error running task %s: %s", aws.StringValue(f.Arn), aws.StringValue(f.Reason))
		}

		task := runResp.Tasks[0]

		if stdio != nil {

			// Ensure that we atleast try to stop the task, after we detach
			// from the process. This ensures that we don't have zombie
			// one-off processes lying around.
			defer s.ecs.StopTask(&ecs.StopTaskInput{
				Cluster: task.ClusterArn,
				Task:    task.TaskArn,
			})

			if err := s.attach(ctx, task, stdio); err != nil {
				return err
			}
		}
	}

	return nil
}

// attach attaches to the given ECS task.
func (m *TaskEngine) attach(ctx context.Context, task *ecs.Task, stdio *empire.IO) error {
	if a, _ := arn.Parse(aws.StringValue(task.TaskArn)); a != nil {
		fmt.Fprintf(stdio.Stderr, "Attaching to %s...\r\n", a.Resource)
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
		InputStream:  stdio.Stdin,
		OutputStream: stdio.Stdout,
		ErrorStream:  stdio.Stderr,
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

// Stop stops the given ECS task.
func (s *TaskEngine) Stop(ctx context.Context, taskID string) error {
	_, err := s.ecs.StopTask(&ecs.StopTaskInput{
		Cluster: aws.String(s.Cluster),
		Task:    aws.String(taskID),
	})
	return err
}

// Tasks returns all of the running tasks for this application.
func (s *TaskEngine) Tasks(ctx context.Context, app *empire.App) ([]*empire.Task, error) {
	stackName, err := s.stackName(app)
	if err != nil {
		return nil, err
	}

	var tasks []*empire.Task

	ecsTasks, err := s.tasks(stackName)
	if err != nil {
		return nil, err
	}

	taskDefinitions := make(map[string]*ecs.TaskDefinition)
	for _, t := range ecsTasks {
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

	for _, t := range ecsTasks {
		k := *t.ClusterArn
		if t.ContainerInstanceArn != nil {
			clusterMap[k] = append(clusterMap[k], t.ContainerInstanceArn)
		}
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

	for _, t := range ecsTasks {
		taskDefinition := taskDefinitions[*t.TaskDefinitionArn]

		id, err := arn.ResourceID(*t.TaskArn)
		if err != nil {
			return tasks, err
		}

		hostId := "FARGATE"
		if t.ContainerInstanceArn != nil {
			hostId = hostMap[*t.ContainerInstanceArn]
		}

		name, p, err := taskDefinitionToProcess(taskDefinition)
		if err != nil {
			return tasks, err
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

		version := p.Environment["EMPIRE_RELEASE"]
		if version == "" {
			version = "v0"
		}

		tasks = append(tasks, &empire.Task{
			Name:        fmt.Sprintf("%s.%s.%s", version, name, id),
			Command:     p.Command,
			Constraints: p.Constraints(),
			State:       state,
			ID:          id,
			Host:        empire.Host{ID: hostId},
			UpdatedAt:   updatedAt,
		})
	}

	return tasks, nil
}

// tasks returns all of the ECS tasks for this app.
func (s *TaskEngine) tasks(stackName string) ([]*ecs.Task, error) {
	services, err := s.Services(stackName)
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
		StartedBy: aws.String(stackName),
	}, func(resp *ecs.ListTasksOutput, lastPage bool) bool {
		arns = append(arns, resp.TaskArns...)
		return true
	}); err != nil {
		return nil, fmt.Errorf("error listing tasks started by %s: %v", stackName, err)
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
func (s *TaskEngine) Services(stackName string) (map[string]string, error) {
	return s.extractProcessData(stackName, servicesOutput)
}

// TaskDefinitions returns a map that maps the name of the process (e.g. web) to the
// ARN of the associated ECS TaskDefinition.
func (s *TaskEngine) TaskDefinitions(stackName string) (map[string]string, error) {
	return s.extractProcessData(stackName, taskDefinitionsOutput)
}

func (s *TaskEngine) extractProcessData(stackName, outputName string) (map[string]string, error) {
	stack, err := s.stack(aws.String(stackName))
	if err != nil {
		return nil, fmt.Errorf("error describing stack: %v", err)
	}

	o := output(stack, outputName)
	if o == nil {
		// Nothing to do but wait until the outputs are set.
		if *stack.StackStatus == cloudformation.StackStatusCreateInProgress {
			return nil, nil
		}

		return nil, fmt.Errorf("stack didn't provide a \"%s\" output key", outputName)
	}

	return extractProcessData(*o.OutputValue), nil
}

// stack returns the cloudformation.Stack for the given stack name.
func (s *TaskEngine) stack(stackName *string) (*cloudformation.Stack, error) {
	resp, err := s.cloudformation.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: stackName,
	})
	if err != nil {
		return nil, err
	}
	return resp.Stacks[0], nil
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

// taskDefinitionToProcess takes an ECS Task Definition and converts it to a
// Process.
func taskDefinitionToProcess(td *ecs.TaskDefinition) (string, *empire.Process, error) {
	// If this task definition has no container definitions, then something
	// funky is up.
	if len(td.ContainerDefinitions) == 0 {
		return "", nil, errors.New("task definition had no container definitions")
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

	return aws.StringValue(container.Name), &empire.Process{
		Command:     command,
		Memory:      constraints.Memory(uint(*container.Memory) * bytesize.MB),
		CPUShare:    constraints.CPUShare(*container.Cpu),
		Environment: env,
	}, nil
}

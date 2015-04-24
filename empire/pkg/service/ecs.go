package service

import (
	"errors"
	"strings"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ecs"
	"github.com/remind101/empire/empire/pkg/arn"
	. "github.com/remind101/empire/empire/pkg/bytesize"
	"github.com/remind101/empire/empire/pkg/ecsutil"
	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
)

var DefaultDelimiter = "-"

// ECSManager is an implementation of the ServiceManager interface that
// is backed by Amazon ECS.
type ECSManager struct {
	// The full name of the ECS cluster to create services in.
	Cluster string

	ecs *ecsutil.Client
}

// NewECSManager returns a new ECSManager instance with a configured ECS client.
func NewECSManager(config *aws.Config) *ECSManager {
	return &ECSManager{
		ecs: ecsutil.NewClient(config),
	}
}

// Submit will create an ECS service for each individual process in the App. New
// task definitions will be created based on the information with each process.
//
// If the app was previously submitted with different process than what are
// provided, any process types that don't exist in the new release will be
// removed from ECS. For example, if you previously submitted an app with a
// `web` and `worker` process, then submit an app with the `web` process, the
// ECS service for the old `worker` process will be removed.
func (m *ECSManager) Submit(ctx context.Context, app *App) error {
	processes, err := m.listProcesses(app.Name)
	if err != nil {
		return err
	}

	for _, p := range app.Processes {
		if err := m.submitProcess(app, p); err != nil {
			return err
		}
	}

	toRemove := diffProcessTypes(processes, app.Processes)
	for _, p := range toRemove {
		if err := m.removeProcess(ctx, app.Name, p); err != nil {
			return err
		}
	}

	return nil
}

// listProcesses lists all of the ecs services for the app.
func (m *ECSManager) listProcesses(app string) ([]*Process, error) {
	var processes []*Process

	list, err := m.ecs.ListAppServices(app, &ecs.ListServicesInput{
		Cluster: aws.String(m.Cluster),
	})
	if err != nil {
		return processes, err
	}

	if len(list.ServiceARNs) == 0 {
		return processes, nil
	}

	desc, err := m.ecs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(m.Cluster),
		Services: list.ServiceARNs,
	})
	if err != nil {
		return processes, err
	}

	for _, s := range desc.Services {
		resp, err := m.ecs.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: s.TaskDefinition,
		})
		if err != nil {
			return processes, err
		}

		p, err := taskDefinitionToProcess(resp.TaskDefinition)
		if err != nil {
			return processes, err
		}

		processes = append(processes, p)
	}

	return processes, nil
}

// submitProcess creates the a task definition based on the information provided
// in the process, then updates or creates the associated ECS service, with the
// new task definition.
func (m *ECSManager) submitProcess(app *App, process *Process) error {
	if _, err := m.createTaskDefinition(app, process); err != nil {
		return err
	}

	_, err := m.updateCreateService(app, process)
	return err
}

// createTaskDefinition creates a Task Definition in ECS for the service.
func (m *ECSManager) createTaskDefinition(app *App, process *Process) (*ecs.TaskDefinition, error) {
	resp, err := m.ecs.RegisterAppTaskDefinition(app.Name, taskDefinitionInput(process))
	return resp.TaskDefinition, err
}

// createService creates a Service in ECS for the service.
func (m *ECSManager) createService(app *App, p *Process) (*ecs.Service, error) {
	resp, err := m.ecs.CreateAppService(app.Name, &ecs.CreateServiceInput{
		Cluster:        aws.String(m.Cluster),
		DesiredCount:   aws.Long(int64(p.Instances)),
		ServiceName:    aws.String(p.Type),
		TaskDefinition: aws.String(p.Type),
	})
	return resp.Service, err
}

// updateService updates an existing Service in ECS.
func (m *ECSManager) updateService(app *App, p *Process) (*ecs.Service, error) {
	resp, err := m.ecs.UpdateAppService(app.Name, &ecs.UpdateServiceInput{
		Cluster:        aws.String(m.Cluster),
		DesiredCount:   aws.Long(int64(p.Instances)),
		Service:        aws.String(p.Type),
		TaskDefinition: aws.String(p.Type),
	})

	// If the service does not exist, return nil.
	if noService(err) {
		return nil, nil
	}

	return resp.Service, err
}

// updateCreateService will perform an upsert for the service in ECS.
func (m *ECSManager) updateCreateService(app *App, p *Process) (*ecs.Service, error) {
	s, err := m.updateService(app, p)
	if err != nil {
		return nil, err
	}

	if s != nil {
		return s, nil
	}

	return m.createService(app, p)
}

// Scale scales an ECS service to the desired number of instances.
func (m *ECSManager) Scale(ctx context.Context, app string, process string, instances uint) error {
	_, err := m.ecs.UpdateAppService(app, &ecs.UpdateServiceInput{
		Cluster:      aws.String(m.Cluster),
		DesiredCount: aws.Long(int64(instances)),
		Service:      aws.String(process),
	})
	return err
}

// Remove removes any ECS services that belong to this app.
func (m *ECSManager) Remove(ctx context.Context, app string) error {
	processes, err := m.listProcesses(app)
	if err != nil {
		return err
	}

	for t, _ := range processTypes(processes) {
		if err := m.removeProcess(ctx, app, t); err != nil {
			return err
		}
	}

	return nil
}

func (m *ECSManager) removeProcess(ctx context.Context, app, process string) error {
	if err := m.Scale(ctx, app, process, 0); err != nil {
		return err
	}

	_, err := m.ecs.DeleteAppService(app, &ecs.DeleteServiceInput{
		Cluster: aws.String(m.Cluster),
		Service: aws.String(process),
	})
	return err
}

// listAppTasks returns all tasks for a given app.
func (m *ECSManager) listAppTasks(app string) ([]*ecs.Task, error) {
	var tasks []*ecs.Task

	resp, err := m.ecs.ListAppServices(app, &ecs.ListServicesInput{
		Cluster: aws.String(m.Cluster),
	})
	if err != nil {
		return tasks, err
	}

	l := len(resp.ServiceARNs)
	serviceTasks := make(chan []*ecs.Task, l)

	for _, s := range resp.ServiceARNs {
		id, err := arn.ResourceID(*s)
		if err != nil {
			return tasks, err
		}

		go func(id string) {
			// TODO handle error
			t, _ := m.serviceTasks(id)

			serviceTasks <- t
		}(id)
	}

	for i := 0; i < l; i++ {
		t := <-serviceTasks
		tasks = append(tasks, t...)
	}

	return tasks, nil
}

// serviceTasks returns all tasks for a specific ECS service.
func (m *ECSManager) serviceTasks(service string) ([]*ecs.Task, error) {
	tr, err := m.ecs.ListTasks(&ecs.ListTasksInput{
		Cluster:     aws.String(m.Cluster),
		ServiceName: aws.String(service),
	})
	if err != nil {
		return nil, err
	}

	if len(tr.TaskARNs) == 0 {
		return []*ecs.Task{}, nil
	}

	dr, err := m.ecs.DescribeTasks(&ecs.DescribeTasksInput{
		Tasks: tr.TaskARNs,
	})
	return dr.Tasks, err
}

// Instances returns all instances that are currently running, pending or
// draining.
func (m *ECSManager) Instances(ctx context.Context, app string) ([]*Instance, error) {
	var instances []*Instance

	tasks, err := m.listAppTasks(app)
	if err != nil {
		return instances, err
	}

	for _, t := range tasks {
		resp, err := m.ecs.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: t.TaskDefinitionARN,
		})
		if err != nil {
			return instances, err
		}

		id, err := arn.ResourceID(*t.TaskARN)
		if err != nil {
			return instances, err
		}

		p, err := taskDefinitionToProcess(resp.TaskDefinition)
		if err != nil {
			return instances, err
		}

		instances = append(instances, &Instance{
			Process:   p,
			State:     safeString(t.LastStatus),
			ID:        id,
			UpdatedAt: timex.Now(),
		})
	}

	return instances, nil
}

func (m *ECSManager) Stop(ctx context.Context, instanceID string) error {
	_, err := m.ecs.StopTask(&ecs.StopTaskInput{
		Cluster: aws.String(m.Cluster),
		Task:    aws.String(instanceID),
	})
	return err
}

// taskDefinitionInput returns an ecs.RegisterTaskDefinitionInput suitable for
// creating a task definition from a Process.
func taskDefinitionInput(p *Process) *ecs.RegisterTaskDefinitionInput {
	var command []*string
	for _, s := range strings.Split(p.Command, " ") {
		ss := s
		command = append(command, &ss)
	}

	var environment []*ecs.KeyValuePair
	for k, v := range p.Env {
		environment = append(environment, &ecs.KeyValuePair{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}

	var ports []*ecs.PortMapping
	for _, m := range p.Ports {
		ports = append(ports, &ecs.PortMapping{
			HostPort:      m.Host,
			ContainerPort: m.Container,
		})
	}

	return &ecs.RegisterTaskDefinitionInput{
		Family: aws.String(p.Type),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			&ecs.ContainerDefinition{
				Name:         aws.String(p.Type),
				CPU:          aws.Long(int64(p.CPUShares)),
				Command:      command,
				Image:        aws.String(p.Image),
				Essential:    aws.Boolean(true),
				Memory:       aws.Long(int64(p.MemoryLimit / MB)),
				Environment:  environment,
				PortMappings: ports,
			},
		},
	}
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func noService(err error) bool {
	if err, ok := err.(aws.APIError); ok {
		if err.Message == "Service was not ACTIVE." {
			return true
		}

		if err.Message == "Service not found." {
			return true
		}
	}

	return false
}

// taskDefinitionToProcess takes an ECS Task Definition and converts it to a
// Process.
func taskDefinitionToProcess(td *ecs.TaskDefinition) (*Process, error) {
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

	return &Process{
		Type:    safeString(container.Name),
		Command: strings.Join(command, " "),
	}, nil
}

func diffProcessTypes(old, new []*Process) []string {
	var types []string

	om := processTypes(old)
	nm := processTypes(new)

	for t, _ := range om {
		if _, ok := nm[t]; !ok {
			types = append(types, t)
		}
	}

	return types
}

func processTypes(processes []*Process) map[string]struct{} {
	m := make(map[string]struct{})

	for _, p := range processes {
		m[p.Type] = struct{}{}
	}

	return m
}

package service

import (
	"errors"
	"strings"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ecs"
	"github.com/remind101/empire/empire/pkg/arn"
	. "github.com/remind101/empire/empire/pkg/bytesize"
	"github.com/remind101/empire/empire/pkg/ecsutil"
	"github.com/remind101/empire/empire/pkg/lb"
	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
)

var DefaultDelimiter = "-"

const ECSServiceRole = "ecsServiceRole"

// ECSManager is an implementation of the ServiceManager interface that
// is backed by Amazon ECS.
type ECSManager struct {
	ProcessManager

	cluster string
	ecs     *ecsutil.Client
}

// ECSConfig holds configuration for generating a new ECS backed Manager
// implementation.
type ECSConfig struct {
	// The ECS cluster to create services and task definitions in.
	Cluster string

	// VPC controls what subnets to attach to ELB's that are created.
	VPC string

	// The hosted zone to create internal DNS records in.
	Zone string

	// The ID of the security group to assign to internal load balancers.
	InternalSecurityGroupID string

	// The ID of the security group to assign to external load balancers.
	ExternalSecurityGroupID string

	// The Subnet IDs to assign when creating internal load balancers.
	InternalSubnetIDs []string

	// The Subnet IDs to assign when creating external load balancers.
	ExternalSubnetIDs []string

	// AWS configuration.
	AWS *aws.Config
}

func (c ECSConfig) validELBConfig() bool {
	return c.InternalSecurityGroupID != "" &&
		c.ExternalSecurityGroupID != "" &&
		len(c.InternalSubnetIDs) > 0 &&
		len(c.ExternalSubnetIDs) > 0
}

// NewECSManager returns a new Manager implementation that:
//
// * Will create internal or external ELB's for ECS services.
// * Will create a CNAME record in route53 under the internal TLD.
func NewECSManager(config ECSConfig) *ECSManager {
	c := ecsutil.NewClient(config.AWS)

	var pm ProcessManager = &ecsProcessManager{
		cluster: config.Cluster,
		ecs:     c,
	}

	// If security group ids are provided, ELB's will be created for ECS
	// services.
	if config.validELBConfig() {
		elb := lb.NewVPCELBManager(config.VPC, config.AWS)
		elb.InternalSecurityGroupID = config.InternalSecurityGroupID
		elb.ExternalSecurityGroupID = config.ExternalSecurityGroupID
		elb.InternalSubnetIDs = config.InternalSubnetIDs
		elb.ExternalSubnetIDs = config.ExternalSubnetIDs

		var l lb.Manager = elb

		if config.Zone != "" {
			n := lb.NewRoute53Nameserver(config.AWS)
			n.Zone = config.Zone

			l = lb.WithCNAME(l, n)
		}

		pm = &LBProcessManager{
			ProcessManager: pm,
			lb:             lb.WithLogging(l),
		}
	}

	return &ECSManager{
		cluster:        config.Cluster,
		ProcessManager: pm,
		ecs:            c,
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
	processes, err := m.Processes(app.ID)
	if err != nil {
		return err
	}

	for _, p := range app.Processes {
		if err := m.CreateProcess(ctx, app, p); err != nil {
			return err
		}
	}

	toRemove := diffProcessTypes(processes, app.Processes)
	for _, p := range toRemove {
		if err := m.RemoveProcess(ctx, app.ID, p); err != nil {
			return err
		}
	}

	return nil
}

// Remove removes any ECS services that belong to this app.
func (m *ECSManager) Remove(ctx context.Context, app string) error {
	processes, err := m.Processes(app)
	if err != nil {
		return err
	}

	for t, _ := range processTypes(processes) {
		if err := m.RemoveProcess(ctx, app, t); err != nil {
			return err
		}
	}

	return nil
}

// listAppTasks returns all tasks for a given app.
func (m *ECSManager) listAppTasks(app string) ([]*ecs.Task, error) {
	var tasks []*ecs.Task

	resp, err := m.ecs.ListAppServices(app, &ecs.ListServicesInput{
		Cluster: aws.String(m.cluster),
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
		Cluster:     aws.String(m.cluster),
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
		Cluster: aws.String(m.cluster),
		Task:    aws.String(instanceID),
	})
	return err
}

var _ ProcessManager = &ecsProcessManager{}

// ecsProcessManager is an implementation of the ProcessManager interface that
// creates ECS services for Processes.
type ecsProcessManager struct {
	cluster string
	ecs     *ecsutil.Client
}

// CreateProcess creates an ECS service for the process.
func (m *ecsProcessManager) CreateProcess(ctx context.Context, app *App, p *Process) error {
	if _, err := m.createTaskDefinition(app, p); err != nil {
		return err
	}

	_, err := m.updateCreateService(app, p)
	return err
}

// createTaskDefinition creates a Task Definition in ECS for the service.
func (m *ecsProcessManager) createTaskDefinition(app *App, process *Process) (*ecs.TaskDefinition, error) {
	resp, err := m.ecs.RegisterAppTaskDefinition(app.ID, taskDefinitionInput(process))
	return resp.TaskDefinition, err
}

// createService creates a Service in ECS for the service.
func (m *ecsProcessManager) createService(app *App, p *Process) (*ecs.Service, error) {
	var role *string
	var loadBalancers []*ecs.LoadBalancer

	if p.LoadBalancer != "" {
		loadBalancers = []*ecs.LoadBalancer{
			{
				ContainerName:    aws.String(p.Type),
				ContainerPort:    p.Ports[0].Container,
				LoadBalancerName: aws.String(p.LoadBalancer),
			},
		}
		role = aws.String(ECSServiceRole)
	}

	resp, err := m.ecs.CreateAppService(app.ID, &ecs.CreateServiceInput{
		Cluster:        aws.String(m.cluster),
		DesiredCount:   aws.Long(int64(p.Instances)),
		ServiceName:    aws.String(p.Type),
		TaskDefinition: aws.String(p.Type),
		LoadBalancers:  loadBalancers,
		Role:           role,
	})
	return resp.Service, err
}

// updateService updates an existing Service in ECS.
func (m *ecsProcessManager) updateService(app *App, p *Process) (*ecs.Service, error) {
	resp, err := m.ecs.UpdateAppService(app.ID, &ecs.UpdateServiceInput{
		Cluster:        aws.String(m.cluster),
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
func (m *ecsProcessManager) updateCreateService(app *App, p *Process) (*ecs.Service, error) {
	s, err := m.updateService(app, p)
	if err != nil {
		return nil, err
	}

	if s != nil {
		return s, nil
	}

	return m.createService(app, p)
}

func (m *ecsProcessManager) Processes(app string) ([]*Process, error) {
	var processes []*Process

	list, err := m.ecs.ListAppServices(app, &ecs.ListServicesInput{
		Cluster: aws.String(m.cluster),
	})
	if err != nil {
		return processes, err
	}

	if len(list.ServiceARNs) == 0 {
		return processes, nil
	}

	desc, err := m.ecs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(m.cluster),
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

func (m *ecsProcessManager) RemoveProcess(ctx context.Context, app string, process string) error {
	if err := m.Scale(ctx, app, process, 0); noService(err) {
		return nil
	} else if err != nil {
		return err
	}

	_, err := m.ecs.DeleteAppService(app, &ecs.DeleteServiceInput{
		Cluster: aws.String(m.cluster),
		Service: aws.String(process),
	})
	if noService(err) {
		return nil
	}

	return err
}

// Scale scales an ECS service to the desired number of instances.
func (m *ecsProcessManager) Scale(ctx context.Context, app string, process string, instances uint) error {
	_, err := m.ecs.UpdateAppService(app, &ecs.UpdateServiceInput{
		Cluster:      aws.String(m.cluster),
		DesiredCount: aws.Long(int64(instances)),
		Service:      aws.String(process),
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

		// Wat
		if err.Message == "Could not find returned type com.amazon.madison.cmb#CMServiceNotActiveException in model" {
			return true
		}
		if err.Message == "Could not find returned type com.amazon.madison.cmb#CMServiceNotFoundException in model" {
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

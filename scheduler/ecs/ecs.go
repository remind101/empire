// Pacakge ecs provides an implementation of the Scheduler interface that uses
// Amazon EC2 Container Service.
package ecs

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/remind101/empire/12factor"
	"github.com/remind101/empire/pkg/arn"
	. "github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/pkg/ecsutil"
	"github.com/remind101/empire/scheduler"
	"github.com/remind101/empire/scheduler/ecs/lb"
	"golang.org/x/net/context"
)

// For HTTP/HTTPS/TCP services, we allocate an ELB and map it's instance port to
// the container port. This is the port that processes within the container
// should bind to. Tihs value is also exposed to the container through the PORT
// environment variable.
const ContainerPort = 8080

var DefaultDelimiter = "-"

// Scheduler is an implementation of the ServiceManager interface that
// is backed by Amazon ECS.
type Scheduler struct {
	cluster          string
	serviceRole      string
	ecs              *ecsutil.Client
	logConfiguration *ecs.LogConfiguration
	lb               lb.Manager
}

// Config holds configuration for generating a new ECS backed Scheduler
// implementation.
type Config struct {
	// The ECS cluster to create services and task definitions in.
	Cluster string

	// The IAM role to use for ECS services with ELBs attached.
	ServiceRole string

	// VPC controls what subnets to attach to ELBs that are created.
	VPC string

	// The hosted zone id to create internal DNS records in
	ZoneID string

	// The ID of the security group to assign to internal load balancers.
	InternalSecurityGroupID string

	// The ID of the security group to assign to external load balancers.
	ExternalSecurityGroupID string

	// The Subnet IDs to assign when creating internal load balancers.
	InternalSubnetIDs []string

	// The Subnet IDs to assign when creating external load balancers.
	ExternalSubnetIDs []string

	// AWS configuration.
	AWS client.ConfigProvider

	// Log configuraton for ECS tasks
	LogConfiguration *ecs.LogConfiguration
}

func newScheduler(config Config) *Scheduler {
	c := ecsutil.NewClient(config.AWS)

	return &Scheduler{
		cluster:          config.Cluster,
		serviceRole:      config.ServiceRole,
		ecs:              c,
		logConfiguration: config.LogConfiguration,
	}
}

// NewScheduler returns a new Scheduler implementation that:
//
// * Creates services with ECS.
func NewScheduler(config Config) (*Scheduler, error) {
	return newScheduler(config), nil
}

// NewLoadBalancedScheduler returns a new Scheduler instance that:
//
// * Creates services with ECS.
// * Creates internal or external ELBs for ECS services.
// * Creates a CNAME record in route53 under the internal TLD.
// * Allocates ports from the ports table.
func NewLoadBalancedScheduler(db *sql.DB, config Config) (*Scheduler, error) {
	lb, err := newLBManager(db, config)
	if err != nil {
		return nil, err
	}

	s := newScheduler(config)
	s.lb = lb
	return s, nil
}

func newLBManager(db *sql.DB, config Config) (lb.Manager, error) {
	if err := validateLoadBalancedConfig(config); err != nil {
		return nil, err
	}

	// Create the ELB Manager
	elb := lb.NewELBManager(config.AWS)
	elb.Ports = lb.NewDBPortAllocator(db)
	elb.InternalSecurityGroupID = config.InternalSecurityGroupID
	elb.ExternalSecurityGroupID = config.ExternalSecurityGroupID
	elb.InternalSubnetIDs = config.InternalSubnetIDs
	elb.ExternalSubnetIDs = config.ExternalSubnetIDs

	// Compose the LB Manager
	var lbm lb.Manager = elb

	n := lb.NewRoute53Nameserver(config.AWS)
	n.ZoneID = config.ZoneID

	lbm = lb.WithCNAME(lbm, n)
	lbm = lb.WithLogging(lbm)

	return lbm, nil
}

func validateLoadBalancedConfig(c Config) error {
	r := func(n string) error {
		return errors.New(fmt.Sprintf("%s is required", n))
	}

	if c.Cluster == "" {
		return r("Cluster")
	}
	if c.ServiceRole == "" {
		return r("ServiceRole")
	}
	if c.ZoneID == "" {
		return r("ZoneID")
	}
	if c.InternalSecurityGroupID == "" {
		return r("InternalSecurityGroupID")
	}
	if c.ExternalSecurityGroupID == "" {
		return r("ExternalSecurityGroupID")
	}
	if len(c.InternalSubnetIDs) == 0 {
		return r("InternalSubnetIDs")
	}
	if len(c.ExternalSubnetIDs) == 0 {
		return r("ExternalSubnetIDs")
	}

	return nil
}

// Submit will create an ECS service for each individual process in the App. New
// task definitions will be created based on the information with each process.
//
// If the app was previously submitted with different process than what are
// provided, any process types that don't exist in the new release will be
// removed from ECS. For example, if you previously submitted an app with a
// `web` and `worker` process, then submit an app with the `web` process, the
// ECS service for the old `worker` process will be removed.
func (m *Scheduler) Submit(ctx context.Context, app twelvefactor.App) error {
	processes, err := m.Processes(ctx, app.ID)
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
func (m *Scheduler) Remove(ctx context.Context, appID string) error {
	processes, err := m.Processes(ctx, appID)
	if err != nil {
		return err
	}

	for t, _ := range processTypes(processes) {
		if err := m.RemoveProcess(ctx, appID, t); err != nil {
			return err
		}
	}

	return nil
}

// Instances returns all instances that are currently running, pending or
// draining.
func (m *Scheduler) Instances(ctx context.Context, appID string) ([]scheduler.Instance, error) {
	var instances []scheduler.Instance

	tasks, err := m.describeAppTasks(ctx, appID)
	if err != nil {
		return instances, err
	}

	taskDefinitions := make(map[string]*ecs.TaskDefinition)
	for _, t := range tasks {
		k := *t.TaskDefinitionArn

		if _, ok := taskDefinitions[k]; !ok {
			resp, err := m.ecs.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
				TaskDefinition: t.TaskDefinitionArn,
			})
			if err != nil {
				return instances, err
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

		instances = append(instances, scheduler.Instance{
			Process:   p,
			State:     state,
			ID:        id,
			UpdatedAt: updatedAt,
		})
	}

	return instances, nil
}

func (m *Scheduler) describeAppTasks(ctx context.Context, appID string) ([]*ecs.Task, error) {
	resp, err := m.ecs.ListAppTasks(ctx, appID, &ecs.ListTasksInput{
		Cluster: aws.String(m.cluster),
	})
	if err != nil {
		return nil, err
	}

	if len(resp.TaskArns) == 0 {
		return []*ecs.Task{}, nil
	}

	tasks, err := m.ecs.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(m.cluster),
		Tasks:   resp.TaskArns,
	})
	return tasks.Tasks, err
}

func (m *Scheduler) Stop(ctx context.Context, instanceID string) error {
	_, err := m.ecs.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: aws.String(m.cluster),
		Task:    aws.String(instanceID),
	})
	return err
}

// CreateProcess creates an ECS service for the process.
func (m *Scheduler) CreateProcess(ctx context.Context, app twelvefactor.App, p twelvefactor.Process) error {
	loadBalancer, err := m.loadBalancer(ctx, app, p)
	if err != nil {
		return err
	}

	if _, err := m.createTaskDefinition(ctx, app, p, loadBalancer); err != nil {
		return err
	}

	_, err = m.updateCreateService(ctx, app, p, loadBalancer)
	return err
}

func (m *Scheduler) Run(ctx context.Context, app twelvefactor.App, process twelvefactor.Process, in io.Reader, out io.Writer) error {
	if out != nil {
		return errors.New("running an attached process is not implemented by the ECS manager.")
	}

	td, err := m.createTaskDefinition(ctx, app, process, nil)
	if err != nil {
		return err
	}

	_, err = m.ecs.RunTask(ctx, &ecs.RunTaskInput{
		TaskDefinition: td.TaskDefinitionArn,
		Cluster:        aws.String(m.cluster),
		Count:          aws.Int64(1),
		StartedBy:      aws.String(app.ID),
	})
	return err
}

// createTaskDefinition creates a Task Definition in ECS for the service.
func (m *Scheduler) createTaskDefinition(ctx context.Context, app twelvefactor.App, process twelvefactor.Process, loadBalancer *lb.LoadBalancer) (*ecs.TaskDefinition, error) {
	taskDef, err := m.taskDefinitionInput(app, process, loadBalancer)
	if err != nil {
		return nil, err
	}

	resp, err := m.ecs.RegisterAppTaskDefinition(ctx, app.ID, taskDef)
	return resp.TaskDefinition, err
}

func (m *Scheduler) taskDefinitionInput(app twelvefactor.App, p twelvefactor.Process, loadBalancer *lb.LoadBalancer) (*ecs.RegisterTaskDefinitionInput, error) {
	// ecs.ContainerDefinition{Command} is expecting a []*string
	var command []*string
	for _, s := range p.Command {
		ss := s
		command = append(command, &ss)
	}

	var environment []*ecs.KeyValuePair
	for k, v := range twelvefactor.ProcessEnv(app, p) {
		environment = append(environment, &ecs.KeyValuePair{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}

	// If there's a load balancer attached, generate the port mappings and
	// expose the container port to the process via the PORT environment
	// variable.
	var ports []*ecs.PortMapping
	if loadBalancer != nil {
		ports = append(ports, &ecs.PortMapping{
			HostPort:      aws.Int64(loadBalancer.InstancePort),
			ContainerPort: aws.Int64(ContainerPort),
		})
		environment = append(environment, &ecs.KeyValuePair{
			Name:  aws.String("PORT"),
			Value: aws.String(fmt.Sprintf("%d", ContainerPort)),
		})
	}

	labels := make(map[string]*string)
	for k, v := range twelvefactor.ProcessLabels(app, p) {
		labels[k] = aws.String(v)
	}

	var ulimits []*ecs.Ulimit
	if p.Nproc != 0 {
		ulimits = []*ecs.Ulimit{
			&ecs.Ulimit{
				Name:      aws.String("nproc"),
				SoftLimit: aws.Int64(int64(p.Nproc)),
				HardLimit: aws.Int64(int64(p.Nproc)),
			},
		}
	}

	return &ecs.RegisterTaskDefinitionInput{
		Family: aws.String(p.Type),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			&ecs.ContainerDefinition{
				Name:             aws.String(p.Type),
				Cpu:              aws.Int64(int64(p.CPUShares)),
				Command:          command,
				Image:            aws.String(app.Image.String()),
				Essential:        aws.Bool(true),
				Memory:           aws.Int64(int64(p.MemoryLimit / MB)),
				Environment:      environment,
				LogConfiguration: m.logConfiguration,
				PortMappings:     ports,
				DockerLabels:     labels,
				Ulimits:          ulimits,
			},
		},
	}, nil
}

// createService creates a Service in ECS for the service.
func (m *Scheduler) createService(ctx context.Context, app twelvefactor.App, p twelvefactor.Process, loadBalancer *lb.LoadBalancer) (*ecs.Service, error) {
	var role *string
	var loadBalancers []*ecs.LoadBalancer

	if loadBalancer != nil {
		loadBalancers = []*ecs.LoadBalancer{
			{
				ContainerName:    aws.String(p.Type),
				ContainerPort:    aws.Int64(ContainerPort),
				LoadBalancerName: aws.String(loadBalancer.Name),
			},
		}
		role = aws.String(m.serviceRole)
	}

	resp, err := m.ecs.CreateAppService(ctx, app.ID, &ecs.CreateServiceInput{
		Cluster:        aws.String(m.cluster),
		DesiredCount:   aws.Int64(int64(p.Instances)),
		ServiceName:    aws.String(p.Type),
		TaskDefinition: aws.String(p.Type),
		LoadBalancers:  loadBalancers,
		Role:           role,
	})
	return resp.Service, err
}

// updateService updates an existing Service in ECS.
func (m *Scheduler) updateService(ctx context.Context, app twelvefactor.App, p twelvefactor.Process) (*ecs.Service, error) {
	_, err := m.loadBalancer(ctx, app, p)
	if err != nil {
		return nil, err
	}

	resp, err := m.ecs.UpdateAppService(ctx, app.ID, &ecs.UpdateServiceInput{
		Cluster:        aws.String(m.cluster),
		DesiredCount:   aws.Int64(int64(p.Instances)),
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
func (m *Scheduler) updateCreateService(ctx context.Context, app twelvefactor.App, p twelvefactor.Process, loadBalancer *lb.LoadBalancer) (*ecs.Service, error) {
	s, err := m.updateService(ctx, app, p)
	if err != nil {
		return nil, err
	}

	if s != nil {
		return s, nil
	}

	return m.createService(ctx, app, p, loadBalancer)
}

// loadBalancer creates (or updates) a a load balancer for the given process, if
// the process is exposed. It returns the name of the load balancer.
func (m *Scheduler) loadBalancer(ctx context.Context, app twelvefactor.App, p twelvefactor.Process) (*lb.LoadBalancer, error) {
	// No exposure, no load balancer.
	if p.Exposure == nil {
		return nil, nil
	}

	// Attempt to find an existing load balancer for this app.
	l, err := m.findLoadBalancer(ctx, app.ID, p.Type)
	if err != nil {
		return nil, err
	}

	// If the load balancer doesn't match the exposure that we
	// want, we'll return an error. Users should manually destroy
	// the app and re-create it with the proper exposure.
	if l != nil {
		var opts *lb.UpdateLoadBalancerOpts
		opts, err = updateOpts(p, l)
		if err != nil {
			return nil, err
		}

		if opts != nil {
			if err = m.lb.UpdateLoadBalancer(ctx, *opts); err != nil {
				return nil, err
			}
		}
	}

	// If this app doesn't have a load balancer yet, create one.
	if l == nil {
		tags := lbTags(app.ID, p.Type)

		// Add "App" tag so that a CNAME can be created.
		tags[lb.AppTag] = app.Name

		opts := lb.CreateLoadBalancerOpts{
			External: p.Exposure.External,
			Tags:     tags,
		}

		if e, ok := p.Exposure.Type.(*twelvefactor.HTTPSExposure); ok {
			opts.SSLCert = e.Cert
		}

		l, err = m.lb.CreateLoadBalancer(ctx, opts)
		if err != nil {
			return nil, err
		}
	}

	return l, nil
}

func (m *Scheduler) removeLoadBalancer(ctx context.Context, app string, p string) error {
	l, err := m.findLoadBalancer(ctx, app, p)
	if err != nil {
		// TODO: Maybe we shouldn't care here.
		return err
	}

	if l != nil {
		if err := m.lb.DestroyLoadBalancer(ctx, l); err != nil {
			// TODO: Maybe we shouldn't care here.
			return err
		}
	}

	return nil
}

// findLoadBalancer attempts to find an existing load balancer for the app.
func (m *Scheduler) findLoadBalancer(ctx context.Context, app string, process string) (*lb.LoadBalancer, error) {
	lbs, err := m.lb.LoadBalancers(ctx, lbTags(app, process))
	if err != nil || len(lbs) == 0 {
		return nil, err
	}

	return lbs[0], nil
}

func (m *Scheduler) Processes(ctx context.Context, appID string) ([]twelvefactor.Process, error) {
	var processes []twelvefactor.Process

	list, err := m.ecs.ListAppServices(ctx, appID, &ecs.ListServicesInput{
		Cluster: aws.String(m.cluster),
	})
	if err != nil {
		return processes, err
	}

	if len(list.ServiceArns) == 0 {
		return processes, nil
	}

	desc, err := m.ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  aws.String(m.cluster),
		Services: list.ServiceArns,
	})
	if err != nil {
		return processes, err
	}

	for _, s := range desc.Services {
		resp, err := m.ecs.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
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

func (m *Scheduler) RemoveProcess(ctx context.Context, app string, process string) error {
	if err := m.Scale(ctx, app, process, 0); noService(err) {
		return nil
	} else if err != nil {
		return err
	}

	_, err := m.ecs.DeleteAppService(ctx, app, &ecs.DeleteServiceInput{
		Cluster: aws.String(m.cluster),
		Service: aws.String(process),
	})
	if noService(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return m.removeLoadBalancer(ctx, app, process)
}

// Scale scales an ECS service to the desired number of instances.
func (m *Scheduler) Scale(ctx context.Context, app string, process string, instances uint) error {
	_, err := m.ecs.UpdateAppService(ctx, app, &ecs.UpdateServiceInput{
		Cluster:      aws.String(m.cluster),
		DesiredCount: aws.Int64(int64(instances)),
		Service:      aws.String(process),
	})
	return err
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func noService(err error) bool {
	if err, ok := err.(awserr.Error); ok {
		if err.Message() == "Service was not ACTIVE." {
			return true
		}

		// Wat
		if err.Message() == "Could not find returned type com.amazon.madison.cmb#CMServiceNotActiveException in model" {
			return true
		}
		if err.Message() == "Could not find returned type com.amazon.madison.cmb#CMServiceNotFoundException in model" {
			return true
		}

		if err.Message() == "Service not found." {
			return true
		}
	}

	return false
}

// taskDefinitionToProcess takes an ECS Task Definition and converts it to a
// Process.
func taskDefinitionToProcess(td *ecs.TaskDefinition) (twelvefactor.Process, error) {
	// If this task definition has no container definitions, then something
	// funky is up.
	if len(td.ContainerDefinitions) == 0 {
		return twelvefactor.Process{}, errors.New("task definition had no container definitions")
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

	return twelvefactor.Process{
		Type:        safeString(container.Name),
		Command:     command,
		Env:         env,
		CPUShares:   uint(*container.Cpu),
		MemoryLimit: uint(*container.Memory) * MB,
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

func diffProcessTypes(old, new []twelvefactor.Process) []string {
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

func processTypes(processes []twelvefactor.Process) map[string]struct{} {
	m := make(map[string]struct{})

	for _, p := range processes {
		m[p.Type] = struct{}{}
	}

	return m
}

// lbTags returns the tags that should be attached to the load balancer so that
// we can find it later.
func lbTags(app string, process string) map[string]string {
	return map[string]string{
		"AppID":       app,
		"ProcessType": process,
	}
}

// LoadBalancerExposureError is returned when the exposure of the process in the data store does not match the exposure of the ELB
type LoadBalancerExposureError struct {
	proc twelvefactor.Process
	lb   *lb.LoadBalancer
}

func (e *LoadBalancerExposureError) Error() string {
	return fmt.Sprintf("Process %s is %s, but load balancer is %s. An update would require me to delete the load balancer.", e.proc.Type, external(e.proc.Exposure.External), external(e.lb.External))
}

type external bool

func (e external) String() string {
	if e {
		return "public"
	}

	return "private"
}

// canUpdate checks if the load balancer is suitable for the process.
func canUpdate(p twelvefactor.Process, lb *lb.LoadBalancer) error {
	if p.Exposure.External && !lb.External {
		return &LoadBalancerExposureError{p, lb}
	}

	if !p.Exposure.External && lb.External {
		return &LoadBalancerExposureError{p, lb}
	}

	return nil
}

func updateOpts(p twelvefactor.Process, b *lb.LoadBalancer) (*lb.UpdateLoadBalancerOpts, error) {
	// This load balancer can't be updated to make it work for the process.
	// Return an error.
	if err := canUpdate(p, b); err != nil {
		return nil, err
	}

	opts := lb.UpdateLoadBalancerOpts{
		Name: b.Name,
	}

	// Requires an update to the Cert.
	if e, ok := p.Exposure.Type.(*twelvefactor.HTTPSExposure); ok {
		if e.Cert != b.SSLCert {
			opts.SSLCert = &e.Cert
		}
	}

	// Load balancer doesn't require an update.
	if opts.SSLCert == nil {
		return nil, nil
	}

	return &opts, nil
}

package cloudformation

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/mitchellh/hashstructure"
	"github.com/remind101/empire/pkg/cloudformation/customresources"
	"github.com/remind101/pkg/reporter"
)

type ecsClient interface {
	CreateService(*ecs.CreateServiceInput) (*ecs.CreateServiceOutput, error)
	DeleteService(*ecs.DeleteServiceInput) (*ecs.DeleteServiceOutput, error)
	UpdateService(*ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error)
	WaitUntilServicesStable(*ecs.DescribeServicesInput) error
	RegisterTaskDefinition(*ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error)
	DeregisterTaskDefinition(*ecs.DeregisterTaskDefinitionInput) (*ecs.DeregisterTaskDefinitionOutput, error)
}

// newECSClient returns a new ecs.ECS instance, that has more relaxed retry
// timeouts.
func newECSClient(config client.ConfigProvider) *ecs.ECS {
	return ecs.New(config, &aws.Config{
		Retryer: newRetryer(),
	})
}

func newRetryer() client.DefaultRetryer {
	return client.DefaultRetryer{
		NumMaxRetries: 10,
	}
}

type LoadBalancer struct {
	ContainerName    *string
	ContainerPort    *customresources.IntValue
	LoadBalancerName *string
	TargetGroupArn   *string
}

// ECSServiceProperties represents the properties for the Custom::ECSService
// resource.
type ECSServiceProperties struct {
	Cluster                 *string
	DeploymentConfiguration *DeploymentConfiguration  `hash:"ignore"`
	DesiredCount            *customresources.IntValue `hash:"ignore"`
	LoadBalancers           []LoadBalancer
	PlacementConstraints    []ECSPlacementConstraint
	PlacementStrategy       []ECSPlacementStrategy
	PropagateTags           *string
	Role                    *string
	ServiceName             *string
	TaskDefinition          *string `hash:"ignore"`
}

func (p *ECSServiceProperties) ReplacementHash() (uint64, error) {
	return hashstructure.Hash(p, nil)
}

type ECSPlacementConstraint struct {
	Type       *string
	Expression *string
}

type ECSPlacementStrategy struct {
	Type  *string
	Field *string
}

// ECSServiceResource is a Provisioner that creates and updates ECS services.
type ECSServiceResource struct {
	ecs ecsClient
}

func newECSServiceProvisioner(resource *ECSServiceResource) *provisioner {
	return &provisioner{
		properties: func() properties {
			return &ECSServiceProperties{}
		},
		Create: resource.Create,
		Update: resource.Update,
		Delete: resource.Delete,
	}
}

func (p *ECSServiceResource) Create(ctx context.Context, req customresources.Request) (string, interface{}, error) {
	properties := req.ResourceProperties.(*ECSServiceProperties)
	clientToken := req.Hash()
	data := make(map[string]string)

	var loadBalancers []*ecs.LoadBalancer
	for _, v := range properties.LoadBalancers {
		loadBalancers = append(loadBalancers, &ecs.LoadBalancer{
			ContainerName:    v.ContainerName,
			ContainerPort:    v.ContainerPort.Value(),
			LoadBalancerName: v.LoadBalancerName,
			TargetGroupArn:   v.TargetGroupArn,
		})
	}

	var placementConstraints []*ecs.PlacementConstraint
	for _, v := range properties.PlacementConstraints {
		placementConstraints = append(placementConstraints, &ecs.PlacementConstraint{
			Type:       v.Type,
			Expression: v.Expression,
		})
	}

	var placementStrategy []*ecs.PlacementStrategy
	for _, v := range properties.PlacementStrategy {
		placementStrategy = append(placementStrategy, &ecs.PlacementStrategy{
			Type:  v.Type,
			Field: v.Field,
		})
	}

	var serviceName *string
	if properties.ServiceName != nil {
		serviceName = aws.String(fmt.Sprintf("%s-%s", *properties.ServiceName, clientToken))
	}

	// Build the DeploymentConfiguration, validating that the fields are set
	// correctly.
	deploymentConfig, err := deploymentConfiguration(properties)
	if err != nil {
		return "", nil, fmt.Errorf("error setting DeploymentConfiguration: %v", err)
	}

	// Create the service
	resp, err := p.ecs.CreateService(&ecs.CreateServiceInput{
		ClientToken:             aws.String(clientToken),
		Cluster:                 properties.Cluster,
		DeploymentConfiguration: deploymentConfig,
		DesiredCount:            properties.DesiredCount.Value(),
		LoadBalancers:           loadBalancers,
		PlacementConstraints:    placementConstraints,
		PlacementStrategy:       placementStrategy,
		PropagateTags:           properties.PropagateTags,
		Role:                    properties.Role,
		ServiceName:             serviceName,
		TaskDefinition:          properties.TaskDefinition,
	})
	if err != nil {
		return "", nil, fmt.Errorf("error creating service: %v", err)
	}
	data["Name"] = *resp.Service.ServiceName

	d := primaryDeployment(resp.Service)
	if d == nil {
		return "", data, fmt.Errorf("no primary deployment found")
	}
	data["DeploymentId"] = *d.Id

	arn := resp.Service.ServiceArn

	stabilized := make(chan struct{})
	go func() {
		if err := p.ecs.WaitUntilServicesStable(&ecs.DescribeServicesInput{
			Cluster:  properties.Cluster,
			Services: []*string{arn},
		}); err != nil {
			// We're ignoring this error, because the service was created,
			// and if the service doesn't stabilize, it's better to just let
			// the stack finish creating than rolling back.
			reporter.Report(ctx, err)
		}
		close(stabilized)
	}()

	select {
	case <-stabilized:
	case <-ctx.Done():
		return *arn, data, ctx.Err()
	}

	return *arn, data, nil
}

func (p *ECSServiceResource) Update(ctx context.Context, req customresources.Request) (interface{}, error) {
	properties := req.ResourceProperties.(*ECSServiceProperties)
	oldProperties := req.OldResourceProperties.(*ECSServiceProperties)
	data := make(map[string]string)

	var desiredCount *int64
	if !properties.DesiredCount.Eq(oldProperties.DesiredCount) {
		desiredCount = properties.DesiredCount.Value()
	}

	// Build the DeploymentConfiguration, validating that the fields are set
	// correctly.
	deploymentConfig, err := deploymentConfiguration(properties)
	if err != nil {
		return "", nil, fmt.Errorf("error setting DeploymentConfiguration: %v", err)
	}

	resp, err := p.ecs.UpdateService(&ecs.UpdateServiceInput{
		Cluster:                 properties.Cluster,
		DeploymentConfiguration: deploymentConfig,
		DesiredCount:            desiredCount,
		Service:                 aws.String(req.PhysicalResourceId),
		TaskDefinition:          properties.TaskDefinition,
	})
	if err != nil {
		return nil, err
	}
	data["Name"] = *resp.Service.ServiceName

	d := primaryDeployment(resp.Service)
	if d == nil {
		return nil, fmt.Errorf("no primary deployment found")
	}

	data["DeploymentId"] = *d.Id
	return data, nil
}

func (p *ECSServiceResource) Delete(ctx context.Context, req customresources.Request) error {
	properties := req.ResourceProperties.(*ECSServiceProperties)
	service := aws.String(req.PhysicalResourceId)
	cluster := properties.Cluster

	// We have to scale the service down to 0, before we're able to
	// destroy it.
	if _, err := p.ecs.UpdateService(&ecs.UpdateServiceInput{
		Cluster:      cluster,
		DesiredCount: aws.Int64(0),
		Service:      service,
	}); err != nil {
		if err, ok := err.(awserr.Error); ok && strings.Contains(err.Message(), "Service was not ACTIVE") {
			// If the service is not active, it was probably manually
			// removed already.
			return nil
		}
		if err, ok := err.(awserr.Error); ok && strings.Contains(err.Message(), "Service not found") {
			// If the service is not active, it was probably manually
			// removed already.
			return nil
		}
		return fmt.Errorf("error scaling service to 0: %v", err)
	}

	if _, err := p.ecs.DeleteService(&ecs.DeleteServiceInput{
		Service: service,
		Cluster: cluster,
	}); err != nil {
		return fmt.Errorf("error deleting service: %v", err)
	}

	return nil
}

type PortMapping struct {
	ContainerPort *customresources.IntValue
	HostPort      *customresources.IntValue
}

type Ulimit struct {
	Name      *string
	HardLimit *customresources.IntValue
	SoftLimit *customresources.IntValue
}

type ContainerDefinition struct {
	Name             *string
	Command          []*string
	Cpu              *customresources.IntValue
	Image            *string
	Essential        *string
	Memory           *customresources.IntValue
	PortMappings     []PortMapping
	DockerLabels     map[string]*string
	Ulimits          []Ulimit
	Environment      []string
	LogConfiguration *ecs.LogConfiguration
}

// TaskDefinitionProperties are properties passed to the
// Custom::ECSTaskDefinition custom resource.
type ECSTaskDefinitionProperties struct {
	Family               *string
	TaskRoleArn          *string
	ContainerDefinitions []ContainerDefinition
	PlacementConstraints []ECSPlacementConstraint
	Tags                 []*ecs.Tag
}

func (p *ECSTaskDefinitionProperties) ReplacementHash() (uint64, error) {
	return hashstructure.Hash(p, nil)
}

// ECSTaskDefinitionResource is a custom resource that provisions ECS task
// definitions.
type ECSTaskDefinitionResource struct {
	ecs              ecsClient
	environmentStore environmentStore
}

func newECSTaskDefinitionProvisioner(resource *ECSTaskDefinitionResource) *provisioner {
	return &provisioner{
		properties: func() properties {
			return &ECSTaskDefinitionProperties{}
		},
		Create: resource.Create,
		Delete: resource.Delete,
	}
}

func (p *ECSTaskDefinitionResource) Create(ctx context.Context, req customresources.Request) (string, interface{}, error) {
	properties := req.ResourceProperties.(*ECSTaskDefinitionProperties)
	id, err := p.register(properties, req.Hash())
	return id, nil, err
}

func (p *ECSTaskDefinitionResource) Delete(ctx context.Context, req customresources.Request) error {
	return p.delete(req.PhysicalResourceId)
}

func (p *ECSTaskDefinitionResource) resolvedEnvironment(ids ...string) ([]*ecs.KeyValuePair, error) {
	var env []*ecs.KeyValuePair
	for _, id := range ids {
		e, err := p.environmentStore.fetch(id)
		if err != nil {
			return nil, err
		}
		env = append(env, e...)
	}
	return env, nil
}

func (p *ECSTaskDefinitionResource) register(properties *ECSTaskDefinitionProperties, postfix string) (string, error) {
	var containerDefinitions []*ecs.ContainerDefinition
	for _, c := range properties.ContainerDefinitions {
		var (
			ulimits      []*ecs.Ulimit
			portMappings []*ecs.PortMapping
			essential    *bool
		)

		env, err := p.resolvedEnvironment(c.Environment...)
		if err != nil {
			return "", err
		}

		for _, u := range c.Ulimits {
			ulimits = append(ulimits, &ecs.Ulimit{
				Name:      u.Name,
				HardLimit: u.HardLimit.Value(),
				SoftLimit: u.SoftLimit.Value(),
			})
		}

		for _, m := range c.PortMappings {
			portMappings = append(portMappings, &ecs.PortMapping{
				ContainerPort: m.ContainerPort.Value(),
				HostPort:      m.HostPort.Value(),
			})
		}

		if c.Essential != nil {
			essential = aws.Bool(*c.Essential == "true")
		}

		containerDefinitions = append(containerDefinitions, &ecs.ContainerDefinition{
			Name:             c.Name,
			Command:          c.Command,
			Cpu:              c.Cpu.Value(),
			Image:            c.Image,
			Essential:        essential,
			Memory:           c.Memory.Value(),
			PortMappings:     portMappings,
			DockerLabels:     c.DockerLabels,
			Ulimits:          ulimits,
			LogConfiguration: c.LogConfiguration,
			Environment:      env,
		})
	}

	var family *string
	if properties.Family != nil {
		family = aws.String(fmt.Sprintf("%s-%s", *properties.Family, postfix))
	}

	var placementConstraints []*ecs.TaskDefinitionPlacementConstraint
	for _, v := range properties.PlacementConstraints {
		placementConstraints = append(placementConstraints, &ecs.TaskDefinitionPlacementConstraint{
			Type:       v.Type,
			Expression: v.Expression,
		})
	}

	resp, err := p.ecs.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
		Family:               family,
		TaskRoleArn:          properties.TaskRoleArn,
		ContainerDefinitions: containerDefinitions,
		PlacementConstraints: placementConstraints,
		Tags:                 properties.Tags,
	})
	if err != nil {
		return "", fmt.Errorf("error creating task definition: %v", err)
	}
	return *resp.TaskDefinition.TaskDefinitionArn, nil
}

func (p *ECSTaskDefinitionResource) delete(arn string) error {
	// We're ignoring errors here because we really don't care if this
	// fails.
	p.ecs.DeregisterTaskDefinition(&ecs.DeregisterTaskDefinitionInput{
		TaskDefinition: aws.String(arn),
	})
	return nil
}

// ECSEnvironmentProperties are the properties provided to the
// Custom::ECSEnvironment custom resource.
type ECSEnvironmentProperties struct {
	Environment []*ecs.KeyValuePair
}

func (p *ECSEnvironmentProperties) ReplacementHash() (uint64, error) {
	return hashstructure.Hash(p, nil)
}

// ECSEnvironmentResource is a custom resource that takes some environment
// variables, stores them, then returns a unique identifier to represent the
// environment, which can be used with the ECSTaskDefinitionResource.
type ECSEnvironmentResource struct {
	environmentStore environmentStore
}

func newECSEnvironmentProvisioner(resource *ECSEnvironmentResource) *provisioner {
	return &provisioner{
		properties: func() properties {
			return &ECSEnvironmentProperties{}
		},
		Create: resource.Create,
		Delete: resource.Delete,
	}
}

func (p *ECSEnvironmentResource) Create(ctx context.Context, req customresources.Request) (string, interface{}, error) {
	properties := req.ResourceProperties.(*ECSEnvironmentProperties)
	id, err := p.environmentStore.store(properties.Environment)
	return id, nil, err
}

func (p *ECSEnvironmentResource) Delete(ctx context.Context, req customresources.Request) error {
	return nil
}

// environmentStore is a storage engine for storing environment variables for
// the Custom::ECSEnvironment resource.
type environmentStore interface {
	store([]*ecs.KeyValuePair) (string, error)
	fetch(string) ([]*ecs.KeyValuePair, error)
}

type ecsKeyValuePair []*ecs.KeyValuePair

func (v *ecsKeyValuePair) Scan(src interface{}) error {
	bytes, ok := src.([]byte)
	if !ok {
		return error(errors.New("Scan source was not []bytes"))
	}

	var kv ecsKeyValuePair
	if err := json.Unmarshal(bytes, &kv); err != nil {
		return err
	}
	*v = kv

	return nil
}

func (v ecsKeyValuePair) Value() (driver.Value, error) {
	return json.Marshal(v)
}

// dbEnvironmentStore implements environmentStore on top of a sql.DB.
type dbEnvironmentStore struct {
	db *sql.DB
}

func (s *dbEnvironmentStore) store(env []*ecs.KeyValuePair) (string, error) {
	sql := `INSERT INTO ecs_environment (environment) VALUES ($1) RETURNING id`
	var id string
	err := s.db.QueryRow(sql, ecsKeyValuePair(env)).Scan(&id)
	return id, err
}

func (s *dbEnvironmentStore) fetch(id string) ([]*ecs.KeyValuePair, error) {
	sql := `SELECT environment FROM ecs_environment WHERE id = $1 LIMIT 1`
	var env ecsKeyValuePair
	err := s.db.QueryRow(sql, id).Scan(&env)
	return env, err
}

func primaryDeployment(service *ecs.Service) *ecs.Deployment {
	for _, d := range service.Deployments {
		if d.Status != nil && *d.Status == "PRIMARY" {
			return d
		}
	}
	return nil
}

func deploymentConfiguration(props *ECSServiceProperties) (*ecs.DeploymentConfiguration, error) {
	// If a deployment configuration was passed, make sure it's valid.
	if properties.DeploymentConfiguration == nil {
		return nil, nil
	}

	// If the MaximumPercent is missing, it isn't valid.
	if properties.DeploymentConfiguration.MaximumPercent == nil {
		return nil, fmt.Errorf("Invalid DeploymentConfiguration: MaximumPercent missing")
	}

	// If the MinimumHealthyPercent is missing, it isn't valid.
	if properties.DeploymentConfiguration.MinimumHealthyPercent == nil {
		return nil, fmt.Errorf("Invalid DeploymentConfiguration: MinimumHealthyPercent missing")
	}

	// Make a DeploymentConfiguration, using the Value() method to convert our
	// custom IntValue (which is actually an int64 parsed from a string) to an
	// *int64.
	return &ecs.DeploymentConfiguration{
		MaximumPercent:        properties.DeploymentConfiguration.MaximumPercent.Value(),
		MinimumHealthyPercent: properties.DeploymentConfiguration.MinimumHealthyPercent.Value(),
	}, nil
}

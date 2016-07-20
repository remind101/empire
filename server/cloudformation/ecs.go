package cloudformation

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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

type LoadBalancer struct {
	ContainerName    *string
	ContainerPort    *customresources.IntValue
	LoadBalancerName *string
}

// ECSServiceProperties represents the properties for the Custom::ECSService
// resource.
type ECSServiceProperties struct {
	ServiceName    *string
	Cluster        *string
	DesiredCount   *customresources.IntValue
	LoadBalancers  []LoadBalancer
	Role           *string
	TaskDefinition *string
}

// ECSServiceResource is a Provisioner that creates and updates ECS services.
type ECSServiceResource struct {
	ecs ecsClient
}

func (p *ECSServiceResource) Properties() interface{} {
	return &ECSServiceProperties{}
}

func (p *ECSServiceResource) Provision(ctx context.Context, req customresources.Request) (string, interface{}, error) {
	properties := req.ResourceProperties.(*ECSServiceProperties)
	oldProperties := req.OldResourceProperties.(*ECSServiceProperties)
	data := make(map[string]string)

	switch req.RequestType {
	case customresources.Create:
		id, deploymentId, err := p.create(ctx, req.Hash(), properties)
		if err == nil {
			data["DeploymentId"] = deploymentId
		}
		return id, data, err
	case customresources.Delete:
		id := req.PhysicalResourceId
		err := p.delete(ctx, aws.String(id), properties.Cluster)
		return id, nil, err
	case customresources.Update:
		id := req.PhysicalResourceId

		// TODO: Update this to use hashstructure.
		if serviceRequiresReplacement(properties, oldProperties) {
			// If we can't update the service, we'll need to create a new
			// one, and destroy the old one.
			oldId := id
			id, deploymentId, err := p.create(ctx, req.Hash(), properties)
			if err != nil {
				return oldId, nil, err
			}
			data["DeploymentId"] = deploymentId

			// There's no need to delete the old service here, since
			// CloudFormation will send us a DELETE request for the old
			// service.

			return id, data, err
		}

		resp, err := p.ecs.UpdateService(&ecs.UpdateServiceInput{
			Service:        aws.String(id),
			Cluster:        properties.Cluster,
			DesiredCount:   properties.DesiredCount.Value(),
			TaskDefinition: properties.TaskDefinition,
		})
		if err == nil {
			d := primaryDeployment(resp.Service)
			if d != nil {
				data["DeploymentId"] = *d.Id
			} else {
				err = fmt.Errorf("no primary deployment found")
			}
		}
		return id, data, err
	default:
		return "", nil, fmt.Errorf("%s is not supported", req.RequestType)
	}
}

func (p *ECSServiceResource) create(ctx context.Context, clientToken string, properties *ECSServiceProperties) (string, string, error) {
	var loadBalancers []*ecs.LoadBalancer
	for _, v := range properties.LoadBalancers {
		loadBalancers = append(loadBalancers, &ecs.LoadBalancer{
			ContainerName:    v.ContainerName,
			ContainerPort:    v.ContainerPort.Value(),
			LoadBalancerName: v.LoadBalancerName,
		})
	}

	var serviceName *string
	if properties.ServiceName != nil {
		serviceName = aws.String(fmt.Sprintf("%s-%s", *properties.ServiceName, clientToken))
	}

	resp, err := p.ecs.CreateService(&ecs.CreateServiceInput{
		ClientToken:    aws.String(clientToken),
		ServiceName:    serviceName,
		Cluster:        properties.Cluster,
		DesiredCount:   properties.DesiredCount.Value(),
		Role:           properties.Role,
		TaskDefinition: properties.TaskDefinition,
		LoadBalancers:  loadBalancers,
	})
	if err != nil {
		return "", "", fmt.Errorf("error creating service: %v", err)
	}

	d := primaryDeployment(resp.Service)
	if d == nil {
		return "", "", fmt.Errorf("no primary deployment found")
	}

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
		return *arn, *d.Id, ctx.Err()
	}

	return *arn, *d.Id, nil
}

func (p *ECSServiceResource) delete(ctx context.Context, service, cluster *string) error {
	// We have to scale the service down to 0, before we're able to
	// destroy it.
	if _, err := p.ecs.UpdateService(&ecs.UpdateServiceInput{
		Service:      service,
		Cluster:      cluster,
		DesiredCount: aws.Int64(0),
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
		Update: resource.Update,
		Delete: resource.Delete,
	}
}

func (p *ECSTaskDefinitionResource) Create(ctx context.Context, req customresources.Request) (string, interface{}, error) {
	properties := req.ResourceProperties.(*ECSTaskDefinitionProperties)
	id, err := p.register(properties, req.Hash())
	return id, nil, err
}

func (p *ECSTaskDefinitionResource) Update(ctx context.Context, req customresources.Request) (interface{}, error) {
	// Updates of ECSTaskDefinition will generate a replacement resource, so
	// if we've reached this point, it means that the environment is the
	// same as it was before.
	return nil, nil
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
	resp, err := p.ecs.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
		Family:               family,
		TaskRoleArn:          properties.TaskRoleArn,
		ContainerDefinitions: containerDefinitions,
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
	Environment []*ecs.KeyValuePair `hash:"set"`
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
		Update: resource.Update,
		Delete: resource.Delete,
	}
}

func (p *ECSEnvironmentResource) Create(ctx context.Context, req customresources.Request) (string, interface{}, error) {
	properties := req.ResourceProperties.(*ECSEnvironmentProperties)
	id, err := p.environmentStore.store(properties.Environment)
	return id, nil, err
}

func (p *ECSEnvironmentResource) Update(ctx context.Context, req customresources.Request) (interface{}, error) {
	// Updates of ECSEnvironment will generate a replacement resource, so if
	// we've reached this point, it means that the environment is the same
	// as it was before.
	return nil, nil
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

// Certain parameters cannot be updated on existing services, so we need to
// create a new physical resource.
func serviceRequiresReplacement(new, old *ECSServiceProperties) bool {
	eq := reflect.DeepEqual

	if !eq(new.Cluster, old.Cluster) {
		return true
	}

	if !eq(new.Role, old.Role) {
		return true
	}

	if !eq(new.ServiceName, old.ServiceName) {
		return true
	}

	if !eq(new.LoadBalancers, old.LoadBalancers) {
		return true
	}

	return false
}

func primaryDeployment(service *ecs.Service) *ecs.Deployment {
	for _, d := range service.Deployments {
		if d.Status != nil && *d.Status == "PRIMARY" {
			return d
		}
	}
	return nil
}

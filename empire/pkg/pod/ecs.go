package pod

import (
	"fmt"
	"strings"
	"sync"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ecs"
)

// ECSManager is an implementation of the Manager interface that can use ECS to
// create and schedule Templates.
type ECSManager struct {
	client *ecs.ECS

	mu sync.Mutex
	// Maps a templateID to a task definition.
	definitions map[string]*ecs.TaskDefinition
}

func NewECSManager() *ECSManager {
	return &ECSManager{
		client:      ecs.New(aws.DefaultConfig),
		definitions: make(map[string]*ecs.TaskDefinition),
	}
}

func (m *ECSManager) Submit(templates ...*Template) error {
	for _, t := range templates {
		if err := m.submit(t); err != nil {
			return err
		}
	}

	return nil
}

func (m *ECSManager) submit(t *Template) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	td, err := m.createTaskDefinition(t)
	if err != nil {
		return err
	}

	m.definitions[t.ID] = td

	if err := m.Scale(t.ID, t.Instances); err != nil {
		return err
	}

	return nil
}

func (m *ECSManager) createTaskDefinition(t *Template) (*ecs.TaskDefinition, error) {
	family := ecsFamily(t.ID)

	var command []*string
	for _, s := range strings.Split(t.Command, " ") {
		command = append(command, &s)
	}

	var environment []*ecs.KeyValuePair
	for k, v := range t.Env {
		environment = append(environment, &ecs.KeyValuePair{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}

	resp, err := m.client.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
		Family: &family,
		ContainerDefinitions: []*ecs.ContainerDefinition{
			&ecs.ContainerDefinition{
				Name:        aws.String(ecsContainerName(t.ID)), // TODO
				CPU:         aws.Long(1024),
				Command:     command,
				Image:       aws.String(t.Image.Repo + ":" + t.Image.ID),
				Essential:   aws.Boolean(true),
				Memory:      aws.Long(128), // TODO Use MemoryLimit
				Environment: environment,
			},
		},
	})
	if err != nil {
		fmt.Println(resp)
		fmt.Println(err)
		return nil, err
	}

	return resp.TaskDefinition, nil
}

func (m *ECSManager) Scale(templateID string, instances uint) error {
	if td, ok := m.definitions[templateID]; ok {
		resp, err := m.client.UpdateService(&ecs.UpdateServiceInput{
			Cluster:        aws.String("default"), // TODO
			DesiredCount:   aws.Long(int64(instances)),
			Service:        aws.String(ecsService(templateID)), // TODO
			TaskDefinition: aws.String(fmt.Sprintf("%s:%d", *td.Family, *td.Revision)),
		})
		if err != nil {
			if err, ok := err.(aws.APIError); ok {
				if err.Message == "Service was not ACTIVE." {
					resp, err := m.client.CreateService(&ecs.CreateServiceInput{
						Cluster:        aws.String("default"), // TODO
						DesiredCount:   aws.Long(int64(instances)),
						ServiceName:    aws.String(ecsService(templateID)), // TODO
						TaskDefinition: aws.String(fmt.Sprintf("%s:%d", *td.Family, *td.Revision)),
					})
					if err != nil {
						fmt.Println(resp)
						fmt.Println(err)
						return err
					}
				}
			} else {
				fmt.Println(resp)
				fmt.Println(err)
			}
			return err
		}
	} else {
		return fmt.Errorf("no task definition found for %s", templateID)
	}

	return nil
}

func (m *ECSManager) Templates(tags map[string]string) ([]*Template, error) {
	return nil, nil
}

func (m *ECSManager) Template(templateID string) (*Template, error) {
	return nil, nil
}

func (m *ECSManager) Instances(templateID string) ([]*Instance, error) {
	return nil, nil
}

func (m *ECSManager) InstanceStates(templateID string) ([]*InstanceState, error) {
	return nil, nil
}

func (m *ECSManager) Restart(*Instance) error {
	return nil
}

// acme-inc.1.web.1 => acme-inc-web
func ecsFamily(templateID string) string {
	parts := strings.Split(templateID, ".")
	return fmt.Sprintf("%s-%s", parts[0], parts[2])
}

// acme-inc.1.web.1 => web
func ecsContainerName(templateID string) string {
	parts := strings.Split(templateID, ".")
	return parts[2]
}

// acme-inc.1.web.1 => acme-inc-web
func ecsService(templateID string) string {
	return ecsFamily(templateID)
}

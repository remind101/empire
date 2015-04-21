package pod

import (
	"errors"
	"fmt"

	"github.com/remind101/empire/empire/pkg/container"
)

var (
	// ErrNoTemplate is returned when a template is not found.
	ErrNoTemplate = errors.New("template does not exist")
)

// Manager is an interface for interacting with Templates and
// Instances.
type Manager interface {
	// Submit submits Templates.
	Submit(...*Template) error

	// Destroy destroys a Template.
	Destroy(...*Template) error

	// Scale scales a Template.
	Scale(templateID string, instances uint) error

	// Templates returns a slice of Templates. A map of tags can be provided to filter
	// by.
	Templates(tags map[string]string) ([]*Template, error)

	// Template returns a single Template by it's ID.
	Template(templateID string) (*Template, error)

	// Instances returns Instances of a Template.
	Instances(templateID string) ([]*Instance, error)

	// InstanceStates returns a slice of InstanceStates for the templateID.
	InstanceStates(templateID string) ([]*InstanceState, error)

	// Restart removes and reschedules an Instance.
	Restart(*Instance) error
}

// ContainerManager is a Manager implementation backed by a
// container scheduler.
type ContainerManager struct {
	// scheduler is the Scheduler that will be used to schedule containers
	// onto the cluster.
	scheduler container.Scheduler

	// store is the store that will be used to persist state.
	store Store
}

// NewContainerManager returns a new ContainerManager backed by the scheduler
// and store.
func NewContainerManager(scheduler container.Scheduler, store Store) *ContainerManager {
	return &ContainerManager{
		scheduler: scheduler,
		store:     store,
	}
}

// Submit will store each template, then schedule a new Instance
// using the scheduler.
func (m *ContainerManager) Submit(templates ...*Template) error {
	for _, template := range templates {
		if err := m.submit(template); err != nil {
			return err
		}
	}

	return nil
}

// Destroy will destroy the Templates and unschedule any containers.
func (m *ContainerManager) Destroy(templates ...*Template) error {
	for _, template := range templates {
		if err := m.destroy(template); err != nil {
			return err
		}
	}

	return nil
}

// Scale scales the template to the desired number of instances.
func (m *ContainerManager) Scale(templateID string, instances uint) error {
	template, err := m.store.Template(templateID)
	if err != nil {
		return err
	}

	if template == nil {
		return ErrNoTemplate
	}

	// The previous number of instances that were desired.
	running := template.Instances

	switch {
	case instances < running: // scale down
		for i := uint(running); i > instances; i-- {
			if err := m.removeInstance(NewInstance(template, i)); err != nil {
				return err
			}
		}
	case instances > running: // scale up
		for i := uint(running + 1); i <= instances; i++ {
			if err := m.createInstance(NewInstance(template, i)); err != nil {
				return err
			}
		}
	default:
		return nil
	}

	// Update the template to match the new desired number of instances.
	template.Instances = instances

	return m.store.UpdateTemplate(template)
}

// Restart removes and reschedules an Instance.
func (m *ContainerManager) Restart(instance *Instance) error {
	if err := m.removeInstance(instance); err != nil {
		return err
	}
	if err := m.createInstance(instance); err != nil {
		return err
	}

	return nil
}

func (m *ContainerManager) Templates(tags map[string]string) ([]*Template, error) {
	return m.store.Templates(tags)
}

func (m *ContainerManager) Template(id string) (*Template, error) {
	return m.store.Template(id)
}

func (m *ContainerManager) Instances(templateID string) ([]*Instance, error) {
	return m.store.Instances(templateID)
}

// InstanceState gets the state of all running containers, filters them to only
// the containers associated with the template, and returns a slice of
// InstanceStates.
func (m *ContainerManager) InstanceStates(templateID string) ([]*InstanceState, error) {
	instances, err := m.store.Instances(templateID)
	if err != nil {
		return nil, err
	}

	containers, err := m.containerStates()
	if err != nil {
		return nil, err
	}

	var states []*InstanceState

	for _, instance := range instances {
		name := containerName(instance)
		state := containers[name]

		states = append(states, newInstanceState(instance, state))
	}

	return states, nil
}

// containerStates returns a of container name to container state.
func (m *ContainerManager) containerStates() (map[string]*container.ContainerState, error) {
	mp := make(map[string]*container.ContainerState)

	states, err := m.scheduler.ContainerStates()
	if err != nil {
		return mp, err
	}

	for _, state := range states {
		mp[state.Container.Name] = state
	}

	return mp, nil
}

// submit submits a single template.
func (m *ContainerManager) submit(template *Template) error {
	if err := m.createTemplate(template); err != nil {
		return err
	}

	for i := uint(1); i <= template.Instances; i++ {
		if err := m.createInstance(NewInstance(template, i)); err != nil {
			return err
		}
	}

	return nil
}

// destroy destroys the template and removes any instances of it.
func (m *ContainerManager) destroy(template *Template) error {
	instances, err := m.store.Instances(template.ID)
	if err != nil {
		return err
	}

	for _, instance := range instances {
		if err := m.removeInstance(instance); err != nil {
			return err
		}
	}

	return m.store.RemoveTemplate(template)
}

// createTemplate creates a template by persisting it to the store.
func (m *ContainerManager) createTemplate(template *Template) error {
	return m.store.CreateTemplate(template)
}

// createInstance schedules the container onto a host and creates an Instance in
// the store.
func (m *ContainerManager) createInstance(instance *Instance) error {
	if err := m.scheduler.Schedule(NewContainer(instance)); err != nil {
		return err
	}

	return m.store.CreateInstance(instance)
}

// removeInstance removes a running instance.
func (m *ContainerManager) removeInstance(instance *Instance) error {
	if err := m.scheduler.Unschedule(containerName(instance)); err != nil {
		return err
	}

	return m.store.RemoveInstance(instance)
}

// containerName returns a container.Container name for an Instance. The
// convention is to append the instance number to the end of the template ID.
// So:
//
//	acme-inc.v1.web
//
// Becomes:
//
//	acme-inc.v1.web.1
//	acme-inc.v1.web.2
//	acme-inc.v1.web.3
func containerName(instance *Instance) string {
	return fmt.Sprintf("%s.%d", instance.Template.ID, instance.Instance)
}

// newContainer takes an Instance and converts it to a container.Container.
func NewContainer(instance *Instance) *container.Container {
	t := instance.Template

	return &container.Container{
		Name:    containerName(instance),
		Env:     t.Env,
		Command: t.Command,
		Image: container.Image{
			Repo: t.Image.Repo,
			ID:   t.Image.ID,
		},
		MemoryLimit: t.MemoryLimit,
	}
}

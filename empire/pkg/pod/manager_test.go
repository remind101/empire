package pod

import (
	"reflect"
	"testing"

	"github.com/remind101/empire/empire/pkg/container"
)

func TestContainerManager_Submit(t *testing.T) {
	m := newFakeContainerManager()

	if err := m.Submit(&Template{
		ID:        "app.v1.web",
		Instances: 2,
	}); err != nil {
		t.Fatal(err)
	}

	assertContainers(t, m.scheduler, []string{
		"app.v1.web.1",
		"app.v1.web.2",
	})
}

func TestContainerManager_Scale(t *testing.T) {
	id, instances := "app.v1.web", uint(3)

	template := &Template{
		ID:        id,
		Instances: instances,
	}

	tests := []struct {
		// number of instances to scale to. Starting from 3.
		instances uint

		// The expected containers to be scheduled after scaling.
		containers []string
	}{
		// Scale up
		{4, []string{"app.v1.web.1", "app.v1.web.2", "app.v1.web.3", "app.v1.web.4"}},
		{5, []string{"app.v1.web.1", "app.v1.web.2", "app.v1.web.3", "app.v1.web.4", "app.v1.web.5"}},

		// Scale down
		{2, []string{"app.v1.web.1", "app.v1.web.2"}},
		{1, []string{"app.v1.web.1"}},
		{0, []string{}},

		// Noop
		{3, []string{"app.v1.web.1", "app.v1.web.2", "app.v1.web.3"}},
	}

	for _, tt := range tests {
		m := newFakeContainerManager()

		if err := m.Submit(template); err != nil {
			t.Fatal(err)
		}

		t.Logf("Scaling to %d", tt.instances)

		if err := m.Scale(id, tt.instances); err != nil {
			t.Fatal(err)
		}

		assertContainers(t, m.scheduler, tt.containers)

		tmpl, err := m.Template(template.ID)
		if err != nil {
			t.Fatal(err)
		}

		if got, want := tmpl.Instances, tt.instances; got != want {
			t.Fatalf("Instances => %d; want %d", got, want)
		}
	}
}

func TestContainerManager_Destroy(t *testing.T) {
	m := newFakeContainerManager()
	s := m.store.(*store)

	if err := m.Submit(&Template{
		ID:        "app.v1.web",
		Instances: 1,
	}); err != nil {
		t.Fatal(err)
	}

	if len(s.instances) != 1 {
		t.Fatal("Expected container 1 instance")
	}

	if err := m.Submit(&Template{
		ID:        "app.v2.web",
		Instances: 1,
	}); err != nil {
		t.Fatal(err)
	}

	if len(s.instances) != 2 {
		t.Fatal("Expected 2 container instances")
	}

	if err := m.Destroy(&Template{
		ID: "app.v1.web",
	}); err != nil {
		t.Fatal(err)
	}

	if len(s.instances) != 1 {
		t.Fatal("Expected container 1 instance")
	}
}

func TestContainerManager_InstanceStates(t *testing.T) {
	m := newFakeContainerManager()

	if err := m.Submit(&Template{
		ID:        "app.v1.web",
		Instances: 2,
	}); err != nil {
		t.Fatal(err)
	}

	states, err := m.InstanceStates("app.v1.web")
	if err != nil {
		t.Fatal(err)
	}

	if len(states) != 2 {
		t.Fatal("Expected 2 instance states")
	}
}

func TestNewContainer(t *testing.T) {
	tests := []struct {
		in  Instance
		out container.Container
	}{
		{
			Instance{
				Template: &Template{
					ID:      "app.v1.web",
					Command: "rackup",
					Env: map[string]string{
						"RAILS_ENV": "production",
					},
					Image: Image{
						Repo: "ejholmes/acme-inc",
						ID:   "abcd",
					},
				},
				Instance: 1,
			},
			container.Container{
				Name:    "app.v1.web.1",
				Command: "rackup",
				Env: map[string]string{
					"RAILS_ENV": "production",
				},
				Image: container.Image{
					Repo: "ejholmes/acme-inc",
					ID:   "abcd",
				},
			},
		},
	}

	for _, tt := range tests {
		out := NewContainer(&tt.in)

		if got, want := out, &tt.out; !reflect.DeepEqual(got, want) {
			t.Errorf("Container => %v; want %v", got, want)
		}
	}
}

// newFakeContainerManager returns a new ContainerManager backed by a fake
// schedule and a fake store.
func newFakeContainerManager() *ContainerManager {
	sched := container.NewFakeScheduler()

	return &ContainerManager{
		scheduler: sched,
		store:     newFakeStore(),
	}
}

func assertContainers(t testing.TB, s container.Scheduler, expected []string) {
	containers, err := containerNames(s)
	if err != nil {
		t.Fatal(err)
	}

	if len(containers) == 0 && len(expected) == 0 {
		return
	}

	if got, want := containers, expected; !reflect.DeepEqual(got, want) {
		t.Errorf("Scheduled => %v; want %v", got, want)
	}
}

func containerNames(s container.Scheduler) ([]string, error) {
	var containers []string

	states, err := s.ContainerStates()
	if err != nil {
		return containers, err
	}

	for _, state := range states {
		containers = append(containers, state.Container.Name)
	}

	return containers, nil
}

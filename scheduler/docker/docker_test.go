package docker

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var ctx = context.Background()

func TestScheduler_InstancesFromAttachedRuns(t *testing.T) {
	d := new(mockDockerClient)
	s := Scheduler{
		docker: d,
	}

	d.On("ListContainers", docker.ListContainersOptions{
		Filters: map[string][]string{
			"label": []string{
				"empire.app.id=2cdc4941-e36d-4855-a0ec-51525db4a500",
				"run=attached",
			},
		},
	}).Return([]docker.APIContainers{
		{ID: "65311c2cc20d671d43118b7d42b3f02df6b48a6bb65b1c5939007214e7587b24"},
	}, nil)

	d.On("InspectContainer", "65311c2cc20d671d43118b7d42b3f02df6b48a6bb65b1c5939007214e7587b24").Return(&docker.Container{
		ID: "65311c2cc20d671d43118b7d42b3f02df6b48a6bb65b1c5939007214e7587b24",
		State: docker.State{
			Running:   true,
			StartedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
		},
		Config: &docker.Config{
			Labels: map[string]string{
				"run":                "attached",
				"empire.app.id":      "2cdc4941-e36d-4855-a0ec-51525db4a500",
				"empire.app.process": "run",
			},
			Cmd: []string{"/bin/sh"},
			Env: []string{"FOO=bar"},
		},
		HostConfig: &docker.HostConfig{
			Memory:    int64(124 * bytesize.MB),
			CPUShares: 512,
		},
	}, nil)

	instances, err := s.InstancesFromAttachedRuns(ctx, "2cdc4941-e36d-4855-a0ec-51525db4a500")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(instances))
	assert.Equal(t, &scheduler.Instance{
		Process: &scheduler.Process{
			Type:    "run",
			Command: []string{"/bin/sh"},
			Env: map[string]string{
				"FOO": "bar",
			},
			MemoryLimit: 124 * bytesize.MB,
			CPUShares:   512,
		},
		ID:        "65311c2cc20d",
		State:     "RUNNING",
		UpdatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
	}, instances[0])

	d.AssertExpectations(t)
}

func TestScheduler_Stop(t *testing.T) {
	d := new(mockDockerClient)
	s := Scheduler{
		docker: d,
	}

	d.On("InspectContainer", "container_id").Return(&docker.Container{
		ID: "container_id",
		Config: &docker.Config{
			Labels: map[string]string{
				"run": "attached",
			},
		},
	}, nil)

	d.On("StopContainer", "container_id", uint(10)).Return(nil)

	err := s.Stop(ctx, "container_id")
	assert.NoError(t, err)

	d.AssertExpectations(t)
}

func TestScheduler_Stop_ContainerNotStartedByEmpire(t *testing.T) {
	d := new(mockDockerClient)
	s := Scheduler{
		docker: d,
	}

	d.On("InspectContainer", "container_id").Return(&docker.Container{
		ID: "container_id",
		Config: &docker.Config{
			Labels: map[string]string{
			// Missing the run label
			},
		},
	}, nil)

	err := s.Stop(ctx, "container_id")
	assert.Error(t, err)

	d.AssertExpectations(t)
}

func TestAttachedScheduler_Stop_ContainerNotFound(t *testing.T) {
	w := new(mockScheduler)
	d := new(mockDockerClient)
	ds := &Scheduler{
		docker: d,
	}
	s := &AttachedScheduler{
		Scheduler:       w,
		dockerScheduler: ds,
		ShowAttached:    true,
	}

	d.On("InspectContainer", "d9ad8d2f-318d-4abd-9d58-ece9a5ca423c").Return(nil, &docker.NoSuchContainer{
		ID: "d9ad8d2f-318d-4abd-9d58-ece9a5ca423c",
	})

	w.On("Stop", "d9ad8d2f-318d-4abd-9d58-ece9a5ca423c").Return(nil)

	err := s.Stop(ctx, "d9ad8d2f-318d-4abd-9d58-ece9a5ca423c")
	assert.NoError(t, err)

	d.AssertExpectations(t)
	w.AssertExpectations(t)
}

func TestParseEnv(t *testing.T) {
	tests := []struct {
		in  []string
		out map[string]string
	}{
		{[]string{"FOO=bar"}, map[string]string{"FOO": "bar"}},
	}

	for _, tt := range tests {
		out := parseEnv(tt.in)
		assert.Equal(t, tt.out, out)
	}
}

type mockDockerClient struct {
	dockerClient
	mock.Mock
}

func (m *mockDockerClient) ListContainers(opts docker.ListContainersOptions) ([]docker.APIContainers, error) {
	args := m.Called(opts)
	return args.Get(0).([]docker.APIContainers), args.Error(1)
}

func (m *mockDockerClient) InspectContainer(id string) (*docker.Container, error) {
	args := m.Called(id)
	var container *docker.Container
	if v := args.Get(0); v != nil {
		container = v.(*docker.Container)
	}
	return container, args.Error(1)
}

func (m *mockDockerClient) StopContainer(ctx context.Context, id string, timeout uint) error {
	args := m.Called(id, timeout)
	return args.Error(0)
}

type mockScheduler struct {
	scheduler.Scheduler
	mock.Mock
}

func (m *mockScheduler) Stop(ctx context.Context, id string) error {
	args := m.Called(id)
	return args.Error(0)
}

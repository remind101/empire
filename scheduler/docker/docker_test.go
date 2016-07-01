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
				"attached-run=true",
				"empire.app.id=2cdc4941-e36d-4855-a0ec-51525db4a500",
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
				"attached-run":       "true",
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
	return args.Get(0).(*docker.Container), args.Error(1)
}

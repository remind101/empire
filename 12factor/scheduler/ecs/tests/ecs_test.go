package ecs_test

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/remind101/empire/12factor"
	"github.com/remind101/empire/12factor/scheduler/ecs"
	"github.com/remind101/empire/12factor/scheduler/ecs/raw"
	"github.com/remind101/empire/pkg/bytesize"
	"github.com/stretchr/testify/assert"
)

// manifest is our test application. This is a valid application that will be run
// with the docker daemon.
var manifest = twelvefactor.Manifest{
	App: twelvefactor.App{
		ID:      "acme",
		Name:    "acme",
		Image:   "remind101/acme-inc",
		Version: "v1",
		Env: map[string]string{
			"RAILS_ENV": "production",
		},
	},
	Processes: []twelvefactor.Process{
		{
			Name:      "web",
			Command:   []string{"acme-inc", "web"},
			CPUShares: 256,
			Memory:    10 * int(bytesize.MB),
		},
	},
}

func TestScheduler(t *testing.T) {
	s := newScheduler(t)
	defer func() {
		err := s.Remove(manifest.ID)
		assert.NoError(t, err)
	}()

	err := s.Up(manifest)
	assert.NoError(t, err)

	err = s.ScaleProcess(manifest.ID, "web", 1)
	assert.NoError(t, err)

	_, err = s.Tasks(manifest.ID)
	assert.NoError(t, err)

	err = s.Restart(manifest.ID)
	assert.NoError(t, err)
}

func newScheduler(t testing.TB) *ecs.Scheduler {
	if testing.Short() {
		t.Skip("skipping ECS scheduler integration tests in short mode")
	}

	creds := &credentials.EnvProvider{}
	if _, err := creds.Retrieve(); err != nil {
		t.Skip("Skipping ECS test because AWS_ environment variables are not present.")
	}

	config := aws.NewConfig().WithCredentials(credentials.NewCredentials(creds))

	cluster := os.Getenv("ECS_CLUSTER")

	b := raw.NewStackBuilder(session.New(config))
	b.Cluster = cluster

	s := ecs.NewScheduler(session.New(config))
	s.StackBuilder = b
	s.Cluster = cluster

	return s
}

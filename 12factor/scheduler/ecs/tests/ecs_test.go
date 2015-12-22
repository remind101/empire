package ecs_test

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/remind101/empire/12factor"
	"github.com/remind101/empire/12factor/scheduler/ecs"
	"github.com/remind101/empire/pkg/bytesize"
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
		if err := s.Remove(manifest.ID); err != nil {
			t.Fatal(err)
		}
	}()

	if err := s.Up(manifest); err != nil {
		t.Fatal(err)
	}

	if err := s.ScaleProcess(manifest.ID, "web", 1); err != nil {
		t.Fatal(err)
	}

	_, err := s.Tasks(manifest.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.ScaleProcess(manifest.ID, "web", 0); err != nil {
		t.Fatal(err)
	}
}

func newScheduler(t testing.TB) *ecs.Scheduler {
	t.Skip("TODO")

	creds := &credentials.EnvProvider{}
	if _, err := creds.Retrieve(); err != nil {
		t.Skip("Skipping ECS test because AWS_ environment variables are not present.")
	}

	config := aws.NewConfig().WithCredentials(credentials.NewCredentials(creds))
	return ecs.NewScheduler(session.New(config))
}

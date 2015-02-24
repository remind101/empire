package empire

import (
	"reflect"
	"testing"

	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/images"
	"github.com/remind101/empire/processes"
	"github.com/remind101/empire/releases"
	"github.com/remind101/empire/scheduler"
)

func TestNewJobName(t *testing.T) {
	n := newJobName("r101-api", "v1", "web", 1)

	if got, want := n, scheduler.JobName("r101-api.v1.web.1"); got != want {
		t.Fatal("newJobName => %s; want %s", got, want)
	}
}

func TestBuildJobs(t *testing.T) {
	image := images.Image{
		Repo: "remind101/r101-api",
		ID:   "1234",
	}

	vars := configs.Vars{"RAILS_ENV": "production"}

	f := processes.ProcessMap{
		"web": &processes.Process{
			Quantity: 2,
			Command:  "./bin/web",
		},
	}

	jobs := buildJobs("r101-api", "v1", image, vars, f)

	expected := []*scheduler.Job{
		{
			Name: "r101-api.v1.web.1",
			Environment: map[string]string{
				"RAILS_ENV": "production",
			},
			Execute: scheduler.Execute{
				Command: "./bin/web",
				Image:   image,
			},
		},
		{
			Name: "r101-api.v1.web.2",
			Environment: map[string]string{
				"RAILS_ENV": "production",
			},
			Execute: scheduler.Execute{
				Command: "./bin/web",
				Image:   image,
			},
		},
	}

	if got, want := jobs, expected; !reflect.DeepEqual(got, want) {
		t.Fatalf("buildJobs => %v; want %v", got, want)
	}
}

type mockManager struct {
	Manager // Just to satisfy the interface.

	ScheduleReleaseFunc func(*releases.Release) error
}

func (m *mockManager) ScheduleRelease(release *releases.Release) error {
	if m.ScheduleReleaseFunc != nil {
		return m.ScheduleReleaseFunc(release)
	}

	return nil
}

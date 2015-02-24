package empire

import (
	"reflect"
	"testing"

	"github.com/remind101/empire/scheduler"
)

func TestNewJobName(t *testing.T) {
	n := newJobName("r101-api", "v1", "web", 1)

	if got, want := n, scheduler.JobName("r101-api.v1.web.1"); got != want {
		t.Fatal("newJobName => %s; want %s", got, want)
	}
}

func TestBuildJobs(t *testing.T) {
	image := Image{
		Repo: "remind101/r101-api",
		ID:   "1234",
	}

	vars := Vars{"RAILS_ENV": "production"}

	f := ProcessMap{
		"web": &Process{
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
				Image: scheduler.Image{
					Repo: string(image.Repo),
					ID:   image.ID,
				},
			},
		},
		{
			Name: "r101-api.v1.web.2",
			Environment: map[string]string{
				"RAILS_ENV": "production",
			},
			Execute: scheduler.Execute{
				Command: "./bin/web",
				Image: scheduler.Image{
					Repo: string(image.Repo),
					ID:   image.ID,
				},
			},
		},
	}

	if got, want := jobs, expected; !reflect.DeepEqual(got, want) {
		t.Fatalf("buildJobs => %v; want %v", got, want)
	}
}

type mockManager struct {
	Manager // Just to satisfy the interface.

	ScheduleReleaseFunc func(*Release) error
}

func (m *mockManager) ScheduleRelease(release *Release) error {
	if m.ScheduleReleaseFunc != nil {
		return m.ScheduleReleaseFunc(release)
	}

	return nil
}

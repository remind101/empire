package empire

import (
	"reflect"
	"testing"

	"github.com/remind101/empire/empire/pkg/container"
)

func TestNewContainerName(t *testing.T) {
	n := newContainerName("r101-api", 1, "web", 1)

	if got, want := n, "r101-api.1.web.1"; got != want {
		t.Fatalf("newContainerName => %s; want %s", got, want)
	}
}

func TestBuildJobs(t *testing.T) {
	image := Image{
		Repo: "remind101/r101-api",
		ID:   "1234",
	}

	vars := Vars{"RAILS_ENV": "production"}

	f := Formation{
		"web": &Process{
			Quantity: 2,
			Command:  "./bin/web",
		},
	}

	jobs := buildJobs("r101-api", 1, image, vars, f)

	expected := []*Job{
		{
			AppName:        "r101-api",
			ReleaseVersion: 1,
			ProcessType:    "web",
			Instance:       1,
			Environment: Vars{
				"RAILS_ENV": "production",
			},
			Image:   image,
			Command: "./bin/web",
		},
		{
			AppName:        "r101-api",
			ReleaseVersion: 1,
			ProcessType:    "web",
			Instance:       2,
			Environment: Vars{
				"RAILS_ENV": "production",
			},
			Image:   image,
			Command: "./bin/web",
		},
	}

	if got, want := jobs, expected; !reflect.DeepEqual(got, want) {
		t.Fatalf("buildJobs => %v; want %v", got, want)
	}
}

type mockScheduler struct {
	ScheduleFunc        func(...*container.Container) error
	UnscheduleFunc      func(...string) error
	ContainerStatesFunc func() ([]*container.ContainerState, error)
}

func (s *mockScheduler) Schedule(containers ...*container.Container) error {
	if s.ScheduleFunc != nil {
		return s.ScheduleFunc(containers...)
	}

	return nil
}

func (s *mockScheduler) Unschedule(names ...string) error {
	if s.UnscheduleFunc != nil {
		s.UnscheduleFunc(names...)
	}

	return nil
}

func (s *mockScheduler) ContainerStates() ([]*container.ContainerState, error) {
	if s.ContainerStatesFunc != nil {
		return s.ContainerStatesFunc()
	}

	return nil, nil
}

type mockManager struct {
	Manager // Just to satisfy the interface.

	ScheduleReleaseFunc func(*Release, *Config, *Slug, Formation) error
}

func (m *mockManager) ScheduleRelease(release *Release, config *Config, slug *Slug, formation Formation) error {
	if m.ScheduleReleaseFunc != nil {
		return m.ScheduleReleaseFunc(release, config, slug, formation)
	}

	return nil
}

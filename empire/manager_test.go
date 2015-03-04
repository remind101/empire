package empire

import (
	"reflect"
	"testing"

	"github.com/remind101/empire/empire/scheduler"
)

func TestNewJobName(t *testing.T) {
	n := newJobName("r101-api", 1, "web", 1)

	if got, want := n, scheduler.JobName("r101-api.1.web.1"); got != want {
		t.Fatalf("newJobName => %s; want %s", got, want)
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
	ScheduleFunc   func(*scheduler.Job) error
	UnscheduleFunc func(scheduler.JobName) error
	JobStatesFunc  func() ([]*scheduler.JobState, error)
}

func (s *mockScheduler) Schedule(j *scheduler.Job) error {
	if s.ScheduleFunc != nil {
		return s.ScheduleFunc(j)
	}

	return nil
}

func (s *mockScheduler) Unschedule(n scheduler.JobName) error {
	if s.UnscheduleFunc != nil {
		s.UnscheduleFunc(n)
	}

	return nil
}

func (s *mockScheduler) JobStates() ([]*scheduler.JobState, error) {
	if s.JobStatesFunc != nil {
		return s.JobStatesFunc()
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

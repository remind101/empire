package empire

import (
	"reflect"
	"testing"

	"github.com/remind101/empire/scheduler"
)

func TestManagerScheduleRelease(t *testing.T) {
	var (
		scheduled bool
		added     bool
	)

	s := &mockScheduler{
		ScheduleFunc: func(j *scheduler.Job) error {
			scheduled = true
			return nil
		},
	}
	r := &mockJobsRepository{
		AddFunc: func(j *Job) error {
			added = true
			return nil
		},
	}
	m := &manager{
		Scheduler:      s,
		JobsRepository: r,
	}

	release := &Release{
		Version: 1,
		App: &App{
			Name: "r101-api",
		},
		Config: &Config{
			Vars: Vars{
				"RAILS_ENV": "production",
			},
		},
		Slug: &Slug{
			Image: &Image{
				Repo: "remind101/r101-api",
				ID:   "1234",
			},
		},
		Formation: Formation{
			"web": &Process{
				Quantity: 1,
				Command:  "./bin/web",
			},
		},
	}

	if err := m.ScheduleRelease(release); err != nil {
		t.Fatal(err)
	}

	if got, want := scheduled, true; got != want {
		t.Fatal("Expected the release to be scheduled")
	}

	if got, want := added, true; got != want {
		t.Fatal("Expected the job to be added to the list of scheduled jobs")
	}
}

func TestManagerScheduleReleaseScaleDown(t *testing.T) {
	var unscheduled bool

	s := &mockScheduler{
		UnscheduleFunc: func(n scheduler.JobName) error {
			unscheduled = true

			if got, want := n, scheduler.JobName("r101-api.1.web.2"); got != want {
				t.Fatalf("Job name => %s; want %s", got, want)
			}

			return nil
		},
	}
	r := &mockJobsRepository{
		ListFunc: func(q JobQuery) ([]*Job, error) {
			jobs := []*Job{
				{
					App:         "r101-api",
					Release:     1,
					ProcessType: "web",
					Instance:    1,
				},
				{
					App:         "r101-api",
					Release:     1,
					ProcessType: "web",
					Instance:    2,
				},
			}

			return jobs, nil
		},
	}
	m := &manager{
		Scheduler:      s,
		JobsRepository: r,
	}

	release := &Release{
		Version: 1,
		App: &App{
			Name: "r101-api",
		},
		Config: &Config{
			Vars: Vars{
				"RAILS_ENV": "production",
			},
		},
		Slug: &Slug{
			Image: &Image{
				Repo: "remind101/r101-api",
				ID:   "1234",
			},
		},
		Formation: Formation{
			"web": &Process{
				Quantity: 1,
				Command:  "./bin/web",
			},
		},
	}

	if err := m.ScheduleRelease(release); err != nil {
		t.Fatal(err)
	}

	if got, want := unscheduled, true; got != want {
		t.Fatal("Expected a job to have been unscheduled")
	}
}

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
			App:         "r101-api",
			Release:     1,
			ProcessType: "web",
			Instance:    1,
			Environment: Vars{
				"RAILS_ENV": "production",
			},
			Image:   image,
			Command: "./bin/web",
		},
		{
			App:         "r101-api",
			Release:     1,
			ProcessType: "web",
			Instance:    2,
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

type mockJobsRepository struct {
	AddFunc    func(*Job) error
	RemoveFunc func(*Job) error
	ListFunc   func(JobQuery) ([]*Job, error)
}

func (r *mockJobsRepository) Add(j *Job) error {
	if r.AddFunc != nil {
		return r.AddFunc(j)
	}

	return nil
}

func (r *mockJobsRepository) Remove(j *Job) error {
	if r.RemoveFunc != nil {
		return r.RemoveFunc(j)
	}

	return nil
}

func (r *mockJobsRepository) List(q JobQuery) ([]*Job, error) {
	if r.ListFunc != nil {
		return r.ListFunc(q)
	}

	return nil, nil
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

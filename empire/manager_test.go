package empire

import (
	"reflect"
	"testing"

	"github.com/remind101/empire/empire/scheduler"
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
	p := &mockProcessesRepository{}

	m := &manager{
		Scheduler: s,
		JobsService: &jobsService{
			JobsRepository: r,
			Scheduler:      s,
		},
		ProcessesRepository: p,
	}

	config := &Config{
		ID: "1",
	}

	slug := &Slug{
		ID: "1",
	}

	formation := Formation{
		"web": &Process{
			Quantity: 1,
			Command:  "./bin/web",
		},
	}

	release := &Release{
		Ver:      1,
		AppName:  "r101-api",
		ConfigID: config.ID,
		SlugID:   slug.ID,
	}

	if err := m.ScheduleRelease(release, config, slug, formation); err != nil {
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
	var updated bool

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
					AppName:        "r101-api",
					ReleaseVersion: 1,
					ProcessType:    "web",
					Instance:       1,
				},
				{
					AppName:        "r101-api",
					ReleaseVersion: 1,
					ProcessType:    "web",
					Instance:       2,
				},
			}

			return jobs, nil
		},
	}
	p := &mockProcessesRepository{
		UpdateFunc: func(p *Process) (int64, error) {
			updated = true
			return 1, nil
		},
	}

	m := &manager{
		Scheduler: s,
		JobsService: &jobsService{
			JobsRepository: r,
			Scheduler:      s,
		},
		ProcessesRepository: p,
	}

	config := &Config{
		ID: "1",
	}

	slug := &Slug{
		ID: "1",
	}

	formation := Formation{
		"web": &Process{
			Quantity: 2,
			Command:  "./bin/web",
		},
	}

	quantityMap := ProcessQuantityMap{"web": 1}

	release := &Release{
		Ver:      1,
		AppName:  "r101-api",
		ConfigID: config.ID,
		SlugID:   slug.ID,
	}

	if err := m.ScaleRelease(release, config, slug, formation, quantityMap); err != nil {
		t.Fatal(err)
	}

	if got, want := unscheduled, true; got != want {
		t.Fatal("Expected a job to have been unscheduled")
	}

	if got, want := updated, true; got != want {
		t.Fatal("Expected formation to have been updated")
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

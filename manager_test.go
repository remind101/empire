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
		ScheduleFunc: func(c *scheduler.Container) error {
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

	s := &mockScheduler{
		UnscheduleFunc: func(n scheduler.ContainerName) error {
			unscheduled = true

			if got, want := n, scheduler.ContainerName("r101-api.1.web.2"); got != want {
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
	m := &manager{
		Scheduler:      s,
		JobsRepository: r,
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
}

func TestNewContainerName(t *testing.T) {
	n := newContainerName("r101-api", 1, "web", 1)

	if got, want := n, scheduler.ContainerName("r101-api.1.web.1"); got != want {
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
	ScheduleFunc        func(*scheduler.Container) error
	UnscheduleFunc      func(scheduler.ContainerName) error
	ContainerStatesFunc func() ([]*scheduler.ContainerState, error)
}

func (s *mockScheduler) Schedule(c *scheduler.Container) error {
	if s.ScheduleFunc != nil {
		return s.ScheduleFunc(c)
	}

	return nil
}

func (s *mockScheduler) Unschedule(n scheduler.ContainerName) error {
	if s.UnscheduleFunc != nil {
		s.UnscheduleFunc(n)
	}

	return nil
}

func (s *mockScheduler) ContainerStates() ([]*scheduler.ContainerState, error) {
	if s.ContainerStatesFunc != nil {
		return s.ContainerStatesFunc()
	}

	return nil, nil
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

	ScheduleReleaseFunc func(*Release, *Config, *Slug, Formation) error
}

func (m *mockManager) ScheduleRelease(release *Release, config *Config, slug *Slug, formation Formation) error {
	if m.ScheduleReleaseFunc != nil {
		return m.ScheduleReleaseFunc(release, config, slug, formation)
	}

	return nil
}

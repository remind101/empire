package scheduler

import "sync"

// FakeScheduler is a fake implementation of a scheduler
type FakeScheduler struct {
	sync.Mutex
	jobs JobMap
}

func NewFakeScheduler() *FakeScheduler {
	return &FakeScheduler{jobs: JobMap{}}
}

func (f *FakeScheduler) Schedule(j *Job) error {
	f.Lock()
	defer f.Unlock()

	f.jobs[j.Name] = *j
	return nil
}

func (f *FakeScheduler) Unschedule(j *Job) error {
	f.Lock()
	defer f.Unlock()

	delete(f.jobs, j.Name)
	return nil
}

func (f *FakeScheduler) Jobs(q *Query) (JobMap, error) {
	f.Lock()
	defer f.Unlock()

	if q == nil {
		return f.jobs, nil
	}

	res := JobMap{}
	for _, j := range f.jobs {
		for key, val := range q.Meta {
			if v, found := j.Meta[key]; found && v == val {
				res[j.Name] = j
			}
		}
	}

	return res, nil
}

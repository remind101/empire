package empire

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

package container

var _ Scheduler = &FakeScheduler{}

// FakeScheduler is an in memory Scheduler implementation.
type FakeScheduler struct {
	containers []*Container
}

func NewFakeScheduler() *FakeScheduler {
	return &FakeScheduler{}
}

func (s *FakeScheduler) Schedule(containers ...*Container) error {
	for _, container := range containers {
		s.containers = append(s.containers, container)
	}

	return nil
}

func (s *FakeScheduler) Unschedule(names ...string) error {
	for i, container := range s.containers {
		for _, name := range names {
			if container.Name == name {
				s.containers = append(s.containers[:i], s.containers[i+1:]...)
			}
		}
	}

	return nil
}

func (s *FakeScheduler) ContainerStates() ([]*ContainerState, error) {
	var states []*ContainerState

	for _, container := range s.containers {
		states = append(states, &ContainerState{
			Container: &Container{
				Name: container.Name,
			},
			State: "running",
		})
	}

	return states, nil
}

package container

var _ Scheduler = &FakeScheduler{}

type FakeScheduler struct{}

func NewFakeScheduler() *FakeScheduler {
	return &FakeScheduler{}
}

func (s *FakeScheduler) Schedule(_ ...*Container) error {
	return nil
}

func (s *FakeScheduler) Unschedule(_ ...string) error {
	return nil
}

func (s *FakeScheduler) ContainerStates() ([]*ContainerState, error) {
	return make([]*ContainerState, 0), nil
}

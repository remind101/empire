package container

import "testing"

func TestFake(t *testing.T) {
	s := &FakeScheduler{}

	if err := s.Schedule(&Container{
		Name: "web",
	}, &Container{
		Name: "worker",
	}); err != nil {
		t.Fatal(err)
	}

	if len(s.containers) != 2 {
		t.Fatal("expected 2 containers")
	}

	if err := s.Unschedule("web"); err != nil {
		t.Fatal(err)
	}

	if len(s.containers) != 1 {
		t.Fatal("expected 1 container")
	}

	if got, want := s.containers[0].Name, "worker"; got != want {
		t.Fatal("wrong container was removed")
	}
}

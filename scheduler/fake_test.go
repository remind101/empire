package scheduler

import (
	"reflect"
	"testing"

	"github.com/remind101/empire/manager"
)

func TestFakeScheduler(t *testing.T) {
	s := NewFakeScheduler()
	jm := JobMap{
		"api.web.1": Job{
			Name:    "api.web.1",
			Execute: manager.Execute{Command: "./bin/web"},
			Meta:    map[string]string{"app": "api", "type": "web"},
		},
		"api.web.2": Job{
			Name:    "api.web.2",
			Execute: manager.Execute{Command: "./bin/web"},
			Meta:    map[string]string{"app": "api", "type": "web"},
		},
		"dash.web.1": Job{
			Name:    "dash.web.1",
			Execute: manager.Execute{Command: "./bin/web"},
			Meta:    map[string]string{"app": "dash", "type": "web"},
		},
	}

	j1 := jm["api.web.1"]
	j2 := jm["api.web.2"]
	j3 := jm["dash.web.1"]

	s.Schedule(&j1)
	s.Schedule(&j2)
	s.Schedule(&j3)

	// A simple query should return both jobs
	jobs, _ := s.Jobs(nil)
	if want, got := jm, jobs; !reflect.DeepEqual(want, got) {
		t.Errorf("s.Jobs(nil) => %v; want %v", got, want)
	}

	// Query based on meta data
	want := JobMap{
		"dash.web.1": Job{
			Name:    "dash.web.1",
			Execute: manager.Execute{Command: "./bin/web"},
			Meta:    map[string]string{"app": "dash", "type": "web"},
		},
	}

	jobs, _ = s.Jobs(&Query{Meta: map[string]string{"app": "dash"}})
	if want, got := want, jobs; !reflect.DeepEqual(want, got) {
		t.Errorf("s.Jobs(Query) => %v; want %v", got, want)
	}

	// Unschedule
	s.Unschedule(&j1)
	s.Unschedule(&j2)
	s.Unschedule(&j3)

	jobs, _ = s.Jobs(nil)
	if want, got := (JobMap{}), jobs; !reflect.DeepEqual(want, got) {
		t.Errorf("s.Jobs(nil) => %v; want %v", got, want)
	}

}

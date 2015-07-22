package tugboat

import (
	"errors"
	"testing"
)

func TestDeploymentStarted(t *testing.T) {
	d := &Deployment{}
	d.Started("fake")

	if got, want := d.Status, StatusStarted; got != want {
		t.Fatalf("Status => %s; want %s", got, want)
	}

	if got, want := d.Provider, "fake"; got != want {
		t.Fatalf("Provider => %s; want %s", got, want)
	}

	if d.StartedAt == nil {
		t.Fatalf("expected StartedAt to not be nil")
	}
}

func TestDeploymentFailed(t *testing.T) {
	d := &Deployment{Status: StatusPending}
	d.Failed()

	if got, want := d.Status, StatusFailed; got != want {
		t.Fatalf("Status => %s; want %s", got, want)
	}

	if got, want := d.prevStatus, StatusPending; got != want {
		t.Fatalf("prevStatus => %s; want %s", got, want)
	}

	if d.CompletedAt == nil {
		t.Fatalf("expected CompletedAt to not be nil")
	}
}

func TestDeploymentSucceeded(t *testing.T) {
	d := &Deployment{Status: StatusPending}
	d.Succeeded()

	if got, want := d.Status, StatusSucceeded; got != want {
		t.Fatalf("Status => %s; want %s", got, want)
	}

	if got, want := d.prevStatus, StatusPending; got != want {
		t.Fatalf("prevStatus => %s; want %s", got, want)
	}

	if d.CompletedAt == nil {
		t.Fatalf("expected CompletedAt to not be nil")
	}
}

func TestDeploymentErrored(t *testing.T) {
	d := &Deployment{Status: StatusPending}
	d.Errored(errors.New("boom"))

	if got, want := d.Error, "boom"; got != want {
		t.Fatalf("Error => %s; want %s", got, want)
	}

	if got, want := d.Status, StatusErrored; got != want {
		t.Fatalf("Status => %s; want %s", got, want)
	}

	if got, want := d.prevStatus, StatusPending; got != want {
		t.Fatalf("prevStatus => %s; want %s", got, want)
	}

	if d.CompletedAt == nil {
		t.Fatalf("expected CompletedAt to not be nil")
	}
}

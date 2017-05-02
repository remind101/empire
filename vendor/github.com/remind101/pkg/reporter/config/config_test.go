package config

import (
	"testing"

	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/hb2"
	"github.com/stvp/rollbar"
)

func TestNewReporterFromConfigString(t *testing.T) {
	configs := []string{
		"hb://api.honeybadger.io/?key=hbkey&environment=hbenv",
		"rollbar://api.rollbar.com/?key=rollbarkey&environment=rollbarenv"}
	rep, err := NewReporterFromUrls(configs)
	multiRep := rep.(reporter.MultiReporter)
	if err != nil {
		t.Fatalf("error parsing urls: %#v", err)
	}
	reps := []reporter.Reporter(multiRep)
	if len(reps) != 2 {
		t.Fatalf("expected two reporters, got %d", len(reps))
	}

	hbReporter := reps[0].(*hb2.HbReporter)
	hbConfig := hbReporter.GetConfig()
	if got, want := hbConfig.APIKey, "hbkey"; got != want {
		t.Errorf("got %#v, but wanted %#v", got, want)
	}
	if got, want := hbConfig.Env, "hbenv"; got != want {
		t.Errorf("got %#v, but wanted %#v", got, want)
	}

	if got, want := rollbar.Token, "rollbarkey"; got != want {
		t.Errorf("got %#v, but wanted %#v", got, want)
	}
	if got, want := rollbar.Environment, "rollbarenv"; got != want {
		t.Errorf("got %#v, but wanted %#v", got, want)
	}
}

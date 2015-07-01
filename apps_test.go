package empire

import (
	"testing"
)

func TestIsValid(t *testing.T) {
	tests := []struct {
		app App
		err error
	}{
		{App{}, ErrInvalidName},
		{App{Name: "api"}, nil},
		{App{Name: "r101-api"}, nil},
	}

	for _, tt := range tests {
		if err := tt.app.IsValid(); err != tt.err {
			t.Fatalf("%v.IsValid() => %v; want %v", tt.app, err, tt.err)
		}
	}
}

func TestAppsQuery(t *testing.T) {
	id := "1234"
	name := "acme-inc"
	repo := "remind101/acme-inc"

	tests := scopeTests{
		{AppsQuery{}, "", []interface{}{}},
		{AppsQuery{ID: &id}, "WHERE (id = $1)", []interface{}{id}},
		{AppsQuery{Name: &name}, "WHERE (name = $1)", []interface{}{name}},
		{AppsQuery{Repo: &repo}, "WHERE (repo = $1)", []interface{}{repo}},
		{AppsQuery{Name: &name, Repo: &repo}, "WHERE (name = $1) AND (repo = $2)", []interface{}{name, repo}},
	}

	tests.Run(t)
}

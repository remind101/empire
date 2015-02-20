package configs

import (
	"testing"

	"github.com/remind101/empire/apps"
)

func TestRepository(t *testing.T) {
	r := newRepository()
	app := &apps.App{Name: "abcd"}

	c, _ := r.Push(&Config{App: app})
	if h, _ := r.Head(app.Name); h != c {
		t.Fatal("Head => %q; want %q", h, c)
	}

	if v, _ := r.Version(app.Name, c.Version); v != c {
		t.Fatal("Version(%s) => %q; want %q", c.Version, v, c)
	}
}

func TestHash(t *testing.T) {
	tests := []struct {
		in  Vars
		out string
	}{
		// Simple
		{
			Vars{"RAILS_ENV": "production"},
			"20f3b833ad1f83353b1ae1d24ea6833693ce067c",
		},

		// More
		{
			Vars{"RAILS_ENV": "production", "FOO": "bar"},
			"e74293df4e696ca0247c3508456712a8541b826c",
		},

		// Swapped
		{
			Vars{"FOO": "bar", "RAILS_ENV": "production"},
			"e74293df4e696ca0247c3508456712a8541b826c",
		},
	}

	for _, tt := range tests {
		if got, want := hash(tt.in), tt.out; got != want {
			t.Errorf("hash(%q) => %s; want %s", tt.in, got, want)
		}
	}
}

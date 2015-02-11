package configs

import (
	"reflect"
	"testing"

	"github.com/remind101/empire/repos"
)

func TestService_Apply(t *testing.T) {
	repo := repos.Repo("remind101/r101-api")
	s := &Service{
		Repository: newRepository(),
	}

	tests := []struct {
		in  Vars
		out *Config
	}{
		{
			Vars{
				"RAILS_ENV": "production",
			},
			&Config{
				Version: "20f3b833ad1f83353b1ae1d24ea6833693ce067c",
				Repo:    repo,
				Vars: Vars{
					"RAILS_ENV": "production",
				},
			},
		},
		{
			Vars{
				"RAILS_ENV":    "production",
				"DATABASE_URL": "postgres://localhost",
			},
			&Config{
				Version: "94a8e2be1e57b07526fee99473255a619563d551",
				Repo:    repo,
				Vars: Vars{
					"RAILS_ENV":    "production",
					"DATABASE_URL": "postgres://localhost",
				},
			},
		},
		{
			Vars{
				"RAILS_ENV": "",
			},
			&Config{
				Version: "aaa6f356d1507b0f5e14bb9adfddbea04d2569eb",
				Repo:    repo,
				Vars: Vars{
					"DATABASE_URL": "postgres://localhost",
				},
			},
		},
	}

	for _, tt := range tests {
		c, err := s.Apply(repo, tt.in)

		if err != nil {
			t.Fatal(err)
		}

		if got, want := c, tt.out; !reflect.DeepEqual(got, want) {
			t.Errorf("want %q; got %q", want, got)
		}
	}
}

func TestRepository(t *testing.T) {
	r := newRepository()
	repo := repos.Repo("r101-api")

	c, _ := r.Push(repo, &Config{})
	if h, _ := r.Head(repo); h != c {
		t.Fatal("Head => %q; want %q", h, c)
	}

	if v, _ := r.Version(repo, c.Version); v != c {
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

package api_test

import (
	"reflect"
	"testing"

	"github.com/remind101/empire/pkg/heroku"
	"github.com/remind101/empire"
)

func TestConfigVarUpdate(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	mustAppCreate(t, c, empire.App{
		Name: "acme-inc",
	})

	env := "production"
	v := mustConfigVarUpdate(t, c, "acme-inc", map[string]*string{
		"RAILS_ENV": &env,
	})

	expected := map[string]string{
		"RAILS_ENV": "production",
	}

	if got, want := v, expected; !reflect.DeepEqual(got, want) {
		t.Fatalf("Config => %v; want %v", got, want)
	}
}

func TestConfigVarUpdateDelete(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	mustAppCreate(t, c, empire.App{
		Name: "acme-inc",
	})

	env := "production"
	mustConfigVarUpdate(t, c, "acme-inc", map[string]*string{
		"RAILS_ENV": &env,
	})

	v := mustConfigVarUpdate(t, c, "acme-inc", map[string]*string{
		"RAILS_ENV": nil,
	})

	expected := map[string]string{}

	if got, want := v, expected; !reflect.DeepEqual(got, want) {
		t.Fatalf("Config => %v; want %v", got, want)
	}
}

func TestConfigVarInfo(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	mustAppCreate(t, c, empire.App{
		Name: "acme-inc",
	})

	env := "production"
	mustConfigVarUpdate(t, c, "acme-inc", map[string]*string{
		"RAILS_ENV": &env,
	})

	v := mustConfigVarInfo(t, c, "acme-inc")

	expected := map[string]string{
		"RAILS_ENV": "production",
	}

	if got, want := v, expected; !reflect.DeepEqual(got, want) {
		t.Fatalf("Config => %v; want %v", got, want)
	}
}

func mustConfigVarUpdate(t testing.TB, c *heroku.Client, appName string, options map[string]*string) map[string]string {
	vars, err := c.ConfigVarUpdate(appName, options)
	if err != nil {
		t.Fatal(err)
	}

	return vars
}

func mustConfigVarInfo(t testing.TB, c *heroku.Client, appName string) map[string]string {
	vars, err := c.ConfigVarInfo(appName)
	if err != nil {
		t.Fatal(err)
	}

	return vars
}

package empire_test

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/remind101/empire"
	client "github.com/remind101/empire/client/empire"
)

type TestClient struct {
	*client.Service
	Server *httptest.Server
	T      testing.TB
}

func NewTestClient(t testing.TB) *TestClient {
	opts := empire.Options{
		DB: "postgres://localhost/empire?sslmode=disable",
	}

	e, err := empire.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	if err := e.Reset(); err != nil {
		t.Fatal(err)
	}

	s := httptest.NewServer(empire.NewServer(e))
	c := client.NewService(nil)
	c.URL = s.URL

	return &TestClient{
		Service: c,
		Server:  s,
		T:       t,
	}
}

func (c *TestClient) Close() {
	c.Server.Close()
}

func TestEmpireDeploy(t *testing.T) {
	c := NewTestClient(t)
	defer c.Close()

	d := mustDeploy(t, c, empire.Image{
		Repo: "remind101/r101-api",
		ID:   "1234",
	})

	if d.Release.ID == "" {
		t.Fatal("Expected a release id")
	}
}

func TestEmpirePatchConfig(t *testing.T) {
	c := NewTestClient(t)
	defer c.Close()

	a := mustAppCreate(t, c, empire.App{
		Name: "api",
		Repo: "remind101/r101-api",
	})

	vars := map[string]*string{"RAILS_ENV": client.String("production")}
	config, err := c.ConfigVarUpdate(a.Name, vars)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]string{"RAILS_ENV": "production"}
	if got, want := config, expected; !reflect.DeepEqual(got, want) {
		t.Fatalf("Vars => %q; want %q", got, want)
	}
}

func TestEmpireScaleProcess(t *testing.T) {
	c := NewTestClient(t)
	defer c.Close()

	mustDeploy(t, c, empire.Image{
		Repo: "remind101/r101-api",
		ID:   "1234",
	})

	o := client.FormationUpdateOpts{
		Updates: []struct {
			Process  string  `json:"process" url:"process,key"`
			Quantity float64 `json:"quantity" url:"quantity,key"`
		}{
			{"web", 2},
		},
	}

	_, err := c.FormationUpdate("r101-api", o)
	if err != nil {
		t.Fatal(err)
	}
}

func mustAppCreate(t testing.TB, c *TestClient, app empire.App) *client.App {
	o := client.AppCreateOpts{}
	o.Name = string(app.Name)
	o.Repo = string(app.Repo)

	a, err := c.AppCreate(o)
	if err != nil {
		t.Fatal(err)
	}

	return a
}

func mustDeploy(t testing.TB, c *TestClient, image empire.Image) *client.Deploy {
	o := client.DeployCreateOpts{}
	o.Image.ID = image.ID
	o.Image.Repo = string(image.Repo)

	d, err := c.DeployCreate(o)
	if err != nil {
		t.Fatal(err)
	}

	return d
}

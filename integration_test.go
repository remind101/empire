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
	e, err := empire.New(empire.DefaultOptions)
	if err != nil {
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

func (c *TestClient) MustAppCreate(repo string) *client.App {
	o := client.AppCreateOpts{}
	o.Repo = repo
	a, err := c.AppCreate(o)
	if err != nil {
		c.T.Fatal(err)
	}
	return a
}

func TestEmpireDeploy(t *testing.T) {
	c := NewTestClient(t)
	defer c.Close()

	o := client.DeployCreateOpts{}
	o.Image.ID = "1234"
	o.Image.Repo = "remind101/r101-api"
	_, err := c.DeployCreate(o)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEmpirePatchConfig(t *testing.T) {
	c := NewTestClient(t)
	defer c.Close()

	a := c.MustAppCreate("remind101/r101-api")

	vars := map[string]string{"RAILS_ENV": "production"}
	o := client.ConfigUpdateOpts{}
	o.Vars = vars
	config, err := c.ConfigUpdate(a.ID, o)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := config.Vars, vars; !reflect.DeepEqual(got, want) {
		t.Fatalf("Vars => %q; want %q", got, want)
	}
}

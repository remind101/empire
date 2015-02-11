package empire_test

import (
	"net/http/httptest"
	"testing"

	"github.com/remind101/empire"
	client "github.com/remind101/empire/client/empire"
)

func TestEmpireDeploy(t *testing.T) {
	e := empire.New()
	s := httptest.NewServer(empire.NewServer(e))
	defer s.Close()

	c := client.NewService(nil)
	c.URL = s.URL

	o := client.DeployCreateOpts{}
	o.Image.ID = "1234"
	o.Image.Repo = "remind101/r101-api"
	_, err := c.DeployCreate(o)
	if err != nil {
		t.Fatal(err)
	}
}

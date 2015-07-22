package github_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/remind101/tugboat"
	"github.com/remind101/tugboat/pkg/hooker"
	"github.com/remind101/tugboat/provider/fake"
	"github.com/remind101/tugboat/tugboattest"
)

// Run the tests with tugboattest.Run, which will lock access to the database
// since it can't be shared by parallel tests.
func TestMain(m *testing.M) {
	tugboattest.Run(m)
}

func TestDeployment(t *testing.T) {
	c, tug, s := newTestClient(t)
	defer s.Close()

	raw := deploymentPayload(t, "ok")
	_, err := c.Trigger("deployment", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}

	ch := make(chan *tugboat.Deployment)

	go func() {
		for {
			<-time.After(1 * time.Second)

			ds, err := tug.Deployments(tugboat.DeploymentsQuery{Limit: 30})
			if err != nil {
				t.Fatal(err)
			}

			if len(ds) != 0 {
				d := ds[0]

				t.Logf("Status: %s", d.Status)

				if d.Status == tugboat.StatusSucceeded {
					ch <- d
					break
				}
			}
		}
	}()

	select {
	case d := <-ch:
		out, err := tug.Logs(d)
		if err != nil {
			t.Fatal(err)
		}

		if got, want := out, fake.DefaultScenarios["Ok"].Logs; got != want {
			t.Fatalf("Logs => %s; want %s", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timedout")
	}
}

// newTestClient will return a new heroku.Client that's configured to interact
// with a instance of the empire HTTP server.
func newTestClient(t testing.TB) (*hooker.Client, *tugboat.Tugboat, *httptest.Server) {
	tug := tugboattest.New(t)

	s := httptest.NewServer(tugboattest.NewServer(tug))
	c := hooker.NewClient(nil)
	c.URL = s.URL
	c.Secret = tugboattest.GitHubSecret

	return c, tug, s
}

func deploymentPayload(t testing.TB, fixture string) []byte {
	raw, err := ioutil.ReadFile(fmt.Sprintf("test-fixtures/deployment/%s.json", fixture))
	if err != nil {
		t.Fatal(err)
	}

	return raw
}

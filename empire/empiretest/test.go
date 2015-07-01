package empiretest

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ejholmes/flock"
	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/server"
)

var (
	// DatabaseURL is a connection string for the postgres database to use
	// during integration tests.
	DatabaseURL = "postgres://localhost/empire?sslmode=disable"
)

// NewEmpire returns a new Empire instance suitable for testing. It ensures that
// the database is clean before returning.
func NewEmpire(t testing.TB) *empire.Empire {
	opts := empire.Options{
		DB:        DatabaseURL,
		AWSConfig: nil,
		Docker: empire.DockerOptions{
			Auth: &docker.AuthConfigurations{
				Configs: map[string]docker.AuthConfiguration{
					"https://index.docker.io/v1/": docker.AuthConfiguration{
						Username: "",
						Password: "",
					},
				},
			},
		},
	}

	e, err := empire.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	if err := e.Reset(); err != nil {
		t.Fatal(err)
	}

	return e
}

// NewServer builds a new empire.Empire instance and returns an httptest.Server
// running the empire API.
func NewServer(t testing.TB, e *empire.Empire) *httptest.Server {
	if e == nil {
		e = NewEmpire(t)
	}

	return httptest.NewServer(server.New(e, server.DefaultOptions))
}

var dblock = "/tmp/empire.lock"

// Run runs testing.M after aquiring a lock against the database.
func Run(m *testing.M) {
	l, err := flock.NewPath(dblock)
	if err != nil {
		panic(err)
	}

	l.Lock()
	defer l.Unlock()

	os.Exit(m.Run())
}

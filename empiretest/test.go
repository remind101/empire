package empiretest

import (
	"io"
	"net/http/httptest"
	"os"
	"testing"

	"golang.org/x/net/context"

	"github.com/ejholmes/flock"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/procfile"
	"github.com/remind101/empire/scheduler"
	"github.com/remind101/empire/server"
	"github.com/remind101/empire/server/auth"
)

var (
	// DatabaseURL is a connection string for the postgres database to use
	// during integration tests.
	DatabaseURL = "postgres://localhost/empire?sslmode=disable"
)

// NewEmpire returns a new Empire instance suitable for testing. It ensures that
// the database is clean before returning.
func NewEmpire(t testing.TB) *empire.Empire {
	db, err := empire.NewDB(DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}

	e := empire.New(db, empire.DefaultOptions)
	e.Scheduler = scheduler.NewFakeScheduler()
	e.ExtractProcfile = ExtractProcfile

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

	server.DefaultOptions.GitHub.Webhooks.Secret = "abcd"
	server.DefaultOptions.GitHub.Deployments.Environment = "test"
	server.DefaultOptions.Authenticator = auth.Anyone(&empire.User{Name: "fake"})
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

// ExtractProcfile extracts a fake procfile.
func ExtractProcfile(ctx context.Context, img image.Image, w io.Writer) (procfile.Procfile, error) {
	return procfile.Procfile{
		"web": "./bin/web",
	}, dockerutil.FakePull(img, w)
}

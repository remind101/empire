package empiretest

import (
	"io"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"
	"text/template"

	"golang.org/x/net/context"

	"github.com/ejholmes/flock"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/procfile"
	"github.com/remind101/empire/scheduler"
	"github.com/remind101/empire/server"
	"github.com/remind101/empire/server/auth"
	"github.com/remind101/empire/server/github"
)

var (
	// DatabaseURL is a connection string for the postgres database to use
	// during integration tests.
	DatabaseURL = "postgres://localhost/empire?sslmode=disable"
)

// NewEmpire returns a new Empire instance suitable for testing. It ensures that
// the database is clean before returning.
func NewEmpire(t testing.TB) *empire.Empire {
	db, err := empire.OpenDB(DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.MigrateUp(); err != nil {
		t.Fatal(err)
	}

	// Log queries if verbose mode is set.
	if testing.Verbose() {
		db.Debug()
	}

	e := empire.New(db, empire.DefaultOptions)
	e.Scheduler = scheduler.NewFakeScheduler()
	e.ProcfileExtractor = empire.ProcfileExtractorFunc(ExtractProcfile)
	e.RunRecorder = empire.RecordTo(ioutil.Discard)

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
	server.DefaultOptions.Authenticator = auth.Anyone(&empire.User{Name: "fake"})
	server.DefaultOptions.GitHub.Deployments.Environments = []string{"test"}
	server.DefaultOptions.GitHub.Deployments.ImageBuilder = github.ImageFromTemplate(template.Must(template.New("image").Parse(github.DefaultTemplate)))
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

// ExtractProcfile extracts a fake Procfile.
func ExtractProcfile(ctx context.Context, img string, w io.Writer) ([]byte, error) {
	p, err := procfile.Marshal(procfile.ExtendedProcfile{
		"web": procfile.Process{
			Command: []string{"./bin/web"},
		},
	})
	if err != nil {
		return nil, err
	}

	return p, dockerutil.FakePull(img, w)
}

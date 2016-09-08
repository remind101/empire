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
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/procfile"
	"github.com/remind101/empire/scheduler"
	"github.com/remind101/empire/server"
	"github.com/remind101/empire/server/acl"
	"github.com/remind101/empire/server/auth"
	"github.com/remind101/empire/server/github"
	"github.com/remind101/empire/server/middleware"
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

	e := empire.New(db)
	e.Scheduler = scheduler.NewFakeScheduler()
	e.ProcfileExtractor = ExtractProcfile(nil)
	e.RunRecorder = empire.RecordTo(ioutil.Discard)

	if err := e.Reset(); err != nil {
		t.Fatal(err)
	}

	return e
}

type Server struct {
	*empire.Empire
	*httptest.Server
}

var TestPolicy = acl.Policy{
	// By default, allow all actions.
	acl.Statement{
		Effect:   acl.Allow,
		Action:   []string{"empire:*"},
		Resource: []string{"*"},
	},

	// Don't let anyone create a new app with the name denied-app-name.
	acl.Statement{
		Effect:   acl.Deny,
		Action:   []string{"empire:Create"},
		Resource: []string{"denied-app-name"},
	},
}

// NewServer builds a new empire.Empire instance and returns an httptest.Server
// running the empire API.
func NewServer(t testing.TB, e *empire.Empire) *Server {
	var opts server.Options
	opts.GitHub.Webhooks.Secret = "abcd"
	opts.Auth = &auth.Auth{
		Authenticator: auth.NewAccessTokenAuthenticator(e),
		Policy:        auth.StaticPolicy(TestPolicy),
	}
	opts.GitHub.Deployments.Environments = []string{"test"}
	opts.GitHub.Deployments.ImageBuilder = github.ImageFromTemplate(template.Must(template.New("image").Parse(github.DefaultTemplate)))
	return NewTestServer(t, e, opts)
}

// NewTestServer returns a new httptest.Server for testing empire's http server.
func NewTestServer(t testing.TB, e *empire.Empire, opts server.Options) *Server {
	if e == nil {
		e = NewEmpire(t)
	}

	s := server.New(e, opts)
	return &Server{
		Empire: e,
		Server: httptest.NewServer(middleware.Handler(context.Background(), s)),
	}
}

func (s *Server) Close() {
	s.Server.Close()
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

// defaultProcfile represents a basic Procfile, which can be used in integration
// tests.
var defaultProcfile = procfile.ExtendedProcfile{
	"web": procfile.Process{
		Command: []string{"./bin/web"},
	},
	"worker": procfile.Process{
		Command: []string{"./bin/worker"},
	},
	"scheduled": procfile.Process{
		Command: []string{"./bin/scheduled"},
		Cron: func() *string {
			everyMinute := "* * * * * *"
			return &everyMinute
		}(),
	},
	"rake": procfile.Process{
		Command:   "bundle exec rake",
		NoService: true,
	},
}

// Returns a function that can be used as a Procfile extract for Empire. It
// writes a fake Docker pull to w, and extracts the given Procfile in yaml
// format.
func ExtractProcfile(pf procfile.Procfile) empire.ProcfileExtractor {
	if pf == nil {
		pf = defaultProcfile
	}

	return empire.ProcfileExtractorFunc(func(ctx context.Context, img image.Image, w io.Writer) ([]byte, error) {
		p, err := procfile.Marshal(pf)
		if err != nil {
			return nil, err
		}

		return p, dockerutil.FakePull(img, w)
	})
}

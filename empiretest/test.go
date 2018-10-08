package empiretest

import (
	"container/list"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"text/template"

	"golang.org/x/net/context"

	"github.com/remind101/empire"
	"github.com/remind101/empire/logs"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/pkg/jsonmessage"
	"github.com/remind101/empire/pkg/timex"
	"github.com/remind101/empire/procfile"
	"github.com/remind101/empire/server"
	"github.com/remind101/empire/server/auth"
	"github.com/remind101/empire/server/github"
	"github.com/remind101/pkg/reporter"
)

// Engine is a fake in memory implementation of the empire.Engine interface.
type Engine struct {
	// Maps an app ID to a list of releases of the application.
	apps map[string]*list.List
}

func newEngine() *Engine {
	s := new(Engine)
	s.Reset()
	return s
}

func (s *Engine) AppsFind(q empire.AppsQuery) (*empire.App, error) {
	releases, ok := s.apps[*q.Name]
	if !ok {
		return nil, &empire.NotFoundError{Err: errors.New("not found")}
	}
	release := releases.Front().Value.(*empire.Release)
	return release.App, nil
}

func (s *Engine) Apps(q empire.AppsQuery) ([]*empire.App, error) {
	var apps []*empire.App
	for _, releases := range s.apps {
		release := releases.Front().Value.(*empire.Release)
		apps = append(apps, release.App)
	}
	return apps, nil
}

func (s *Engine) AppsDestroy(app *empire.App) error {
	delete(s.apps, app.Name)
	return nil
}

func (s *Engine) ReleasesCreate(app *empire.App, event empire.Event) (*empire.Release, error) {
	app.BeforeCreate()

	releases, ok := s.apps[app.Name]
	if !ok {
		releases = list.New()
		s.apps[app.Name] = releases
	}

	app2 := *app
	app2.Version = app.Version + 1
	release := &empire.Release{
		App:         &app2,
		Description: event.String(),
	}
	release.BeforeCreate()
	releases.PushFront(release)
	return release, nil
}

func (s *Engine) Releases(q empire.ReleasesQuery) ([]*empire.Release, error) {
	l, ok := s.apps[q.App.Name]
	if !ok {
		return nil, errors.New("no releases")
	}
	var releases []*empire.Release
	for e := l.Front(); e != nil; e = e.Next() {
		releases = append(releases, e.Value.(*empire.Release))
	}
	return releases, nil
}

func (s *Engine) ReleasesFind(q empire.ReleasesQuery) (*empire.Release, error) {
	releases := s.apps[q.App.Name]
	release := releases.Front().Value.(*empire.Release)
	return release, nil
}

func (s *Engine) Run(ctx context.Context, app *empire.App, stdio *empire.IO) error {
	for _, p := range app.Formation {
		if stdio != nil {
			fmt.Fprintf(stdio.Stdout, "Attaching to container\n")
			fmt.Fprintf(stdio.Stdout, "Fake output for `%s` on %s\n", p.Command, app.Name)
		}
	}
	return nil
}

func (s *Engine) Stop(ctx context.Context, taskID string) error {
	return nil
}

func (s *Engine) Tasks(ctx context.Context, app *empire.App) ([]*empire.Task, error) {
	var tasks []*empire.Task
	now := timex.Now()
	for name, p := range app.Formation {
		for i := 1; i <= p.Quantity; i++ {
			id := fmt.Sprintf("%d", i)
			tasks = append(tasks, &empire.Task{
				Name:        fmt.Sprintf("v%d.%s.%s", app.Version, name, id),
				Type:        name,
				Command:     p.Command,
				ID:          id,
				Host:        empire.Host{ID: "i-aa111aa1"},
				State:       "running",
				Constraints: p.Constraints(),
				UpdatedAt:   now,
			})
		}
	}
	return tasks, nil
}

func (s *Engine) IsHealthy() error {
	return nil
}

func (s *Engine) Reset() error {
	s.apps = make(map[string]*list.List)
	return nil
}

// Marks the test as skipped when running in CI.
func SkipCI(t testing.TB) {
	if _, ok := os.LookupEnv("CI"); ok {
		t.Skip("Skipping test in CI")
	}
}

// NewEmpire returns a new Empire instance suitable for testing. It ensures that
// the database is clean before returning.
func NewEmpire(t testing.TB) *empire.Empire {
	e := empire.New(newEngine())
	e.ImageRegistry = ExtractProcfile(nil, nil)
	e.RunRecorder = logs.RecordTo(ioutil.Discard)

	if err := e.Reset(); err != nil {
		t.Fatal(err)
	}

	return e
}

// Server wraps an Empire instance being served by the canonical server.Server
// http.Handler, for testing.
type Server struct {
	*empire.Empire
	*server.Server
	svr *httptest.Server
}

// NewServer builds a new empire.Empire instance and returns an httptest.Server
// running the Empire API.
//
// The Server is unstarted so that you can perform additional configuration.
// Consumers should call Start() before makeing any requests.
func NewServer(t testing.TB, e *empire.Empire) *Server {
	var opts server.Options
	opts.GitHub.Webhooks.Secret = "abcd"
	opts.GitHub.Deployments.Environments = []string{"test"}
	opts.GitHub.Deployments.ImageBuilder = github.ImageFromTemplate(template.Must(template.New("image").Parse(github.DefaultTemplate)))
	s := newTestServer(t, e, opts)
	s.Heroku.Auth = &auth.Auth{}
	return s
}

// newTestServer returns a new httptest.Server for testing empire's http server.
func newTestServer(t testing.TB, e *empire.Empire, opts server.Options) *Server {
	if e == nil {
		e = NewEmpire(t)
	}

	s := server.New(e, opts)
	h := func(w http.ResponseWriter, r *http.Request) {
		// Log reporter errors to stderr
		ctx := reporter.WithReporter(r.Context(), reporter.ReporterFunc(func(ctx context.Context, err error) error {
			fmt.Fprintf(os.Stderr, "reported error: %v\n", err)
			return nil
		}))
		s.ServeHTTP(w, r.WithContext(ctx))
	}
	svr := httptest.NewUnstartedServer(http.HandlerFunc(h))
	u, _ := url.Parse(svr.URL)
	s.URL = u
	return &Server{
		Empire: e,
		Server: s,
		svr:    svr,
	}
}

// URL returns that URL that this Empire server is (or will be) located.
func (s *Server) URL() string {
	if s.svr.URL == "" {
		return "http://" + s.svr.Listener.Addr().String()
	}
	return s.svr.URL
}

// Start starts the underlying httptest.Server.
func (s *Server) Start() {
	s.svr.Start()
}

// Close closes the underlying httptest.Server.
func (s *Server) Close() {
	s.svr.Close()
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

// ImageRegistry is a fake implementation of the empire.ImageRegistry interface.
type ImageRegistry struct {
	procfile   procfile.Procfile
	extractErr error
}

// ExtractProcfile returns an empire.ImageRegistry implementation that writes a
// fake Docker pull to w, and extracts the given Procfile in yaml format when
// ExtractProcfile is called.
func ExtractProcfile(pf procfile.Procfile, err error) empire.ImageRegistry {
	if pf == nil {
		pf = defaultProcfile
	}

	return &ImageRegistry{
		procfile:   pf,
		extractErr: err,
	}
}

func (r *ImageRegistry) ExtractProcfile(ctx context.Context, img image.Image, w *jsonmessage.Stream) ([]byte, error) {
	if err := r.extractErr; err != nil {
		return nil, err
	}

	p, err := procfile.Marshal(r.procfile)
	if err != nil {
		return nil, err
	}

	return p, dockerutil.FakePull(img, w)
}

func (r *ImageRegistry) Resolve(ctx context.Context, img image.Image, w *jsonmessage.Stream) (image.Image, error) {
	return img, nil
}

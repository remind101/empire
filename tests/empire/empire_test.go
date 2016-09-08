package empire_test

import (
	"errors"
	"io"
	"io/ioutil"
	"sort"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/remind101/empire"
	"github.com/remind101/empire/empiretest"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/procfile"
	"github.com/remind101/empire/scheduler"
	"github.com/remind101/empire/server/acl"
	"github.com/remind101/pkg/timex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var fakeNow = time.Date(2015, time.January, 1, 1, 1, 1, 1, time.UTC)

var ctx = acl.WithPolicies(context.Background(), empiretest.TestPolicies)

// Stubs out time.Now in empire.
func init() {
	timex.Now = func() time.Time {
		return fakeNow
	}
}

// Run the tests with empiretest.Run, which will lock access to the database
// since it can't be shared by parallel tests.
func TestMain(m *testing.M) {
	empiretest.Run(m)
}

func TestEmpire_AccessTokens(t *testing.T) {
	e := empiretest.NewEmpire(t)

	token := &empire.AccessToken{
		User: &empire.User{Name: "ejholmes"},
	}
	_, err := e.AccessTokensCreate(token)
	assert.NoError(t, err)

	token, err = e.AccessTokensFind(token.Token)
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "ejholmes", token.User.Name)

	token, err = e.AccessTokensFind("invalid")
	assert.NoError(t, err)
	assert.Nil(t, token)

	token = &empire.AccessToken{
		User: &empire.User{Name: ""},
	}
	_, err = e.AccessTokensCreate(token)
	assert.Equal(t, empire.ErrUserName, err)
}

func TestEmpire_CertsAttach(t *testing.T) {
	e := empiretest.NewEmpire(t)
	s := new(mockScheduler)
	e.Scheduler = s

	user := &empire.User{Name: "ejholmes"}

	app, err := e.Create(ctx, empire.CreateOpts{
		User: user,
		Name: "acme-inc",
	})
	assert.NoError(t, err)

	cert := "serverCertificate"
	err = e.CertsAttach(ctx, app, cert)
	assert.NoError(t, err)

	app, err = e.AppsFind(empire.AppsQuery{ID: &app.ID})
	assert.NoError(t, err)
	assert.Equal(t, cert, app.Cert)

	s.AssertExpectations(t)
}

func TestEmpire_Deploy(t *testing.T) {
	e := empiretest.NewEmpire(t)
	s := new(mockScheduler)
	e.Scheduler = s

	user := &empire.User{Name: "ejholmes"}

	app, err := e.Create(ctx, empire.CreateOpts{
		User: user,
		Name: "acme-inc",
	})
	assert.NoError(t, err)

	img := image.Image{Repository: "remind101/acme-inc"}
	s.On("Submit", &scheduler.App{
		ID:      app.ID,
		Name:    "acme-inc",
		Release: "v1",
		Env: map[string]string{
			"EMPIRE_APPID":   app.ID,
			"EMPIRE_APPNAME": "acme-inc",
			"EMPIRE_RELEASE": "v1",
		},
		Labels: map[string]string{
			"empire.app.name":    "acme-inc",
			"empire.app.id":      app.ID,
			"empire.app.release": "v1",
		},
		Processes: []*scheduler.Process{
			{
				Type:        "scheduled",
				Image:       img,
				Command:     []string{"./bin/scheduled"},
				Schedule:    scheduler.CRONSchedule("* * * * * *"),
				Instances:   0,
				MemoryLimit: 536870912,
				CPUShares:   256,
				Nproc:       256,
				Env: map[string]string{
					"EMPIRE_PROCESS":       "scheduled",
					"EMPIRE_PROCESS_SCALE": "0",
					"SOURCE":               "acme-inc.scheduled.v1",
				},
				Labels: map[string]string{
					"empire.app.process": "scheduled",
				},
			},
			{
				Type:    "web",
				Image:   img,
				Command: []string{"./bin/web"},
				Exposure: &scheduler.Exposure{
					Type: &scheduler.HTTPExposure{},
				},
				Instances:   1,
				MemoryLimit: 536870912,
				CPUShares:   256,
				Nproc:       256,
				Env: map[string]string{
					"EMPIRE_PROCESS":       "web",
					"EMPIRE_PROCESS_SCALE": "1",
					"SOURCE":               "acme-inc.web.v1",
				},
				Labels: map[string]string{
					"empire.app.process": "web",
				},
			},
			{
				Type:        "worker",
				Image:       img,
				Command:     []string{"./bin/worker"},
				Instances:   0,
				MemoryLimit: 536870912,
				CPUShares:   256,
				Nproc:       256,
				Env: map[string]string{
					"EMPIRE_PROCESS":       "worker",
					"EMPIRE_PROCESS_SCALE": "0",
					"SOURCE":               "acme-inc.worker.v1",
				},
				Labels: map[string]string{
					"empire.app.process": "worker",
				},
			},
		},
	}).Return(nil)

	_, err = e.Deploy(ctx, empire.DeployOpts{
		App:    app,
		User:   user,
		Output: empire.NewDeploymentStream(ioutil.Discard),
		Image:  img,
	})
	assert.NoError(t, err)

	s.AssertExpectations(t)
}

func TestEmpire_Deploy_ImageNotFound(t *testing.T) {
	e := empiretest.NewEmpire(t)
	s := new(mockScheduler)
	e.Scheduler = s
	e.ProcfileExtractor = empire.ProcfileExtractorFunc(func(ctx context.Context, img image.Image, w io.Writer) ([]byte, error) {
		return nil, errors.New("image not found")
	})

	// Deploying an image to an app that doesn't exist will create a new
	// app.
	_, err := e.Deploy(ctx, empire.DeployOpts{
		User:   &empire.User{Name: "ejholmes"},
		Output: empire.NewDeploymentStream(ioutil.Discard),
		Image:  image.Image{Repository: "remind101/acme-inc"},
	})
	assert.Error(t, err)

	// If there's an error deploying, then the transaction should be rolled
	// backed and no apps should exist.
	apps, err := e.Apps(empire.AppsQuery{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(apps))

	s.AssertExpectations(t)
}

func TestEmpire_Deploy_Concurrent(t *testing.T) {
	e := empiretest.NewEmpire(t)
	s := new(mockScheduler)
	e.Scheduler = scheduler.NewFakeScheduler()
	e.ProcfileExtractor = empiretest.ExtractProcfile(procfile.ExtendedProcfile{
		"web": procfile.Process{
			Command: []string{"./bin/web"},
		},
	})

	user := &empire.User{Name: "ejholmes"}

	// Create the first release for this app.
	r, err := e.Deploy(ctx, empire.DeployOpts{
		User:   user,
		Output: empire.NewDeploymentStream(ioutil.Discard),
		Image:  image.Image{Repository: "remind101/acme-inc"},
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, r.Version)

	// We'll use the procfile extractor to synchronize two concurrent
	// deployments.
	v2Started, v3Started := make(chan struct{}), make(chan struct{})
	e.ProcfileExtractor = empire.ProcfileExtractorFunc(func(ctx context.Context, img image.Image, w io.Writer) ([]byte, error) {
		switch img.Tag {
		case "v2":
			close(v2Started)
			<-v3Started
		case "v3":
			close(v3Started)
		}
		return procfile.Marshal(procfile.ExtendedProcfile{
			"web": procfile.Process{
				Command: []string{"./bin/web"},
			},
		})
	})

	v2Done := make(chan struct{})
	go func() {
		r, err := e.Deploy(ctx, empire.DeployOpts{
			User:   user,
			Output: empire.NewDeploymentStream(ioutil.Discard),
			Image:  image.Image{Repository: "remind101/acme-inc", Tag: "v2"},
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, r.Version)
		close(v2Done)
	}()

	<-v2Started

	r, err = e.Deploy(ctx, empire.DeployOpts{
		User:   user,
		Output: empire.NewDeploymentStream(ioutil.Discard),
		Image:  image.Image{Repository: "remind101/acme-inc", Tag: "v3"},
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, r.Version)

	<-v2Done

	s.AssertExpectations(t)
}

func TestEmpire_Run(t *testing.T) {
	e := empiretest.NewEmpire(t)

	user := &empire.User{Name: "ejholmes"}

	app, err := e.Create(ctx, empire.CreateOpts{
		User: user,
		Name: "acme-inc",
	})
	assert.NoError(t, err)

	img := image.Image{Repository: "remind101/acme-inc"}
	_, err = e.Deploy(ctx, empire.DeployOpts{
		App:    app,
		User:   user,
		Output: empire.NewDeploymentStream(ioutil.Discard),
		Image:  img,
	})
	assert.NoError(t, err)

	s := new(mockScheduler)
	e.Scheduler = s

	s.On("Run", &scheduler.App{
		ID:      app.ID,
		Name:    "acme-inc",
		Release: "v1",
		Env: map[string]string{
			"EMPIRE_APPID":   app.ID,
			"EMPIRE_APPNAME": "acme-inc",
			"EMPIRE_RELEASE": "v1",
		},
		Labels: map[string]string{
			"empire.app.name":    "acme-inc",
			"empire.app.id":      app.ID,
			"empire.app.release": "v1",
		},
	},
		&scheduler.Process{
			Type:        "run",
			Image:       img,
			Command:     []string{"bundle", "exec", "rake", "db:migrate"},
			Instances:   1,
			MemoryLimit: 536870912,
			CPUShares:   256,
			Nproc:       256,
			Env: map[string]string{
				"EMPIRE_PROCESS":       "run",
				"EMPIRE_PROCESS_SCALE": "1",
				"SOURCE":               "acme-inc.run.v1",
				"TERM":                 "xterm",
			},
			Labels: map[string]string{
				"empire.app.process": "run",
				"empire.user":        "ejholmes",
			},
		}, nil, nil).Return(nil)

	err = e.Run(ctx, empire.RunOpts{
		User:    user,
		App:     app,
		Command: empire.MustParseCommand("bundle exec rake db:migrate"),

		// Detached Process
		Output: nil,
		Input:  nil,

		Env: map[string]string{
			"TERM": "xterm",
		},
	})
	assert.NoError(t, err)

	s.AssertExpectations(t)
}

func TestEmpire_Run_WithConstraints(t *testing.T) {
	e := empiretest.NewEmpire(t)

	user := &empire.User{Name: "ejholmes"}

	app, err := e.Create(ctx, empire.CreateOpts{
		User: user,
		Name: "acme-inc",
	})
	assert.NoError(t, err)

	img := image.Image{Repository: "remind101/acme-inc"}
	_, err = e.Deploy(ctx, empire.DeployOpts{
		App:    app,
		User:   user,
		Output: empire.NewDeploymentStream(ioutil.Discard),
		Image:  img,
	})
	assert.NoError(t, err)

	s := new(mockScheduler)
	e.Scheduler = s

	s.On("Run", &scheduler.App{
		ID:      app.ID,
		Name:    "acme-inc",
		Release: "v1",
		Env: map[string]string{
			"EMPIRE_APPID":   app.ID,
			"EMPIRE_APPNAME": "acme-inc",
			"EMPIRE_RELEASE": "v1",
		},
		Labels: map[string]string{
			"empire.app.name":    "acme-inc",
			"empire.app.id":      app.ID,
			"empire.app.release": "v1",
		},
	},
		&scheduler.Process{
			Type:        "run",
			Image:       img,
			Command:     []string{"bundle", "exec", "rake", "db:migrate"},
			Instances:   1,
			MemoryLimit: 1073741824,
			CPUShares:   512,
			Nproc:       512,
			Env: map[string]string{
				"EMPIRE_PROCESS":       "run",
				"EMPIRE_PROCESS_SCALE": "1",
				"SOURCE":               "acme-inc.run.v1",
				"TERM":                 "xterm",
			},
			Labels: map[string]string{
				"empire.app.process": "run",
				"empire.user":        "ejholmes",
			},
		}, nil, nil).Return(nil)

	constraints := empire.NamedConstraints["2X"]
	err = e.Run(ctx, empire.RunOpts{
		User:    user,
		App:     app,
		Command: empire.MustParseCommand("bundle exec rake db:migrate"),

		// Detached Process
		Output: nil,
		Input:  nil,

		Env: map[string]string{
			"TERM": "xterm",
		},

		Constraints: &constraints,
	})
	assert.NoError(t, err)

	s.AssertExpectations(t)
}

func TestEmpire_Run_WithAllowCommandProcfile(t *testing.T) {
	e := empiretest.NewEmpire(t)
	e.AllowedCommands = empire.AllowCommandProcfile

	user := &empire.User{Name: "ejholmes"}

	app, err := e.Create(ctx, empire.CreateOpts{
		User: user,
		Name: "acme-inc",
	})
	assert.NoError(t, err)

	img := image.Image{Repository: "remind101/acme-inc"}
	_, err = e.Deploy(ctx, empire.DeployOpts{
		App:    app,
		User:   user,
		Output: empire.NewDeploymentStream(ioutil.Discard),
		Image:  img,
	})
	assert.NoError(t, err)

	s := new(mockScheduler)
	e.Scheduler = s

	err = e.Run(ctx, empire.RunOpts{
		User:    user,
		App:     app,
		Command: empire.MustParseCommand("bundle exec rake db:migrate"),

		// Detached Process
		Output: nil,
		Input:  nil,

		Env: map[string]string{
			"TERM": "xterm",
		},
	})
	assert.IsType(t, &empire.CommandNotAllowedError{}, err)

	s.On("Run", &scheduler.App{
		ID:      app.ID,
		Name:    "acme-inc",
		Release: "v1",
		Env: map[string]string{
			"EMPIRE_APPID":   app.ID,
			"EMPIRE_APPNAME": "acme-inc",
			"EMPIRE_RELEASE": "v1",
		},
		Labels: map[string]string{
			"empire.app.id":      app.ID,
			"empire.app.name":    "acme-inc",
			"empire.app.release": "v1",
		},
	},
		&scheduler.Process{
			Type:        "rake",
			Image:       img,
			Command:     []string{"bundle", "exec", "rake", "db:migrate"},
			Instances:   1,
			MemoryLimit: 536870912,
			CPUShares:   256,
			Nproc:       256,
			Env: map[string]string{
				"EMPIRE_PROCESS":       "rake",
				"EMPIRE_PROCESS_SCALE": "1",
				"SOURCE":               "acme-inc.rake.v1",
				"TERM":                 "xterm",
			},
			Labels: map[string]string{
				"empire.app.process": "rake",
				"empire.user":        "ejholmes",
			},
		}, nil, nil).Return(nil)

	err = e.Run(ctx, empire.RunOpts{
		User:    user,
		App:     app,
		Command: empire.MustParseCommand("rake db:migrate"),

		// Detached Process
		Output: nil,
		Input:  nil,

		Env: map[string]string{
			"TERM": "xterm",
		},
	})
	assert.NoError(t, err)

	s.AssertExpectations(t)
}

func TestEmpire_Set(t *testing.T) {
	e := empiretest.NewEmpire(t)
	s := new(mockScheduler)
	e.Scheduler = s
	e.ProcfileExtractor = empiretest.ExtractProcfile(procfile.ExtendedProcfile{
		"web": procfile.Process{
			Command: []string{"./bin/web"},
		},
	})

	user := &empire.User{Name: "ejholmes"}

	// Create an app
	app, err := e.Create(ctx, empire.CreateOpts{
		User: user,
		Name: "acme-inc",
	})
	assert.NoError(t, err)

	// Add some environment variables to it.
	prod := "production"
	_, err = e.Set(ctx, empire.SetOpts{
		User: user,
		App:  app,
		Vars: empire.Vars{
			"RAILS_ENV": &prod,
		},
	})
	assert.NoError(t, err)

	// Deploy a new image to the app.
	img := image.Image{Repository: "remind101/acme-inc"}
	s.On("Submit", &scheduler.App{
		ID:      app.ID,
		Name:    "acme-inc",
		Release: "v1",
		Env: map[string]string{
			"EMPIRE_APPID":   app.ID,
			"EMPIRE_APPNAME": "acme-inc",
			"EMPIRE_RELEASE": "v1",
			"RAILS_ENV":      "production",
		},
		Labels: map[string]string{
			"empire.app.name":    "acme-inc",
			"empire.app.id":      app.ID,
			"empire.app.release": "v1",
		},
		Processes: []*scheduler.Process{
			{
				Type:    "web",
				Image:   img,
				Command: []string{"./bin/web"},
				Exposure: &scheduler.Exposure{
					Type: &scheduler.HTTPExposure{},
				},
				Instances:   1,
				MemoryLimit: 536870912,
				CPUShares:   256,
				Nproc:       256,
				Env: map[string]string{
					"EMPIRE_PROCESS":       "web",
					"EMPIRE_PROCESS_SCALE": "1",
					"SOURCE":               "acme-inc.web.v1",
				},
				Labels: map[string]string{
					"empire.app.process": "web",
				},
			},
		},
	}).Once().Return(nil)

	_, err = e.Deploy(ctx, empire.DeployOpts{
		App:    app,
		User:   user,
		Output: empire.NewDeploymentStream(ioutil.Discard),
		Image:  img,
	})
	assert.NoError(t, err)

	// Remove the environment variable
	s.On("Submit", &scheduler.App{
		ID:      app.ID,
		Name:    "acme-inc",
		Release: "v2",
		Env: map[string]string{
			"EMPIRE_APPID":   app.ID,
			"EMPIRE_APPNAME": "acme-inc",
			"EMPIRE_RELEASE": "v2",
		},
		Labels: map[string]string{
			"empire.app.name":    "acme-inc",
			"empire.app.id":      app.ID,
			"empire.app.release": "v2",
		},
		Processes: []*scheduler.Process{
			{
				Type:    "web",
				Image:   img,
				Command: []string{"./bin/web"},
				Exposure: &scheduler.Exposure{
					Type: &scheduler.HTTPExposure{},
				},
				Instances:   1,
				MemoryLimit: 536870912,
				CPUShares:   256,
				Nproc:       256,
				Env: map[string]string{
					"EMPIRE_PROCESS":       "web",
					"EMPIRE_PROCESS_SCALE": "1",
					"SOURCE":               "acme-inc.web.v2",
				},
				Labels: map[string]string{
					"empire.app.process": "web",
				},
			},
		},
	}).Once().Return(nil)

	_, err = e.Set(ctx, empire.SetOpts{
		User: user,
		App:  app,
		Vars: empire.Vars{
			"RAILS_ENV": nil,
		},
	})
	assert.NoError(t, err)

	s.AssertExpectations(t)
}

type mockScheduler struct {
	scheduler.Scheduler
	mock.Mock
}

type processesByType []*scheduler.Process

func (e processesByType) Len() int           { return len(e) }
func (e processesByType) Less(i, j int) bool { return e[i].Type < e[j].Type }
func (e processesByType) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }

func (m *mockScheduler) Submit(_ context.Context, app *scheduler.App, ss scheduler.StatusStream) error {
	// mock.Mock checks the order of slices, so sort by process name.
	p := processesByType(app.Processes)
	sort.Sort(p)
	app.Processes = p

	args := m.Called(app)
	return args.Error(0)
}

func (m *mockScheduler) Run(_ context.Context, app *scheduler.App, process *scheduler.Process, in io.Reader, out io.Writer) error {
	app.Processes = nil // This is bogus and doesn't actually matter for Runs.
	args := m.Called(app, process, in, out)
	return args.Error(0)
}

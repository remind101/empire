package empire_test

import (
	"errors"
	"io"
	"io/ioutil"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/remind101/empire"
	"github.com/remind101/empire/12factor"
	"github.com/remind101/empire/empiretest"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/procfile"
	"github.com/remind101/empire/scheduler"
	"github.com/remind101/pkg/timex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var fakeNow = time.Date(2015, time.January, 1, 1, 1, 1, 1, time.UTC)

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

	app, err := e.Create(context.Background(), empire.CreateOpts{
		User: user,
		Name: "acme-inc",
	})
	assert.NoError(t, err)

	cert := "serverCertificate"
	err = e.CertsAttach(context.Background(), app, cert)
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

	app, err := e.Create(context.Background(), empire.CreateOpts{
		User: user,
		Name: "acme-inc",
	})
	assert.NoError(t, err)

	img := image.Image{Repository: "remind101/acme-inc"}
	s.On("Submit", twelvefactor.Manifest{
		App: twelvefactor.App{
			ID:    app.ID,
			Name:  "acme-inc",
			Image: img,
			Env: map[string]string{
				"EMPIRE_APPID":      app.ID,
				"EMPIRE_APPNAME":    "acme-inc",
				"EMPIRE_RELEASE":    "v1",
				"EMPIRE_CREATED_AT": "2015-01-01T01:01:01Z",
			},
			Labels: map[string]string{
				"empire.app.name":    "acme-inc",
				"empire.app.id":      app.ID,
				"empire.app.release": "v1",
			},
		},
		Processes: []twelvefactor.Process{
			{
				Type:    "web",
				Command: []string{"./bin/web"},
				Exposure: &twelvefactor.Exposure{
					Type: &twelvefactor.HTTPExposure{},
				},
				Instances:   1,
				MemoryLimit: 536870912,
				CPUShares:   256,
				Nproc:       256,
				Env: map[string]string{
					"EMPIRE_PROCESS": "web",
					"SOURCE":         "acme-inc.web.v1",
				},
				Labels: map[string]string{
					"empire.app.process": "web",
				},
			},
		},
	}).Return(nil)

	_, err = e.Deploy(context.Background(), empire.DeploymentsCreateOpts{
		App:    app,
		User:   user,
		Output: ioutil.Discard,
		Image:  img,
	})
	assert.NoError(t, err)

	s.AssertExpectations(t)
}

func TestEmpire_Deploy_ImageNotFound(t *testing.T) {
	e := empiretest.NewEmpire(t)
	s := new(mockScheduler)
	e.Scheduler = s
	e.ExtractProcfile = func(ctx context.Context, img image.Image, w io.Writer) (procfile.Procfile, error) {
		return nil, errors.New("image not found")
	}

	// Deploying an image to an app that doesn't exist will create a new
	// app.
	_, err := e.Deploy(context.Background(), empire.DeploymentsCreateOpts{
		User:   &empire.User{Name: "ejholmes"},
		Output: ioutil.Discard,
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
	e.ExtractProcfile = func(ctx context.Context, img image.Image, w io.Writer) (procfile.Procfile, error) {
		return nil, nil
	}

	user := &empire.User{Name: "ejholmes"}

	// Create the first release for this app.
	r, err := e.Deploy(context.Background(), empire.DeploymentsCreateOpts{
		User:   user,
		Output: ioutil.Discard,
		Image:  image.Image{Repository: "remind101/acme-inc"},
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, r.Version)

	// We'll use the procfile extractor to synchronize two concurrent
	// deployments.
	v2Started, v3Started := make(chan struct{}), make(chan struct{})
	e.ExtractProcfile = func(ctx context.Context, img image.Image, w io.Writer) (procfile.Procfile, error) {
		switch img.Tag {
		case "v2":
			close(v2Started)
			<-v3Started
		case "v3":
			close(v3Started)
		}
		return nil, nil
	}

	v2Done := make(chan struct{})
	go func() {
		r, err = e.Deploy(context.Background(), empire.DeploymentsCreateOpts{
			User:   user,
			Output: ioutil.Discard,
			Image:  image.Image{Repository: "remind101/acme-inc", Tag: "v2"},
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, r.Version)
		close(v2Done)
	}()

	<-v2Started

	r, err = e.Deploy(context.Background(), empire.DeploymentsCreateOpts{
		User:   user,
		Output: ioutil.Discard,
		Image:  image.Image{Repository: "remind101/acme-inc", Tag: "v3"},
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, r.Version)

	<-v2Done

	s.AssertExpectations(t)
}

func TestEmpire_Set(t *testing.T) {
	e := empiretest.NewEmpire(t)
	s := new(mockScheduler)
	e.Scheduler = s

	user := &empire.User{Name: "ejholmes"}

	// Create an app
	app, err := e.Create(context.Background(), empire.CreateOpts{
		User: user,
		Name: "acme-inc",
	})
	assert.NoError(t, err)

	// Add some environment variables to it.
	prod := "production"
	_, err = e.Set(context.Background(), empire.SetOpts{
		User: user,
		App:  app,
		Vars: empire.Vars{
			"RAILS_ENV": &prod,
		},
	})
	assert.NoError(t, err)

	// Deploy a new image to the app.
	img := image.Image{Repository: "remind101/acme-inc"}
	s.On("Submit", twelvefactor.Manifest{
		App: twelvefactor.App{
			ID:    app.ID,
			Name:  "acme-inc",
			Image: img,
			Env: map[string]string{
				"EMPIRE_APPID":      app.ID,
				"EMPIRE_APPNAME":    "acme-inc",
				"EMPIRE_RELEASE":    "v1",
				"EMPIRE_CREATED_AT": "2015-01-01T01:01:01Z",
				"RAILS_ENV":         "production",
			},
			Labels: map[string]string{
				"empire.app.name":    "acme-inc",
				"empire.app.id":      app.ID,
				"empire.app.release": "v1",
			},
		},
		Processes: []twelvefactor.Process{
			{
				Type:    "web",
				Command: []string{"./bin/web"},
				Exposure: &twelvefactor.Exposure{
					Type: &twelvefactor.HTTPExposure{},
				},
				Instances:   1,
				MemoryLimit: 536870912,
				CPUShares:   256,
				Nproc:       256,
				Env: map[string]string{
					"EMPIRE_PROCESS": "web",
					"SOURCE":         "acme-inc.web.v1",
				},
				Labels: map[string]string{
					"empire.app.process": "web",
				},
			},
		},
	}).Once().Return(nil)

	_, err = e.Deploy(context.Background(), empire.DeploymentsCreateOpts{
		App:    app,
		User:   user,
		Output: ioutil.Discard,
		Image:  img,
	})
	assert.NoError(t, err)

	// Remove the environment variable
	s.On("Submit", twelvefactor.Manifest{
		App: twelvefactor.App{
			ID:    app.ID,
			Name:  "acme-inc",
			Image: img,
			Env: map[string]string{
				"EMPIRE_APPID":      app.ID,
				"EMPIRE_APPNAME":    "acme-inc",
				"EMPIRE_RELEASE":    "v2",
				"EMPIRE_CREATED_AT": "2015-01-01T01:01:01Z",
			},
			Labels: map[string]string{
				"empire.app.name":    "acme-inc",
				"empire.app.id":      app.ID,
				"empire.app.release": "v2",
			},
		},
		Processes: []twelvefactor.Process{
			{
				Type:    "web",
				Command: []string{"./bin/web"},
				Exposure: &twelvefactor.Exposure{
					Type: &twelvefactor.HTTPExposure{},
				},
				Instances:   1,
				MemoryLimit: 536870912,
				CPUShares:   256,
				Nproc:       256,
				Env: map[string]string{
					"EMPIRE_PROCESS": "web",
					"SOURCE":         "acme-inc.web.v2",
				},
				Labels: map[string]string{
					"empire.app.process": "web",
				},
			},
		},
	}).Once().Return(nil)

	_, err = e.Set(context.Background(), empire.SetOpts{
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

func (m *mockScheduler) Submit(_ context.Context, manifest twelvefactor.Manifest) error {
	args := m.Called(manifest)
	return args.Error(0)
}

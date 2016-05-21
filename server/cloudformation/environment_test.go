package cloudformation

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/hashicorp/go-multierror"
	"github.com/remind101/empire"
	"github.com/remind101/empire/empiretest"
	"github.com/remind101/empire/pkg/image"
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

func TestProvision_Create(t *testing.T) {
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
	s.On("Submit", &scheduler.App{
		ID:   app.ID,
		Name: "acme-inc",
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

	// Add cloudformation environment
	s.On("Submit", &scheduler.App{
		ID:   app.ID,
		Name: "acme-inc",
		Env: map[string]string{
			"EMPIRE_APPID":      app.ID,
			"EMPIRE_APPNAME":    "acme-inc",
			"EMPIRE_RELEASE":    "v2",
			"EMPIRE_CREATED_AT": "2015-01-01T01:01:01Z",
			"RAILS_ENV":         "production",
			"COMPUTERS":         "woo",
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
					"EMPIRE_PROCESS": "web",
					"SOURCE":         "acme-inc.web.v2",
				},
				Labels: map[string]string{
					"empire.app.process": "web",
				},
			},
		},
	}).Once().Return(nil)

	resource := &EnvironmentResource{empire: e}
	var req Request
	err = json.Unmarshal([]byte(fmt.Sprintf(`{"RequestType": "Create", "ResourceProperties": {"AppId": "%s", "Variables": [{"Name": "COMPUTERS", "Value": "woo"}]}}`, app.ID)), &req)
	assert.NoError(t, err)

	id, _, err := resource.Provision(req)
	assert.NoError(t, err)
	assert.Equal(t, id, app.ID)
	s.AssertExpectations(t)
}

func TestProvision_Update(t *testing.T) {
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
	comp := "woo"
	_, err = e.Set(context.Background(), empire.SetOpts{
		User: user,
		App:  app,
		Vars: empire.Vars{
			"RAILS_ENV": &prod,
			"COMPUTERS": &comp,
		},
	})
	assert.NoError(t, err)

	// Deploy a new image to the app.
	img := image.Image{Repository: "remind101/acme-inc"}
	s.On("Submit", &scheduler.App{
		ID:   app.ID,
		Name: "acme-inc",
		Env: map[string]string{
			"EMPIRE_APPID":      app.ID,
			"EMPIRE_APPNAME":    "acme-inc",
			"EMPIRE_RELEASE":    "v1",
			"EMPIRE_CREATED_AT": "2015-01-01T01:01:01Z",
			"RAILS_ENV":         "production",
			"COMPUTERS":         "woo",
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

	// Add cloudformation environment
	s.On("Submit", &scheduler.App{
		ID:   app.ID,
		Name: "acme-inc",
		Env: map[string]string{
			"EMPIRE_APPID":      app.ID,
			"EMPIRE_APPNAME":    "acme-inc",
			"EMPIRE_RELEASE":    "v2",
			"EMPIRE_CREATED_AT": "2015-01-01T01:01:01Z",
			"RAILS_ENV":         "development",
			"FOO":               "bar",
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
					"EMPIRE_PROCESS": "web",
					"SOURCE":         "acme-inc.web.v2",
				},
				Labels: map[string]string{
					"empire.app.process": "web",
				},
			},
		},
	}).Once().Return(nil)

	resource := &EnvironmentResource{empire: e}
	var req Request
	err = json.Unmarshal([]byte(fmt.Sprintf(`{"RequestType": "Update", "PhysicalResourceId": "%s", "ResourceProperties": {"Variables": [{"Name": "RAILS_ENV", "Value": "development"}, {"Name": "FOO", "Value": "bar"}]}, "OldResourceProperties": {"Variables": [{"Name": "RAILS_ENV", "Value": "production"}, {"Name": "COMPUTERS", "Value": "woo"}]}}`, app.ID)), &req)
	assert.NoError(t, err)

	id, _, err := resource.Provision(req)
	assert.NoError(t, err)
	assert.Equal(t, id, app.ID)
	s.AssertExpectations(t)
}

func TestProvision_Delete(t *testing.T) {
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
	comp := "woo"
	_, err = e.Set(context.Background(), empire.SetOpts{
		User: user,
		App:  app,
		Vars: empire.Vars{
			"RAILS_ENV": &prod,
			"COMPUTERS": &comp,
		},
	})
	assert.NoError(t, err)

	// Deploy a new image to the app.
	img := image.Image{Repository: "remind101/acme-inc"}
	s.On("Submit", &scheduler.App{
		ID:   app.ID,
		Name: "acme-inc",
		Env: map[string]string{
			"EMPIRE_APPID":      app.ID,
			"EMPIRE_APPNAME":    "acme-inc",
			"EMPIRE_RELEASE":    "v1",
			"EMPIRE_CREATED_AT": "2015-01-01T01:01:01Z",
			"RAILS_ENV":         "production",
			"COMPUTERS":         "woo",
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

	// Add cloudformation environment
	s.On("Submit", &scheduler.App{
		ID:   app.ID,
		Name: "acme-inc",
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
					"EMPIRE_PROCESS": "web",
					"SOURCE":         "acme-inc.web.v2",
				},
				Labels: map[string]string{
					"empire.app.process": "web",
				},
			},
		},
	}).Once().Return(nil)

	resource := &EnvironmentResource{empire: e}
	var req Request
	err = json.Unmarshal([]byte(fmt.Sprintf(`{"RequestType": "Delete", "PhysicalResourceId": "%s", "ResourceProperties": {"Variables": [{"Name": "RAILS_ENV", "Value": "development"}, {"Name": "COMPUTERS", "Value": "woo"}]}}`, app.ID)), &req)
	assert.NoError(t, err)

	id, _, err := resource.Provision(req)
	assert.NoError(t, err)
	assert.Equal(t, id, app.ID)
	s.AssertExpectations(t)
}

func TestVarsFromRequest(t *testing.T) {
	var req Request
	err := json.Unmarshal([]byte(`{"RequestType": "Create", "ResourceProperties": {"Variables": [{"Name": "FOO", "Value": "bar"}, {"Name": "BAR", "Value": null}]}}`), &req)
	assert.NoError(t, err)

	bar := "bar"
	vars, err := varsFromRequest(req)
	assert.NoError(t, err)
	assert.Equal(t, empire.Vars{
		"FOO": &bar,
		"BAR": nil,
	}, vars)
}

func TestVarsFromRequestMissingRequiredFields(t *testing.T) {
	var req Request
	err := json.Unmarshal([]byte(`{"RequestType": "Create", "ResourceProperties": {"Variables": [{"Name": "FOO"}, {"Value": "bar"}, {}, "invalid"]}}`), &req)
	assert.NoError(t, err)

	_, err = varsFromRequest(req)
	assert.Error(t, err)
	if merr, ok := err.(*multierror.Error); ok {
		assert.Equal(t, len(merr.Errors), 5)
	}
}

func TestVarsFromUpdateRequest_DeletedVars(t *testing.T) {
	var req Request
	err := json.Unmarshal([]byte(`{"RequestType": "Update", "ResourceProperties": {"Variables": [{"Name": "FOO", "Value": "bar"}, {"Name": "BAR", "Value": null}]}, "OldResourceProperties": {"Variables": [{"Name": "FOOBAR", "Value": "foobar"}]}}`), &req)
	assert.NoError(t, err)

	bar := "bar"
	vars, err := varsFromRequest(req)
	assert.NoError(t, err)
	assert.Equal(t, empire.Vars{
		"FOO":    &bar,
		"BAR":    nil,
		"FOOBAR": nil,
	}, vars)
}

func TestVarsFromDeleteRequest(t *testing.T) {
	var req Request
	err := json.Unmarshal([]byte(`{"RequestType": "Delete", "ResourceProperties": {"Variables": [{"Name": "FOO", "Value": "bar"}, {"Name": "BAR", "Value": null}]}}`), &req)
	assert.NoError(t, err)

	vars, err := varsFromRequest(req)
	assert.NoError(t, err)
	assert.Equal(t, empire.Vars{
		"FOO": nil,
		"BAR": nil,
	}, vars)
}

type mockScheduler struct {
	scheduler.Scheduler
	mock.Mock
}

func (m *mockScheduler) Submit(_ context.Context, app *scheduler.App) error {
	args := m.Called(app)
	return args.Error(0)
}

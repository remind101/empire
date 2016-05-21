package cloudformation

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"golang.org/x/net/context"

	"github.com/remind101/empire"
	"github.com/remind101/empire/empiretest"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/scheduler"
	"github.com/stretchr/testify/assert"
)

func TestAppResourceProvision_Create(t *testing.T) {
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

func TestAppResource_Update(t *testing.T) {
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

	resource := &AppResource{empire: e}
	var req Request
	err = json.Unmarshal([]byte(fmt.Sprintf(`{"RequestType": "Update", "PhysicalResourceId": "%s", "ResourceProperties": {"Name": "renamed"}, "OldResourceProperties": {"Name": "acme-inc"}}`, app.ID)), &req)
	assert.NoError(t, err)

	_, _, err = resource.Provision(req)
	assert.EqualError(t, err, "Updates are not supported")
	s.AssertExpectations(t)
}

func TestAppResourceProvision_Delete(t *testing.T) {
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

	// Remove the app
	s.On("Remove", app.ID).Once().Return(nil)

	resource := &AppResource{empire: e}
	var req Request
	err = json.Unmarshal([]byte(fmt.Sprintf(`{"RequestType": "Delete", "PhysicalResourceId": "%s", "ResourceProperties": {"Name": "acme-inc"}}`, app.ID)), &req)
	assert.NoError(t, err)

	id, _, err := resource.Provision(req)
	assert.NoError(t, err)
	assert.Equal(t, id, app.ID)
	s.AssertExpectations(t)
}

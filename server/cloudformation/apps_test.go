package cloudformation

import (
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire"
	"github.com/stretchr/testify/assert"
)

func TestAppResourceProvision_Create(t *testing.T) {
	e := new(mockEmpire)
	user := NewUser()

	app := empire.App{
		ID:   "1234",
		Name: "acme-inc",
	}
	e.On("AppsFind", empire.AppsQuery{
		Name: &app.Name,
	}).Once().Return(&empire.App{}, gorm.RecordNotFound)
	e.On("Create", empire.CreateOpts{
		User:    user,
		Name:    app.Name,
		Message: "Creating app via Cloudformation",
	}).Once().Return(&app, nil)

	resource := &AppResource{empire: e}
	id, _, err := resource.Provision(Request{
		RequestType: Create,
		ResourceProperties: &AppProperties{
			Name: app.Name,
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, id, app.ID)
	e.AssertExpectations(t)
}

func TestAppResource_Update(t *testing.T) {
	e := new(mockEmpire)

	app := empire.App{
		ID:   "1234",
		Name: "acme-inc",
	}
	e.On("AppsFind", empire.AppsQuery{
		ID: &app.ID,
	}).Once().Return(&app, nil)

	resource := &AppResource{empire: e}
	_, _, err := resource.Provision(Request{
		RequestType:        Update,
		PhysicalResourceId: app.ID,
		ResourceProperties: &AppProperties{},
	})
	assert.EqualError(t, err, "Updates are not supported")
	e.AssertExpectations(t)
}

func TestAppResourceProvision_Delete(t *testing.T) {
	e := new(mockEmpire)
	user := NewUser()

	app := empire.App{
		ID:   "1234",
		Name: "acme-inc",
	}
	e.On("AppsFind", empire.AppsQuery{
		ID: &app.ID,
	}).Once().Return(&app, nil)
	e.On("Destroy", empire.DestroyOpts{
		User:    user,
		App:     &app,
		Message: "Destroying app via Cloudformation",
	}).Once().Return(nil)

	resource := &AppResource{empire: e}
	id, _, err := resource.Provision(Request{
		RequestType:        Delete,
		PhysicalResourceId: app.ID,
		ResourceProperties: &AppProperties{},
	})
	assert.NoError(t, err)
	assert.Equal(t, id, app.ID)
	e.AssertExpectations(t)
}

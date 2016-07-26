package cloudformation

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/jinzhu/gorm"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/cloudformation/customresources"
	"github.com/stretchr/testify/assert"
)

func TestEmpireAppResourceProvision_Create(t *testing.T) {
	e := new(mockEmpire)
	user := newUser()

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

	resource := &EmpireAppResource{empire: e}
	id, _, err := resource.Provision(ctx, customresources.Request{
		RequestType: customresources.Create,
		ResourceProperties: &AppProperties{
			Name: aws.String(app.Name),
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, id, app.ID)
	e.AssertExpectations(t)
}

func TestEmpireAppResource_Update(t *testing.T) {
	e := new(mockEmpire)

	app := empire.App{
		ID:   "1234",
		Name: "acme-inc",
	}
	e.On("AppsFind", empire.AppsQuery{
		ID: &app.ID,
	}).Once().Return(&app, nil)

	resource := &EmpireAppResource{empire: e}
	_, _, err := resource.Provision(ctx, customresources.Request{
		RequestType:        customresources.Update,
		PhysicalResourceId: app.ID,
		ResourceProperties: &AppProperties{},
	})
	assert.EqualError(t, err, "Updates are not supported")
	e.AssertExpectations(t)
}

func TestEmpireAppResourceProvision_Delete(t *testing.T) {
	e := new(mockEmpire)
	user := newUser()

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

	resource := &EmpireAppResource{empire: e}
	id, _, err := resource.Provision(ctx, customresources.Request{
		RequestType:        customresources.Delete,
		PhysicalResourceId: app.ID,
		ResourceProperties: &AppProperties{},
	})
	assert.NoError(t, err)
	assert.Equal(t, id, app.ID)
	e.AssertExpectations(t)
}

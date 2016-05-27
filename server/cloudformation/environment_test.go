package cloudformation

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/hashicorp/go-multierror"
	"github.com/remind101/empire"
	"github.com/stretchr/testify/assert"
)

func TestEnvironmentResourceProvision_Create(t *testing.T) {
	e := new(mockEmpire)
	user := NewUser()

	app := empire.App{
		ID:   "1234",
		Name: "acme-inc",
	}

	vars := empire.Vars{
		"RAILS_ENV": aws.String("production"),
	}

	e.On("AppsFind", empire.AppsQuery{
		ID: &app.ID,
	}).Once().Return(&app, nil)
	e.On("Set", empire.SetOpts{
		User:    user,
		App:     &app,
		Vars:    vars,
		Message: "Setting variables via Cloudformation",
	}).Once().Return(&empire.Config{}, nil)

	resource := &EnvironmentResource{empire: e}
	id, _, err := resource.Provision(Request{
		RequestType: Create,
		ResourceProperties: &EnvironmentProperties{
			AppId: &app.ID,
			Variables: []Variable{
				Variable{Name: aws.String("RAILS_ENV"), Value: aws.String("production")},
			},
		},
		OldResourceProperties: &EnvironmentProperties{},
	})
	assert.NoError(t, err)
	assert.Equal(t, id, app.ID)
	e.AssertExpectations(t)
}

func TestEnvironmentResourceProvision_Update(t *testing.T) {
	e := new(mockEmpire)
	user := NewUser()

	app := empire.App{
		ID:   "1234",
		Name: "acme-inc",
	}

	vars := empire.Vars{
		"RAILS_ENV": aws.String("development"),
		"FOO":       aws.String("bar"),
		"BIZ":       nil,
	}

	e.On("AppsFind", empire.AppsQuery{
		ID: &app.ID,
	}).Once().Return(&app, nil)
	e.On("Set", empire.SetOpts{
		User:    user,
		App:     &app,
		Vars:    vars,
		Message: "Updating variables via Cloudformation",
	}).Once().Return(&empire.Config{}, nil)

	resource := &EnvironmentResource{empire: e}
	id, _, err := resource.Provision(Request{
		RequestType:        Update,
		PhysicalResourceId: app.ID,
		ResourceProperties: &EnvironmentProperties{
			AppId: &app.ID,
			Variables: []Variable{
				Variable{Name: aws.String("RAILS_ENV"), Value: aws.String("development")},
				Variable{Name: aws.String("FOO"), Value: aws.String("bar")},
			},
		},
		OldResourceProperties: &EnvironmentProperties{
			AppId: &app.ID,
			Variables: []Variable{
				Variable{Name: aws.String("RAILS_ENV"), Value: aws.String("production")},
				Variable{Name: aws.String("BIZ"), Value: aws.String("buzz")},
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, id, app.ID)
	e.AssertExpectations(t)
}

func TestEnvironmentResourceProvision_Delete(t *testing.T) {
	e := new(mockEmpire)
	user := NewUser()

	app := empire.App{
		ID:   "1234",
		Name: "acme-inc",
	}

	vars := empire.Vars{
		"RAILS_ENV": nil,
		"FOO":       nil,
	}

	e.On("AppsFind", empire.AppsQuery{
		ID: &app.ID,
	}).Once().Return(&app, nil)
	e.On("Set", empire.SetOpts{
		User:    user,
		App:     &app,
		Vars:    vars,
		Message: "Unsetting variables via Cloudformation",
	}).Once().Return(&empire.Config{}, nil)

	resource := &EnvironmentResource{empire: e}
	id, _, err := resource.Provision(Request{
		RequestType:        Delete,
		PhysicalResourceId: app.ID,
		ResourceProperties: &EnvironmentProperties{
			AppId: &app.ID,
			Variables: []Variable{
				Variable{Name: aws.String("RAILS_ENV"), Value: aws.String("development")},
				Variable{Name: aws.String("FOO"), Value: aws.String("bar")},
			},
		},
		OldResourceProperties: &EnvironmentProperties{},
	})
	assert.NoError(t, err)
	assert.Equal(t, id, app.ID)
	e.AssertExpectations(t)
}

func TestVarsFromRequest(t *testing.T) {
	vars, err := varsFromRequest(Request{
		RequestType: Create,
		ResourceProperties: &EnvironmentProperties{
			Variables: []Variable{
				Variable{Name: aws.String("FOO"), Value: aws.String("bar")},
				Variable{Name: aws.String("BAR"), Value: nil},
			},
		},
		OldResourceProperties: &EnvironmentProperties{},
	})
	assert.NoError(t, err)
	assert.Equal(t, empire.Vars{
		"FOO": aws.String("bar"),
		"BAR": nil,
	}, vars)
}

func TestVarsFromRequestMissingRequiredFields(t *testing.T) {
	_, err := varsFromRequest(Request{
		RequestType: Create,
		ResourceProperties: &EnvironmentProperties{
			Variables: []Variable{
				Variable{Value: aws.String("bar")},
				Variable{Name: aws.String(""), Value: aws.String("bizz")},
			},
		},
		OldResourceProperties: &EnvironmentProperties{},
	})
	assert.Error(t, err)
	if merr, ok := err.(*multierror.Error); ok {
		assert.Equal(t, len(merr.Errors), 2)
	}
}

func TestVarsFromUpdateRequest_DeletedVars(t *testing.T) {
	vars, err := varsFromRequest(Request{
		RequestType: Update,
		ResourceProperties: &EnvironmentProperties{
			Variables: []Variable{
				Variable{Name: aws.String("FOO"), Value: aws.String("bar")},
				Variable{Name: aws.String("BAR"), Value: nil},
			},
		},
		OldResourceProperties: &EnvironmentProperties{
			Variables: []Variable{
				Variable{Name: aws.String("FOOBAR"), Value: aws.String("foobar")},
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, empire.Vars{
		"FOO":    aws.String("bar"),
		"BAR":    nil,
		"FOOBAR": nil,
	}, vars)
}

func TestVarsFromDeleteRequest(t *testing.T) {
	vars, err := varsFromRequest(Request{
		RequestType: Delete,
		ResourceProperties: &EnvironmentProperties{
			Variables: []Variable{
				Variable{Name: aws.String("FOO"), Value: aws.String("bar")},
				Variable{Name: aws.String("BAR"), Value: nil},
			},
		},
		OldResourceProperties: &EnvironmentProperties{},
	})
	assert.NoError(t, err)
	assert.Equal(t, empire.Vars{
		"FOO": nil,
		"BAR": nil,
	}, vars)
}
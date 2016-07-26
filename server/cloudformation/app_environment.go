package cloudformation

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/cloudformation/customresources"
	"golang.org/x/net/context"
)

// envClient mocks the Empire interface we use.
type envClient interface {
	AppsFind(empire.AppsQuery) (*empire.App, error)
	Set(context.Context, empire.SetOpts) (*empire.Config, error)
}

type Variable struct {
	Name  *string
	Value *string
}

// EnvironmentProperties represents the properties for the
// Custom::EmpireAppEnvironment
type EnvironmentProperties struct {
	AppId     *string
	Variables []Variable
}

// EmpireAppEnvironmentResource is a Provisioner that manages environmental variables
// within an Empire application.
type EmpireAppEnvironmentResource struct {
	empire envClient
}

func (p *EmpireAppEnvironmentResource) Properties() interface{} {
	return &EnvironmentProperties{}
}

type VariableError struct {
	index int
	err   string
}

func (v *VariableError) Error() string {
	return fmt.Sprintf("invalid variable [%d]: %s", v.index, v.err)
}

func (p *EmpireAppEnvironmentResource) Provision(ctx context.Context, req customresources.Request) (id string, data interface{}, err error) {
	properties := req.ResourceProperties.(*EnvironmentProperties)
	user := newUser()

	switch req.RequestType {
	case customresources.Create:
		if properties.AppId == nil || *properties.AppId == "" {
			return "", nil, fmt.Errorf("missing parameter: AppId")
		}
		id = *properties.AppId
	default:
		id = req.PhysicalResourceId
	}

	app, err := p.empire.AppsFind(empire.AppsQuery{
		ID: &id,
	})
	if err != nil {
		return id, nil, err
	}

	if err := p.setEnvironment(ctx, user, app, req); err != nil {
		return id, nil, err
	}
	return
}

func (p *EmpireAppEnvironmentResource) setEnvironment(ctx context.Context, user *empire.User, app *empire.App, req customresources.Request) error {
	vars, err := varsFromRequest(req)
	if err != nil {
		return err
	}

	var action string
	switch req.RequestType {
	case customresources.Create:
		action = "Setting"
	case customresources.Update:
		action = "Updating"
	case customresources.Delete:
		action = "Unsetting"
	}

	_, err = p.empire.Set(ctx, empire.SetOpts{
		User:    user,
		App:     app,
		Vars:    vars,
		Message: fmt.Sprintf("%s variables via Cloudformation", action),
	})
	return err
}

func isValid(index int, variable *Variable) error {
	var err error
	if variable.Name == nil || *variable.Name == "" {
		err = &VariableError{index, "key 'Name' is required"}
	}
	return err
}

func varsFromRequest(req customresources.Request) (empire.Vars, error) {
	vars := make(empire.Vars)
	var errors *multierror.Error

	properties := req.ResourceProperties.(*EnvironmentProperties)
	oldProperties := req.OldResourceProperties.(*EnvironmentProperties)

	for i, v := range properties.Variables {
		err := isValid(i, &v)
		if err != nil {
			errors = multierror.Append(errors, err)
			continue
		}

		var val *string
		// If we're deleting the resource, we want to unset the variable
		if req.RequestType != customresources.Delete {
			val = v.Value
		}
		vars[empire.Variable(*v.Name)] = val
	}

	if req.RequestType == customresources.Update {
		for i, v := range oldProperties.Variables {
			err := isValid(i, &v)
			if err != nil {
				errors = multierror.Append(errors, err)
				continue
			}

			if _, ok := vars[empire.Variable(*v.Name)]; !ok {
				vars[empire.Variable(*v.Name)] = nil
			}
		}
	}
	return vars, errors.ErrorOrNil()
}

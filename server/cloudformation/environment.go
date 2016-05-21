package cloudformation

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/remind101/empire"
	"golang.org/x/net/context"
)

type EnvironmentResource struct {
	empire *empire.Empire
}

type VariableError struct {
	index int
	err   string
}

func (v *VariableError) Error() string {
	return fmt.Sprintf("invalid variable [%d]: %s", v.index, v.err)
}

func (p *EnvironmentResource) Provision(req Request) (id string, data interface{}, err error) {
	ctx := context.Background()
	user := NewUser()

	var ok bool
	switch req.RequestType {
	case Create:
		id, ok = req.ResourceProperties["AppId"].(string)
		if !ok {
			return "", nil, fmt.Errorf("missing parameter: AppId")
		}
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

func (p *EnvironmentResource) setEnvironment(ctx context.Context, user *empire.User, app *empire.App, req Request) error {
	vars, err := varsFromRequest(req)
	if err != nil {
		return err
	}

	var action string
	switch req.RequestType {
	case Create:
		action = "Setting"
	case Update:
		action = "Updating"
	case Delete:
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

type Variable struct {
	Key   string
	Value *string
}

func parseVariable(index int, input interface{}) (*Variable, error) {
	var variable *Variable
	var err *multierror.Error
	if m, ok := input.(map[string]interface{}); ok {
		n, ok := m["Name"]
		if !ok {
			err = multierror.Append(err, &VariableError{index, "key 'Name' is required"})
		}

		var key string
		switch n := n.(type) {
		case string:
			key = n
		default:
		}

		if ok && key == "" {
			err = multierror.Append(err, &VariableError{index, "key 'Name' is required"})
		}

		v, ok := m["Value"]
		if !ok {
			err = multierror.Append(err, &VariableError{index, "key 'Value' is required"})
		}

		var val *string
		switch v := v.(type) {
		case string:
			vv := v
			val = &vv
		default:
		}

		if key != "" {
			variable = &Variable{Key: key, Value: val}
		}
	} else {
		err = multierror.Append(err, &VariableError{index, "keys 'Name' and 'Value' are required"})
	}
	return variable, err.ErrorOrNil()
}

func varsFromRequest(req Request) (empire.Vars, error) {
	vars := make(empire.Vars)
	var errors *multierror.Error

	if variables, ok := req.ResourceProperties["Variables"].([]interface{}); ok {
		for i, variable := range variables {
			v, err := parseVariable(i, variable)
			if err != nil {
				errors = multierror.Append(errors, err)
				continue
			}
			var val *string
			if req.RequestType != Delete {
				val = v.Value
			}
			vars[empire.Variable(v.Key)] = val
		}
	}

	if req.RequestType == Update {
		if variables, ok := req.OldResourceProperties["Variables"].([]interface{}); ok {
			for i, variable := range variables {
				v, err := parseVariable(i, variable)
				if err != nil {
					errors = multierror.Append(errors, err)
					continue
				}
				if _, ok := vars[empire.Variable(v.Key)]; !ok {
					vars[empire.Variable(v.Key)] = nil
				}
			}
		}
	}

	return vars, errors.ErrorOrNil()
}

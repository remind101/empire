package cloudformation

import (
	"github.com/jinzhu/gorm"
	"github.com/remind101/empire"
	"golang.org/x/net/context"
)

type AppsProvisioner struct {
	empire *empire.Empire
}

func (p *AppsProvisioner) Provision(req Request) (id string, data interface{}, err error) {
	ctx := context.Background()
	user := &empire.User{Name: "cloudformation"}

	switch req.RequestType {
	case Create:
		name := req.ResourceProperties["Name"].(string)
		app, err := p.empire.AppsFind(empire.AppsQuery{
			Name: &name,
		})
		if err != nil && err != gorm.RecordNotFound {
			return "", nil, err
		}

		app, err = p.empire.Create(ctx, empire.CreateOpts{
			User: user,
			Name: name,
		})
		if err != nil {
			return "", nil, err
		}

		if err := p.setEnvironment(ctx, user, app, req); err != nil {
			return "", nil, err
		}

		return app.ID, map[string]string{"Id": app.ID}, nil
	case Delete:
		id := req.PhysicalResourceId
		app, err := p.empire.AppsFind(empire.AppsQuery{
			ID: &id,
		})
		if err != nil {
			return id, nil, err
		}

		// TODO: Handle error
		_ = p.empire.Destroy(ctx, empire.DestroyOpts{
			User: user,
			App:  app,
		})

		return id, nil, nil
	case Update:
		id := req.PhysicalResourceId

		app, err := p.empire.AppsFind(empire.AppsQuery{
			ID: &id,
		})
		if err != nil {
			return id, nil, err
		}

		if err := p.setEnvironment(ctx, user, app, req); err != nil {
			return id, nil, err
		}

		return id, map[string]string{"Id": app.ID}, nil
	}

	return
}

func (p *AppsProvisioner) setEnvironment(ctx context.Context, user *empire.User, app *empire.App, req Request) error {
	vars := varsFromRequest(req)

	_, err := p.empire.Set(ctx, empire.SetOpts{
		User: user,
		App:  app,
		Vars: vars,
	})

	return err
}

func varsFromRequest(req Request) empire.Vars {
	vars := make(empire.Vars)

	if env, ok := req.ResourceProperties["Environment"].(map[string]interface{}); ok {
		for k, v := range env {
			var val *string
			switch v := v.(type) {
			case string:
				vv := v
				val = &vv
			default:
			}
			vars[empire.Variable(k)] = val
		}
	}

	if req.RequestType == Update {
		if env, ok := req.OldResourceProperties["Environment"].(map[string]interface{}); ok {
			// Find any environment variables that were removed.
			for k := range env {
				if _, ok := vars[empire.Variable(k)]; !ok {
					vars[empire.Variable(k)] = nil
				}
			}
		}
	}

	return vars
}

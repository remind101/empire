package cloudformation

import (
	"context"
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/cloudformation/customresources"
)

// appClient mocks the Empire interface we use.
type appClient interface {
	AppsFind(empire.AppsQuery) (*empire.App, error)
	Create(context.Context, empire.CreateOpts) (*empire.App, error)
	Destroy(context.Context, empire.DestroyOpts) error
}

// AppProperties represents the properties for the Custom::EmpireApp
type AppProperties struct {
	Name *string
}

// EmpireAppResource is a Provisioner that will manage an Empire application
type EmpireAppResource struct {
	empire appClient
}

func (p *EmpireAppResource) Properties() interface{} {
	return &AppProperties{}
}

func (p *EmpireAppResource) Provision(ctx context.Context, req customresources.Request) (id string, data interface{}, err error) {
	user := newUser()
	properties := req.ResourceProperties.(*AppProperties)

	switch req.RequestType {
	case customresources.Create:
		name := properties.Name
		app, err := p.empire.AppsFind(empire.AppsQuery{
			Name: name,
		})
		if err != nil && err != gorm.RecordNotFound {
			return "", nil, err
		}

		app, err = p.empire.Create(ctx, empire.CreateOpts{
			User:    user,
			Name:    *name,
			Message: "Creating app via Cloudformation",
		})
		if err != nil {
			return "", nil, err
		}

		return app.ID, nil, nil
	case customresources.Delete:
		id := req.PhysicalResourceId
		app, err := p.empire.AppsFind(empire.AppsQuery{
			ID: &id,
		})
		if err != nil {
			return id, nil, err
		}

		err = p.empire.Destroy(ctx, empire.DestroyOpts{
			User:    user,
			App:     app,
			Message: "Destroying app via Cloudformation",
		})
		if err != nil {
			return id, nil, err
		}

		return id, nil, nil
	case customresources.Update:
		id := req.PhysicalResourceId

		_, err := p.empire.AppsFind(empire.AppsQuery{
			ID: &id,
		})
		if err != nil {
			return id, nil, err
		}

		return id, nil, fmt.Errorf("Updates are not supported")
	}

	return
}

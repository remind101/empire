package cloudformation

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire"
)

// empireClient mocks the Empire interface we use.
type empireClient interface {
	AppsFind(empire.AppsQuery) (*empire.App, error)
	Create(context.Context, empire.CreateOpts) (*empire.App, error)
	Destroy(context.Context, empire.DestroyOpts) error
}

// AppProperties represents the properties for the Custom::EmpirApp
type AppProperties struct {
	Name string
}

// AppResource is a Provisioner that will manage an Empire application
type AppResource struct {
	empire empireClient
}

func (p *AppResource) Provision(req Request) (id string, data interface{}, err error) {
	ctx := context.Background()
	user := NewUser()

	properties := req.ResourceProperties.(*AppProperties)

	switch req.RequestType {
	case Create:
		name := properties.Name
		app, err := p.empire.AppsFind(empire.AppsQuery{
			Name: &name,
		})
		if err != nil && err != gorm.RecordNotFound {
			return "", nil, err
		}

		app, err = p.empire.Create(ctx, empire.CreateOpts{
			User:    user,
			Name:    name,
			Message: "Creating app via Cloudformation",
		})
		if err != nil {
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

		err = p.empire.Destroy(ctx, empire.DestroyOpts{
			User:    user,
			App:     app,
			Message: "Destroying app via Cloudformation",
		})
		if err != nil {
			return id, nil, err
		}

		return id, nil, nil
	case Update:
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

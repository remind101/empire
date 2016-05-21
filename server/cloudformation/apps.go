package cloudformation

import (
	"github.com/jinzhu/gorm"
	"github.com/remind101/empire"
	"golang.org/x/net/context"
)

type AppResource struct {
	empire *empire.Empire
}

func (p *AppResource) Provision(req Request) (id string, data interface{}, err error) {
	ctx := context.Background()
	user := NewUser()

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

		return id, map[string]string{"Id": app.ID}, nil
	}

	return
}

package github

import (
	"io/ioutil"
	"testing"

	"golang.org/x/net/context"

	"github.com/ejholmes/hookshot/events"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEmpireDeployer_Deploy(t *testing.T) {
	e := new(mockEmpire)
	d := &EmpireDeployer{
		empire:       e,
		ImageBuilder: ImageFromTemplate(defaultTemplate),
	}

	var event events.Deployment
	event.Repository.FullName = "remind101/acme-inc"
	event.Deployment.Sha = "abcd123"
	event.Deployment.Creator.Login = "ejholmes"

	w := ioutil.Discard

	e.On("Deploy", empire.DeploymentsCreateOpts{
		User: &empire.User{Name: "ejholmes"},
		Image: image.Image{
			Repository: "remind101/acme-inc",
			Tag:        "abcd123",
		},
		Output: w,
	}).Return(nil)

	err := d.Deploy(context.Background(), event, w)
	assert.NoError(t, err)
}

type mockEmpire struct {
	mock.Mock
}

func (m *mockEmpire) Deploy(ctx context.Context, opts empire.DeploymentsCreateOpts) (*empire.Release, error) {
	args := m.Called(opts)
	return nil, args.Error(0)
}

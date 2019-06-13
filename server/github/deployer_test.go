package github

import (
	"bytes"
	"testing"

	"golang.org/x/net/context"

	"github.com/ejholmes/hookshot/events"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/dockerutil"
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
	event.Deployment.ID = 53252
	event.Deployment.Ref = "master"
	event.Deployment.Task = "deploy"
	event.Deployment.Environment = "test"

	b := new(bytes.Buffer)

	e.On("Deploy", empire.DeployOpts{
		User: &empire.User{Name: "ejholmes"},
		Image: image.Image{
			Repository: "remind101/acme-inc",
			Tag:        "abcd123",
		},
		GitSHA:     "abcd123",
		Stream: true,
		Message: `GitHub deployment 53252 of remind101/acme-inc to test

Repository: remind101/acme-inc
SHA: abcd123
Ref: master
Deployment-Id: 53252
Task: deploy`,
	}).Return(nil)

	err := d.Deploy(context.Background(), event, b)
	assert.NoError(t, err)
	assert.Equal(t, `Pulling repository remind101/acme-inc
345c7524bc96: Pulling image (latest) from remind101/acme-inc
345c7524bc96: Pulling image (latest) from remind101/acme-inc, endpoint: https://registry-1.docker.io/v1/
345c7524bc96: Pulling dependent layers
a1dd7097a8e8: Download complete
Status: Image is up to date for remind101/acme-inc:latest
`, b.String())
}

type mockEmpire struct {
	mock.Mock
}

func (m *mockEmpire) Deploy(ctx context.Context, opts empire.DeployOpts) (*empire.Release, error) {
	w := opts.Output
	if err := dockerutil.FakePull(image.Image{Repository: "remind101/acme-inc", Tag: "latest"}, w); err != nil {
		panic(err)
	}
	opts.Output = nil
	args := m.Called(opts)
	return nil, args.Error(0)
}

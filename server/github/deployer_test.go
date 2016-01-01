package github

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"testing"

	"golang.org/x/net/context"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/ejholmes/hookshot/events"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/pkg/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEmpireDeployer_Deploy(t *testing.T) {
	e := new(mockEmpire)
	d := &empireDeployer{
		Empire: e,
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

func TestPrettyDeployer_Deploy(t *testing.T) {
	d := &prettyDeployer{
		deployer: deployerFunc(func(ctx context.Context, event events.Deployment, w io.Writer) error {
			img := image.Image{
				Repository: "remind101/acme-inc",
				Tag:        "abcd1234",
			}
			return dockerutil.FakePull(img, w)
		}),
	}

	var event events.Deployment
	buf := new(bytes.Buffer)
	err := d.Deploy(context.Background(), event, buf)
	assert.NoError(t, err)
	assert.Equal(t, `Pulling repository remind101/acme-inc
345c7524bc96: Pulling image (abcd1234) from remind101/acme-inc
345c7524bc96: Pulling image (abcd1234) from remind101/acme-inc, endpoint: https://registry-1.docker.io/v1/
345c7524bc96: Pulling dependent layers
a1dd7097a8e8: Download complete
Status: Image is up to date for remind101/acme-inc:abcd1234
`, buf.String())
}

func TestPrettyDeployer_Deploy_Error(t *testing.T) {
	errMsg := "Get https://registry-1.docker.io/v1/repositories/remind101/acme-inc/tags: read tcp 54.208.52.137:443: i/o timeout"
	d := &prettyDeployer{
		deployer: deployerFunc(func(ctx context.Context, event events.Deployment, w io.Writer) error {
			err := jsonmessage.JSONMessage{
				Error:        &jsonmessage.JSONError{Message: errMsg},
				ErrorMessage: errMsg,
			}
			return json.NewEncoder(w).Encode(&err)
		}),
	}

	var event events.Deployment
	buf := new(bytes.Buffer)
	err := d.Deploy(context.Background(), event, buf)
	assert.EqualError(t, err, errMsg)
}

type mockEmpire struct {
	mock.Mock
}

func (m *mockEmpire) Deploy(ctx context.Context, opts empire.DeploymentsCreateOpts) (*empire.Release, error) {
	args := m.Called(opts)
	return nil, args.Error(0)
}

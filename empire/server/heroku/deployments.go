package heroku

import (
	"net/http"

	"github.com/remind101/empire/empire"
	"golang.org/x/net/context"
)

type Deployment struct {
	ID      string   `json:"id"`
	Release *Release `json:"release"`
}

func newDeployment(d *empire.Deployment) *Deployment {
	return &Deployment{
		ID: d.ID,
	}
}

// PostDeploys is a Handler for the POST /v1/deploys endpoint.
type PostDeploys struct {
	*empire.Empire
}

// PostDeployForm is the form object that represents the POST body.
type PostDeployForm struct {
	Image struct {
		ID   string `json:"id"`
		Repo string `json:"repo"`
	} `json:"image"`
}

// jsonMessage represents a streamed status message from the docker remote api.
// https://docs.docker.com/reference/api/docker_remote_api_v1.9/#create-an-image
type jsonMessage struct {
	Status         string      `json:"status,omitempty"`
	Progress       string      `json:"progress,omitempty"`
	ProgressDetail interface{} `json:"progressDetail,omitempty"`
	Error          string      `json:"error,omitempty"`
	Stream         string      `json:"stream,omitempty"`
	Deployment     *Deployment `json:"deployment,omitempty"`
}

// Serve implements the Handler interface.
func (h *PostDeploys) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var form PostDeployForm

	if err := Decode(r, &form); err != nil {
		return err
	}

	image := empire.Image{
		Repo: empire.Repo(form.Image.Repo),
		ID:   form.Image.ID,
	}

	w.Header().Set("Content-Type", "application/json; boundary=NL")

	var (
		d   *empire.Deployment
		err error
	)

	ch := make(chan empire.Event)
	errCh := make(chan error)
	go func() {
		d, err = h.DeployImage(ctx, image, ch)
		errCh <- err
	}()

	ok := true
	for ok {
		select {
		case evt := <-ch:
			if err := Stream(w, evt); err != nil {
				return err
			}
		case err := <-errCh:
			if err != nil {
				return err
			}
			ok = false
		}
	}

	return Encode(w, newDeployment(d))
}

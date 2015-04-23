package heroku

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/remind101/empire/empire"
	"github.com/remind101/pkg/logger"
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

	pr, pw := io.Pipe()

	var (
		d     *empire.Deployment
		err   error
		errCh chan error
	)

	errCh = make(chan error, 1)
	go func() {
		d, err = h.DeployImage(ctx, image, pw)
		errCh <- err
	}()

	// Stream output from DeployImage, adding newlines after each json message.
	if err := h.streamJSON(pr, w); err != nil {
		logger.Log(ctx, "at", "streamJSON", "err", err)
		h.streamErr(err, w)
		return nil
	}

	// Wait for DeployImage to finish.
	if err := <-errCh; err != nil {
		h.streamErr(err, w)
		return nil
	}

	return Encode(w, &jsonMessage{
		Status:     "Deployed",
		Deployment: newDeployment(d),
	})
}

func (h *PostDeploys) streamJSON(r io.Reader, w io.Writer) error {
	dec := json.NewDecoder(r)
	enc := json.NewEncoder(w)
	for {
		var m jsonMessage
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if err := enc.Encode(&m); err != nil {
			return err
		}
		w.Write([]byte("\n"))
	}
	return nil
}

func (h *PostDeploys) streamErr(err error, w io.Writer) error {
	enc := json.NewEncoder(w)
	return enc.Encode(&jsonMessage{
		Status: "Errored during deploy",
		Error:  err.Error(),
	})
}

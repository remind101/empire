package empire

import (
	"io"
	"net/http"

	"github.com/remind101/tugboat/pkg/heroku"
)

type client struct {
	*heroku.Service
}

func newClient(c *http.Client) *client {
	if c == nil {
		c = http.DefaultClient
	}

	return &client{
		Service: heroku.NewService(c),
	}
}

func (c *client) Deploy(image string, w io.Writer) error {
	d := struct {
		Image string `json:"image"`
	}{
		Image: image,
	}

	return c.Post(w, "/deploys", &d)
}

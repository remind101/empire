package empire

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/remind101/tugboat"
	"github.com/remind101/tugboat/pkg/heroku"
	"golang.org/x/net/context"
)

type Provider struct {
	client *client
}

func NewProvider(url, token string) *Provider {
	c := newClient(&http.Client{
		Transport: newTransport(token),
	})
	c.URL = url

	return &Provider{
		client: c,
	}
}

func (p *Provider) Name() string {
	return "empire"
}

func (p *Provider) Deploy(ctx context.Context, d *tugboat.Deployment, w io.Writer) error {
	image := newImage(d)
	fmt.Fprintf(w, "Deploying %s to %s...\n", image, p.client.URL)

	if err := p.client.Deploy(newImage(d), w); err != nil {
		return err
	}

	return nil
}

func newTransport(token string) http.RoundTripper {
	proxy := &http.Transport{
		Proxy: func(_ *http.Request) (*url.URL, error) {
			proxy := os.Getenv("EMPIRE_PROXY")
			if proxy == "" {
				return nil, nil
			}

			fmt.Println("Using proxy", proxy)

			return url.Parse(proxy)
		},
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &heroku.Transport{
		Password:  token,
		Transport: proxy,
	}
}

// newImage returns the Docker image that should be used for the given
// deployment. It assumes that the docker images are tagged with the full git
// Sha and that the Docker repository matches the github repository.
func newImage(d *tugboat.Deployment) string {
	return fmt.Sprintf("%s:%s", d.Repo, d.Sha)
}

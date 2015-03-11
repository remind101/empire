// package registry provides an http client for the docker registry.
package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// DefaultRegistry is the official docker registry.
const DefaultRegistry = "index.docker.io"

type Client struct {
	// The domain of the docker registry. The zero value will be the
	// official docker registry.
	Registry string

	// Username is a username to authenticate with.
	Username string

	// Password is a password to authenticate with.
	Password string

	// Whether to use http or https.
	DisableTLS bool

	client *http.Client
}

func NewClient(c *http.Client) *Client {
	if c == nil {
		c = http.DefaultClient
	}

	return &Client{
		client: c,
	}
}

// ResolveTag resolves a tag in a given repo to an image id.
func (c *Client) ResolveTag(repo, tag string) (string, error) {
	var imageID string

	path := fmt.Sprintf("/v1/repositories/%s/tags/%s", repo, tag)
	req, err := c.NewRequest("GET", path)
	if err != nil {
		return imageID, err
	}

	if _, err := c.Do(req, &imageID); err != nil {
		return imageID, err
	}

	return imageID, nil
}

// NewRequest builds a new http.Request.
func (c *Client) NewRequest(method, path string) (*http.Request, error) {
	proto := "https"
	if c.DisableTLS {
		proto = "http"
	}

	registry := c.Registry
	if registry == "" {
		registry = DefaultRegistry
	}

	url := fmt.Sprintf("%s://%s%s", proto, registry, path)

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return req, err
	}

	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	return req, nil
}

// Do performs an request and decodes the response body into v.
func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return resp, fmt.Errorf("registry: unexpected response %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return resp, err
	}

	return resp, nil
}

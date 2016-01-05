package tugboat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/context"
)

const AcceptHeader = "application/vnd.tugboat+json; version=1"

type Client struct {
	// URL is the base url for the Tugboat API.
	URL string

	client *http.Client
}

// NewClient returns a new Client instance.
func NewClient(c *http.Client) *Client {
	if c == nil {
		c = http.DefaultClient
	}

	return &Client{
		client: c,
	}
}

// Deploy performs a deployment using a Provider.
func (c *Client) Deploy(ctx context.Context, opts DeployOpts, p Provider) (*Deployment, error) {
	return deploy(ctx, opts, p, c)
}

// DeploymentsCreate creates a deployment.
func (c *Client) DeploymentsCreate(opts DeployOpts) (*Deployment, error) {
	req, err := c.NewRequest("POST", "/deployments", opts)
	if err != nil {
		return nil, err
	}

	var d Deployment
	if _, err := c.Do(req, &d); err != nil {
		return nil, err
	}

	return &d, nil
}

// WriteLogs streams the reader to the API creating log lines for each line.
func (c *Client) WriteLogs(d *Deployment, r io.Reader) error {
	req, err := c.NewRequest("POST", fmt.Sprintf("/deployments/%s/logs", d.ID), r)
	if err != nil {
		return err
	}

	_, err = c.Do(req, nil)
	return err
}

// UpdateStatus updates the status of the deployment.
func (c *Client) UpdateStatus(d *Deployment, update StatusUpdate) error {
	req, err := c.NewRequest("POST", fmt.Sprintf("/deployments/%s/status", d.ID), update)
	if err != nil {
		return err
	}

	_, err = c.Do(req, nil)
	return err
}

func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		raw, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("unexpected response: %d - %s", resp.StatusCode, string(raw))
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

func (c *Client) NewRequest(method, path string, v interface{}) (*http.Request, error) {
	var r io.Reader

	switch v := v.(type) {
	case io.Reader:
		r = v
	default:
		buf := new(bytes.Buffer)

		if err := json.NewEncoder(buf).Encode(v); err != nil {
			return nil, err
		}

		r = buf
	}

	req, err := http.NewRequest(method, c.URL+path, r)
	if err != nil {
		return req, err
	}
	req.Header.Set("Accept", AcceptHeader)

	return req, err
}

// Package hooker can generate github webhooks. It's only real use is for
// testing endpoints that handle github webhooks.
package hooker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/ejholmes/hookshot"
	"github.com/ejholmes/hookshot/events"
)

// DefaultPing is a Ping event that makes you feel good.
var DefaultPing = events.Ping{
	Zen: "Practicality beats purity.",
}

// Client represents a client that can send GitHub webhooks.
type Client struct {
	// Secret is a secret to sign request bodies with.
	Secret string

	// URL is the url to make requests against.
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

// Trigger triggers a webhook event. The sha1 digest of v is calculated and
// including in the X-Hub-Signature header.
func (c *Client) Trigger(event string, v interface{}) (*http.Response, error) {
	b := new(bytes.Buffer)

	switch v := v.(type) {
	case io.Reader:
		if _, err := io.Copy(b, v); err != nil {
			return nil, err
		}
	default:
		if err := json.NewEncoder(b).Encode(v); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest("POST", c.URL, b)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-GitHub-Event", event)
	req.Header.Set("X-Hub-Signature", fmt.Sprintf("sha1=%s", hookshot.Signature(b.Bytes(), c.Secret)))

	return c.Do(req)
}

// Ping sends a ping event.
func (c *Client) Ping(p events.Ping) (*http.Response, error) {
	return c.Trigger("ping", p)
}

// Do performs the request.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return resp, err
	}

	if resp.StatusCode/100 != 2 {
		raw, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return resp, err
		}
		resp.Body.Close()

		return resp, fmt.Errorf("hooker: request failed with status %d: %s", resp.StatusCode, raw)
	}

	return resp, err
}

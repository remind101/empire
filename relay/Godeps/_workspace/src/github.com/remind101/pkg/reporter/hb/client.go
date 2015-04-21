package hb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	DefaultURL     = "https://api.honeybadger.io"
	DefaultVersion = "v1"
)

type Notifier struct {
	Name     string `json:"name"`
	Url      string `json:"url"`
	Version  string `json:"version"`
	Language string `json:"language"`
}

type BacktraceLine struct {
	Method string `json:"method"`
	File   string `json:"file"`
	Number string `json:"number"`
}

type Error struct {
	Class     string                 `json:"class"`
	Message   string                 `json:"message"`
	Backtrace []*BacktraceLine       `json:"backtrace"`
	Source    map[string]interface{} `json:"source"`
	Tags      []string               `json:"tags"`
}

type Request struct {
	Url       string                 `json:"url"`
	Component string                 `json:"component"`
	Action    string                 `json:"action"`
	Params    map[string]interface{} `json:"params"`
	Session   map[string]interface{} `json:"session"`
	CgiData   map[string]interface{} `json:"cgi_data"`
	Context   map[string]interface{} `json:"context"`
}

type Server struct {
	ProjectRoot     map[string]interface{} `json:"project_root"`
	EnvironmentName string                 `json:"environment_name"`
	Hostname        string                 `json:"hostname"`
}

type Report struct {
	Notifier *Notifier `json:"notifier"`
	Error    *Error    `json:"error"`
	Request  *Request  `json:"request"`
	Server   *Server   `json:"server"`
}

type Client struct {
	// URL is the location for the honeybadger api. The zero value is DefaultURL.
	URL string

	// Version is the API version to use. The zero value is DefaultVersion.
	Version string

	client *http.Client
}

// NewClient returns a new Client instance.
func NewClient(c *http.Client) *Client {
	if c == nil {
		c = http.DefaultClient
	}

	return &Client{client: c}
}

// NewClientFromKey returns a new Client with an http.Client configured to add
// the key as the api token.
func NewClientFromKey(key string) *Client {
	t := &Transport{Key: key}
	return NewClient(t.Client())
}

func (c *Client) Send(r *Report) error {
	req, err := c.NewRequest("POST", "/notices", r)
	if err != nil {
		return err
	}

	if _, err := c.Do(req); err != nil {
		return err
	}

	return nil
}

func (c *Client) NewRequest(method, path string, v interface{}) (*http.Request, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	url := c.URL
	if url == "" {
		url = DefaultURL
	}

	version := c.Version
	if version == "" {
		version = DefaultVersion
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s%s", url, version, path), bytes.NewBuffer(raw))
	if err != nil {
		return req, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return req, nil
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	client := c.client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return resp, fmt.Errorf("hb: unexpected response: %d", resp.StatusCode)
	}

	return resp, nil
}

// Transport is an http.RoundTripper that adds the api key to the request
// headers.
type Transport struct {
	Key string

	Transport http.RoundTripper
}

func (t *Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("X-API-Key", t.Key)

	if t.Transport == nil {
		t.Transport = http.DefaultTransport
	}

	return t.Transport.RoundTrip(r)
}

func (t *Transport) Client() *http.Client {
	return &http.Client{
		Transport: t,
	}
}

// Generated service client for empire API.
//
// To be able to interact with this API, you have to
// create a new service:
//
//     s := empire.NewService(nil)
//
// The Service struct has all the methods you need
// to interact with empire API.
//
package empire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"runtime"

	"github.com/ernesto-jimenez/go-querystring/query"
)

const (
	Version          = ""
	DefaultUserAgent = "empire/" + Version + " (" + runtime.GOOS + "; " + runtime.GOARCH + ")"
	DefaultURL       = "http://localhost:8080"
)

// Service represents your API.
type Service struct {
	client *http.Client
	URL    string
}

// NewService creates a Service using the given, if none is provided
// it uses http.DefaultClient.
func NewService(c *http.Client) *Service {
	if c == nil {
		c = http.DefaultClient
	}
	return &Service{
		client: c,
		URL:    DefaultURL,
	}
}

// NewRequest generates an HTTP request, but does not perform the request.
func (s *Service) NewRequest(method, path string, body interface{}, q interface{}) (*http.Request, error) {
	var ctype string
	var rbody io.Reader
	switch t := body.(type) {
	case nil:
	case string:
		rbody = bytes.NewBufferString(t)
	case io.Reader:
		rbody = t
	default:
		v := reflect.ValueOf(body)
		if !v.IsValid() {
			break
		}
		if v.Type().Kind() == reflect.Ptr {
			v = reflect.Indirect(v)
			if !v.IsValid() {
				break
			}
		}
		j, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rbody = bytes.NewReader(j)
		ctype = "application/json"
	}
	req, err := http.NewRequest(method, s.URL+path, rbody)
	if err != nil {
		return nil, err
	}
	if q != nil {
		v, err := query.Values(q)
		if err != nil {
			return nil, err
		}
		query := v.Encode()
		if req.URL.RawQuery != "" && query != "" {
			req.URL.RawQuery += "&"
		}
		req.URL.RawQuery += query
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", DefaultUserAgent)
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	return req, nil
}

// Do sends a request and decodes the response into v.
func (s *Service) Do(v interface{}, method, path string, body interface{}, q interface{}, lr *ListRange) error {
	req, err := s.NewRequest(method, path, body, q)
	if err != nil {
		return err
	}
	if lr != nil {
		lr.SetHeader(req)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch t := v.(type) {
	case nil:
	case io.Writer:
		_, err = io.Copy(t, resp.Body)
	default:
		err = json.NewDecoder(resp.Body).Decode(v)
	}
	return err
}

// Get sends a GET request and decodes the response into v.
func (s *Service) Get(v interface{}, path string, query interface{}, lr *ListRange) error {
	return s.Do(v, "GET", path, nil, query, lr)
}

// Patch sends a Path request and decodes the response into v.
func (s *Service) Patch(v interface{}, path string, body interface{}) error {
	return s.Do(v, "PATCH", path, body, nil, nil)
}

// Post sends a POST request and decodes the response into v.
func (s *Service) Post(v interface{}, path string, body interface{}) error {
	return s.Do(v, "POST", path, body, nil, nil)
}

// Put sends a PUT request and decodes the response into v.
func (s *Service) Put(v interface{}, path string, body interface{}) error {
	return s.Do(v, "PUT", path, body, nil, nil)
}

// Delete sends a DELETE request.
func (s *Service) Delete(v interface{}, path string) error {
	return s.Do(v, "DELETE", path, nil, nil, nil)
}

// ListRange describes a range.
type ListRange struct {
	Field      string
	Max        int
	Descending bool
	FirstID    string
	LastID     string
}

// SetHeader set headers on the given Request.
func (lr *ListRange) SetHeader(req *http.Request) {
	var hdrval string
	if lr.Field != "" {
		hdrval += lr.Field + " "
	}
	hdrval += lr.FirstID + ".." + lr.LastID
	if lr.Max != 0 {
		hdrval += fmt.Sprintf("; max=%d", lr.Max)
		if lr.Descending {
			hdrval += ", "
		}
	}
	if lr.Descending {
		hdrval += ", order=desc"
	}
	req.Header.Set("Range", hdrval)
	return
}

// Bool allocates a new int value returns a pointer to it.
func Bool(v bool) *bool {
	p := new(bool)
	*p = v
	return p
}

// Int allocates a new int value returns a pointer to it.
func Int(v int) *int {
	p := new(int)
	*p = v
	return p
}

// Float64 allocates a new float64 value returns a pointer to it.
func Float64(v float64) *float64 {
	p := new(float64)
	*p = v
	return p
}

// String allocates a new string value returns a pointer to it.
func String(v string) *string {
	p := new(string)
	*p = v
	return p
}

type App struct {
	Name string `json:"name" url:"name,key"` // unique name of app
	Repo string `json:"repo" url:"repo,key"` // the name of the repo
}
type AppCreateOpts struct {
	Name string `json:"name" url:"name,key"` // unique name of app
	Repo string `json:"repo" url:"repo,key"` // the name of the repo
}

// Create a new app.
func (s *Service) AppCreate(o AppCreateOpts) (*App, error) {
	var app App
	return &app, s.Post(&app, fmt.Sprintf("/apps"), o)
}

type Config struct {
	Vars    map[string]string `json:"vars" url:"vars,key"`       // a hash of configuration values
	Version string            `json:"version" url:"version,key"` // unique identifier of config
}

// Get the latest version of an app's config
func (s *Service) ConfigHead(appIdentity string) (*Config, error) {
	var config Config
	return &config, s.Get(&config, fmt.Sprintf("/apps/%v/configs/head", appIdentity), nil, nil)
}

// Get a specific version of an app's config
func (s *Service) ConfigInfo(appIdentity string, configIdentity string) (*Config, error) {
	var config Config
	return &config, s.Get(&config, fmt.Sprintf("/apps/%v/configs/%v", appIdentity, configIdentity), nil, nil)
}

type ConfigUpdateOpts struct {
	Vars map[string]string `json:"vars" url:"vars,key"` // a hash of configuration values
}

// Updates the config for an app
func (s *Service) ConfigUpdate(appIdentity string, o ConfigUpdateOpts) (*Config, error) {
	var config Config
	return &config, s.Patch(&config, fmt.Sprintf("/apps/%v/configs", appIdentity), o)
}

type Deploy struct {
	ID      string `json:"id" url:"id,key"` // unique identifier of deploy
	Release struct {
		App struct {
			Name string `json:"name" url:"name,key"` // unique name of app
		} `json:"app" url:"app,key"`
		Config struct {
			Version string `json:"version" url:"version,key"` // unique identifier of config
		} `json:"config" url:"config,key"`
		ID   string `json:"id" url:"id,key"` // unique identifier of release
		Slug struct {
			ID string `json:"id" url:"id,key"` // unique identifier of slug
		} `json:"slug" url:"slug,key"`
		Version string `json:"version" url:"version,key"` // an incremental identifier for the version
	} `json:"release" url:"release,key"`
}
type DeployCreateOpts struct {
	Image struct {
		ID   string `json:"id" url:"id,key"`     // unique identifier of image
		Repo string `json:"repo" url:"repo,key"` // the name of the repo
	} `json:"image" url:"image,key"`
}

// Create a new deploy.
func (s *Service) DeployCreate(o DeployCreateOpts) (*Deploy, error) {
	var deploy Deploy
	return &deploy, s.Post(&deploy, fmt.Sprintf("/deploys"), o)
}

type Image struct {
	ID   string `json:"id" url:"id,key"`     // unique identifier of image
	Repo string `json:"repo" url:"repo,key"` // the name of the repo
}
type Procdef struct {
	InstanceCount float64 `json:"instance_count" url:"instance_count,key"` // the number of running processes to maintain
	ProcessType   string  `json:"process_type" url:"process_type,key"`     // the type of process
	Release       struct {
		ID string `json:"id" url:"id,key"` // unique identifier of release
	} `json:"release" url:"release,key"`
}
type Release struct {
	ID      string `json:"id" url:"id,key"`           // unique identifier of release
	Version string `json:"version" url:"version,key"` // an incremental identifier for the version
}
type Repo struct {
	Name string `json:"name" url:"name,key"` // the name of the repo
}
type Slug struct {
	ID string `json:"id" url:"id,key"` // unique identifier of slug
}

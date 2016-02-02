// Generated service client for conveyor API.
//
// To be able to interact with this API, you have to
// create a new service:
//
//     s := conveyor.NewService(nil)
//
// The Service struct has all the methods you need
// to interact with conveyor API.
//
package conveyor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ernesto-jimenez/go-querystring/query"
	"io"
	"net/http"
	"reflect"
	"runtime"
	"time"
)

const (
	Version          = ""
	DefaultUserAgent = "conveyor/" + Version + " (" + runtime.GOOS + "; " + runtime.GOARCH + ")"
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

// An artifact is the result of a successful build. It represents a
// built Docker image and will tell what what you need to pull to obtain
// the image.
type Artifact struct {
	Build struct {
		ID string `json:"id" url:"id,key"` // unique identifier of build
	} `json:"build" url:"build,key"`
	ID    string `json:"id" url:"id,key"`       // unique identifier of artifact
	Image string `json:"image" url:"image,key"` // the name of the Docker image. This can be pulled with `docker pull`
}

func (s *Service) ArtifactInfo(artifactIdentity string) (*Artifact, error) {
	var artifact Artifact
	return &artifact, s.Get(&artifact, fmt.Sprintf("/artifacts/%v", artifactIdentity), nil, nil)
}

// A build represents a request to build a git commit for a repo.
type Build struct {
	Branch string `json:"branch" url:"branch,key"` // the branch within the GitHub repository that the build was triggered
	// from
	CompletedAt *time.Time `json:"completed_at" url:"completed_at,key"` // when the build moved to the `"succeeded"` or `"failed"` state
	CreatedAt   time.Time  `json:"created_at" url:"created_at,key"`     // when the build was created
	ID          string     `json:"id" url:"id,key"`                     // unique identifier of build
	Repository  string     `json:"repository" url:"repository,key"`     // the GitHub repository that this build is for
	Sha         string     `json:"sha" url:"sha,key"`                   // the git commit to build
	StartedAt   *time.Time `json:"started_at" url:"started_at,key"`     // when the build moved to the `"building"` state
	State       string     `json:"state" url:"state,key"`               // the current state of the build
}
type BuildCreateOpts struct {
	Branch *string `json:"branch,omitempty" url:"branch,omitempty,key"` // the branch within the GitHub repository that the build was triggered
	// from
	Repository string  `json:"repository" url:"repository,key"`       // the GitHub repository that this build is for
	Sha        *string `json:"sha,omitempty" url:"sha,omitempty,key"` // the git commit to build
}

// Create a new build and start it. Note that you cannot start a new
// build for a sha that is already in a "pending" or "building" state.
// You should cancel the existing build first, or wait for it to
// complete. You must specify either a `branch` OR a `sha`. If you
// provide a `branch` but no `sha`, Conveyor will use the GitHub API to
// resolve the HEAD commit on that branch to a sha. If you provide a
// `sha` but no `branch`, branch caching will be disabled.
func (s *Service) BuildCreate(o BuildCreateOpts) (*Build, error) {
	var build Build
	return &build, s.Post(&build, fmt.Sprintf("/builds"), o)
}

// Info for existing build.
func (s *Service) BuildInfo(buildIdentity string) (*Build, error) {
	var build Build
	return &build, s.Get(&build, fmt.Sprintf("/builds/%v", buildIdentity), nil, nil)
}

// Defines the format that errors are returned in
type Error struct {
	ID      string `json:"id" url:"id,key"`           // unique identifier of error
	Message string `json:"message" url:"message,key"` // human readable message
}


// Copyright (c) 2013 Blake Gentry. All rights reserved. Use of
// this source code is governed by an MIT license that can be
// found in the LICENSE file.

package heroku

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"reflect"
	"runtime"
	"strings"

	"code.google.com/p/go-uuid/uuid"
)

const (
	Version          = "0.10.2"
	DefaultAPIURL    = "https://api.heroku.com"
	DefaultUserAgent = "heroku-go/" + Version + " (" + runtime.GOOS + "; " + runtime.GOARCH + ")"
)

// A Client is a Heroku API client. Its zero value is a usable client that uses
// default settings for the Heroku API. The Client has an internal HTTP client
// (HTTP) which defaults to http.DefaultClient.
//
// As with all http.Clients, this Client's Transport has internal state (cached
// HTTP connections), so Clients should be reused instead of created as needed.
// Clients are safe for use by multiple goroutines.
type Client struct {
	// HTTP is the Client's internal http.Client, handling HTTP requests to the
	// Heroku API.
	HTTP *http.Client

	// The URL of the Heroku API to communicate with. Defaults to
	// "https://api.heroku.com".
	URL string

	// Username is the HTTP basic auth username for API calls made by this Client.
	Username string

	// Password is the HTTP basic auth password for API calls made by this Client.
	Password string

	// UserAgent to be provided in API requests. Set to DefaultUserAgent if not
	// specified.
	UserAgent string

	// Debug mode can be used to dump the full request and response to stdout.
	Debug bool

	// AdditionalHeaders are extra headers to add to each HTTP request sent by
	// this Client.
	AdditionalHeaders http.Header
}

func (c *Client) Get(v interface{}, path string) error {
	return c.APIReq(v, "GET", path, nil)
}

func (c *Client) Patch(v interface{}, path string, body interface{}) error {
	return c.APIReq(v, "PATCH", path, body)
}

func (c *Client) Post(v interface{}, path string, body interface{}) error {
	return c.APIReq(v, "POST", path, body)
}

func (c *Client) Put(v interface{}, path string, body interface{}) error {
	return c.APIReq(v, "PUT", path, body)
}

func (c *Client) Delete(path string) error {
	return c.APIReq(nil, "DELETE", path, nil)
}

// Generates an HTTP request for the Heroku API, but does not
// perform the request. The request's Accept header field will be
// set to:
//
//   Accept: application/vnd.heroku+json; version=3
//
// The Request-Id header will be set to a random UUID. The User-Agent header
// will be set to the Client's UserAgent, or DefaultUserAgent if UserAgent is
// not set.
//
// The type of body determines how to encode the request:
//
//   nil         no body
//   io.Reader   body is sent verbatim
//   else        body is encoded as application/json
func (c *Client) NewRequest(method, path string, body interface{}) (*http.Request, error) {
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
			log.Fatal(err)
		}
		rbody = bytes.NewReader(j)
		ctype = "application/json"
	}
	apiURL := strings.TrimRight(c.URL, "/")
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}
	req, err := http.NewRequest(method, apiURL+path, rbody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.heroku+json; version=3")
	req.Header.Set("Request-Id", uuid.New())
	useragent := c.UserAgent
	if useragent == "" {
		useragent = DefaultUserAgent
	}
	req.Header.Set("User-Agent", useragent)
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	req.SetBasicAuth(c.Username, c.Password)
	for k, v := range c.AdditionalHeaders {
		req.Header[k] = v
	}
	return req, nil
}

// Sends a Heroku API request and decodes the response into v. As
// described in NewRequest(), the type of body determines how to
// encode the request body. As described in DoReq(), the type of
// v determines how to handle the response body.
func (c *Client) APIReq(v interface{}, meth, path string, body interface{}) error {
	req, err := c.NewRequest(meth, path, body)
	if err != nil {
		return err
	}
	return c.DoReq(req, v)
}

// Submits an HTTP request, checks its response, and deserializes
// the response into v. The type of v determines how to handle
// the response body:
//
//   nil        body is discarded
//   io.Writer  body is copied directly into v
//   else       body is decoded into v as json
//
func (c *Client) DoReq(req *http.Request, v interface{}) error {
	if c.Debug {
		dump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			log.Println(err)
		} else {
			os.Stderr.Write(dump)
			os.Stderr.Write([]byte{'\n', '\n'})
		}
	}

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if c.Debug {
		dump, err := httputil.DumpResponse(res, true)
		if err != nil {
			log.Println(err)
		} else {
			os.Stderr.Write(dump)
			os.Stderr.Write([]byte{'\n'})
		}
	}
	if err = checkResp(res); err != nil {
		return err
	}
	switch t := v.(type) {
	case nil:
	case io.Writer:
		_, err = io.Copy(t, res.Body)
	default:
		err = json.NewDecoder(res.Body).Decode(v)
	}
	return err
}

// An Error represents a Heroku API error.
type Error struct {
	error
	Id  string
	URL string
}

type errorResp struct {
	Message string
	Id      string
	URL     string `json:"url"`
}

func checkResp(res *http.Response) error {
	if res.StatusCode/100 != 2 { // 200, 201, 202, etc
		var e errorResp
		err := json.NewDecoder(res.Body).Decode(&e)
		if err != nil {
			return errors.New("Unexpected error: " + res.Status)
		}
		return Error{error: errors.New(e.Message), Id: e.Id, URL: e.URL}
	}
	if msg := res.Header.Get("X-Heroku-Warning"); msg != "" {
		fmt.Fprintln(os.Stderr, strings.TrimSpace(msg))
	}
	return nil
}

type ListRange struct {
	Field      string
	Max        int
	Descending bool
	FirstId    string
	LastId     string
}

func (lr *ListRange) SetHeader(req *http.Request) {
	var hdrval string
	if lr.Field != "" {
		hdrval += lr.Field + " "
	}
	hdrval += lr.FirstId + ".." + lr.LastId
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

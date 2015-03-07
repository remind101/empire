package heroku

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/bgentry/testnet"
)

// Tests

func TestAdditionalHeaders(t *testing.T) {
	multival := []string{"awesome", "multival"}
	c := &Client{AdditionalHeaders: http.Header{
		"Fake-Header":     []string{"value"},
		"X-Heroku-Header": multival,
	}}
	req, err := c.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if val := req.Header.Get("Fake-Header"); val != "value" {
		t.Errorf("Fake-Header expected %q, got %q", "value", val)
	}
	val := req.Header["X-Heroku-Header"]
	if len(val) != len(multival) {
		t.Errorf("X-Heroku-Header len expected %d, got %d", len(multival), len(val))
	}
	for i, v := range val {
		if v != multival[i] {
			t.Errorf("X-Heroku-Header value[%d] expected %q, got %q", i, multival[i], v)
		}
	}
}

func TestRequestId(t *testing.T) {
	c := &Client{}
	req, err := c.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Request-Id") == "" {
		t.Error("Request-Id not set")
	}
}

func TestUserAgent(t *testing.T) {
	c := &Client{}
	req, err := c.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if ua := req.Header.Get("User-Agent"); ua != DefaultUserAgent {
		t.Errorf("Default User-Agent expected %q, got %q", DefaultUserAgent, ua)
	}

	// try a custom User-Agent
	customAgent := "custom-client 2.1 " + DefaultUserAgent
	c.UserAgent = customAgent
	req, err = c.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if ua := req.Header.Get("User-Agent"); ua != customAgent {
		t.Errorf("User-Agent expected %q, got %q", customAgent, ua)
	}
}

func TestGet(t *testing.T) {
	resp := testnet.TestResponse{
		Status: http.StatusOK,
		Body:   `{"omg": "wtf"}`,
	}
	req := newTestRequest("GET", "/", "", resp)

	ts, _, c := newTestServerAndClient(t, req)
	defer ts.Close()

	var respBody struct {
		Omg string
	}
	err := c.Get(&respBody, "/")
	if err != nil {
		t.Fatal(err)
	}
	if respBody.Omg != "wtf" {
		t.Errorf("expected %q, got %q", "wtf", respBody.Omg)
	}
}

func TestPost(t *testing.T) {
	resp := testnet.TestResponse{
		Status: http.StatusOK,
		Body:   `{"omg": "wtf"}`,
	}
	req := newTestRequest("POST", "/", `{"Wtf": "bbq"}`, resp)

	ts, _, c := newTestServerAndClient(t, req)
	defer ts.Close()

	var reqBody struct {
		Wtf string
	}
	reqBody.Wtf = "bbq"
	var respBody struct {
		Omg string
	}
	err := c.Post(&respBody, "/", reqBody)
	if err != nil {
		t.Fatal(err)
	}
	if respBody.Omg != "wtf" {
		t.Errorf("expected %q, got %q", "wtf", respBody.Omg)
	}
}

type respTest struct {
	Response http.Response
	Expected error
}

func newTestResponse(statuscode int, body string) http.Response {
	return http.Response{
		StatusCode:    statuscode,
		Status:        http.StatusText(statuscode),
		ContentLength: int64(len(body)),
		Body:          ioutil.NopCloser(bytes.NewBufferString(body)),
	}
}

var respTests = []respTest{
	{newTestResponse(200, `{"code": "OK"}`), nil},
	{newTestResponse(201, `{"code": "OK"}`), nil},
	{
		newTestResponse(403, `{"id": "forbidden", "message": "You do not have access to the app myapp."}`),
		Error{
			error: errors.New("You do not have access to the app myapp."),
			Id:    "forbidden",
			URL:   "",
		},
	},
	{
		newTestResponse(401, `{"id": "unauthorized", "message": "Long error message."}`),
		Error{
			error: errors.New("Long error message."),
			Id:    "unauthorized",
			URL:   "",
		},
	},
	{
		newTestResponse(422, `{"id": "invalid_params", "message": "Cannot scale to more than 5 PX size dynos per process type.", "url": "https://bit.ly/1gK1TvU"}`),
		Error{
			error: errors.New("Cannot scale to more than 5 PX size dynos per process type."),
			Id:    "invalid_params",
			URL:   "https://bit.ly/1gK1TvU",
		},
	},
	{
		newTestResponse(500, `not valid json {} ""`),
		errors.New("Unexpected error: Internal Server Error"),
	},
}

func TestCheckResp(t *testing.T) {
	for i, rt := range respTests {
		resp := checkResp(&rt.Response)
		if !reflect.DeepEqual(rt.Expected, resp) {
			t.Errorf("checkResp respTests[%d] expected %v, got %v", i, rt.Expected, resp)
		}
	}
}

// test helpers

func newTestRequest(method, path, body string, resp testnet.TestResponse) testnet.TestRequest {
	headers := http.Header{}
	headers.Set("Accept", "application/vnd.heroku+json; version=3")
	req := testnet.TestRequest{
		Method:   method,
		Path:     path,
		Response: resp,
		Header:   headers,
	}
	if method != "GET" && body != "" {
		req.Matcher = testnet.RequestBodyMatcher(body)
	}
	return req
}

func newTestServerAndClient(t *testing.T, requests ...testnet.TestRequest) (*httptest.Server, *testnet.Handler, *Client) {
	ts, handler := testnet.NewServer(t, requests)
	c := &Client{}
	c.URL = ts.URL

	return ts, handler, c
}

func testStringsEqual(t *testing.T, fieldName, expected, actual string) {
	if actual != expected {
		t.Errorf("%s expected %s, got %s", fieldName, expected, actual)
	}
}

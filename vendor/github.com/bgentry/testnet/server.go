// This package provides helpers for testing interactions with an HTTP API in
// the Go (#golang) programming language. It allows you to test that the
// expected HTTP requests are received by the `httptest.Server`, and return mock
// responses.
package testnet

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type TestRequest struct {
	Method   string
	Path     string
	Header   http.Header
	Matcher  RequestMatcher
	Response TestResponse
}

type RequestMatcher func(*testing.T, *http.Request)

type TestResponse struct {
	Body   string
	Status int
	Header http.Header
}

type Handler struct {
	Requests  []TestRequest
	CallCount int
	T         *testing.T
}

func (h *Handler) AllRequestsCalled() bool {
	return h.CallCount == len(h.Requests)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(h.Requests) <= h.CallCount {
		h.logError("Index out of range! Test server called too many times. Final Request:", r.Method, r.RequestURI)
		return
	}

	tester := h.Requests[h.CallCount]
	h.CallCount++

	// match method
	if tester.Method != r.Method {
		h.logError("Method does not match.\nExpected: %s\nActual:   %s", tester.Method, r.Method)
	}

	// match path
	paths := strings.Split(tester.Path, "?")
	if paths[0] != r.URL.Path {
		h.logError("Path does not match.\nExpected: %s\nActual:   %s", paths[0], r.URL.Path)
	}
	// match query string
	if len(paths) > 1 {
		if !strings.Contains(r.URL.RawQuery, paths[1]) {
			h.logError("Query string does not match.\nExpected: %s\nActual:   %s", paths[1], r.URL.RawQuery)
		}
	}

	for key, values := range tester.Header {
		key = http.CanonicalHeaderKey(key)
		actualValues := strings.Join(r.Header[key], ";")
		expectedValues := strings.Join(values, ";")

		if key == "Authorization" && !strings.Contains(actualValues, expectedValues) {
			h.logError("%s header is not contained in actual value.\nExpected: %s\nActual:   %s", key, expectedValues, actualValues)
		}
		if key != "Authorization" && actualValues != expectedValues {
			h.logError("%s header did not match.\nExpected: %s\nActual:   %s", key, expectedValues, actualValues)
		}
	}

	// match custom request matcher
	if tester.Matcher != nil {
		tester.Matcher(h.T, r)
	}

	// set response headers
	header := w.Header()
	for name, values := range tester.Response.Header {
		if len(values) < 1 {
			continue
		}
		header.Set(name, values[0])
	}

	// write response
	w.WriteHeader(tester.Response.Status)
	fmt.Fprintln(w, tester.Response.Body)
}

func NewServer(t *testing.T, requests []TestRequest) (s *httptest.Server, h *Handler) {
	h = &Handler{
		Requests: requests,
		T:        t,
	}
	s = httptest.NewServer(h)
	return
}

func NewTLSServer(t *testing.T, requests []TestRequest) (s *httptest.Server, h *Handler) {
	h = &Handler{
		Requests: requests,
		T:        t,
	}
	s = httptest.NewTLSServer(h)
	return
}

func (h *Handler) logError(msg string, args ...interface{}) {
	h.T.Logf(msg, args...)
	h.T.Fail()
}

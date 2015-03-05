// Copyright 2014 Eric Holmes.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hookshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type payload struct {
	event string `json:"event"`
}

func TestRouterAuthorized(t *testing.T) {
	tests := []struct {
		secret    string
		event     string
		body      string
		signature string

		status int
	}{
		{
			secret:    "1234",
			event:     "",
			body:      `{"event":"data"}`,
			signature: "invalid",
			status:    404,
		},
		{
			secret:    "1234",
			event:     "foobar",
			body:      `{"event":"data"}`,
			signature: "invalid",
			status:    404,
		},
		{
			secret:    "1234",
			event:     "deployment",
			body:      `{"event":"data"}`,
			signature: "invalid",
			status:    403,
		},
		{
			secret:    "1234",
			event:     "deployment",
			body:      `{"event":"data"}`,
			signature: "sha1=ade133892a181fba3a21c163cd5cbc3f5f8e915c",
			status:    200,
		},
		{
			secret: "1234",
			event:  "deployment",
			body:   `{"event":"data"}`,
			status: 403,
		},
		{
			secret: "",
			event:  "deployment",
			body:   `{"event":"data"}`,
			status: 200,
		},
	}

	for _, tt := range tests {
		router := NewRouter()

		router.Handle("deployment", Authorize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var p payload
			err := json.NewDecoder(r.Body).Decode(&p)

			if err != nil {
				t.Fatalf("Could not read from request body: %v", err)
			}

			w.WriteHeader(200)
			w.Write([]byte("ok\n"))
		}), tt.secret))

		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/", bytes.NewReader([]byte(tt.body)))

		if tt.event != "" {
			req.Header.Set("X-GitHub-Event", tt.event)
		}

		if tt.signature != "" {
			req.Header.Set("X-Hub-Signature", tt.signature)
		}

		router.ServeHTTP(resp, req)

		if resp.Code != tt.status {
			t.Errorf("resp.Code => %v; want %v", resp.Code, tt.status)
		}

		expectedBody := ""
		switch tt.status {
		case 200:
			expectedBody = "ok\n"
		case 404:
			expectedBody = "404 page not found\n"
		case 403:
			expectedBody = "The provided signature in the X-Hub-Signature header does not match.\n"
		}

		if resp.Body.String() != expectedBody {
			t.Errorf("resp.Body => %q; want %q", resp.Body.String(), expectedBody)
		}

		if resp.Header().Get("X-Calculated-Signature") != "" {
			t.Errorf("resp.Header[X-Calculated-Signature] => %q; want %q", resp.Header().Get("X-Calculated-Signature"), "")
		}
	}
}

func TestSignature(t *testing.T) {
	tests := []struct {
		in     string
		secret string

		signature string
	}{
		{
			`{"event":"data"}`,
			"1234",
			"ade133892a181fba3a21c163cd5cbc3f5f8e915c",
		},
	}

	for _, tt := range tests {
		signature := Signature([]byte(tt.in), tt.secret)

		if signature != tt.signature {
			t.Errorf("Signature(%q, %q) => %q; want %q", tt.in, tt.secret, signature, tt.signature)
		}
	}
}

func ExampleRouterHandleFunc() {
	r := NewRouter()
	r.Handle("ping", Authorize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`pong`))
	}), "secret"))

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "", bytes.NewBufferString(`{"data":"foo"}`))
	req.Header.Set("X-GitHub-Event", "ping")
	req.Header.Set("X-Hub-Signature", "sha1=b3dc4e9a2d727ee1e60bb6828c2dcef88b5ec970")

	r.ServeHTTP(res, req)

	fmt.Print(res.Body)
	// Output: pong
}

func ExampleIsAuthorized() {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := IsAuthorized(r, "secret"); !ok {
			http.Error(w, "The provided signature in the "+HeaderSignature+" header does not match.", 403)
			return
		}

		w.Write([]byte(`Ok`))
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "", bytes.NewBufferString(`{"data":"foo"}`))
	req.Header.Set("X-Hub-Signature", "sha1=b3dc4e9a2d727ee1e60bb6828c2dcef88b5ec970")

	h.ServeHTTP(res, req)

	fmt.Print(res.Body)
	// Output: Ok
}

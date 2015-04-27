package hb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

var (
	// boom
	errBoom = errors.New("boom")

	// boom with backtrace.
	errBoomMore = reporter.NewError(errBoom, 0)
)

func TestSend(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("X-Api-Key"), "1234"; got != want {
			t.Fatal("API Key => %s; want %s", got, want)
		}

		w.WriteHeader(200)
	}))
	defer s.Close()

	c := NewClientFromKey("1234")
	c.URL = s.URL
	r := &Reporter{
		client: c,
	}

	if err := r.Report(context.Background(), reporter.NewError(errBoom, 0)); err != nil {
		t.Fatal(err)
	}
}

func TestNewReport(t *testing.T) {
	tests := []struct {
		err     error
		fixture string

		// if set to true, removes environment specific information from
		// the backtrace.
		truncBacktrace bool
	}{
		// With a basic error
		{
			err:     errBoom,
			fixture: "boom.json",
		},

		// With a typed error
		{
			err:     &typedError{errBoom},
			fixture: "boom-typed.json",
		},

		// With a reporter.Error.
		{
			err:            errBoomMore,
			fixture:        "boom-more.json",
			truncBacktrace: true,
		},

		// reporter.Error with contextual information.
		{
			err: &reporter.Error{
				Err: errBoom,
				Context: map[string]interface{}{
					"request_id": "1234",
				},
				Request: func() *http.Request {
					req, _ := http.NewRequest("GET", "/api/foo", nil)
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("X-Forwarded-For", "127.0.0.1")
					req.SetBasicAuth("Foo", "bar")
					return req
				}(),
			},
			fixture: "boom-request.json",
		},
	}

	for i, tt := range tests {
		report := NewReport(tt.err)

		if tt.truncBacktrace {
			for _, line := range report.Error.Backtrace {
				line.File = fmt.Sprintf("(removed)/%s", path.Base(line.File))
				line.Number = "(removed)"
			}
		}

		fixture := fmt.Sprintf("test-fixtures/%s", tt.fixture)

		f, err := ioutil.ReadFile(fixture)
		if err != nil {
			t.Fatal(err)
		}

		raw, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatal(err)
		}

		if got, want := string(raw), string(f); got != want {
			if err := ioutil.WriteFile(fixture, raw, 0644); err != nil {
				t.Fatal(err)
			}

			t.Errorf("#%d: Report => %v; want %v", i, got, want)
		}
	}
}

type typedError struct {
	error
}

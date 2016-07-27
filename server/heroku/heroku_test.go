package heroku

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/remind101/empire"
)

func TestNew(t *testing.T) {
	New(nil)
}

func TestEncode(t *testing.T) {
	tests := []struct {
		in  interface{}
		out string
	}{
		{nil, "{}\n"},
		{map[string]string{"foo": "bar"}, `{"foo":"bar"}` + "\n"},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()

		if err := Encode(w, tt.in); err != nil {
			t.Fatal(err)
		}

		if got, want := w.Body.String(), tt.out; got != want {
			t.Errorf("Encode(%v) => %q; want %q", tt.in, got, want)
		}
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		err    error
		status int

		out  string
		code int
	}{
		{errors.New("fuck"), 400, `{"id":"","message":"fuck","url":""}` + "\n", 400},
		{ErrNotFound, 400, `{"id":"not_found","message":"Request failed, the specified resource does not exist","url":""}` + "\n", 404},
		{&ErrorResource{Message: "custom"}, 400, `{"id":"","message":"custom","url":""}` + "\n", 400},
		{&empire.ValidationError{Err: errors.New("boom")}, 500, `{"id":"bad_request","message":"Request invalid, validate usage and try again","url":""}` + "\n", 400},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()

		if err := Error(w, tt.err, tt.status); err != nil {
			t.Fatal(err)
		}

		if got, want := w.Body.String(), tt.out; got != want {
			t.Errorf("Error(%v, %d) => %s; want %s", tt.err, tt.status, got, want)
		}

		if got, want := w.Code, tt.code; got != want {
			t.Errorf("Error(%v, %d) Status => %d; want %d", tt.err, tt.status, tt.status, tt.code)
		}
	}
}

package httpmock

import (
	"net/http"
	"strings"
	"testing"
)

// ServeReplay is an http.Handler
// It contains a list of handlers and calls the next handler in the list for each incoming request.
//
// If a request is received and no more handlers are available, ServeReplay calls NoneLeftFunc.
// The default behavior of NoneLeftFunc is to call t.Errorf with some information about the request.
type ServeReplay struct {
	t            *testing.T
	i            int
	Handlers     []http.Handler
	NoneLeftFunc func(*testing.T, *http.Request)
}

func defaultNoneLeftFunc(t *testing.T, r *http.Request) {
	t.Errorf("http request: %s %s; no more handlers to call", r.Method, r.URL.Path)
}

// NewServeReplay returns a new ServeReplay.
func NewServeReplay(t *testing.T) *ServeReplay {
	return &ServeReplay{
		t:            t,
		Handlers:     make([]http.Handler, 0),
		NoneLeftFunc: defaultNoneLeftFunc,
	}
}

// Add appends a handler to ServReplay's handler list.
// It returns itself to allow chaining.
func (h *ServeReplay) Add(handler http.Handler) *ServeReplay {
	h.Handlers = append(h.Handlers, handler)
	return h
}

// ServeHTTP dispatches the request to the next handler in the list.
func (h *ServeReplay) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.i >= len(h.Handlers) {
		h.NoneLeftFunc(h.t, r)
	} else {
		h.Handlers[h.i].ServeHTTP(w, r)
		h.i++
	}
}

// PathHandler will fail if the request doesn't match based on reqPath.
// If it matches it returns the given response status and body.
//
// reqPath should be of the form `<METHOD> <PATH>`. For example: `GET /foo`.
func PathHandler(t *testing.T, reqPath string, respStatus int, respBody string) http.Handler {
	var meth, path string

	s := strings.SplitN(reqPath, " ", 2)
	if len(s) == 1 {
		t.Fatal("reqPath must be of the form `<METHOD> <PATH>`")
	}

	meth = s[0]
	path = s[1]

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if meth == r.Method && path == r.URL.Path {
			w.WriteHeader(respStatus)
			w.Write([]byte(respBody))
		} else {
			w.WriteHeader(http.StatusNotFound)
			t.Errorf("http request => %s %s; want %s %s", r.Method, r.URL.Path, meth, path)
		}
	})
}

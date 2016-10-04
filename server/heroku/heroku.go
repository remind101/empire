// Package heroku provides a Heroku Platform API compatible http.Handler
// implementation for Empire.
package heroku

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"runtime"
	"time"

	"github.com/gorilla/mux"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/headerutil"
	"github.com/remind101/empire/pkg/heroku"
	"github.com/remind101/empire/server/auth"
	"github.com/remind101/empire/stats"
	"github.com/remind101/pkg/reporter"
)

// The Accept header that controls the api version. See
// https://devcenter.heroku.com/articles/platform-api-reference#clients
const AcceptHeader = "application/vnd.heroku+json; version=3"

// Server provides an http.Handler for serving the Heroku compatible API.
type Server struct {
	*empire.Empire

	// Authenticator is the auth.Authenticator that will be used to
	// authenticate requests.
	Authenticator auth.Authenticator

	mux *mux.Router
}

// handler represents an http.Handler within this package, which augments the
// standard http.HandlerFunc signature to allow for an error to be returned, for
// generic error handling.
type handler func(w http.ResponseWriter, r *http.Request) error

// New returns a new Server instance to serve the Heroku compatible API.
func New(e *empire.Empire) *Server {
	r := &Server{
		Empire: e,
		mux:    mux.NewRouter(),
	}

	// Apps
	r.handle("GET", "/apps", r.GetApps)                  // hk apps
	r.handle("GET", "/apps/{app}", r.GetAppInfo)         // hk info
	r.handle("DELETE", "/apps/{app}", r.DeleteApp)       // hk destroy
	r.handle("PATCH", "/apps/{app}", r.PatchApp)         // hk destroy
	r.handle("POST", "/apps/{app}/deploys", r.DeployApp) // Deploy an image to an app
	r.handle("POST", "/apps", r.PostApps)                // hk create
	r.handle("POST", "/organizations/apps", r.PostApps)  // hk create

	// Domains
	r.handle("GET", "/apps/{app}/domains", r.GetDomains)                 // hk domains
	r.handle("POST", "/apps/{app}/domains", r.PostDomains)               // hk domain-add
	r.handle("DELETE", "/apps/{app}/domains/{hostname}", r.DeleteDomain) // hk domain-remove

	// Deploys
	r.handle("POST", "/deploys", r.PostDeploys) // Deploy an app

	// Releases
	r.handle("GET", "/apps/{app}/releases", r.GetReleases)          // hk releases
	r.handle("GET", "/apps/{app}/releases/{version}", r.GetRelease) // hk release-info
	r.handle("POST", "/apps/{app}/releases", r.PostReleases)        // hk rollback

	// Configs
	r.handle("GET", "/apps/{app}/config-vars", r.GetConfigs)     // hk env, hk get
	r.handle("PATCH", "/apps/{app}/config-vars", r.PatchConfigs) // hk set, hk unset

	// Processes
	r.handle("GET", "/apps/{app}/dynos", r.GetProcesses)                     // hk dynos
	r.handle("POST", "/apps/{app}/dynos", r.PostProcess)                     // hk run
	r.handle("DELETE", "/apps/{app}/dynos", r.DeleteProcesses)               // hk restart
	r.handle("DELETE", "/apps/{app}/dynos/{ptype}.{pid}", r.DeleteProcesses) // hk restart web.1
	r.handle("DELETE", "/apps/{app}/dynos/{pid}", r.DeleteProcesses)         // hk restart web

	// Formations
	r.handle("GET", "/apps/{app}/formation", r.GetFormation)     // hk scale -l
	r.handle("PATCH", "/apps/{app}/formation", r.PatchFormation) // hk scale

	// OAuth
	r.handle("POST", "/oauth/authorizations", r.PostAuthorizations)

	// SSL
	sslRemoved := errHandler(ErrSSLRemoved)
	r.handle("GET", "/apps/{app}/ssl-endpoints", sslRemoved)           // hk ssl
	r.handle("POST", "/apps/{app}/ssl-endpoints", sslRemoved)          // hk ssl-cert-add
	r.handle("PATCH", "/apps/{app}/ssl-endpoints/{cert}", sslRemoved)  // hk ssl-cert-add, hk ssl-cert-rollback
	r.handle("DELETE", "/apps/{app}/ssl-endpoints/{cert}", sslRemoved) // hk ssl-destroy

	// Logs
	r.handle("POST", "/apps/{app}/log-sessions", r.PostLogs) // hk log

	return r
}

// ServeHTTPContext implements the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
	return
}

// handle adds a new handler to the router, which also increments a counter.
func (s *Server) handle(method, path string, h handler) {
	name := handlerName(h)

	// Wrap handler to perform authentication, and track metrics.
	m := handler(func(w http.ResponseWriter, r *http.Request) error {
		r, err := s.Authenticate(r)
		if err != nil {
			return err
		}

		ctx := r.Context()

		tags := []string{
			fmt.Sprintf("handler:%s", name),
			fmt.Sprintf("user:%s", UserFromContext(ctx).Name),
		}
		start := time.Now()
		err = h(w, r)
		d := time.Since(start)
		stats.Timing(ctx, fmt.Sprintf("heroku.request"), d, 1.0, tags)
		stats.Timing(ctx, fmt.Sprintf("heroku.request.%s", name), d, 1.0, tags)
		return err
	})

	// Encode and report errors.
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		err := m(w, r)
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
			reporter.Report(ctx, err)
		}
		return
	})

	s.mux.HandleFunc(path, fn).Methods(method)
}

// Encode json encodes v into w.
func Encode(w http.ResponseWriter, v interface{}) error {
	if v == nil {
		// Empty JSON body "{}"
		v = map[string]interface{}{}
	}

	return json.NewEncoder(w).Encode(v)
}

// DecodeRequest json decodes the request body into v, optionally ignoring EOF
// errors to handle cases where the request body might be empty.
func DecodeRequest(r *http.Request, v interface{}, ignoreEOF bool) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		if err == io.EOF && ignoreEOF {
			return nil
		}
		return fmt.Errorf("error decoding request body: %v", err)
	}
	return nil
}

// Decode json decodes the request body into v.
func Decode(r *http.Request, v interface{}) error {
	return DecodeRequest(r, v, false)
}

// Stream encodes and flushes data to the client.
func Stream(w http.ResponseWriter, v interface{}) error {
	if err := Encode(w, v); err != nil {
		return err
	}

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	return nil
}

// Error is used to respond with errors in the heroku error format, which is
// specified at
// https://devcenter.heroku.com/articles/platform-api-reference#errors
//
// If an ErrorResource is provided as the error, and it provides a non-zero
// status, that will be used as the response status code.
func Error(w http.ResponseWriter, err error, status int) error {
	res := newError(err)

	// If the ErrorResource provides and exit status, we'll use that
	// instead.
	if res.Status != 0 {
		status = res.Status
	}

	w.WriteHeader(status)
	return Encode(w, res)
}

// NoContent responds with a 404 and an empty body.
func NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// RangeHeader parses the Range header and returns an headerutil.Range.
func RangeHeader(r *http.Request) (headerutil.Range, error) {
	header := r.Header.Get("Range")
	if header == "" {
		return headerutil.Range{}, nil
	}

	rangeHeader, err := headerutil.ParseRange(header)
	if err != nil {
		return headerutil.Range{}, err
	}
	return *rangeHeader, nil
}

// key used to store context values from within this package.
type key int

const (
	userKey key = 0
)

// WithUser adds a user to the context.Context.
func WithUser(ctx context.Context, u *empire.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// UserFromContext returns a user from a context.Context if one is present.
func UserFromContext(ctx context.Context) *empire.User {
	u, ok := ctx.Value(userKey).(*empire.User)
	if !ok {
		panic("expected user to be authenticated")
	}
	return u
}

func findMessage(r *http.Request) (string, error) {
	h := r.Header.Get(heroku.CommitMessageHeader)
	return h, nil
}

var nameRegexp = regexp.MustCompile(`^.*\.(.*)-fm$`)

// handlerName returns the name of the handler, which can be used as a metrics
// postfix.
func handlerName(h handler) string {
	name := runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	matches := nameRegexp.FindStringSubmatch(name)
	if len(matches) < 2 {
		return "Unknown"
	}
	return matches[1]
}

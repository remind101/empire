package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/remind101/empire/empire"
	"golang.org/x/net/context"
)

// Named matching heroku's error codes. See
// https://devcenter.heroku.com/articles/platform-api-reference#error-responses
var (
	ErrBadRequest = &ErrorResource{
		Status:  http.StatusBadRequest,
		ID:      "bad_request",
		Message: "Request invalid, validate usage and try again",
	}
	ErrUnauthorized = &ErrorResource{
		Status:  http.StatusUnauthorized,
		ID:      "unauthorized",
		Message: "Request not authenticated, API token is missing, invalid or expired",
	}
	ErrForbidden = &ErrorResource{
		Status:  http.StatusForbidden,
		ID:      "forbidden",
		Message: "Request not authorized, provided credentials do not provide access to specified resource",
	}
	ErrNotFound = &ErrorResource{
		Status:  http.StatusNotFound,
		ID:      "not_found",
		Message: "Request failed, the specified resource does not exist",
	}
	ErrTwoFactor = &ErrorResource{
		Status:  http.StatusUnauthorized,
		ID:      "two_factor",
		Message: "Two factor code is required.",
	}
)

var DefaultOptions = Options{}

type Options struct {
	GitHub struct {
		ClientID     string
		ClientSecret string
		Organization string
	}
}

// Server represents the API.
type Server struct {
	http.Handler
}

// New creates the API routes and returns a new Server instance.
func New(e *empire.Empire, options Options) *Server {
	r := newRouter()

	// Apps
	r.Handle("GET", "/apps", Authenticate(e, &GetApps{e}))                 // hk apps
	r.Handle("DELETE", "/apps/{app}", Authenticate(e, &DeleteApp{e}))      // hk destroy
	r.Handle("POST", "/apps", Authenticate(e, &PostApps{e}))               // hk create
	r.Handle("POST", "/organizations/apps", Authenticate(e, &PostApps{e})) // hk create

	// Deploys
	r.Handle("POST", "/deploys", Authenticate(e, &PostDeploys{e})) // Deploy an app

	// Releases
	r.Handle("GET", "/apps/{app}/releases", Authenticate(e, &GetReleases{e}))          // hk releases
	r.Handle("GET", "/apps/{app}/releases/{version}", Authenticate(e, &GetRelease{e})) // hk release-info
	r.Handle("POST", "/apps/{app}/releases", Authenticate(e, &PostReleases{e}))        // hk rollback

	// Configs
	r.Handle("GET", "/apps/{app}/config-vars", Authenticate(e, &GetConfigs{e}))     // hk env, hk get
	r.Handle("PATCH", "/apps/{app}/config-vars", Authenticate(e, &PatchConfigs{e})) // hk set

	// Processes
	r.Handle("GET", "/apps/{app}/dynos", Authenticate(e, &GetProcesses{e})) // hk dynos

	// Formations
	r.Handle("PATCH", "/apps/{app}/formation", Authenticate(e, &PatchFormation{e})) // hk scale

	// OAuth
	auth := NewAuthorizer(options.GitHub.ClientID, options.GitHub.ClientSecret, options.GitHub.Organization)
	r.Handle("POST", "/oauth/authorizations", &PostAuthorizations{e, auth})

	n := negroni.Classic()
	n.UseHandler(r)

	return &Server{n}
}

// ErrorResource represents the error response format that we return.
type ErrorResource struct {
	Status  int    `json:"-"`
	ID      string `json:"id"`
	Message string `json:"message"`
	URL     string `json:"url"`
}

// Error implements error interface.
func (e *ErrorResource) Error() string {
	return e.Message
}

// router is an http router for Handlers.
type router struct {
	*mux.Router
}

// newRouter returns a new router instance.
func newRouter() *router {
	return &router{Router: mux.NewRouter()}
}

func (r *router) Handle(method, path string, h Handler) {
	r.Router.Handle(path, &handler{h}).Methods(method)
}

// Handler is represents a Handler that can take a context.Context as the
// first argument.
type Handler interface {
	ServeHTTPContext(context.Context, http.ResponseWriter, *http.Request) error
}

type HandlerFunc func(context.Context, http.ResponseWriter, *http.Request) error

func (fn HandlerFunc) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return fn(ctx, w, r)
}

// handler adapts a Handler to an http.Handler. It's the entrypoint from the
// http.Handler router to Handlers within package server.
type handler struct {
	Handler
}

// ServeHTTP calls the Handler. If an error is returned, the error will be
// encoded into the response.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := context.Background()

	if err := h.Handler.ServeHTTPContext(ctx, w, r); err != nil {
		Error(w, err, http.StatusInternalServerError)
	}
}

// Encode json ecnodes v into w.
func Encode(w http.ResponseWriter, v interface{}) error {
	if v == nil {
		// Empty JSON body "{}"
		v = map[string]interface{}{}
	}

	return json.NewEncoder(w).Encode(v)
}

// Decode json decodes the request body into v.
func Decode(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// Error is used to respond with errors in the heroku error format, which is
// specified at
// https://devcenter.heroku.com/articles/platform-api-reference#errors
//
// If an ErrorResource is provided as the error, and it provides a non-zero
// status, that will be used as the response status code.
func Error(w http.ResponseWriter, err error, status int) error {
	var v interface{}
	switch err := err.(type) {
	case *ErrorResource:
		if err.Status != 0 {
			status = err.Status
		}

		v = err
	case *empire.ValidationError:
		v = ErrBadRequest
	default:
		v = &ErrorResource{
			Message: err.Error(),
		}
	}

	log.Printf("error=%+v\n", v)
	w.WriteHeader(status)
	return Encode(w, v)
}

// NoContent responds with a 404 and an empty body.
func NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return Encode(w, nil)
}

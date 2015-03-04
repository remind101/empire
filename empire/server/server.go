package server

import (
	"encoding/json"
	"net/http"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/remind101/empire/empire"
)

// Named matching heroku's error codes. See
// https://devcenter.heroku.com/articles/platform-api-reference#error-responses
var (
	ErrBadRequest = &ErrorResource{
		Status:  http.StatusBadRequest,
		ID:      "bad_request",
		Message: "Request invalid, validate usage and try again",
	}
	ErrNotFound = &ErrorResource{
		Status:  http.StatusNotFound,
		ID:      "not_found",
		Message: "Request failed, the specified resource does not exist",
	}
)

// Server represents the API.
type Server struct {
	http.Handler
}

// New creates the API routes and returns a new Server instance.
func New(e *empire.Empire) *Server {
	r := newRouter()

	// Apps
	r.Handle("GET", "/apps", &GetApps{e})                 // hk apps
	r.Handle("DELETE", "/apps/{app}", &DeleteApp{e})      // hk destroy
	r.Handle("POST", "/apps", &PostApps{e})               // hk create
	r.Handle("POST", "/organizations/apps", &PostApps{e}) // hk create

	// Deploys
	r.Handle("POST", "/deploys", &PostDeploys{e}) // Deploy an app

	// Releases
	r.Handle("GET", "/apps/{app}/releases", &GetReleases{e})   // hk releases
	r.Handle("POST", "/apps/{app}/releases", &PostReleases{e}) // hk rollback

	// Configs
	r.Handle("GET", "/apps/{app}/config-vars", &GetConfigs{e})     // hk env, hk get
	r.Handle("PATCH", "/apps/{app}/config-vars", &PatchConfigs{e}) // hk set

	// Processes
	r.Handle("GET", "/apps/{app}/dynos", &GetProcesses{e}) // hk dynos

	// Formations
	r.Handle("PATCH", "/apps/{app}/formation", &PatchFormation{e}) // hk scale

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

type Handler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request) error
}

func (r *router) Handle(method, path string, h Handler) {
	r.Router.Handle(path, &handler{h}).Methods(method)
}

// handler adapts a Handler to an http.Handler.
type handler struct {
	Handler
}

// ServeHTTP calls the Hander. If an error is returned, the error will be
// encoded into the response.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h.Handler.ServeHTTP(w, r); err != nil {
		Error(w, err, http.StatusInternalServerError)
	}
}

// Encode json ecnodes v into w.
func Encode(w http.ResponseWriter, v interface{}) error {
	if v == nil {
		// Empty JSON body "{}"
		v = map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
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
	default:
		v = &ErrorResource{
			Message: err.Error(),
		}
	}

	w.WriteHeader(status)
	return Encode(w, v)
}

// NoContent responds with a 404 and an empty body.
func NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return Encode(w, nil)
}

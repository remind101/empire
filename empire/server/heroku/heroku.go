package heroku

import (
	"encoding/json"
	"net/http"

	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/pkg/httpx"
	"github.com/remind101/empire/empire/pkg/httpx/middleware"
)

// The Accept header that controls the api version. See
// https://devcenter.heroku.com/articles/platform-api-reference#clients
const AcceptHeader = "application/vnd.heroku+json; version=3"

// New creates the API routes and returns a new http.Handler to serve them.
func New(e *empire.Empire, auth Authorizer) http.Handler {
	r := httpx.NewRouter()

	// Apps
	r.Handle("GET", "/apps", Middleware(e, &GetApps{e}, nil))                 // hk apps
	r.Handle("DELETE", "/apps/{app}", Middleware(e, &DeleteApp{e}, nil))      // hk destroy
	r.Handle("POST", "/apps", Middleware(e, &PostApps{e}, nil))               // hk create
	r.Handle("POST", "/organizations/apps", Middleware(e, &PostApps{e}, nil)) // hk create

	// Deploys
	r.Handle("POST", "/deploys", Middleware(e, &PostDeploys{e}, nil)) // Deploy an app

	// Releases
	r.Handle("GET", "/apps/{app}/releases", Middleware(e, &GetReleases{e}, nil))          // hk releases
	r.Handle("GET", "/apps/{app}/releases/{version}", Middleware(e, &GetRelease{e}, nil)) // hk release-info
	r.Handle("POST", "/apps/{app}/releases", Middleware(e, &PostReleases{e}, nil))        // hk rollback

	// Configs
	r.Handle("GET", "/apps/{app}/config-vars", Middleware(e, &GetConfigs{e}, nil))     // hk env, hk get
	r.Handle("PATCH", "/apps/{app}/config-vars", Middleware(e, &PatchConfigs{e}, nil)) // hk set

	// Processes
	r.Handle("GET", "/apps/{app}/dynos", Middleware(e, &GetProcesses{e}, nil)) // hk dynos

	// Formations
	r.Handle("PATCH", "/apps/{app}/formation", Middleware(e, &PatchFormation{e}, nil)) // hk scale

	// OAuth
	r.Handle("POST", "/oauth/authorizations", Middleware(e, &PostAuthorizations{e, auth}, &MiddlewareOpts{DisableAuthenticate: true}))

	// Wrap the router in middleware to handle errors.
	h := middleware.HandleError(r, func(err error, w http.ResponseWriter, r *http.Request) {
		Error(w, err, http.StatusInternalServerError)
	})

	// Wrap the route in middleware to add a context.Context.
	b := middleware.BackgroundContext(h)

	return b
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

	w.WriteHeader(status)
	return Encode(w, v)
}

// NoContent responds with a 404 and an empty body.
func NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return Encode(w, nil)
}

package heroku

import (
	"encoding/json"
	"net/http"

	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/server/authorization"
	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/httpx/middleware"
)

// The Accept header that controls the api version. See
// https://devcenter.heroku.com/articles/platform-api-reference#clients
const AcceptHeader = "application/vnd.heroku+json; version=3"

// New creates the API routes and returns a new http.Handler to serve them.
func New(e *empire.Empire, auth authorization.Authorizer) httpx.Handler {
	r := httpx.NewRouter()

	// Apps
	r.Handle("/apps", Authenticate(e, &GetApps{e})).Methods("GET")                 // hk apps
	r.Handle("/apps/{app}", Authenticate(e, &DeleteApp{e})).Methods("DELETE")      // hk destroy
	r.Handle("/apps", Authenticate(e, &PostApps{e})).Methods("POST")               // hk create
	r.Handle("/organizations/apps", Authenticate(e, &PostApps{e})).Methods("POST") // hk create

	// Domains
	r.Handle("/apps/{app}/domains", Authenticate(e, &GetDomains{e})).Methods("GET")                 // hk domains
	r.Handle("/apps/{app}/domains", Authenticate(e, &PostDomains{e})).Methods("POST")               // hk domain-add
	r.Handle("/apps/{app}/domains/{hostname}", Authenticate(e, &DeleteDomain{e})).Methods("DELETE") // hk domain-remove

	// Deploys
	r.Handle("/deploys", Authenticate(e, &PostDeploys{e})).Methods("POST") // Deploy an app

	// Releases
	r.Handle("/apps/{app}/releases", Authenticate(e, &GetReleases{e})).Methods("GET")          // hk releases
	r.Handle("/apps/{app}/releases/{version}", Authenticate(e, &GetRelease{e})).Methods("GET") // hk release-info
	r.Handle("/apps/{app}/releases", Authenticate(e, &PostReleases{e})).Methods("POST")        // hk rollback

	// Configs
	r.Handle("/apps/{app}/config-vars", Authenticate(e, &GetConfigs{e})).Methods("GET")     // hk env, hk get
	r.Handle("/apps/{app}/config-vars", Authenticate(e, &PatchConfigs{e})).Methods("PATCH") // hk set, hk unset

	// Processes
	r.Handle("/apps/{app}/dynos", Authenticate(e, &GetProcesses{e})).Methods("GET")                     // hk dynos
	r.Handle("/apps/{app}/dynos", Authenticate(e, &PostProcess{e})).Methods("POST")                     // hk run
	r.Handle("/apps/{app}/dynos", Authenticate(e, &DeleteProcesses{e})).Methods("DELETE")               // hk restart
	r.Handle("/apps/{app}/dynos/{ptype}.{pid}", Authenticate(e, &DeleteProcesses{e})).Methods("DELETE") // hk restart web.1
	r.Handle("/apps/{app}/dynos/{ptype}", Authenticate(e, &DeleteProcesses{e})).Methods("DELETE")       // hk restart web

	// Formations
	r.Handle("/apps/{app}/formation", Authenticate(e, &PatchFormation{e})).Methods("PATCH") // hk scale

	// OAuth
	r.Handle("/oauth/authorizations", &PostAuthorizations{e, auth}).Methods("POST")

	// SSL
	r.Handle("/apps/{app}/ssl-endpoints", &GetSSLEndpoints{e}).Methods("GET")             // hk ssl
	r.Handle("/apps/{app}/ssl-endpoints", &PostSSLEndpoints{e}).Methods("POST")           // hk ssl-cert-add
	r.Handle("/apps/{app}/ssl-endpoints/{cert}", &PatchSSLEndpoint{e}).Methods("PATCH")   // hk ssl-cert-add, hk ssl-cert-rollback
	r.Handle("/apps/{app}/ssl-endpoints/{cert}", &DeleteSSLEndpoint{e}).Methods("DELETE") // hk ssl-destroy

	errorHandler := func(err error, w http.ResponseWriter, r *http.Request) {
		Error(w, err, http.StatusInternalServerError)
	}

	return middleware.HandleError(r, errorHandler)
}

// Encode json encodes v into w.
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

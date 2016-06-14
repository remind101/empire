package heroku

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/net/context"

	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/headerutil"
	"github.com/remind101/empire/pkg/heroku"
	"github.com/remind101/empire/server/auth"
	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/reporter"
)

// The Accept header that controls the api version. See
// https://devcenter.heroku.com/articles/platform-api-reference#clients
const AcceptHeader = "application/vnd.heroku+json; version=3"

// New creates the API routes and returns a new http.Handler to serve them.
func New(e *empire.Empire, authenticator auth.Authenticator) httpx.Handler {
	r := httpx.NewRouter()

	// Apps
	r.Handle("/apps", &GetApps{e}).Methods("GET")                  // hk apps
	r.Handle("/apps/{app}", &GetAppInfo{e}).Methods("GET")         // hk info
	r.Handle("/apps/{app}", &DeleteApp{e}).Methods("DELETE")       // hk destroy
	r.Handle("/apps/{app}", &PatchApp{e}).Methods("PATCH")         // hk destroy
	r.Handle("/apps/{app}/deploys", &DeployApp{e}).Methods("POST") // Deploy an image to an app
	r.Handle("/apps", &PostApps{e}).Methods("POST")                // hk create
	r.Handle("/organizations/apps", &PostApps{e}).Methods("POST")  // hk create

	// Domains
	r.Handle("/apps/{app}/domains", &GetDomains{e}).Methods("GET")                 // hk domains
	r.Handle("/apps/{app}/domains", &PostDomains{e}).Methods("POST")               // hk domain-add
	r.Handle("/apps/{app}/domains/{hostname}", &DeleteDomain{e}).Methods("DELETE") // hk domain-remove

	// Deploys
	r.Handle("/deploys", &PostDeploys{e}).Methods("POST") // Deploy an app

	// Releases
	r.Handle("/apps/{app}/releases", &GetReleases{e}).Methods("GET")          // hk releases
	r.Handle("/apps/{app}/releases/{version}", &GetRelease{e}).Methods("GET") // hk release-info
	r.Handle("/apps/{app}/releases", &PostReleases{e}).Methods("POST")        // hk rollback

	// Configs
	r.Handle("/apps/{app}/config-vars", &GetConfigs{e}).Methods("GET")     // hk env, hk get
	r.Handle("/apps/{app}/config-vars", &PatchConfigs{e}).Methods("PATCH") // hk set, hk unset

	// Processes
	r.Handle("/apps/{app}/dynos", &GetProcesses{e}).Methods("GET")                     // hk dynos
	r.Handle("/apps/{app}/dynos", &PostProcess{e}).Methods("POST")                     // hk run
	r.Handle("/apps/{app}/dynos", &DeleteProcesses{e}).Methods("DELETE")               // hk restart
	r.Handle("/apps/{app}/dynos/{ptype}.{pid}", &DeleteProcesses{e}).Methods("DELETE") // hk restart web.1
	r.Handle("/apps/{app}/dynos/{either}", &DeleteProcesses{e}).Methods("DELETE")      // hk restart web|1e5a2da0-88ae-4888-8762-71d465d9c9c5

	// Formations
	r.Handle("/apps/{app}/formation", &GetFormation{e}).Methods("GET")     // hk scale -l
	r.Handle("/apps/{app}/formation", &PatchFormation{e}).Methods("PATCH") // hk scale

	// OAuth
	r.Handle("/oauth/authorizations", &PostAuthorizations{e}).Methods("POST")

	// SSL
	sslRemoved := errHandler(ErrSSLRemoved)
	r.Handle("/apps/{app}/ssl-endpoints", sslRemoved).Methods("GET")           // hk ssl
	r.Handle("/apps/{app}/ssl-endpoints", sslRemoved).Methods("POST")          // hk ssl-cert-add
	r.Handle("/apps/{app}/ssl-endpoints/{cert}", sslRemoved).Methods("PATCH")  // hk ssl-cert-add, hk ssl-cert-rollback
	r.Handle("/apps/{app}/ssl-endpoints/{cert}", sslRemoved).Methods("DELETE") // hk ssl-destroy

	// Logs
	r.Handle("/apps/{app}/log-sessions", &PostLogs{e}).Methods("POST") // hk log

	api := Authenticate(r, authenticator)

	return httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		err := api.ServeHTTPContext(ctx, w, r)
		if err != nil {
			Error(w, err, http.StatusInternalServerError)
			reporter.Report(ctx, err)
		}
		return nil
	})
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

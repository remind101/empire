package httpx

import (
	"net/http"

	"github.com/gorilla/mux"
	"golang.org/x/net/context"
)

// ErrorHandler represents a function that can handle the error returned from a
// Handler.
type ErrorHandler func(error, http.ResponseWriter, *http.Request)

// Router is an httpx.Handler router.
type Router struct {
	// ErrorHandler is a function that will be called when a handler returns
	// an error.
	ErrorHandler ErrorHandler

	// NotFoundHandler is a Handler that will be called when a route is not
	// found.
	NotFoundHandler Handler

	// This router is ultimately backed by a gorilla mux router.
	mux *mux.Router
}

// NewRouter returns a new Router instance.
func NewRouter() *Router {
	return &Router{
		mux: mux.NewRouter(),
	}
}

// Handle adds a new router that routes requests using the method verb against
// path to the given Handler.
func (r *Router) Handle(method, path string, h Handler) {
	// mux.Handler expects an http.Handler. We wrap the Hander in a handler,
	// which satisfies the http.Handler interface. When this route is
	// eventually used, it's type asserted back to a Handler.
	hh := &handler{h}

	r.mux.Handle(path, hh).Methods(method)
}

// Handler returns a Handler that can be used to serve the request. Most of this
// is pulled from http://goo.gl/tyxad8.
func (r *Router) Handler(req *http.Request) (h Handler, vars map[string]string) {
	var match mux.RouteMatch

	if r.mux.Match(req, &match) {
		h = match.Handler.(Handler)
		vars = match.Vars
		return
	}

	if r.NotFoundHandler == nil {
		h = HandlerFunc(NotFound)
		return
	}

	h = r.NotFoundHandler
	return
}

// ServeHTTP implements the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.ServeHTTPContext(context.Background(), w, req)
}

// ServeHTTPContext implements the Handler interface.
func (r *Router) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	h, vars := r.Handler(req)
	return h.ServeHTTPContext(WithVars(ctx, vars), w, req)
}

// Vars extracts the route vars from a context.Context.
func Vars(ctx context.Context) map[string]string {
	vars, ok := ctx.Value(varsKey).(map[string]string)
	if !ok {
		return map[string]string{}
	}

	return vars
}

// WithVars adds the vars to the context.Context.
func WithVars(ctx context.Context, vars map[string]string) context.Context {
	return context.WithValue(ctx, varsKey, vars)
}

// handler adapts a Handler to an http.Handler.
type handler struct {
	Handler
}

// ServeHTTP implements the http.Handler interface. This method is never
// actually called by this package, it's only used as a means to pass a Handler
// in and out of mux.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.ServeHTTPContext(context.Background(), w, r)
}

// NotFound is a HandlerFunc that just delegates off to http.NotFound.
func NotFound(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	http.NotFound(w, r)
	return nil
}

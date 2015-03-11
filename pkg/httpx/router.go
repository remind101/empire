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
//
// Note that Router does not implement the httpx.Handler interface, primarily
// because most Go http routers are tightly coupled to the http.Handler
// interface, or their own interface.
type Router struct {
	// ErrorHandler is a function that will be called when a handler returns
	// an error.
	ErrorHandler ErrorHandler

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
	r.mux.Handle(path, &handler{r.ErrorHandler, h}).Methods(method)
}

// ServeHTTP implements the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// handler adapts a Handler to an http.Handler.
type handler struct {
	errorHandler ErrorHandler
	handler      Handler
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	if err := h.handler.ServeHTTPContext(ctx, w, r); err != nil {
		h.errorHandler(err, w, r)
	}
}

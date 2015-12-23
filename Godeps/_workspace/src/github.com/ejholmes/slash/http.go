package slash

import (
	"net/http"

	"golang.org/x/net/context"
)

// Server adapts a Handler to be served over http.
type Server struct {
	Handler
	Context func() context.Context
}

// NewServer returns a new Server instance.
func NewServer(h Handler) *Server {
	return &Server{
		Handler: h,
	}
}

// ServeHTTP parses the Command from the incoming request then serves it using
// the Handler.
func (h *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var ctx = h.Context
	if ctx == nil {
		ctx = context.Background
	}

	if err := h.ServeHTTPContext(ctx(), w, r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	return
}

// ServeHTTPContext serves the http request with context.Context support.
func (h *Server) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	command, err := ParseRequest(r)
	if err != nil {
		return err
	}

	go h.ServeCommand(ctx, newResponder(command), command)

	return nil
}

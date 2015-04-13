package relay

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/httpx/middleware"
	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

func NewHTTPServer(ctx context.Context, r *Relay) http.Handler {
	m := httpx.NewRouter()

	m.Handle("GET", "/containers", &PostContainers{r})

	var h httpx.Handler

	// Recover from panics.
	h = middleware.Recover(m, reporter.NewLogReporter())

	// Add a logger to the context.
	h = middleware.NewLogger(h, os.Stdout)

	// Add the request id to the context.
	h = middleware.ExtractRequestID(h)

	// Wrap the route in middleware to add a context.Context.
	b := middleware.BackgroundContext(h)
	b.Generate = func() context.Context {
		return ctx
	}

	return http.Handler(b)
}

type Container struct {
	AttachURL string `json:"attach_url"`
}

type PostContainers struct {
	*Relay
}

func (h *PostContainers) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	id := h.NewSession()
	logger.Log(ctx, "at", "PostContainers", "session", id, "starting new container session")

	w.WriteHeader(201)
	return Encode(w, Container{AttachURL: id})
}

func Encode(w http.ResponseWriter, v interface{}) error {
	if v == nil {
		// Empty JSON body "{}"
		v = map[string]interface{}{}
	}

	return json.NewEncoder(w).Encode(v)
}

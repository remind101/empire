package relay

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/httpx/middleware"
	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

func NewHTTPHandler(r *Relay) http.Handler {
	m := httpx.NewRouter()

	m.Handle("POST", "/containers", &PostContainers{r})

	var h httpx.Handler

	// Recover from panics.
	h = middleware.Recover(m, reporter.NewLogReporter())

	// Add a logger to the context.
	h = middleware.NewLogger(h, os.Stdout)

	// Add the request id to the context.
	h = middleware.ExtractRequestID(h)

	// Wrap the route in middleware to add a context.Context.
	return middleware.BackgroundContext(h)
}

type PostContainersForm struct {
	Image   string            `json:"image"`
	Command string            `json:"command"`
	Env     map[string]string `json:"env"`
	Attach  bool              `json:"attach"`
}

type PostContainers struct {
	*Relay
}

func (h *PostContainers) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var form PostContainersForm

	if err := Decode(r, &form); err != nil {
		return err
	}

	id := h.NewSession()
	logger.Log(ctx, "at", "PostContainers", "session", id, "starting new container session")

	c := &Container{
		Image:     form.Image,
		Name:      strings.Join([]string{"run", id}, "."),
		Command:   form.Command,
		State:     "starting",
		Env:       form.Env,
		Attach:    form.Attach,
		AttachURL: strings.Join([]string{h.Host, id}, "/"),
	}

	if err := h.CreateContainer(c); err != nil {
		return err
	}

	w.WriteHeader(201)
	return Encode(w, c)
}

func Encode(w http.ResponseWriter, v interface{}) error {
	if v == nil {
		// Empty JSON body "{}"
		v = map[string]interface{}{}
	}

	return json.NewEncoder(w).Encode(v)
}

func Decode(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

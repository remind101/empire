package relay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

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

	// Handle errors
	errorHandler := func(err error, w http.ResponseWriter, r *http.Request) {
		Error(w, err, http.StatusInternalServerError)
	}

	h = middleware.HandleError(m, errorHandler)

	// Recover from panics.
	h = middleware.Recover(h, reporter.NewLogReporter())

	// Add a logger to the context.
	h = middleware.NewLogger(h, os.Stdout)

	// Add the request id to the context.
	h = middleware.ExtractRequestID(h)

	// Wrap the route in middleware to add a context.Context.
	return middleware.BackgroundContext(h)
}

type PostContainersForm struct {
	User    string            `json:"user"`
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

	if form.User == "" {
		form.User = "unknown"
	}

	name := h.GenContainerName(fmt.Sprintf("run.%s", form.User))
	logger.Log(ctx, "at", "PostContainers", "container-name", name, "starting new container session")

	c := &Container{
		Image:     form.Image,
		Name:      name,
		Command:   form.Command,
		State:     "starting",
		Env:       form.Env,
		Attach:    form.Attach,
		AttachURL: fmt.Sprintf("%s/%s", h.Host, name),
	}

	h.RegisterContainer(name, c)

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

// Error is used to respond with errors in the heroku error format, which is
// specified at
// https://devcenter.heroku.com/articles/platform-api-reference#errors
//
// If an ErrorResource is provided as the error, and it provides a non-zero
// status, that will be used as the response status code.
func Error(w http.ResponseWriter, err error, status int) error {
	var res *ErrorResource

	switch err := err.(type) {
	case *ErrorResource:
		res = err
	default:
		res = &ErrorResource{
			Message: err.Error(),
		}
	}

	// If the ErrorResource provides and exit status, we'll use that
	// instead.
	if res.Status != 0 {
		status = res.Status
	}

	w.WriteHeader(status)
	return Encode(w, res)
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

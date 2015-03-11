package heroku

import (
	"net/http"
	"time"

	"golang.org/x/net/context"
)

// dyno is a heroku compatible response struct to the hk dynos command.
type dyno struct {
	Command   string    `json:"command"`
	Name      string    `json:"name"`
	State     string    `json:"state"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GetProcesses struct {
	Empire
}

func (h *GetProcesses) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(r, h)
	if err != nil {
		return err
	}

	// Retrieve job states
	js, err := h.JobStatesByApp(a)
	if err != nil {
		return err
	}

	// Convert to hk compatible format
	dynos := make([]dyno, len(js))
	for i, j := range js {
		dynos[i] = dyno{
			Command:   string(j.Job.Command),
			Name:      string(j.Name),
			State:     j.State,
			UpdatedAt: j.Job.UpdatedAt,
		}
	}

	w.WriteHeader(200)
	return Encode(w, dynos)
}

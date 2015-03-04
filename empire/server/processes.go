package server

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/remind101/empire/empire"
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

func (h *GetProcesses) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	name := empire.AppName(vars["app"])

	a, err := h.AppsFind(name)
	if err != nil {
		return err
	}

	if a == nil {
		return ErrNotFound
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

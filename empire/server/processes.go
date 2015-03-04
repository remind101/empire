package server

import (
	"net/http"
	"time"

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

func (h *GetProcesses) Serve(req *Request) (int, interface{}, error) {
	name := empire.AppName(req.Vars["app"])

	a, err := h.AppsFind(name)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if a == nil {
		return http.StatusNotFound, nil, nil
	}

	// Retrieve job states
	js, err := h.JobStatesByApp(a)
	if err != nil {
		return http.StatusInternalServerError, nil, err
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

	return 200, dynos, nil
}

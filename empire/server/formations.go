package server

import (
	"net/http"
	"time"

	"github.com/remind101/empire/empire"
)

// formation is a heroku compatible response struct to the hk scale command.
type formation struct {
	Command   string    `json:"command"`
	CreatedAt time.Time `json:"created_at"`
	Id        string    `json:"id"`
	Quantity  int       `json:"quantity"`
	Size      string    `json:"size"`
	Type      string    `json:"type"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PatchFormation struct {
	AppsService      empire.AppsService
	ReleasesService  empire.ReleasesService
	ConfigsService   empire.ConfigsService
	SlugsService     empire.SlugsService
	ProcessesService empire.ProcessesRepository
	Manager          empire.Manager
}

type PatchFormationForm struct {
	Updates []struct {
		Process  string `json:"process"` // Refers to process type
		Quantity int    `json:"quantity"`
		Size     string `json:"size"`
	} `json:"updates"`
}

func (h *PatchFormation) Serve(req *Request) (int, interface{}, error) {
	var form PatchFormationForm

	if err := req.Decode(&form); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	name := empire.AppName(req.Vars["app"])

	a, err := h.AppsService.Find(name)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if a == nil {
		return http.StatusNotFound, nil, nil
	}

	qm := empire.ProcessQuantityMap{}
	for _, up := range form.Updates {
		qm[empire.ProcessType(up.Process)] = up.Quantity
	}

	r, err := h.ReleasesService.Head(a)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if r == nil {
		return http.StatusNotFound, nil, nil
	}

	config, err := h.ConfigsService.Find(r.ConfigID)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	slug, err := h.SlugsService.Find(r.SlugID)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	f, err := h.ProcessesService.All(r.ID)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	err = h.Manager.ScaleRelease(r, config, slug, f, qm)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	// Create the response object
	resp := make([]formation, len(form.Updates))
	for i, up := range form.Updates {
		resp[i] = formation{Type: up.Process, Quantity: up.Quantity, Size: "1X"}
	}

	return 200, resp, nil
}

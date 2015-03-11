package server

import (
	"net/http"
	"time"

	"github.com/remind101/empire/empire"
	"golang.org/x/net/context"
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
	Empire
}

type PatchFormationForm struct {
	Updates []struct {
		Process  string `json:"process"` // Refers to process type
		Quantity int    `json:"quantity"`
		Size     string `json:"size"`
	} `json:"updates"`
}

func (h *PatchFormation) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var form PatchFormationForm

	if err := Decode(r, &form); err != nil {
		return err
	}

	a, err := findApp(r, h)
	if err != nil {
		return err
	}

	qm := empire.ProcessQuantityMap{}
	for _, up := range form.Updates {
		qm[empire.ProcessType(up.Process)] = up.Quantity
	}

	release, err := h.ReleasesLast(a)
	if err != nil {
		return err
	}

	if release == nil {
		return ErrNotFound
	}

	config, err := h.ConfigsFind(release.ConfigID)
	if err != nil {
		return err
	}

	slug, err := h.SlugsFind(release.SlugID)
	if err != nil {
		return err
	}

	f, err := h.ProcessesAll(release)
	if err != nil {
		return err
	}

	err = h.ScaleRelease(release, config, slug, f, qm)
	if err != nil {
		return err
	}

	// Create the response object
	resp := make([]formation, len(form.Updates))
	for i, up := range form.Updates {
		resp[i] = formation{Type: up.Process, Quantity: up.Quantity, Size: "1X"}
	}

	w.WriteHeader(200)
	return Encode(w, resp)
}

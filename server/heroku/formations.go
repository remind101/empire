package heroku

import (
	"net/http"

	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/heroku"
	"golang.org/x/net/context"
)

type Formation heroku.Formation

type PatchFormation struct {
	*empire.Empire
}

type PatchFormationForm struct {
	Updates []struct {
		Process  string              `json:"process"` // Refers to process type
		Quantity int                 `json:"quantity"`
		Size     *empire.Constraints `json:"size"`
	} `json:"updates"`
}

func (h *PatchFormation) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var form PatchFormationForm

	if err := Decode(r, &form); err != nil {
		return err
	}

	app, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	// Create the response object
	var resp []*Formation
	for _, up := range form.Updates {
		p, err := h.Scale(ctx, empire.ScaleOpts{
			User:        UserFromContext(ctx),
			App:         app,
			Process:     up.Process,
			Quantity:    up.Quantity,
			Constraints: up.Size,
		})
		if err != nil {
			return err
		}
		resp = append(resp, &Formation{
			Type:     up.Process,
			Quantity: p.Quantity,
			Size:     p.Constraints().String(),
		})
	}

	w.WriteHeader(200)
	return Encode(w, resp)
}

// GetFormation returns the current Formation info for an App
type GetFormation struct {
	*empire.Empire
}

// ServeHTTPContext handles the http response
func (h *GetFormation) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	app, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	formation, err := h.ListScale(ctx, app)
	if err != nil {
		return err
	}

	var resp []*Formation
	for name, proc := range formation {
		resp = append(resp, &Formation{
			Type:     name,
			Quantity: proc.Quantity,
			Size:     proc.Constraints().String(),
		})
	}

	w.WriteHeader(200)
	return Encode(w, resp)
}

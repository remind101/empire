package heroku

import (
	"net/http"

	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/heroku"
	"golang.org/x/net/context"
)

type Formation heroku.Formation

type PatchFormationForm struct {
	Updates []struct {
		Process  string              `json:"process"` // Refers to process type
		Quantity int                 `json:"quantity"`
		Size     *empire.Constraints `json:"size"`
	} `json:"updates"`
}

func (h *Server) PatchFormation(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var form PatchFormationForm

	if err := Decode(r, &form); err != nil {
		return err
	}

	app, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	m, err := findMessage(r)
	if err != nil {
		return err
	}

	var updates []*empire.ProcessUpdate
	for _, up := range form.Updates {
		updates = append(updates, &empire.ProcessUpdate{
			Process:     up.Process,
			Quantity:    up.Quantity,
			Constraints: up.Size,
		})
	}
	ps, err := h.Scale(ctx, empire.ScaleOpts{
		User:    UserFromContext(ctx),
		App:     app,
		Updates: updates,
		Message: m,
	})
	if err != nil {
		return err
	}

	var resp []*Formation
	for i, p := range ps {
		up := updates[i]
		resp = append(resp, &Formation{
			Type:     up.Process,
			Quantity: p.Quantity,
			Size:     p.Constraints().String(),
		})
	}

	w.WriteHeader(200)
	return Encode(w, resp)
}

// ServeHTTPContext handles the http response
func (h *Server) GetFormation(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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

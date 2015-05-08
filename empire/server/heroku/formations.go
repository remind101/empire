package heroku

import (
	"net/http"

	"github.com/bgentry/heroku-go"
	"github.com/remind101/empire/empire"
	"golang.org/x/net/context"
)

type Formation heroku.Formation

type PatchFormation struct {
	*empire.Empire
}

type PatchFormationForm struct {
	Updates []struct {
		Process  empire.ProcessType  `json:"process"` // Refers to process type
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
		p, err := h.AppsScale(ctx, app, up.Process, up.Quantity, up.Size)
		if err != nil {
			return err
		}
		resp = append(resp, &Formation{
			Type:     string(p.Type),
			Quantity: p.Quantity,
			Size:     p.Constraints.String(),
		})
	}

	w.WriteHeader(200)
	return Encode(w, resp)
}

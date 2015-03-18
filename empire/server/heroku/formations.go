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
		Process  empire.ProcessType `json:"process"` // Refers to process type
		Quantity int                `json:"quantity"`
		Size     string             `json:"size"`
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

	for _, up := range form.Updates {
		if err := h.AppsScale(app, up.Process, up.Quantity); err != nil {
			return err
		}
	}

	// Create the response object
	resp := make([]*Formation, len(form.Updates))
	for i, up := range form.Updates {
		resp[i] = &Formation{
			Type:     string(up.Process),
			Quantity: up.Quantity,
			Size:     "1X",
		}
	}

	w.WriteHeader(200)
	return Encode(w, resp)
}

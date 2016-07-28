package heroku

import (
	"net/http"

	"github.com/remind101/empire"
	"golang.org/x/net/context"
)

func (h *Server) GetConfigs(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	c, err := h.Config(a)
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, c.Vars)
}

func (h *Server) PatchConfigs(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var configVars empire.Vars

	if err := Decode(r, &configVars); err != nil {
		return err
	}

	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	m, err := findMessage(r)
	if err != nil {
		return err
	}

	// Update the config
	c, err := h.Set(ctx, empire.SetOpts{
		User:    UserFromContext(ctx),
		App:     a,
		Vars:    configVars,
		Message: m,
	})
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, c.Vars)
}

package heroku

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/remind101/empire/empire"
	"golang.org/x/net/context"
)

type GetConfigs struct {
	Empire
}

func (h *GetConfigs) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(r, h)
	if err != nil {
		return err
	}

	c, err := h.ConfigsCurrent(a)
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, c.Vars)
}

type PatchConfigs struct {
	Empire
}

func (h *PatchConfigs) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var configVars empire.Vars

	if err := Decode(r, &configVars); err != nil {
		return err
	}

	a, err := findApp(r, h)
	if err != nil {
		return err
	}

	// Update the config
	c, err := h.ConfigsApply(a, configVars)
	if err != nil {
		return err
	}

	// Find current release
	release, err := h.ReleasesLast(a)
	if err != nil {
		return err
	}

	// If there is an existing release, create a new one
	if release != nil {
		slug, err := h.SlugsFind(release.SlugID)
		if err != nil {
			return err
		}

		keys := make([]string, 0, len(configVars))
		for k, _ := range configVars {
			keys = append(keys, string(k))
		}

		desc := fmt.Sprintf("Set %s config vars", strings.Join(keys, ","))

		// Create new release based on new config and old slug
		_, err = h.ReleasesCreate(a, c, slug, desc)
		if err != nil {
			return err
		}
	}

	w.WriteHeader(200)
	return Encode(w, c.Vars)
}

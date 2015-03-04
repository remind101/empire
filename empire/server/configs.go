package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/remind101/empire/empire"
)

type GetConfigs struct {
	Empire interface {
		empire.AppsFinder
		empire.ConfigsFinder
	}
}

func (h *GetConfigs) Serve(req *Request) (int, interface{}, error) {
	name := empire.AppName(req.Vars["app"])

	a, err := h.Empire.AppsFind(name)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if a == nil {
		return http.StatusNotFound, nil, nil
	}

	c, err := h.Empire.ConfigsCurrent(a)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 200, c.Vars, nil
}

type PatchConfigs struct {
	Empire interface {
		empire.AppsFinder
		empire.ConfigsApplier
		empire.SlugsFinder
		empire.ReleasesService
	}
}

func (h *PatchConfigs) Serve(req *Request) (int, interface{}, error) {
	var configVars empire.Vars

	if err := req.Decode(&configVars); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	name := empire.AppName(req.Vars["app"])

	// Find app
	a, err := h.Empire.AppsFind(name)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if a == nil {
		return http.StatusNotFound, nil, nil
	}

	// Update the config
	c, err := h.Empire.ConfigsApply(a, configVars)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	// Find current release
	r, err := h.Empire.ReleasesLast(a)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	// If there is an existing release, create a new one
	if r != nil {
		slug, err := h.Empire.SlugsFind(r.SlugID)
		if err != nil {
			return http.StatusInternalServerError, nil, err
		}

		keys := make([]string, 0, len(configVars))
		for k, _ := range configVars {
			keys = append(keys, string(k))
		}

		desc := fmt.Sprintf("Set %s config vars", strings.Join(keys, ","))

		// Create new release based on new config and old slug
		_, err = h.Empire.ReleasesCreate(a, c, slug, desc)
		if err != nil {
			return http.StatusInternalServerError, nil, err
		}
	}

	return 200, c.Vars, nil
}

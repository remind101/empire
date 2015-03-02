package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/remind101/empire/empire"
)

type GetConfigs struct {
	AppsService    empire.AppsService
	ConfigsService empire.ConfigsService
}

func (h *GetConfigs) Serve(req *Request) (int, interface{}, error) {
	name := empire.AppName(req.Vars["app"])

	a, err := h.AppsService.FindByName(name)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if a == nil {
		return http.StatusNotFound, nil, nil
	}

	c, err := h.ConfigsService.Head(a)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 200, c.Vars, nil
}

type PatchConfigs struct {
	AppsService     empire.AppsService
	ReleasesService empire.ReleasesService
	ConfigsService  empire.ConfigsService
	SlugsService    empire.SlugsService
}

func (h *PatchConfigs) Serve(req *Request) (int, interface{}, error) {
	var configVars empire.Vars

	if err := req.Decode(&configVars); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	name := empire.AppName(req.Vars["app"])

	// Find app
	a, err := h.AppsService.FindByName(name)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if a == nil {
		return http.StatusNotFound, nil, nil
	}

	// Update the config
	c, err := h.ConfigsService.Apply(a, configVars)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	// Find current release
	r, err := h.ReleasesService.Head(a)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	// If there is an existing release, create a new one
	if r != nil {
		slug, err := h.SlugsService.Find(r.SlugID)
		if err != nil {
			return http.StatusInternalServerError, nil, err
		}

		keys := make([]string, 0, len(configVars))
		for k, _ := range configVars {
			keys = append(keys, string(k))
		}

		desc := fmt.Sprintf("Set %s config vars", strings.Join(keys, ","))

		// Create new release based on new config and old slug
		_, err = h.ReleasesService.Create(a, c, slug, desc)
		if err != nil {
			return http.StatusInternalServerError, nil, err
		}
	}

	return 200, c.Vars, nil
}

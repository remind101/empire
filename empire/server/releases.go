package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/remind101/empire/empire"
)

type GetReleases struct {
	Empire
}

func (h *GetReleases) Serve(req *Request) (int, interface{}, error) {
	name := empire.AppName(req.Vars["app"])

	a, err := h.AppsFind(name)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if a == nil {
		return http.StatusNotFound, nil, nil
	}

	rels, err := h.ReleasesFindByApp(a)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 200, rels, nil
}

type PostReleases struct {
	Empire
}

type PostReleasesForm struct {
	Version string `json:"release"`
}

func (p *PostReleasesForm) ReleaseVersion() (empire.ReleaseVersion, error) {
	var r empire.ReleaseVersion
	i, err := strconv.Atoi(p.Version)
	if err != nil {
		return r, err
	}

	return empire.ReleaseVersion(i), nil
}

func (h *PostReleases) Serve(req *Request) (int, interface{}, error) {
	var form PostReleasesForm

	if err := req.Decode(&form); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	version, err := form.ReleaseVersion()
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	name := empire.AppName(req.Vars["app"])

	// Find app
	app, err := h.AppsFind(name)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if app == nil {
		return http.StatusNotFound, nil, nil
	}

	// Find previous release
	rel, err := h.ReleasesFindByAppAndVersion(app, version)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if rel == nil {
		return http.StatusNotFound, nil, nil
	}

	// Find config
	config, err := h.ConfigsFind(rel.ConfigID)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if config == nil {
		return http.StatusNotFound, nil, nil
	}

	// Find slug
	slug, err := h.SlugsFind(rel.SlugID)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if slug == nil {
		return http.StatusNotFound, nil, nil
	}

	// Create new release
	desc := fmt.Sprintf("Rollback to v%d", version)
	release, err := h.ReleasesCreate(app, config, slug, desc)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 200, release, nil
}

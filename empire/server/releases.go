package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/remind101/empire/empire"
)

type GetReleases struct {
	Empire
}

func (h *GetReleases) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	name := empire.AppName(vars["app"])

	a, err := h.AppsFind(name)
	if err != nil {
		return err
	}

	if a == nil {
		return ErrNotFound
	}

	rels, err := h.ReleasesFindByApp(a)
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, rels)
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

func (h *PostReleases) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	var form PostReleasesForm

	if err := Decode(r, &form); err != nil {
		return err
	}

	version, err := form.ReleaseVersion()
	if err != nil {
		return err
	}

	vars := mux.Vars(r)
	name := empire.AppName(vars["app"])

	// Find app
	app, err := h.AppsFind(name)
	if err != nil {
		return err
	}

	if app == nil {
		return ErrNotFound
	}

	// Find previous release
	rel, err := h.ReleasesFindByAppAndVersion(app, version)
	if err != nil {
		return err
	}

	if rel == nil {
		return ErrNotFound
	}

	// Find config
	config, err := h.ConfigsFind(rel.ConfigID)
	if err != nil {
		return err
	}

	if config == nil {
		return ErrNotFound
	}

	// Find slug
	slug, err := h.SlugsFind(rel.SlugID)
	if err != nil {
		return err
	}

	if slug == nil {
		return ErrNotFound
	}

	// Create new release
	desc := fmt.Sprintf("Rollback to v%d", version)
	release, err := h.ReleasesCreate(app, config, slug, desc)
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, release)
}

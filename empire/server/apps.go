package server

import (
	"net/http"

	"github.com/remind101/empire"
)

type GetApps struct {
	AppsService empire.AppsService
}

func (h *GetApps) Serve(req *Request) (int, interface{}, error) {
	apps, err := h.AppsService.FindAll()
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 200, apps, nil
}

type DeleteApp struct {
	AppsService empire.AppsService
}

func (h *DeleteApp) Serve(req *Request) (int, interface{}, error) {
	name := empire.AppName(req.Vars["app"])

	a, err := h.AppsService.FindByName(name)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if a == nil {
		return http.StatusNotFound, nil, nil
	}

	if err := h.AppsService.Destroy(a); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 200, nil, nil
}

type PostAppsForm struct {
	Name string `json:"name"`
	Repo string `json:"repo"`
}

type PostApps struct {
	AppsService empire.AppsService
}

func (h *PostApps) Serve(req *Request) (int, interface{}, error) {
	var form PostAppsForm

	if err := req.Decode(&form); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	app, err := empire.NewApp(empire.AppName(form.Name), empire.Repo(form.Repo))
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	a, err := h.AppsService.Create(app)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 201, a, nil
}

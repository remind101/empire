package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/empire"
)

type dockerConfig struct {
	Auth  string `json:"auth"`
	Email string `json:"email"`
}

// authConfigs converts a dockerConfigs map to a docker.AuthConfigurations object.
func authConfigs(confs map[string]dockerConfig) (*docker.AuthConfigurations, error) {
	c := &docker.AuthConfigurations{
		Configs: make(map[string]docker.AuthConfiguration),
	}

	for reg, conf := range confs {
		data, err := base64.StdEncoding.DecodeString(conf.Auth)
		if err != nil {
			return nil, err
		}

		userpass := strings.Split(string(data), ":")

		c.Configs[reg] = docker.AuthConfiguration{
			Email:         conf.Email,
			Username:      userpass[0],
			Password:      userpass[1],
			ServerAddress: reg,
		}
	}

	return c, nil
}

// PostDeploys is a Handler for the POST /v1/deploys endpoint.
type PostDeploys struct {
	Empire
}

// PostDeployForm is the form object that represents the POST body.
type PostDeployForm struct {
	Image struct {
		ID   string `json:"id"`
		Repo string `json:"repo"`
	} `json:"image"`
}

// Serve implements the Handler interface.
func (h *PostDeploys) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	var form PostDeployForm
	var auth *docker.AuthConfigurations

	regConfigHeader := req.Header.Get("X-Docker-Config")
	if regConfigHeader != "" {
		data, err := base64.URLEncoding.DecodeString(regConfigHeader)
		if err != nil {
			return http.StatusInternalServerError, nil, err
		}

		var confs map[string]dockerConfig
		if err := json.Unmarshal(data, &confs); err != nil {
			return http.StatusInternalServerError, nil, err
		}

		auth, err = authConfigs(confs)
		if err != nil {
			return http.StatusInternalServerError, nil, err
		}
	}

	if err := Decode(r, &form); err != nil {
		return err
	}

	d, err := h.Deploy(empire.Image{
		Repo: empire.Repo(form.Image.Repo),
		ID:   form.Image.ID,
	}, auth)
	if err != nil {
		return err
	}

	w.WriteHeader(201)
	return Encode(w, d)
}

package registry

import (
	"errors"
	"strings"
)

// ErrInvalidRepo is returned by Split when the repo is not a valid repo.
var ErrInvalidRepo = errors.New("registry: not a valid docker repo")

// Split splits a full docker repo into registry and path segments.
func Split(fullRepo string) (registry string, path string, err error) {
	parts := strings.Split(fullRepo, "/")

	if len(parts) < 2 {
		return "", "", ErrInvalidRepo
	}

	if len(parts) == 2 {
		return "", strings.Join(parts, "/"), nil
	}

	return parts[0], strings.Join(parts[1:], "/"), nil
}

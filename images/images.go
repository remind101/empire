package images

import "github.com/remind101/empire/repos"

// Image represents a container image, which is tied to a repository.
type Image struct {
	Repo repos.Repo `json:"repo"`
	ID   string     `json:"id"`
}

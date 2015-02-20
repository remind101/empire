package deploys

import "github.com/remind101/empire/releases"

// ID represents the unique identifier for a Deploy.
type ID string

// Deploy represents a deployment to the platform.
type Deploy struct {
	ID      ID                `json:"id"`
	Status  string            `json:"status"`
	Release *releases.Release `json:"release"`
}

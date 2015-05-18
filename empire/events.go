package empire

import "fmt"

// Event defines an interface for events within empire.
type Event interface {
	// The type of event, e.g. "deploy", "error", "docker"
	Event() string
}

type dockerProgress struct {
	Current int   `json:"current,omitempty"`
	Total   int   `json:"total,omitempty"`
	Start   int64 `json:"start,omitempty"`
}

type dockerError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// DockerEvent represents an event received from the docker remote API.
type DockerEvent struct {
	Stream       string          `json:"stream,omitempty"`
	Status       string          `json:"status,omitempty"`
	Progress     *dockerProgress `json:"progressDetail,omitempty"`
	ID           string          `json:"id,omitempty"`
	From         string          `json:"from,omitempty"`
	Time         int64           `json:"time,omitempty"`
	Error        *dockerError    `json:"errorDetail,omitempty"`
	ErrorMessage string          `json:"error,omitempty"`
}

func (e *DockerEvent) Event() string {
	return "docker"
}

// FakeDockerPull returns a slice of events that look like a docker pull.
func FakeDockerPull(image Image) []DockerEvent {
	return []DockerEvent{
		{Status: fmt.Sprintf("Pulling repository %s", image.Repo)},
		{Status: fmt.Sprintf("Pulling image (%s) from %s", image.ID, image.Repo), Progress: &dockerProgress{}, ID: "345c7524bc96"},
		{Status: fmt.Sprintf("Pulling image (%s) from %s, endpoint: https://registry-1.docker.io/v1/", image.ID, image.Repo), Progress: &dockerProgress{}, ID: "345c7524bc96"},
		{Status: "Pulling dependent layers", Progress: &dockerProgress{}, ID: "345c7524bc96"},
		{Status: "Download complete", Progress: &dockerProgress{}, ID: "a1dd7097a8e8"},
		{Status: fmt.Sprintf("Status: Image is up to date for %s", image)},
	}
}

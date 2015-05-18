package empire

import (
	"fmt"

	"github.com/docker/docker/pkg/jsonmessage"
)

// Event defines an interface for events within empire.
type Event interface {
	// The type of event, e.g. "deploy", "error", "docker"
	Event() string
}

type DockerEvent jsonmessage.JSONMessage

func (e *DockerEvent) Event() string {
	return "docker"
}

// FakeDockerPull returns a slice of events that look like a docker pull.
func FakeDockerPull(image Image) []DockerEvent {
	return []DockerEvent{
		{Status: fmt.Sprintf("Pulling repository %s", image.Repo)},
		{Status: fmt.Sprintf("Pulling image (%s) from %s", image.ID, image.Repo), Progress: &jsonmessage.JSONProgress{}, ID: "345c7524bc96"},
		{Status: fmt.Sprintf("Pulling image (%s) from %s, endpoint: https://registry-1.docker.io/v1/", image.ID, image.Repo), Progress: &jsonmessage.JSONProgress{}, ID: "345c7524bc96"},
		{Status: "Pulling dependent layers", Progress: &jsonmessage.JSONProgress{}, ID: "345c7524bc96"},
		{Status: "Download complete", Progress: &jsonmessage.JSONProgress{}, ID: "a1dd7097a8e8"},
		{Status: fmt.Sprintf("Status: Image is up to date for %s", image)},
	}
}

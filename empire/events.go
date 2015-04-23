package empire

// Event defines an interface for events within empire.
type Event interface {
	// The type of event, e.g. "deploy", "error", "docker"
	Event() string
}

type DeploymentEvent struct {
	Deployment *Deployment
}

func (e *DeploymentEvent) Event() string {
	return "deploy"
}

type dockerProgress struct {
	Current int   `json:"current,omitempty"`
	Total   int   `json:"total,omitempty"`
	Start   int64 `json:"start,omitempty"`
}

// DockerEvent represents an event received from the docker remote API.
type DockerEvent struct {
	Stream   string          `json:"stream,omitempty"`
	Status   string          `json:"status,omitempty"`
	Progress *dockerProgress `json:"progressDetail,omitempty"`
	ID       string          `json:"id,omitempty"`
	From     string          `json:"from,omitempty"`
	Time     int64           `json:"time,omitempty"`
	Error    error           `json:"errorDetail,omitempty"`
}

func (e *DockerEvent) Event() string {
	return "docker"
}

package empire

import (
	"fmt"
	"log"
	"strings"
)

type multiError struct {
	Errors []error
}

func (e *multiError) Error() string {
	points := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		points[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Sprintf(
		"%d error(s) occurred:\n\n%s",
		len(e.Errors), strings.Join(points, "\n"))
}

type BaseEvent struct {
	user    *User
	message string
}

func NewBaseEvent(user *User, message string) BaseEvent {
	return BaseEvent{user: user, message: message}
}

func (e BaseEvent) User() *User {
	return e.user
}

func (e BaseEvent) Message() string {
	return e.message
}

// RunEvent is triggered when a user starts or stops a one off process.
type RunEvent struct {
	BaseEvent

	App      string
	Command  Command
	URL      string
	Attached bool
	Finished bool

	app *App
}

func (e RunEvent) Event() string {
	return "run"
}

func (e *RunEvent) Finish() {
	e.Finished = true
}

func (e RunEvent) String() string {
	attachment := "detached"
	if e.Attached {
		attachment = "attached"
	}

	action := "Started running"
	if e.Finished {
		action = "Ran"
	}
	msg := fmt.Sprintf("%s `%s` (%s) on %s", action, e.Command.String(), attachment, e.App)
	if e.URL != "" {
		msg = fmt.Sprintf("%s (<%s|logs>)", msg, e.URL)
	}
	return msg
}

func (e RunEvent) GetApp() *App {
	return e.app
}

// RestartEvent is triggered when a user restarts an application.
type RestartEvent struct {
	BaseEvent

	App string
	PID string

	app *App
}

func (e RestartEvent) Event() string {
	return "restart"
}

func (e RestartEvent) String() string {
	msg := ""
	if e.PID == "" {
		msg = fmt.Sprintf("Restarted %s", e.App)
	} else {
		msg = fmt.Sprintf("Restarted `%s` on %s", e.PID, e.App)
	}
	return msg
}

func (e RestartEvent) GetApp() *App {
	return e.app
}

type MaintenanceEvent struct {
	BaseEvent

	App         string
	Maintenance bool

	app *App
}

func (e MaintenanceEvent) Event() string {
	return "maintenance"
}

func (e MaintenanceEvent) String() string {
	state := "Disabled"
	if e.Maintenance {
		state = "Enabled"
	}
	msg := fmt.Sprintf("%s maintenance mode on %s", state, e.App)
	return msg
}

func (e MaintenanceEvent) GetApp() *App {
	return e.app
}

type ScaleEventUpdate struct {
	Process             string
	Quantity            int
	PreviousQuantity    int
	Constraints         Constraints
	PreviousConstraints Constraints
}

// ScaleEvent is triggered when a manual scaling event happens.
type ScaleEvent struct {
	BaseEvent

	App     string
	Updates []*ScaleEventUpdate

	app *App
}

func (e ScaleEvent) Event() string {
	return "scale"
}

func (e ScaleEvent) String() string {
	var updates []string
	for _, up := range e.Updates {
		// Deal with no new constraints by copying previous constraint settings.
		newConstraints := up.Constraints
		previousConstraints := up.PreviousConstraints
		if newConstraints.CPUShare == 0 {
			newConstraints.CPUShare = previousConstraints.CPUShare
		}

		if newConstraints.Memory == 0 {
			newConstraints.Memory = previousConstraints.Memory
		}

		updates = append(updates, fmt.Sprintf(
			"`%s` on %s from %d(%s) to %d(%s)",
			up.Process,
			e.App,
			up.PreviousQuantity,
			up.PreviousConstraints,
			up.Quantity,
			newConstraints,
		))
	}
	return fmt.Sprintf("Scaled %s", strings.Join(updates, ", and"))
}

func (e ScaleEvent) GetApp() *App {
	return e.app
}

// DeployEvent is triggered when a user deploys a new image to an app.
type DeployEvent struct {
	BaseEvent

	App   string
	Image string

	app *App
}

func (e DeployEvent) Event() string {
	return "deploy"
}

func (e DeployEvent) String() string {
	msg := ""
	if e.App == "" {
		msg = fmt.Sprintf("Deployed %s", e.Image)
	} else {
		msg = fmt.Sprintf("Deployed %s to %s", e.Image, e.App)
	}
	return msg
}

func (e DeployEvent) GetApp() *App {
	return e.app
}

// RollbackEvent is triggered when a user rolls back to an old version.
type RollbackEvent struct {
	BaseEvent

	App     string
	Version int

	app *App
}

func (e RollbackEvent) Event() string {
	return "rollback"
}

func (e RollbackEvent) String() string {
	msg := fmt.Sprintf("Rolled back %s to v%d", e.App, e.Version)
	return msg
}

func (e RollbackEvent) GetApp() *App {
	return e.app
}

// SetEvent is triggered when environment variables are changed on an
// application.
type SetEvent struct {
	BaseEvent

	App     string
	Changed []string

	app *App
}

func (e SetEvent) Event() string {
	return "set"
}

func (e SetEvent) String() string {
	return fmt.Sprintf("Changed environment variables on %s (%s)", e.App, strings.Join(e.Changed, ", "))
}

func (e SetEvent) GetApp() *App {
	return e.app
}

// CreateEvent is triggered when a user creates a new application.
type CreateEvent struct {
	BaseEvent

	Name string
}

func (e CreateEvent) Event() string {
	return "create"
}

func (e CreateEvent) String() string {
	return fmt.Sprintf("Created %s", e.Name)
}

// DestroyEvent is triggered when a user destroys an application.
type DestroyEvent struct {
	BaseEvent

	App string
}

func (e DestroyEvent) Event() string {
	return "destroy"
}

func (e DestroyEvent) String() string {
	return fmt.Sprintf("Destroyed %s", e.App)
}

// Event represents an event triggered within Empire.
type Event interface {
	// Returns the name of the event.
	Event() string

	// Returns a human readable string about the event.
	String() string

	// Returns the commit message that was supplied at the time of the
	// change.
	Message() string

	// Returns the user that made the change.
	User() *User
}

// AppEvent is an Event that relates to a specific App.
type AppEvent interface {
	Event
	GetApp() *App
}

// EventStream is an interface for publishing events that happen within
// Empire.
type EventStream interface {
	PublishEvent(Event) error
}

// EventStreamFunc is a function that implements the Events interface.
type EventStreamFunc func(Event) error

func (fn EventStreamFunc) PublishEvent(event Event) error {
	return fn(event)
}

// NullEventStream an events service that does nothing.
var NullEventStream = EventStreamFunc(func(event Event) error {
	return nil
})

// MultiEventStream is an EventStream implementation that publishes the event to multiple EventStreams, returning any errors after publishing to all streams.
type MultiEventStream []EventStream

func (streams MultiEventStream) PublishEvent(e Event) error {
	result := new(multiError)
	for _, s := range streams {
		if err := s.PublishEvent(e); err != nil {
			result.Errors = append(result.Errors, err)
		}
	}
	if len(result.Errors) == 0 {
		return nil
	}
	return result
}

// asyncEventStream wraps an array of EventStreams to publish events
// asynchronously in a goroutine
type asyncEventStream struct {
	e      EventStream
	events chan Event
}

// AsyncEvents returns a new AsyncEventStream that will buffer upto 100 events
// before applying backpressure.
func AsyncEvents(e EventStream) EventStream {
	s := &asyncEventStream{
		e:      e,
		events: make(chan Event, 100),
	}
	go s.start()
	return s
}

func (e *asyncEventStream) PublishEvent(event Event) error {
	e.events <- event
	return nil
}

func (e *asyncEventStream) start() {
	for event := range e.events {
		err := e.publishEvent(event)
		if err != nil {
			log.Printf("event stream error: %v\n", err)
		}
	}
}

func (e *asyncEventStream) publishEvent(event Event) (err error) {
	defer func() {
		if v := recover(); v != nil {
			var ok bool
			if err, ok = v.(error); ok {
				return
			}

			err = fmt.Errorf("panic: %v", v)
		}
	}()
	err = e.e.PublishEvent(event)
	return
}

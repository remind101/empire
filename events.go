package empire

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-multierror"
)

func appendCommitMessage(main, commit string) string {
	output := main
	if commit != "" {
		output = fmt.Sprintf("%s: '%s'", main, commit)
	}
	return output
}

// RunEvent is triggered when a user starts a one off process.
type RunEvent struct {
	User     string
	App      string
	Command  Command
	URL      string
	Attached bool
	Message  string

	app *App
}

func (e RunEvent) Event() string {
	return "run"
}

func (e RunEvent) String() string {
	attachment := "detached"
	if e.Attached {
		attachment = "attached"
	}
	msg := fmt.Sprintf("%s ran `%s` (%s) on %s", e.User, e.Command.String(), attachment, e.App)
	if e.URL != "" {
		msg = fmt.Sprintf("%s (<%s|logs>)", msg, e.URL)
	}
	return appendCommitMessage(msg, e.Message)
}

func (e RunEvent) GetApp() *App {
	return e.app
}

// RestartEvent is triggered when a user restarts an application.
type RestartEvent struct {
	User    string
	App     string
	PID     string
	PTYPE   string
	Message string

	app *App
}

func (e RestartEvent) Event() string {
	return "restart"
}

func (e RestartEvent) String() string {
	msg := ""
	if e.PID == "" {
		msg = fmt.Sprintf("%s restarted %s", e.User, e.App)
	} else {
		msg = fmt.Sprintf("%s restarted `%s` on %s", e.User, e.PID, e.App)
	}
	return appendCommitMessage(msg, e.Message)
}

func (e RestartEvent) GetApp() *App {
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
	User    string
	App     string
	Updates []*ScaleEventUpdate
	Message string

	app *App
}

func (e ScaleEvent) Event() string {
	return "scale"
}

func (e ScaleEvent) String() string {
	var msg, sep string
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

		if newConstraints.Nproc == 0 {
			newConstraints.Nproc = previousConstraints.Nproc
		}

		msg += fmt.Sprintf(
			"%s%s scaled `%s` on %s from %d(%s) to %d(%s)",
			sep,
			e.User,
			up.Process,
			e.App,
			up.PreviousQuantity,
			up.PreviousConstraints,
			up.Quantity,
			newConstraints,
		)
		sep = "\n"
	}
	return appendCommitMessage(msg, e.Message)
}

func (e ScaleEvent) GetApp() *App {
	return e.app
}

// DeployEvent is triggered when a user deploys a new image to an app.
type DeployEvent struct {
	User        string
	App         string
	Image       string
	Environment string
	Release     int
	Message     string

	app *App
}

func (e DeployEvent) Event() string {
	return "deploy"
}

func (e DeployEvent) String() string {
	msg := ""
	if e.App == "" {
		msg = fmt.Sprintf("%s deployed %s", e.User, e.Image)
	} else {
		msg = fmt.Sprintf("%s deployed %s to %s %s (v%d)", e.User, e.Image, e.App, e.Environment, e.Release)
	}
	return appendCommitMessage(msg, e.Message)
}

func (e DeployEvent) GetApp() *App {
	return e.app
}

// RollbackEvent is triggered when a user rolls back to an old version.
type RollbackEvent struct {
	User    string
	App     string
	Version int
	Message string

	app *App
}

func (e RollbackEvent) Event() string {
	return "rollback"
}

func (e RollbackEvent) String() string {
	msg := fmt.Sprintf("%s rolled back %s to v%d", e.User, e.App, e.Version)
	return appendCommitMessage(msg, e.Message)
}

func (e RollbackEvent) GetApp() *App {
	return e.app
}

// SetEvent is triggered when environment variables are changed on an
// application.
type SetEvent struct {
	User    string
	App     string
	Changed []string
	Message string

	app *App
}

func (e SetEvent) Event() string {
	return "set"
}

func (e SetEvent) String() string {
	msg := fmt.Sprintf("%s changed environment variables on %s (%s)", e.User, e.App, strings.Join(e.Changed, ", "))
	return appendCommitMessage(msg, e.Message)
}

func (e SetEvent) GetApp() *App {
	return e.app
}

// CreateEvent is triggered when a user creates a new application.
type CreateEvent struct {
	User    string
	Name    string
	Message string
}

func (e CreateEvent) Event() string {
	return "create"
}

func (e CreateEvent) String() string {
	msg := fmt.Sprintf("%s created %s", e.User, e.Name)
	return appendCommitMessage(msg, e.Message)
}

// DestroyEvent is triggered when a user destroys an application.
type DestroyEvent struct {
	User    string
	App     string
	Message string
}

func (e DestroyEvent) Event() string {
	return "destroy"
}

func (e DestroyEvent) String() string {
	msg := fmt.Sprintf("%s destroyed %s", e.User, e.App)
	return appendCommitMessage(msg, e.Message)
}

// Event represents an event triggered within Empire.
type Event interface {
	// Returns the name of the event.
	Event() string

	// Returns a human readable string about the event.
	String() string
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
	var result *multierror.Error
	for _, s := range streams {
		if err := s.PublishEvent(e); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result.ErrorOrNil()
}

// AsyncEventStream wraps an array of EventStreams to publish events
// asynchronously in a goroutine
type AsyncEventStream struct {
	EventStream
	events chan Event
}

// AsyncEvents returns a new AsyncEventStream that will buffer upto 100 events
// before applying backpressure.
func AsyncEvents(e EventStream) *AsyncEventStream {
	s := &AsyncEventStream{
		EventStream: e,
		events:      make(chan Event, 100),
	}
	go s.start()
	return s
}

func (e *AsyncEventStream) PublishEvent(event Event) error {
	e.events <- event
	return nil
}

func (e *AsyncEventStream) start() {
	for event := range e.events {
		err := e.publishEvent(event)
		if err != nil {
			log.Printf("event stream error: %v\n", err)
		}
	}
}

func (e *AsyncEventStream) publishEvent(event Event) (err error) {
	defer func() {
		if v := recover(); v != nil {
			var ok bool
			if err, ok = v.(error); ok {
				return
			}

			err = fmt.Errorf("panic: %v", v)
		}
	}()
	err = e.EventStream.PublishEvent(event)
	return
}

// prettyDelta is a fmt.Stringer that formats a delta by prefixing it with a `+`
// or `-`.
type prettyDelta struct {
	delta int
}

// String implements the fmt.Stringer interface.
func (d prettyDelta) String() string {
	if d.delta > 0 {
		return fmt.Sprintf("+%d", d.delta)
	}

	if d.delta < 0 {
		return fmt.Sprintf("-%d", d.delta*-1)
	}

	return fmt.Sprintf("no change")
}

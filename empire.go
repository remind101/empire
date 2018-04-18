package empire // import "github.com/remind101/empire"

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/remind101/empire/pkg/image"
	"golang.org/x/net/context"
)

// Various errors that may be returned.
var (
	ErrUserName = errors.New("Name is required")
	// ErrInvalidName is used to indicate that the app name is not valid.
	ErrInvalidName = &ValidationError{
		errors.New("An app name must be alphanumeric and dashes only, 3-30 chars in length."),
	}
)

// NotFoundError is returned when an entity doesn't exist.
type NotFoundError struct {
	Err error
}

func (e *NotFoundError) Error() string {
	return e.Err.Error()
}

// AllowedCommands specifies what commands are allowed to be Run with Empire.
type AllowedCommands int

const (
	// AllowCommandAny will allow any command to be run.
	AllowCommandAny AllowedCommands = iota
	// AllowCommandProcfile will only allow commands specified in the
	// Procfile (the key itself) to be run. Any other command will return an
	// error.
	AllowCommandProcfile
)

// An error that is returned when a command is not whitelisted to be Run.
type CommandNotAllowedError struct {
	Command Command
}

// commandNotInFormation returns a new CommandNotAllowedError for a command
// that's not in the formation.
func commandNotInFormation(command Command, formation Formation) *CommandNotAllowedError {
	return &CommandNotAllowedError{Command: command}
}

// Error implements the error interface.
func (c *CommandNotAllowedError) Error() string {
	return fmt.Sprintf("command not allowed: %v\n", c.Command)
}

// NoCertError is returned when the Procfile specifies an https/ssl listener but
// there is no attached certificate.
type NoCertError struct {
	Process string
}

func (e *NoCertError) Error() string {
	return fmt.Sprintf("the %s process does not have a certificate attached", e.Process)
}

// Empire provides the core public API for Empire. Refer to the package
// documentation for details.
type Empire struct {
	storage Storage

	tasks  *tasksService
	runner *runnerService
	slugs  *slugsService

	// LogsStreamer is the backend used to stream application logs.
	LogsStreamer LogsStreamer

	// ImageRegistry is used to interract with container images.
	ImageRegistry ImageRegistry

	// Environment represents the environment this Empire server is responsible for
	Environment string

	// EventStream service for publishing Empire events.
	EventStream

	// RunRecorder is used to record the logs from interactive runs.
	RunRecorder RunRecorder

	// MessagesRequired is a boolean used to determine if messages should be required for events.
	MessagesRequired bool

	// Configures what type of commands are allowed to be run with the Run
	// method. The zero value allows all commands to be run.
	AllowedCommands AllowedCommands
}

// New returns a new Empire instance.
func New(storage Storage) *Empire {
	e := &Empire{
		storage:      storage,
		LogsStreamer: logsDisabled,
		EventStream:  NullEventStream,
	}

	e.slugs = &slugsService{Empire: e}
	e.tasks = &tasksService{Empire: e}
	e.runner = &runnerService{Empire: e}
	return e
}

// AppsFind finds the first app matching the query.
func (e *Empire) AppsFind(q AppsQuery) (*App, error) {
	return e.storage.AppsFind(q)
}

// Apps returns all Apps.
func (e *Empire) Apps(q AppsQuery) ([]*App, error) {
	return e.storage.Apps(q)
}

func (e *Empire) requireMessages(m string) error {
	if e.MessagesRequired && m == "" {
		return &MessageRequiredError{}
	}
	return nil
}

// CreateOpts are options that are provided when creating a new application.
type CreateOpts struct {
	// User performing the action.
	User *User

	// Name of the application.
	Name string

	// Commit message
	Message string
}

func (opts CreateOpts) Event() CreateEvent {
	return CreateEvent{
		User:    opts.User.Name,
		Name:    opts.Name,
		Message: opts.Message,
	}
}

func (opts CreateOpts) Validate(e *Empire) error {
	return e.requireMessages(opts.Message)
}

// Create creates a new app.
func (e *Empire) Create(ctx context.Context, opts CreateOpts) (*Release, error) {
	if err := opts.Validate(e); err != nil {
		return nil, err
	}

	app := &App{
		Name:        opts.Name,
		Environment: make(map[string]string),
	}

	desc := fmt.Sprintf("Creating new application")
	release, err := e.storage.ReleasesCreate(app, desc)
	if err != nil {
		return nil, err
	}

	return release, e.PublishEvent(opts.Event())
}

// DestroyOpts are options provided when destroying an application.
type DestroyOpts struct {
	// User performing the action.
	User *User

	// The associated app.
	App *App

	// Commit message
	Message string
}

func (opts DestroyOpts) Event() DestroyEvent {
	return DestroyEvent{
		User:    opts.User.Name,
		App:     opts.App.Name,
		Message: opts.Message,
	}
}

func (opts DestroyOpts) Validate(e *Empire) error {
	return e.requireMessages(opts.Message)
}

// Destroy destroys an app.
func (e *Empire) Destroy(ctx context.Context, opts DestroyOpts) error {
	if err := opts.Validate(e); err != nil {
		return err
	}

	if err := e.storage.AppsDestroy(opts.App); err != nil {
		return err
	}

	return e.PublishEvent(opts.Event())
}

// Config returns the current Config for a given app.
func (e *Empire) Config(app *App) (map[string]string, error) {
	return app.Environment, nil
}

type SetMaintenanceModeOpts struct {
	// User performing the action.
	User *User

	// The associated app.
	App *App

	// Wheather maintenance mode should be enabled or not.
	Maintenance bool

	// Commit message
	Message string
}

func (opts SetMaintenanceModeOpts) Event() MaintenanceEvent {
	return MaintenanceEvent{
		User:        opts.User.Name,
		App:         opts.App.Name,
		Maintenance: opts.Maintenance,
		Message:     opts.Message,
	}
}

func (opts SetMaintenanceModeOpts) Validate(e *Empire) error {
	return e.requireMessages(opts.Message)
}

// SetMaintenanceMode enables or disables "maintenance mode" on the app. When an
// app is in maintenance mode, all processes will be scaled down to 0. When
// taken out of maintenance mode, all processes will be scaled up back to their
// existing values.
func (e *Empire) SetMaintenanceMode(ctx context.Context, opts SetMaintenanceModeOpts) error {
	if err := opts.Validate(e); err != nil {
		return err
	}

	app := opts.App
	app.Maintenance = opts.Maintenance

	if _, err := e.storage.ReleasesCreate(app, "Enabling maintenance mode"); err != nil {
		return err
	}

	return e.PublishEvent(opts.Event())
}

// SetOpts are options provided when setting new config vars on an app.
type SetOpts struct {
	// User performing the action.
	User *User

	// The associated app.
	App *App

	// The new vars to merge into the old config.
	Vars Vars

	// Commit message
	Message string
}

func (opts SetOpts) Event() SetEvent {
	var changed []string
	for k := range opts.Vars {
		changed = append(changed, string(k))
	}

	return SetEvent{
		User:    opts.User.Name,
		App:     opts.App.Name,
		Changed: changed,
		Message: opts.Message,
		app:     opts.App,
	}
}

func (opts SetOpts) Validate(e *Empire) error {
	return e.requireMessages(opts.Message)
}

// Set applies the new config vars to the apps current Config, returning the new
// Config. If the app has a running release, a new release will be created and
// run.
func (e *Empire) Set(ctx context.Context, opts SetOpts) (map[string]string, error) {
	if err := opts.Validate(e); err != nil {
		return nil, err
	}

	app, vars := opts.App, opts.Vars

	app.Environment = newConfig(app.Environment, vars)

	if _, err := e.storage.ReleasesCreate(app, "Setting environment variables"); err != nil {
		return nil, err
	}

	return app.Environment, e.PublishEvent(opts.Event())
}

// Tasks returns the Tasks for the given app.
func (e *Empire) Tasks(ctx context.Context, app *App) ([]*Task, error) {
	return e.tasks.Tasks(ctx, app)
}

// RestartOpts are options provided when restarting an app.
type RestartOpts struct {
	// User performing the action.
	User *User

	// The associated app.
	App *App

	// If provided, a PID that will be killed. Generally used for killing
	// detached processes.
	PID string

	// Commit message
	Message string
}

func (opts RestartOpts) Event() RestartEvent {
	return RestartEvent{
		User:    opts.User.Name,
		App:     opts.App.Name,
		PID:     opts.PID,
		Message: opts.Message,
		app:     opts.App,
	}
}

func (opts RestartOpts) Validate(e *Empire) error {
	return e.requireMessages(opts.Message)
}

// Restart restarts processes matching the given prefix for the given Release.
// If the prefix is empty, it will match all processes for the release.
func (e *Empire) Restart(ctx context.Context, opts RestartOpts) error {
	return fmt.Errorf("`emp restart` is currently unsupported")
}

// RunOpts are options provided when running an attached/detached process.
type RunOpts struct {
	// User performing this action.
	User *User

	// Related app to run.
	App *App

	// The command to run.
	Command Command

	// Commit message
	Message string

	// Input/Output streams. The caller is responsible for closing these
	// streams.
	Stdin          io.Reader
	Stdout, Stderr io.Writer

	// Extra environment variables to set.
	Env map[string]string

	// Optional memory/cpu/nproc constraints.
	Constraints *Constraints
}

func (opts RunOpts) Event() RunEvent {
	var attached bool
	if opts.Stdout != nil || opts.Stderr != nil {
		attached = true
	}

	return RunEvent{
		User:     opts.User.Name,
		App:      opts.App.Name,
		Command:  opts.Command,
		Attached: attached,
		Message:  opts.Message,
		app:      opts.App,
	}
}

func (opts RunOpts) Validate(e *Empire) error {
	return e.requireMessages(opts.Message)
}

// Run runs a one-off process for a given App and command.
func (e *Empire) Run(ctx context.Context, opts RunOpts) error {
	event := opts.Event()

	if err := opts.Validate(e); err != nil {
		return err
	}

	if e.RunRecorder != nil && (opts.Stdout != nil || opts.Stderr != nil) {
		w, err := e.RunRecorder()
		if err != nil {
			return err
		}

		// Add the log url to the event, if there is one.
		if w, ok := w.(interface {
			URL() string
		}); ok {
			event.URL = w.URL()
		}

		msg := fmt.Sprintf("Running `%s` on %s as %s", opts.Command, opts.App.Name, opts.User.Name)
		msg = appendCommitMessage(msg, opts.Message)
		io.WriteString(w, fmt.Sprintf("%s\n", msg))

		// Write output to both the original output as well as the
		// record.
		if opts.Stdout != nil {
			opts.Stdout = io.MultiWriter(w, opts.Stdout)
		}
		if opts.Stderr != nil {
			opts.Stderr = io.MultiWriter(w, opts.Stderr)
		}
	}

	if err := e.PublishEvent(event); err != nil {
		return err
	}

	if err := e.runner.Run(ctx, opts); err != nil {
		return err
	}

	event.Finish()
	return e.PublishEvent(event)
}

// Releases returns all Releases for a given App.
func (e *Empire) Releases(q ReleasesQuery) ([]*Release, error) {
	return e.storage.Releases(q)
}

// ReleasesFind returns the first releases for a given App.
func (e *Empire) ReleasesFind(q ReleasesQuery) (*Release, error) {
	return e.storage.ReleasesFind(q)
}

// RollbackOpts are options provided when rolling back to an old release.
type RollbackOpts struct {
	// The user performing the action.
	User *User

	// The associated app.
	App *App

	// The release version to rollback to.
	Version int

	// Commit message
	Message string
}

func (opts RollbackOpts) Event() RollbackEvent {
	return RollbackEvent{
		User:    opts.User.Name,
		App:     opts.App.Name,
		Version: opts.Version,
		Message: opts.Message,
		app:     opts.App,
	}
}

func (opts RollbackOpts) Validate(e *Empire) error {
	return e.requireMessages(opts.Message)
}

// Rollback rolls an app back to a specific release version. Returns a
// new release.
func (e *Empire) Rollback(ctx context.Context, opts RollbackOpts) (*Release, error) {
	if err := opts.Validate(e); err != nil {
		return nil, err
	}

	// TODO
	return nil, errors.New("not implemented")

	//return r, e.PublishEvent(opts.Event())
}

// DeployOpts represents options that can be passed when deploying to
// an application.
type DeployOpts struct {
	// User the user that is triggering the deployment.
	User *User

	// App is the app that is being deployed to.
	App *App

	// Image is the image that's being deployed.
	Image image.Image

	// Environment is the environment where the image is being deployed
	Environment string

	// Output is a DeploymentStream where deployment output and events will
	// be streamed in jsonmessage format.
	Output *DeploymentStream

	// Commit message
	Message string

	// Stream boolean for whether or not a status stream should be created.
	Stream bool
}

func (opts DeployOpts) Event() DeployEvent {
	e := DeployEvent{
		User:    opts.User.Name,
		Image:   opts.Image.String(),
		Message: opts.Message,
	}
	if opts.App != nil {
		e.App = opts.App.Name
		e.app = opts.App
	}

	return e
}

func (opts DeployOpts) Validate(e *Empire) error {
	return e.requireMessages(opts.Message)
}

// Deploy deploys an image and streams the output to w.
func (e *Empire) Deploy(ctx context.Context, opts DeployOpts) (*Release, error) {
	w := opts.Output

	r, err := e.deploy(ctx, opts)
	if err != nil {
		return nil, w.Error(err)
	}

	if err := w.Status(fmt.Sprintf("Created new release v%d for %s", r.App.Version, r.App.Name)); err != nil {
		return r, err
	}

	return r, w.Status(fmt.Sprintf("Finished processing events for release v%d of %s", r.App.Version, r.App.Name))
}

func (e *Empire) deploy(ctx context.Context, opts DeployOpts) (*Release, error) {
	if err := opts.Validate(e); err != nil {
		return nil, err
	}

	app, img := opts.App, opts.Image

	if app == nil {
		var err error
		name := appNameFromRepo(img.Repository)
		app, err = e.AppsFind(AppsQuery{Name: &name})
		if err != nil {
			if err, ok := err.(*NotFoundError); !ok {
				return nil, err
			}
			release, err := e.Create(ctx, CreateOpts{
				User:    opts.User,
				Message: fmt.Sprintf("Creating new app from deployment of %s", img),
				Name:    name,
			})
			if err != nil {
				return nil, err
			}
			app = release.App
		}
	}

	slug, err := e.slugs.Create(ctx, img, opts.Output)
	if err != nil {
		return nil, err
	}

	formation, err := slug.Formation()
	if err != nil {
		return nil, err
	}

	app.Image = &slug.Image
	app.Formation = formation.Merge(app.Formation)

	desc := fmt.Sprintf("Deploy %s", img.String())
	desc = appendMessageToDescription(desc, opts.User, opts.Message)

	r, err := e.storage.ReleasesCreate(app, desc)
	if err != nil {
		return nil, err
	}

	event := opts.Event()
	event.Release = r.App.Version
	event.Environment = e.Environment
	// Deals with new app creation on first deploy
	if event.App == "" && r.App != nil {
		event.App = r.App.Name
		event.app = r.App
	}

	return r, e.PublishEvent(event)
}

type ProcessUpdate struct {
	// The process to scale.
	Process string

	// The desired quantity of processes.
	Quantity int

	// If provided, new memory and CPU constraints for the process.
	Constraints *Constraints
}

// ScaleOpts are options provided when scaling a process.
type ScaleOpts struct {
	// User that's performing the action.
	User *User

	// The associated app.
	App *App

	Updates []*ProcessUpdate

	// Commit message
	Message string
}

func (opts ScaleOpts) Event() ScaleEvent {
	e := ScaleEvent{
		User:    opts.User.Name,
		App:     opts.App.Name,
		Message: opts.Message,
		app:     opts.App,
	}

	var updates []*ScaleEventUpdate
	for _, up := range opts.Updates {
		event := &ScaleEventUpdate{
			Process:  up.Process,
			Quantity: up.Quantity,
		}
		if up.Constraints != nil {
			event.Constraints = *up.Constraints
		}
		updates = append(updates, event)
	}
	e.Updates = updates
	return e
}

func (opts ScaleOpts) Validate(e *Empire) error {
	return e.requireMessages(opts.Message)
}

// Scale scales an apps processes.
func (e *Empire) Scale(ctx context.Context, opts ScaleOpts) ([]*Process, error) {
	if err := opts.Validate(e); err != nil {
		return nil, err
	}

	app := opts.App
	event := opts.Event()

	var ps []*Process
	for i, up := range opts.Updates {
		t, q, c := up.Process, up.Quantity, up.Constraints

		p, ok := app.Formation[t]
		if !ok {
			return nil, &ValidationError{Err: fmt.Errorf("no %s process type in release", t)}
		}

		eventUpdate := event.Updates[i]
		eventUpdate.PreviousQuantity = p.Quantity
		eventUpdate.PreviousConstraints = p.Constraints()

		// Update quantity for this process in the formation
		p.Quantity = q
		if c != nil {
			p.SetConstraints(*c)
		}

		app.Formation[t] = p
		ps = append(ps, &p)
	}

	// TODO
	desc := fmt.Sprintf("Scaled processes")
	_, err := e.storage.ReleasesCreate(app, desc)
	if err != nil {
		return nil, err
	}

	return ps, nil
}

// ListScale lists the current scale settings for a given App
func (e *Empire) ListScale(ctx context.Context, app *App) (Formation, error) {
	return app.Formation, nil
}

// Streamlogs streams logs from an app.
func (e *Empire) StreamLogs(app *App, w io.Writer, duration time.Duration) error {
	if err := e.LogsStreamer.StreamLogs(app, w, duration); err != nil {
		return fmt.Errorf("error streaming logs: %v", err)
	}

	return nil
}

// Reset resets empire.
func (e *Empire) Reset() error {
	return e.storage.Reset()
}

// IsHealthy returns true if Empire is healthy, which means it can connect to
// the services it depends on.
func (e *Empire) IsHealthy() error {
	return e.storage.IsHealthy()
}

// ValidationError is returned when a model is not valid.
type ValidationError struct {
	Err error
}

func (e *ValidationError) Error() string {
	return e.Err.Error()
}

// MessageRequiredError is an error implementation, which is returned by Empire
// when a commit message is required for the operation.
type MessageRequiredError struct{}

func (e *MessageRequiredError) Error() string {
	return "Missing required option: 'Message'"
}

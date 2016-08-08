package empire // import "github.com/remind101/empire"

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/scheduler"
	"golang.org/x/net/context"
)

const (
	// webProcessType is the process type we assume are web server processes.
	webProcessType = "web"
)

// Various errors that may be returned.
var (
	ErrDomainInUse        = errors.New("Domain currently in use by another app.")
	ErrDomainAlreadyAdded = errors.New("Domain already added to this app.")
	ErrDomainNotFound     = errors.New("Domain could not be found.")
	ErrUserName           = errors.New("Name is required")
	ErrNoReleases         = errors.New("no releases")
	// ErrInvalidName is used to indicate that the app name is not valid.
	ErrInvalidName = &ValidationError{
		errors.New("An app name must be alphanumeric and dashes only, 3-30 chars in length."),
	}
)

// An error that is returned when RequireWhitelistedRuns is enabled, and
// the command is not whitelisted in the Procfile.
type CommandWhitelistError struct {
	Command Command
}

// Error implements the error interface.
func (c *CommandWhitelistError) Error() string {
	return fmt.Sprintf("command not whitelisted: %v", c.Command)
}

// Empire provides the core public API for Empire. Refer to the package
// documentation for details.
type Empire struct {
	DB *DB
	db *gorm.DB

	accessTokens *accessTokensService
	apps         *appsService
	configs      *configsService
	domains      *domainsService
	tasks        *tasksService
	releases     *releasesService
	deployer     *deployerService
	runner       *runnerService
	slugs        *slugsService
	certs        *certsService

	// Secret is used to sign JWT access tokens.
	Secret []byte

	// Scheduler is the backend scheduler used to run applications.
	Scheduler scheduler.Scheduler

	// LogsStreamer is the backend used to stream application logs.
	LogsStreamer LogsStreamer

	// ProcfileExtractor is called during deployments to extract the
	// Formation from the Procfile in the newly deployed image.
	ProcfileExtractor ProcfileExtractor

	// Environment represents the environment this Empire server is responsible for
	Environment string

	// EventStream service for publishing Empire events.
	EventStream

	// RunRecorder is used to record the logs from interactive runs.
	RunRecorder RunRecorder

	// MessagesRequired is a boolean used to determine if messages should be required for events.
	MessagesRequired bool

	// If enabled, only commands that are marked as `run: true` in the
	// Procfile will be allowed with `emp run`.
	RequireWhitelistedRuns bool
}

// New returns a new Empire instance.
func New(db *DB) *Empire {
	e := &Empire{
		LogsStreamer: logsDisabled,
		EventStream:  NullEventStream,

		DB: db,
		db: db.DB,
	}

	e.accessTokens = &accessTokensService{Empire: e}
	e.apps = &appsService{Empire: e}
	e.configs = &configsService{Empire: e}
	e.deployer = &deployerService{Empire: e}
	e.domains = &domainsService{Empire: e}
	e.slugs = &slugsService{Empire: e}
	e.tasks = &tasksService{Empire: e}
	e.runner = &runnerService{Empire: e}
	e.releases = &releasesService{Empire: e}
	e.certs = &certsService{Empire: e}
	return e
}

// AccessTokensFind finds an access token.
func (e *Empire) AccessTokensFind(token string) (*AccessToken, error) {
	return e.accessTokens.AccessTokensFind(token)
}

// AccessTokensCreate creates a new AccessToken.
func (e *Empire) AccessTokensCreate(accessToken *AccessToken) (*AccessToken, error) {
	return e.accessTokens.AccessTokensCreate(accessToken)
}

// AppsFind finds the first app matching the query.
func (e *Empire) AppsFind(q AppsQuery) (*App, error) {
	return appsFind(e.db, q)
}

// Apps returns all Apps.
func (e *Empire) Apps(q AppsQuery) ([]*App, error) {
	return apps(e.db, q)
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
func (e *Empire) Create(ctx context.Context, opts CreateOpts) (*App, error) {
	if err := opts.Validate(e); err != nil {
		return nil, err
	}

	a, err := appsCreate(e.db, &App{Name: opts.Name})
	if err != nil {
		return a, err
	}

	return a, e.PublishEvent(opts.Event())
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

	tx := e.db.Begin()

	if err := e.apps.Destroy(ctx, tx, opts.App); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return e.PublishEvent(opts.Event())
}

// Config returns the current Config for a given app.
func (e *Empire) Config(app *App) (*Config, error) {
	tx := e.db.Begin()

	c, err := e.configs.Config(tx, app)
	if err != nil {
		tx.Rollback()
		return c, err
	}

	if err := tx.Commit().Error; err != nil {
		return c, err
	}

	return c, nil
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
func (e *Empire) Set(ctx context.Context, opts SetOpts) (*Config, error) {
	if err := opts.Validate(e); err != nil {
		return nil, err
	}

	tx := e.db.Begin()

	c, err := e.configs.Set(ctx, tx, opts)
	if err != nil {
		tx.Rollback()
		return c, err
	}

	if err := tx.Commit().Error; err != nil {
		return c, err
	}

	return c, e.PublishEvent(opts.Event())
}

// DomainsFind returns the first domain matching the query.
func (e *Empire) DomainsFind(q DomainsQuery) (*Domain, error) {
	return domainsFind(e.db, q)
}

// Domains returns all domains matching the query.
func (e *Empire) Domains(q DomainsQuery) ([]*Domain, error) {
	return domains(e.db, q)
}

// DomainsCreate adds a new Domain for an App.
func (e *Empire) DomainsCreate(ctx context.Context, domain *Domain) (*Domain, error) {
	tx := e.db.Begin()

	d, err := e.domains.DomainsCreate(ctx, tx, domain)
	if err != nil {
		tx.Rollback()
		return d, err
	}

	if err := tx.Commit().Error; err != nil {
		return d, err
	}

	return d, nil
}

// DomainsDestroy removes a Domain for an App.
func (e *Empire) DomainsDestroy(ctx context.Context, domain *Domain) error {
	tx := e.db.Begin()

	if err := e.domains.DomainsDestroy(ctx, tx, domain); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
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
	if err := opts.Validate(e); err != nil {
		return err
	}

	if err := e.apps.Restart(ctx, e.db, opts); err != nil {
		return err
	}

	return e.PublishEvent(opts.Event())

}

// RunOpts are options provided when running an attached/detached process.
type RunOpts struct {
	// User performing this action.
	User *User

	// Related app.
	App *App

	// The command to run.
	Command Command

	// Commit message
	Message string

	// If provided, input will be read from this.
	Input io.Reader

	// If provided, output will be written to this.
	Output io.Writer

	// Extra environment variables to set.
	Env map[string]string

	// Optional memory/cpu/nproc constraints.
	Constraints *Constraints
}

func (opts RunOpts) Event() RunEvent {
	var attached bool
	if opts.Output != nil {
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

	if opts.Input != nil && opts.Output != nil && e.RunRecorder != nil {
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
		opts.Output = io.MultiWriter(w, opts.Output)
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
	return releases(e.db, q)
}

// ReleasesFind returns the first releases for a given App.
func (e *Empire) ReleasesFind(q ReleasesQuery) (*Release, error) {
	return releasesFind(e.db, q)
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

	tx := e.db.Begin()

	r, err := e.releases.Rollback(ctx, tx, opts)
	if err != nil {
		tx.Rollback()
		return r, err
	}

	if err := tx.Commit().Error; err != nil {
		return r, err
	}

	return r, e.PublishEvent(opts.Event())
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
	if err := opts.Validate(e); err != nil {
		return nil, err
	}

	r, err := e.deployer.Deploy(ctx, opts)
	if err != nil {
		return r, err
	}

	event := opts.Event()
	event.Release = r.Version
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

	tx := e.db.Begin()

	ps, err := e.apps.Scale(ctx, tx, opts)
	if err != nil {
		tx.Rollback()
		return ps, err
	}

	return ps, tx.Commit().Error
}

// ListScale lists the current scale settings for a given App
func (e *Empire) ListScale(ctx context.Context, app *App) (Formation, error) {
	return currentFormation(e.db, app)
}

// Streamlogs streams logs from an app.
func (e *Empire) StreamLogs(app *App, w io.Writer, duration time.Duration) error {
	if err := e.LogsStreamer.StreamLogs(app, w, duration); err != nil {
		return fmt.Errorf("error streaming logs: %v", err)
	}

	return nil
}

// CertsAttach attaches an SSL certificate to the app.
func (e *Empire) CertsAttach(ctx context.Context, app *App, cert string) error {
	tx := e.db.Begin()

	if err := e.certs.CertsAttach(ctx, tx, app, cert); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// Reset resets empire.
func (e *Empire) Reset() error {
	return e.DB.Reset()
}

// IsHealthy returns true if Empire is healthy, which means it can connect to
// the services it depends on.
func (e *Empire) IsHealthy() error {
	return e.DB.IsHealthy()
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

// PullAndExtract returns a ProcfileExtractor that will pull the image using the
// docker client, then attempt to extract the the Procfile, or fallback to the
// CMD directive in the Dockerfile.
func PullAndExtract(c *dockerutil.Client) ProcfileExtractor {
	e := multiExtractor(
		newFileExtractor(c),
		newCMDExtractor(c),
	)

	return ProcfileExtractorFunc(func(ctx context.Context, img image.Image, w io.Writer) ([]byte, error) {
		if err := c.PullImage(ctx, docker.PullImageOptions{
			Registry:      img.Registry,
			Repository:    img.Repository,
			Tag:           img.Tag,
			OutputStream:  w,
			RawJSONStream: true,
		}); err != nil {
			return nil, err
		}

		return e.Extract(ctx, img, w)
	})
}

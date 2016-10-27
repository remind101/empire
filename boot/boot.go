// Package boot provides a Go package for initialzing Empire.
package boot

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	htmltemplate "html/template"
	"net/http"
	"net/url"
	"os"
	"path"
	texttemplate "text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/remind101/conveyor/client/conveyor"
	"github.com/remind101/empire"
	"github.com/remind101/empire/events/app"
	"github.com/remind101/empire/events/sns"
	"github.com/remind101/empire/events/stdout"
	"github.com/remind101/empire/pkg/dockerauth"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/pkg/saml"
	"github.com/remind101/empire/pkg/troposphere"
	"github.com/remind101/empire/scheduler"
	"github.com/remind101/empire/scheduler/cloudformation"
	"github.com/remind101/empire/scheduler/docker"
	"github.com/remind101/empire/server"
	"github.com/remind101/empire/server/auth"
	githubauth "github.com/remind101/empire/server/auth/github"
	customresources "github.com/remind101/empire/server/cloudformation"
	"github.com/remind101/empire/server/github"
	"github.com/remind101/empire/server/heroku"
	"github.com/remind101/empire/server/middleware"
	"github.com/remind101/empire/stats"
	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/hb"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

// Database defaults
var (
	DefaultDatabaseURL = "postgres://localhost/empire?sslmode=disable"
)

// Server defaults
var (
	DefaultPort = "8080"
)

// Docker defaults
var (
	DefaultDockerHost = "unix:///var/run/docker.sock"
	DefaultDockerAuth = path.Join(os.Getenv("HOME"), ".dockercfg")
)

// ECS Defaults
var (
	DefaultECSLogDriver = ""
)

// Empire wraps the 3 main components of Empire under a single easily
// bootstrapped object. The components are:
//
// 1. An empire.Empire instance, serving the core internal Empire API.
// 2. A server.Server instance, serving the Heroku compatibly RESTful API.
// 3. The CloudFormation resource provisioner.
type Empire struct {
	DB                        *empire.DB
	Empire                    *empire.Empire
	Server                    *server.Server
	CustomResourceProvisioner *customresources.CustomResourceProvisioner

	context *Context

	handler http.Handler
}

// Boot boots up an Empire instance using the given Config.
func Boot(config *Config) (*Empire, error) {
	ctx, err := NewContext(config)
	if err != nil {
		return nil, err
	}
	return BootContext(ctx)
}

// BootContext boots up an Empire instance using the given Context.
func BootContext(ctx *Context) (*Empire, error) {
	logger.Info(ctx, "Booting Empire...")

	db, err := ctx.DB()
	if err != nil {
		return nil, err
	}

	// AutoMigrate the database.
	if v := ctx.Config.DB.NoAutoMigrate; v != nil && *v != true {
		if err := db.MigrateUp(); err != nil {
			return nil, err
		}
	}

	docker, err := newDockerClient(ctx)
	if err != nil {
		return nil, err
	}

	scheduler, err := newScheduler(ctx)
	if err != nil {
		return nil, err
	}

	logs, err := newLogsStreamer(ctx)
	if err != nil {
		return nil, err
	}

	streams, err := newEventStreams(ctx)
	if err != nil {
		return nil, err
	}

	runRecorder, err := newRunRecorder(ctx)
	if err != nil {
		return nil, err
	}

	e := empire.New(db)
	e.Scheduler = scheduler
	e.EventStream = empire.AsyncEvents(streams)
	e.ProcfileExtractor = empire.PullAndExtract(docker)
	e.RunRecorder = runRecorder
	e.LogsStreamer = logs

	if v := ctx.Environment; v != nil {
		e.Environment = *v
	}

	if v := ctx.MessagesRequired; v != nil {
		e.MessagesRequired = *v
	}

	if v := ctx.AllowedCommands; v != nil {
		switch *v {
		case "procfile":
			e.AllowedCommands = empire.AllowCommandProcfile
		default:
		}
	}

	s, err := newServer(e, ctx)
	if err != nil {
		return nil, err
	}

	return &Empire{
		DB:     db,
		Empire: e,
		Server: s,
		CustomResourceProvisioner: newCloudFormationCustomResourceProvisioner(e, ctx),

		context: ctx,
		handler: middleware.Handler(ctx, middleware.Common(s)),
	}, nil
}

// ServeHTTP implements the http.Handler interface to serve the Empire HTTP
// server.
func (e *Empire) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.handler.ServeHTTP(w, r)
}

// ListenAndServe starts the Empire http server.
func (e *Empire) ListenAndServe() error {
	port := DefaultPort
	if v := e.context.Server.Port; v != nil {
		port = *v
	}
	logger.Info(e.context, fmt.Sprintf("Starting http server on port %s", port))
	return http.ListenAndServe(":"+port, e)
}

// Start starts this Empire instance. It:
//
// 1. Starts the HTTP server listening.
// 2. Starts the CloudFormation custom resource provisioner.
func (e *Empire) Start() error {
	// Send runtime metrics to stats backend.
	go stats.Runtime(e.context.Stats())

	// Do a preliminary health check to make sure everything is good at
	// boot.
	if err := e.Empire.IsHealthy(); err != nil {
		if err, ok := err.(*empire.IncompatibleSchemaError); ok {
			return fmt.Errorf("%v. You can resolve this error by running the migrations with `empire migrate` or with the `--automigrate` flag", err)
		}

		return err
	}

	logger.Info(e.context, "Preliminary health checks passed")

	if e.CustomResourceProvisioner != nil {
		logger.Info(e.context, "Starting CloudFormation custom resource provisioner")
		// TODO: Track this goroutine.
		go e.CustomResourceProvisioner.Start()
	}
	return e.ListenAndServe()
}

// MigrateUp migrates the database up.
func MigrateUp(ctx *Context) error {
	db, err := ctx.DB()
	if err != nil {
		return err
	}

	return db.MigrateUp()
}

// Context provides lazy loaded, memoized instances of services instantiated
// instances. It also implements the context.Context interfaces with embedded
// reporter.Repoter, and stats.Stats implementations, so it can
// be injected as a top level context object.
type Context struct {
	context.Context
	*Config

	db *empire.DB

	// Error reporting, logging and stats.
	reporter reporter.Reporter
	stats    stats.Stats

	// AWS stuff
	awsConfigProvider client.ConfigProvider

	samlServiceProvider *saml.ServiceProvider
}

// NewContext returns a new Context object.
func NewContext(config *Config) (*Context, error) {
	return NewRootContext(context.Background(), config)
}

// NewRootContext returns a new Context, using the given context.Context as the
// root.
func NewRootContext(root context.Context, c *Config) (ctx *Context, err error) {
	ctx = &Context{
		Config:  c,
		Context: root,
	}

	ctx.reporter, err = newReporter(ctx)
	if err != nil {
		return
	}

	ctx.stats, err = newStats(ctx)
	if err != nil {
		return
	}

	if ctx.reporter != nil {
		ctx.Context = reporter.WithReporter(ctx.Context, ctx.reporter)
	}
	if ctx.stats != nil {
		ctx.Context = stats.WithStats(ctx.Context, ctx.stats)
	}

	return
}

func (c *Context) Reporter() reporter.Reporter { return c.reporter }
func (c *Context) Stats() stats.Stats          { return c.stats }

func (c *Context) DB() (*empire.DB, error) {
	if c.db == nil {
		db, err := newDB(c)
		if err != nil {
			return nil, err
		}
		c.db = db
	}
	return c.db, nil
}

func (c *Context) SQLDB() (*sql.DB, error) {
	db, err := c.DB()
	if err != nil {
		return nil, err
	}
	return db.DB.DB(), nil
}

// ClientConfig implements the client.ConfigProvider interface. This will return
// a mostly standard client.Config, but also includes middleware that will
// generate metrics for retried requests, and enables debug mode if
// `FlagAWSDebug` is set.
func (c *Context) ClientConfig(serviceName string, cfgs ...*aws.Config) client.Config {
	if c.awsConfigProvider == nil {
		c.awsConfigProvider = newConfigProvider(c)
	}

	return c.awsConfigProvider.ClientConfig(serviceName, cfgs...)
}

func (c *Context) SAMLServiceProvider() (*saml.ServiceProvider, error) {
	if c.samlServiceProvider == nil {
		opts := c.Config.Server.Auth.SAML

		if opts.Metadata == nil {
			return nil, nil
		}

		metadataContent, err := uriContentOrValue(*opts.Metadata)
		if err != nil {
			return nil, err
		}

		baseURL := *c.Config.Server.URL

		var metadata saml.Metadata
		if err := xml.Unmarshal(metadataContent, &metadata); err != nil {
			return nil, fmt.Errorf("error parsing SAML metadata: %v", err)
		}

		c.samlServiceProvider = &saml.ServiceProvider{
			IDPMetadata: &metadata,
			MetadataURL: fmt.Sprintf("%s/saml/metadata", baseURL),
			AcsURL:      fmt.Sprintf("%s/saml/acs", baseURL),
		}

		if v := opts.Key; v != nil {
			key, err := uriContentOrValue(*v)
			if err != nil {
				return nil, err
			}
			c.samlServiceProvider.Key = string(key)
		}
		if v := opts.Cert; v != nil {
			cert, err := uriContentOrValue(*v)
			if err != nil {
				return nil, err
			}
			c.samlServiceProvider.Certificate = string(cert)
		}
	}

	return c.samlServiceProvider, nil
}

// newDB returns a new empire.DB instance from the given Config.
func newDB(c *Context) (*empire.DB, error) {
	connstr := DefaultDatabaseURL
	if v := c.Config.DB.URL; v != nil {
		connstr = *v
	}

	if uri, err := url.Parse(connstr); err == nil {
		logger.Info(c, fmt.Sprintf("Opening database connection to %s...", safeURL(*uri)))
	} else {
		logger.Info(c, "Opening database connection...")
	}

	db, err := empire.OpenDB(connstr)
	if err != nil {
		return db, err
	}

	return db, nil
}

func newServer(e *empire.Empire, c *Context) (*server.Server, error) {
	// TODO: Get rid of server.Options
	var opts server.Options
	opts.GitHub.Deployments.ImageBuilder = newImageBuilder(c)
	if v := c.Server.GitHub.Secret; v != nil {
		opts.GitHub.Webhooks.Secret = *v
	}
	if v := c.Server.GitHub.Deployments.Environments; v != nil {
		opts.GitHub.Deployments.Environments = *v
	}
	if v := c.Server.GitHub.Deployments.Tugboat; v != nil {
		opts.GitHub.Deployments.TugboatURL = *c.Tugboat.URL
	}

	s := server.New(e, opts)
	if v := c.Server.Auth.Secret; v != nil {
		s.Heroku.Secret = []byte(*v)
	}

	auth, err := newAuth(e, c)
	if err != nil {
		return nil, err
	}
	s.Heroku.Auth = auth

	sp, err := c.SAMLServiceProvider()
	if err != nil {
		return nil, err
	}

	if sp != nil {
		s.ServiceProvider = sp
		s.Heroku.Unauthorized = heroku.SAMLUnauthorized(*c.Config.Server.URL + "/saml/login")
	}

	return s, nil
}

func newAuth(e *empire.Empire, c *Context) (*auth.Auth, error) {
	var authBackend string
	if v := c.Config.Server.Auth.Backend; v != nil {
		authBackend = *v
	} else {
		// For backwards compatibility. If the auth backend is unspecified, but
		// a github client id is provided, assume the GitHub auth backend.
		if c.Config.Server.Auth.GitHub.ClientID != nil {
			authBackend = "github"
		} else {
			authBackend = "fake"
		}
	}

	// If a GitHub client id is provided, we'll use GitHub as an
	// authentication backend. Otherwise, we'll just use a static username
	// and password backend.
	switch authBackend {
	case "fake":
		logger.Info(c, "Using fake authentication backend")
		// Fake authentication password where the user is "fake" and
		// password is blank.
		return &auth.Auth{
			Strategies: auth.Strategies{
				{
					Name:          auth.StrategyUsernamePassword,
					Authenticator: auth.StaticAuthenticator("fake", "", "", &empire.User{Name: "fake"}),
				},
			},
		}, nil
	case "github":
		opts := c.Config.Server.Auth.GitHub

		config := &oauth2.Config{
			ClientID:     *opts.ClientID,
			ClientSecret: *opts.ClientSecret,
			Scopes:       []string{"repo_deployment", "read:org"},
		}

		var apiURL string
		if v := opts.APIURL; v != nil {
			apiURL = *v
		}
		client := githubauth.NewClient(config)
		client.URL = apiURL

		logger.Info(c, "Using GitHub authentication backend with the following configuration:")
		logger.Info(c, fmt.Sprintf("  ClientID: %v", config.ClientID))
		logger.Info(c, fmt.Sprintf("  ClientSecret: ****"))
		logger.Info(c, fmt.Sprintf("  Scopes: %v", config.Scopes))
		logger.Info(c, fmt.Sprintf("  GitHubAPI: %v", client.URL))

		// an authenticator for authenticating requests with a users github
		// credentials.
		authenticator := githubauth.NewAuthenticator(client)
		a := &auth.Auth{
			Strategies: auth.Strategies{
				{
					Name:          auth.StrategyUsernamePassword,
					Authenticator: authenticator,
				},
			},
		}

		// After the user is authenticated, check their GitHub Organization membership.
		if org := opts.Organization; org != nil {
			authorizer := githubauth.NewOrganizationAuthorizer(client)
			authorizer.Organization = *org

			logger.Info(c, "Adding GitHub Organization authorizer with the following configuration:")
			logger.Info(c, fmt.Sprintf("  Organization: %v ", *org))

			a.Authorizer = auth.CacheAuthorization(authorizer, 30*time.Minute)
		}

		// After the user is authenticated, check their GitHub Team membership.
		if teamID := opts.TeamID; teamID != nil {
			authorizer := githubauth.NewTeamAuthorizer(client)
			authorizer.TeamID = *teamID

			logger.Info(c, "Adding GitHub Team authorizer with the following configuration:")
			logger.Info(c, fmt.Sprintf("  Team ID: %v ", *teamID))

			// Cache the team check for 30 minutes
			a.Authorizer = auth.CacheAuthorization(authorizer, 30*time.Minute)
		}

		return a, nil
	case "saml":
		sp, err := c.SAMLServiceProvider()
		if err != nil {
			return nil, err
		}
		logger.Info(c, "Using SAML authentication backend:")
		logger.Info(c, fmt.Sprintf("  EntityID: %s", sp.IDPMetadata.EntityID))

		loginURL := *c.Config.Server.URL + "/saml/login"

		// When using the SAML authentication backend, access tokens are
		// created through the browser, so username/password
		// authentication should be disabled.
		usernamePasswordDisabled := auth.AuthenticatorFunc(func(username, password, otp string) (*empire.User, error) {
			return nil, fmt.Errorf("Authentication via username/password is disabled. Login at %s", loginURL)
		})

		return &auth.Auth{
			Strategies: auth.Strategies{
				{
					Name:          auth.StrategyUsernamePassword,
					Authenticator: usernamePasswordDisabled,
					// Ensure that this strategy isn't used
					// by default.
					Disabled: true,
				},
			},
		}, nil
	default:
		panic("unreachable")
	}
}

// Scheduler ============================

var schedulerBackends = map[string]func(*Context) (scheduler.Scheduler, error){
	"cloudformation": newCloudFormationScheduler,
}

func newScheduler(c *Context) (scheduler.Scheduler, error) {
	backend := ""
	if v := c.Scheduler.Backend; v != nil {
		backend = *v
	}

	f := schedulerBackends[backend]
	if f == nil {
		return nil, nil
	}

	s, err := f(c)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize %s scheduler: %v", backend, err)
	}

	// TODO: Use a memoized instance.
	d, err := newDockerClient(c)
	if err != nil {
		return nil, err
	}

	a := docker.RunAttachedWithDocker(s, d)
	if v := c.ShowAttachedProcesses; v != nil {
		a.ShowAttached = *v
	}
	return a, nil
}

func newCloudFormationScheduler(c *Context) (scheduler.Scheduler, error) {
	opts := c.Scheduler.CloudFormation

	zoneID := *opts.Route53InternalHostedZoneID

	logger.Info(c, fmt.Sprintf("Fetching HostedZone information for %s", zoneID))
	zone, err := cloudformation.HostedZone(c, zoneID)
	if err != nil {
		return nil, err
	}

	t := &cloudformation.EmpireTemplate{
		VpcId:                   *opts.VpcID,
		Cluster:                 *opts.ECSCluster,
		InternalSecurityGroupID: *opts.ELBPrivateSecurityGroup,
		ExternalSecurityGroupID: *opts.ELBPublicSecurityGroup,
		InternalSubnetIDs:       *opts.EC2PrivateSubnets,
		ExternalSubnetIDs:       *opts.EC2PublicSubnets,
		HostedZone:              zone,
		ServiceRole:             *opts.ECSServiceRole,
		CustomResourcesTopic:    *c.CloudFormationCustomResources.Topic,
		// TODO:
		//LogConfiguration:        logConfiguration,
		ExtraOutputs: map[string]troposphere.Output{
			"EmpireVersion": troposphere.Output{Value: empire.Version},
		},
	}

	// TODO: Move this out. No longer needed
	if err := t.Validate(); err != nil {
		return nil, fmt.Errorf("error validating CloudFormation template: %v", err)
	}

	db, err := c.SQLDB()
	if err != nil {
		return nil, err
	}

	s := cloudformation.NewScheduler(db, c)
	s.Cluster = *opts.ECSCluster
	s.Template = t
	s.Bucket = *opts.TemplateBucket
	if env := c.Environment; env != nil {
		s.StackNameTemplate = prefixedStackName(*env)
		s.Tags = []*cf.Tag{{Key: aws.String("environment"), Value: env}}
	}

	logger.Info(c, "Using CloudFormation backend with the following configuration:")
	logger.Info(c, fmt.Sprintf("  Cluster: %v", s.Cluster))
	logger.Info(c, fmt.Sprintf("  InternalSecurityGroupID: %v", t.InternalSecurityGroupID))
	logger.Info(c, fmt.Sprintf("  ExternalSecurityGroupID: %v", t.ExternalSecurityGroupID))
	logger.Info(c, fmt.Sprintf("  InternalSubnetIDs: %v", t.InternalSubnetIDs))
	logger.Info(c, fmt.Sprintf("  ExternalSubnetIDs: %v", t.ExternalSubnetIDs))
	logger.Info(c, fmt.Sprintf("  ZoneID: %v", zoneID))
	logger.Info(c, fmt.Sprintf("  LogConfiguration: %v", t.LogConfiguration))

	return s, nil
}

// prefixedStackName returns a text/template that prefixes the stack name with
// the given prefix, if it's set.
func prefixedStackName(prefix string) *htmltemplate.Template {
	t := `{{ if "` + prefix + `" }}{{"` + prefix + `"}}-{{ end }}{{.Name}}`
	// TODO: Don't use html/template. Mistake.
	return htmltemplate.Must(htmltemplate.New("stack_name").Parse(t))
}

// DockerClient ========================

// newDockerClient builds a new dockerutil.Client from the given Config.
func newDockerClient(c *Context) (*dockerutil.Client, error) {
	host := DefaultDockerHost
	if v := c.Docker.Host; v != nil {
		host = *v
	}

	logger.Info(c, fmt.Sprintf("Connecting to Docker Daemon at %s...", host))

	certPath := ""
	if v := c.Docker.CertPath; v != nil {
		certPath = *v
	}

	authProvider, err := newDockerAuthProvider(c)
	if err != nil {
		return nil, err
	}

	return dockerutil.NewClient(authProvider, host, certPath)
}

func newDockerAuthProvider(c *Context) (dockerauth.AuthProvider, error) {
	provider := dockerauth.NewMultiAuthProvider()
	provider.AddProvider(dockerauth.NewECRAuthProvider(func(region string) dockerauth.ECR {
		return ecr.New(c, &aws.Config{Region: aws.String(region)})
	}))

	if v := c.Docker.DockerCfg; v != nil {
		path := *v
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}

		defer f.Close()

		dockerConfigProvider, err := dockerauth.NewDockerConfigAuthProvider(f)
		if err != nil {
			return nil, err
		}

		provider.AddProvider(dockerConfigProvider)
	}

	return provider, nil
}

// LogStreamer =========================

var logsStreamerBackends = map[string]func(*Context) (empire.LogsStreamer, error){
	"kinesis": newKinesisLogsStreamer,
}

func newLogsStreamer(c *Context) (empire.LogsStreamer, error) {
	backend := ""
	if c.RunLogs.Backend != nil {
		backend = *c.RunLogs.Backend
	}

	f := logsStreamerBackends[backend]
	if f == nil {
		return nil, nil
	}
	return f(c)
}

func newKinesisLogsStreamer(c *Context) (empire.LogsStreamer, error) {
	logger.Info(c, "Using Kinesis backend for log streaming")
	return empire.NewKinesisLogsStreamer(), nil
}

// Events ==============================

var eventStreamBackends = map[string]func(*Context) (empire.EventStream, error){
	"sns":    newSNSEventStream,
	"stdout": newStdoutEventStream,
}

func newEventStreams(c *Context) (empire.MultiEventStream, error) {
	backend := ""
	if v := c.Events.Backend; v != nil {
		backend = *v
	}

	var streams empire.MultiEventStream
	f := eventStreamBackends[backend]
	if f != nil {
		e, err := f(c)
		if err != nil {
			return streams, err
		}
		streams = append(streams, e)
	}

	if v := c.RunLogs.Backend; v != nil && *v == "kinesis" {
		e, err := newAppEventStream(c)
		if err != nil {
			return streams, err
		}
		streams = append(streams, e)
	}

	return streams, nil
}

func newAppEventStream(c *Context) (empire.EventStream, error) {
	e := app.NewEventStream(c)
	logger.Info(c, "Using App (Kinesis) events backend")
	return e, nil
}

func newSNSEventStream(c *Context) (empire.EventStream, error) {
	e := sns.NewEventStream(c)
	e.TopicARN = *c.Events.SNS.Topic

	logger.Info(c, "Using SNS events backend with the following configuration:")
	logger.Info(c, fmt.Sprintf("  TopicARN: %s", e.TopicARN))

	return e, nil
}

func newStdoutEventStream(c *Context) (empire.EventStream, error) {
	e := stdout.NewEventStream(c)
	logger.Info(c, "Using Stdout events backend")
	return e, nil
}

// RunRecorder =========================

var runRecorderBackends = map[string]func(*Context) (empire.RunRecorder, error){
	"cloudwatch": newCloudWatchRunRecorder,
	"stdout":     newStdoutRunRecorder,
}

func newRunRecorder(c *Context) (empire.RunRecorder, error) {
	backend := ""
	if v := c.RunLogs.Backend; v != nil {
		backend = *v
	}

	f := runRecorderBackends[backend]
	if f == nil {
		return nil, nil
	}

	return f(c)
}

func newCloudWatchRunRecorder(c *Context) (empire.RunRecorder, error) {
	group := *c.RunLogs.CloudWatch.LogGroup
	logger.Info(c, "Using CloudWatch run logs backend with the following configuration:")
	logger.Info(c, fmt.Sprintf("  LogGroup: %s", group))
	return empire.RecordToCloudWatch(group, c), nil
}

func newStdoutRunRecorder(c *Context) (empire.RunRecorder, error) {
	logger.Info(c, "Using Stdout run logs backend")
	return empire.RecordTo(os.Stdout), nil
}

// Reporter ============================

var reporterBackends = map[string]func(*Context) (reporter.Reporter, error){
	"honeybadger": newHBReporter,
}

func newReporter(c *Context) (reporter.Reporter, error) {
	backend := ""
	if v := c.ErrorReporter.Backend; v != nil {
		backend = *v
	}

	f := reporterBackends[backend]
	if f == nil {
		return reporter.NewLogReporter(), nil
	}
	return f(c)
}

func newHBReporter(c *Context) (reporter.Reporter, error) {
	r := hb.NewReporter(*c.ErrorReporter.Honeybadger.ApiKey)
	r.Environment = *c.ErrorReporter.Honeybadger.Environment
	logger.Info(c, "Using Honeybadger to report errors")

	// Append here because `go vet` will complain about unkeyed fields,
	// since it thinks MultiReporter is a struct literal.
	return append(reporter.MultiReporter{}, reporter.NewLogReporter(), r), nil
}

// Stats =======================

func newStats(c *Context) (stats.Stats, error) {
	backend := ""
	if v := c.Config.Stats.Backend; v != nil {
		backend = *v
	}

	switch backend {
	case "statsd":
		return newStatsdStats(c)
	case "dogstatsd":
		return newDogstatsdStats(c)
	default:
		return stats.Null, nil
	}
}

func newStatsdStats(c *Context) (stats.Stats, error) {
	return stats.NewStatsd(*c.Config.Stats.Statsd.Addr, "empire")
}

func newDogstatsdStats(c *Context) (stats.Stats, error) {
	s, err := stats.NewDogstatsd(*c.Config.Stats.DogStatsd.Addr)
	if err != nil {
		return nil, err
	}
	s.Namespace = "empire."
	s.Tags = []string{
		fmt.Sprintf("empire_version:%s", empire.Version),
	}
	return s, nil
}

func newConfigProvider(c *Context) client.ConfigProvider {
	stats := c.Stats()
	config := aws.NewConfig()

	if v := c.AWS.Debug; v != nil && *v == true {
		config.WithLogLevel(aws.LogDebug)
	}

	s := session.New(config)

	requestTags := func(r *request.Request) []string {
		return []string{
			fmt.Sprintf("service_name:%s", r.ClientInfo.ServiceName),
			fmt.Sprintf("api_version:%s", r.ClientInfo.APIVersion),
			fmt.Sprintf("operation:%s", r.Operation.Name),
		}
	}

	s.Handlers.Send.PushBackNamed(request.NamedHandler{
		Name: "empire.RequestMetrics",
		Fn: func(r *request.Request) {
			tags := requestTags(r)
			stats.Inc(fmt.Sprintf("aws.request"), 1, 1.0, tags)
			stats.Inc(fmt.Sprintf("aws.request.%s", r.ClientInfo.ServiceName), 1, 1.0, tags)
			stats.Inc(fmt.Sprintf("aws.request.%s.%s", r.ClientInfo.ServiceName, r.Operation.Name), 1, 1.0, tags)
		},
	})
	s.Handlers.Retry.PushFrontNamed(request.NamedHandler{
		Name: "empire.ErrorMetrics",
		Fn: func(r *request.Request) {
			tags := requestTags(r)
			if r.Error != nil {
				if err, ok := r.Error.(awserr.Error); ok {
					tags = append(tags, fmt.Sprintf("error:%s", err.Code()))
					stats.Inc(fmt.Sprintf("aws.request.error"), 1, 1.0, tags)
					stats.Inc(fmt.Sprintf("aws.request.%s.error", r.ClientInfo.ServiceName), 1, 1.0, tags)
					stats.Inc(fmt.Sprintf("aws.request.%s.%s.error", r.ClientInfo.ServiceName, r.Operation.Name), 1, 1.0, tags)
				}
			}
		},
	})

	return s
}

func newImageBuilder(c *Context) github.ImageBuilder {
	backend := c.Config.Server.GitHub.Deployments.ImageBuilder
	if backend == nil {
		return nil
	}

	switch *backend {
	case "template":
		tmpl := texttemplate.Must(texttemplate.New("image").Parse(*c.Config.Server.GitHub.Deployments.ImageTemplate))
		return github.ImageFromTemplate(tmpl)
	case "conveyor":
		s := conveyor.NewService(conveyor.DefaultClient)
		s.URL = *c.Conveyor.URL
		return github.NewConveyorImageBuilder(s)
	default:
		return nil
	}
}

func newCloudFormationCustomResourceProvisioner(e *empire.Empire, c *Context) *customresources.CustomResourceProvisioner {
	queue := c.CloudFormationCustomResources.Queue
	if queue == nil {
		return nil
	}

	p := customresources.NewCustomResourceProvisioner(e, c)
	p.QueueURL = *queue
	p.Context = c
	return p
}

type safeURL url.URL

func (url safeURL) String() string {
	var username string
	if url.User != nil {
		username = url.User.Username()
	}
	return fmt.Sprintf("%s://%s:***@%s%s...", url.Scheme, username, url.Host, url.Path)
}

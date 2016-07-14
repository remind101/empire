package main

import (
	"fmt"
	"html/template"
	"log"
	"net/url"
	"os"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/codegangsta/cli"
	"github.com/inconshreveable/log15"
	"github.com/remind101/empire"
	"github.com/remind101/empire/events/app"
	"github.com/remind101/empire/events/sns"
	"github.com/remind101/empire/events/stdout"
	"github.com/remind101/empire/pkg/dockerauth"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/pkg/ecsutil"
	"github.com/remind101/empire/pkg/troposphere"
	"github.com/remind101/empire/scheduler"
	"github.com/remind101/empire/scheduler/cloudformation"
	"github.com/remind101/empire/scheduler/docker"
	"github.com/remind101/empire/scheduler/ecs"
	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/hb"
)

// newRootContext returns a new root context.Context with an error reporter and
// logger embedded in the context.
func newRootContext(c *cli.Context) (context.Context, error) {
	r, err := newReporter(c)
	if err != nil {
		return nil, err
	}
	l := newLogger()

	ctx := context.Background()
	if r != nil {
		ctx = reporter.WithReporter(ctx, r)
	}
	if l != nil {
		ctx = logger.WithLogger(ctx, l)
	}

	return ctx, nil
}

// DB ===================================

func newDB(c *cli.Context) (*empire.DB, error) {
	return empire.OpenDB(c.String(FlagDB))
}

// Empire ===============================

func newEmpire(db *empire.DB, c *cli.Context) (*empire.Empire, error) {
	docker, err := newDockerClient(c)
	if err != nil {
		return nil, err
	}

	scheduler, err := newScheduler(db, c)
	if err != nil {
		return nil, err
	}

	logs, err := newLogsStreamer(c)
	if err != nil {
		return nil, err
	}

	streams, err := newEventStreams(c)
	if err != nil {
		return nil, err
	}

	runRecorder, err := newRunRecorder(c)
	if err != nil {
		return nil, err
	}

	e := empire.New(db)
	e.Scheduler = scheduler
	e.Secret = []byte(c.String(FlagSecret))
	e.EventStream = empire.AsyncEvents(streams)
	e.ProcfileExtractor = empire.PullAndExtract(docker)
	e.Environment = c.String(FlagEnvironment)
	e.RunRecorder = runRecorder
	e.MessagesRequired = c.Bool(FlagMessagesRequired)
	if logs != nil {
		e.LogsStreamer = logs
	}

	return e, nil
}

// Scheduler ============================

func newScheduler(db *empire.DB, c *cli.Context) (scheduler.Scheduler, error) {
	var (
		s   scheduler.Scheduler
		err error
	)

	switch c.String(FlagScheduler) {
	case "ecs":
		s, err = newECSScheduler(db, c)
	case "cloudformation-migration":
		s, err = newMigrationScheduler(db, c)
	case "cloudformation":
		s, err = newCloudFormationScheduler(db, c)
	default:
		return nil, fmt.Errorf("unknown scheduler: %s", c.String(FlagScheduler))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize %s scheduler: %v", c.String(FlagScheduler), err)
	}

	d, err := newDockerClient(c)
	if err != nil {
		return nil, err
	}

	a := docker.RunAttachedWithDocker(s, d)
	a.ShowAttached = c.Bool(FlagXShowAttached)
	return a, nil
}

func newMigrationScheduler(db *empire.DB, c *cli.Context) (*cloudformation.MigrationScheduler, error) {
	log.Println("Using the CloudFormation Migration backend")

	es, err := newECSScheduler(db, c)
	if err != nil {
		return nil, fmt.Errorf("error creating ecs scheduler: %v", err)
	}

	cs, err := newCloudFormationScheduler(db, c)
	if err != nil {
		return nil, fmt.Errorf("error creating cloudformation scheduler: %v", err)
	}

	return cloudformation.NewMigrationScheduler(db.DB.DB(), cs, es), nil
}

func newCloudFormationScheduler(db *empire.DB, c *cli.Context) (*cloudformation.Scheduler, error) {
	logDriver := c.String(FlagECSLogDriver)
	logOpts := c.StringSlice(FlagECSLogOpts)
	logConfiguration := ecsutil.NewLogConfiguration(logDriver, logOpts)

	config := newConfigProvider(c)

	zoneID := c.String(FlagRoute53InternalZoneID)
	zone, err := cloudformation.HostedZone(config, zoneID)
	if err != nil {
		return nil, err
	}

	t := &cloudformation.EmpireTemplate{
		Cluster:                 c.String(FlagECSCluster),
		InternalSecurityGroupID: c.String(FlagELBSGPrivate),
		ExternalSecurityGroupID: c.String(FlagELBSGPublic),
		InternalSubnetIDs:       c.StringSlice(FlagEC2SubnetsPrivate),
		ExternalSubnetIDs:       c.StringSlice(FlagEC2SubnetsPublic),
		HostedZone:              zone,
		ServiceRole:             c.String(FlagECSServiceRole),
		CustomResourcesTopic:    c.String(FlagCustomResourcesTopic),
		LogConfiguration:        logConfiguration,
		ExtraOutputs: map[string]troposphere.Output{
			"EmpireVersion": troposphere.Output{Value: empire.Version},
		},
	}

	if err := t.Validate(); err != nil {
		return nil, fmt.Errorf("error validating CloudFormation template: %v", err)
	}

	var tags []*cf.Tag
	if env := c.String(FlagEnvironment); env != "" {
		tags = append(tags, &cf.Tag{Key: aws.String("environment"), Value: aws.String(env)})
	}

	s := cloudformation.NewScheduler(db.DB.DB(), config)
	s.Cluster = c.String(FlagECSCluster)
	s.Template = t
	s.StackNameTemplate = prefixedStackName(c.String(FlagEnvironment))
	s.Bucket = c.String(FlagS3TemplateBucket)
	s.Tags = tags

	log.Println("Using CloudFormation backend with the following configuration:")
	log.Println(fmt.Sprintf("  Cluster: %v", s.Cluster))
	log.Println(fmt.Sprintf("  InternalSecurityGroupID: %v", t.InternalSecurityGroupID))
	log.Println(fmt.Sprintf("  ExternalSecurityGroupID: %v", t.ExternalSecurityGroupID))
	log.Println(fmt.Sprintf("  InternalSubnetIDs: %v", t.InternalSubnetIDs))
	log.Println(fmt.Sprintf("  ExternalSubnetIDs: %v", t.ExternalSubnetIDs))
	log.Println(fmt.Sprintf("  ZoneID: %v", zoneID))
	log.Println(fmt.Sprintf("  LogConfiguration: %v", t.LogConfiguration))

	return s, nil
}

// prefixedStackName returns a text/template that prefixes the stack name with
// the given prefix, if it's set.
func prefixedStackName(prefix string) *template.Template {
	t := `{{ if "` + prefix + `" }}{{"` + prefix + `"}}-{{ end }}{{.Name}}`
	return template.Must(template.New("stack_name").Parse(t))
}

func newECSScheduler(db *empire.DB, c *cli.Context) (*ecs.Scheduler, error) {
	logDriver := c.String(FlagECSLogDriver)
	logOpts := c.StringSlice(FlagECSLogOpts)
	logConfiguration := ecsutil.NewLogConfiguration(logDriver, logOpts)

	config := ecs.Config{
		AWS:                     newConfigProvider(c),
		Cluster:                 c.String(FlagECSCluster),
		ServiceRole:             c.String(FlagECSServiceRole),
		InternalSecurityGroupID: c.String(FlagELBSGPrivate),
		ExternalSecurityGroupID: c.String(FlagELBSGPublic),
		InternalSubnetIDs:       c.StringSlice(FlagEC2SubnetsPrivate),
		ExternalSubnetIDs:       c.StringSlice(FlagEC2SubnetsPublic),
		ZoneID:                  c.String(FlagRoute53InternalZoneID),
		LogConfiguration:        logConfiguration,
	}

	s, err := ecs.NewLoadBalancedScheduler(db.DB.DB(), config)
	if err != nil {
		return nil, err
	}

	log.Println("Using ECS backend with the following configuration:")
	log.Println(fmt.Sprintf("  Cluster: %v", config.Cluster))
	log.Println(fmt.Sprintf("  ServiceRole: %v", config.ServiceRole))
	log.Println(fmt.Sprintf("  InternalSecurityGroupID: %v", config.InternalSecurityGroupID))
	log.Println(fmt.Sprintf("  ExternalSecurityGroupID: %v", config.ExternalSecurityGroupID))
	log.Println(fmt.Sprintf("  InternalSubnetIDs: %v", config.InternalSubnetIDs))
	log.Println(fmt.Sprintf("  ExternalSubnetIDs: %v", config.ExternalSubnetIDs))
	log.Println(fmt.Sprintf("  ZoneID: %v", config.ZoneID))
	log.Println(fmt.Sprintf("  LogConfiguration: %v", logConfiguration))

	return s, nil
}

func newConfigProvider(c *cli.Context) client.ConfigProvider {
	p := session.New()

	if c.Bool(FlagAWSDebug) {
		config := &aws.Config{}
		config.WithLogLevel(aws.LogDebug)
		p = session.New(config)
	}

	return p
}

// DockerClient ========================

func newDockerClient(c *cli.Context) (*dockerutil.Client, error) {
	socket := c.String(FlagDockerSocket)
	certPath := c.String(FlagDockerCert)
	authProvider, err := newAuthProvider(c)
	if err != nil {
		return nil, err
	}

	return dockerutil.NewClient(authProvider, socket, certPath)
}

// LogStreamer =========================

func newLogsStreamer(c *cli.Context) (empire.LogsStreamer, error) {
	switch c.String(FlagLogsStreamer) {
	case "kinesis":
		return newKinesisLogsStreamer(c)
	default:
		log.Println("Streaming logs are disabled")
		return nil, nil
	}
}

func newKinesisLogsStreamer(c *cli.Context) (empire.LogsStreamer, error) {
	log.Println("Using Kinesis backend for log streaming")
	return empire.NewKinesisLogsStreamer(), nil
}

// Events ==============================

func newEventStreams(c *cli.Context) (empire.MultiEventStream, error) {
	var streams empire.MultiEventStream
	switch c.String(FlagEventsBackend) {
	case "sns":
		e, err := newSNSEventStream(c)
		if err != nil {
			return streams, err
		}
		streams = append(streams, e)
	case "stdout":
		e, err := newStdoutEventStream(c)
		if err != nil {
			return streams, err
		}
		streams = append(streams, e)
	default:
		e := empire.NullEventStream
		streams = append(streams, e)
	}

	if c.String(FlagLogsStreamer) == "kinesis" {
		e, err := newAppEventStream(c)
		if err != nil {
			return streams, err
		}
		streams = append(streams, e)
	}
	return streams, nil
}

func newAppEventStream(c *cli.Context) (empire.EventStream, error) {
	e := app.NewEventStream(newConfigProvider(c))
	log.Println("Using App (Kinesis) events backend")
	return e, nil
}

func newSNSEventStream(c *cli.Context) (empire.EventStream, error) {
	e := sns.NewEventStream(newConfigProvider(c))
	e.TopicARN = c.String(FlagSNSTopic)

	log.Println("Using SNS events backend with the following configuration:")
	log.Println(fmt.Sprintf("  TopicARN: %s", e.TopicARN))

	return e, nil
}

func newStdoutEventStream(c *cli.Context) (empire.EventStream, error) {
	e := stdout.NewEventStream(newConfigProvider(c))
	log.Println("Using Stdout events backend")
	return e, nil
}

// RunRecorder =========================

func newRunRecorder(c *cli.Context) (empire.RunRecorder, error) {
	backend := c.String(FlagRunLogsBackend)
	switch backend {
	case "cloudwatch":
		group := c.String(FlagCloudWatchLogGroup)

		log.Println("Using CloudWatch run logs backend with the following configuration:")
		log.Println(fmt.Sprintf("  LogGroup: %s", group))

		return empire.RecordToCloudWatch(group, newConfigProvider(c)), nil
	case "stdout":
		log.Println("Using Stdout run logs backend:")
		return empire.RecordTo(os.Stdout), nil
	default:
		panic(fmt.Sprintf("unknown run logs backend: %v", backend))
	}
}

// Logger ==============================

func newLogger() log15.Logger {
	l := log15.New()
	h := log15.StreamHandler(os.Stdout, log15.LogfmtFormat())
	l.SetHandler(log15.LazyHandler(h))
	return l
}

// Reporter ============================

func newReporter(c *cli.Context) (reporter.Reporter, error) {
	u := c.String(FlagReporter)
	if u == "" {
		return nil, nil
	}

	uri, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	switch uri.Scheme {
	case "hb":
		log.Println("Using Honeybadger to report errors")
		q := uri.Query()
		return newHBReporter(q.Get("key"), q.Get("environment"))
	default:
		panic(fmt.Errorf("unknown reporter: %s", u))
	}
}

func newHBReporter(key, env string) (reporter.Reporter, error) {
	r := hb.NewReporter(key)
	r.Environment = env

	// Append here because `go vet` will complain about unkeyed fields,
	// since it thinks MultiReporter is a struct literal.
	return append(reporter.MultiReporter{}, reporter.NewLogReporter(), r), nil
}

// Auth provider =======================

func newAuthProvider(c *cli.Context) (dockerauth.AuthProvider, error) {
	awsSession := newConfigProvider(c)
	provider := dockerauth.NewMultiAuthProvider()
	provider.AddProvider(dockerauth.NewECRAuthProvider(func(region string) dockerauth.ECR {
		return ecr.New(awsSession, &aws.Config{Region: aws.String(region)})
	}))

	if dockerConfigPath := c.String(FlagDockerAuth); dockerConfigPath != "" {
		dockerConfigFile, err := os.Open(dockerConfigPath)
		if err != nil {
			return nil, err
		}

		defer dockerConfigFile.Close()

		dockerConfigProvider, err := dockerauth.NewDockerConfigAuthProvider(dockerConfigFile)
		if err != nil {
			return nil, err
		}

		provider.AddProvider(dockerConfigProvider)
	}

	return provider, nil
}

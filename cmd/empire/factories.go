package main

import (
	"fmt"
	"html/template"
	"log"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/codegangsta/cli"
	"github.com/inconshreveable/log15"
	"github.com/remind101/empire"
	"github.com/remind101/empire/events/sns"
	"github.com/remind101/empire/pkg/dockerauth"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/pkg/ecsutil"
	"github.com/remind101/empire/pkg/runner"
	"github.com/remind101/empire/scheduler"
	"github.com/remind101/empire/scheduler/cloudformation"
	"github.com/remind101/empire/scheduler/ecs"
	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/hb"
)

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

	reporter, err := newReporter(c)
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

	events, err := newEventStream(c)
	if err != nil {
		return nil, err
	}

	runRecorder, err := newRunRecorder(c)
	if err != nil {
		return nil, err
	}

	e := empire.New(db, empire.Options{
		Secret: c.String(FlagSecret),
	})
	e.Reporter = reporter
	e.Scheduler = scheduler
	e.LogsStreamer = logs
	e.EventStream = empire.AsyncEvents(events)
	e.ProcfileExtractor = empire.PullAndExtract(docker)
	e.Logger = newLogger()
	e.Environment = c.String(FlagEnvironment)
	e.RunRecorder = runRecorder
	e.MessagesRequired = c.Bool(FlagMessagesRequired)

	return e, nil
}

// Scheduler ============================

func newScheduler(db *empire.DB, c *cli.Context) (scheduler.Scheduler, error) {
	r, err := newDockerRunner(c)
	if err != nil {
		return nil, err
	}

	s, err := newMigrationScheduler(db, c)
	if err != nil {
		return nil, err
	}

	return &scheduler.AttachedRunner{
		Scheduler: s,
		Runner:    r,
	}, nil
}

func newMigrationScheduler(db *empire.DB, c *cli.Context) (*cloudformation.MigrationScheduler, error) {
	es, err := newECSScheduler(db, c)
	if err != nil {
		return nil, err
	}

	cs, err := newCloudFormationScheduler(db, c)
	if err != nil {
		return nil, err
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

func newDockerRunner(c *cli.Context) (*runner.Runner, error) {
	client, err := newDockerClient(c)
	if err != nil {
		return nil, err
	}
	return runner.NewRunner(client), nil
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

func newEventStream(c *cli.Context) (empire.EventStream, error) {
	switch c.String(FlagEventsBackend) {
	case "sns":
		return newSNSEventStream(c)
	default:
		return empire.NullEventStream, nil
	}
}

func newSNSEventStream(c *cli.Context) (empire.EventStream, error) {
	e := sns.NewEventStream(newConfigProvider(c))
	e.TopicARN = c.String(FlagSNSTopic)

	log.Println("Using SNS events backend with the following configuration:")
	log.Println(fmt.Sprintf("  TopicARN: %s", e.TopicARN))

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
		return empire.DefaultReporter, nil
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
	return append(reporter.MultiReporter{}, empire.DefaultReporter, r), nil
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

package main

import (
	"fmt"
	"html/template"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/inconshreveable/log15"
	"github.com/remind101/empire"
	"github.com/remind101/empire/events/app"
	"github.com/remind101/empire/events/sns"
	"github.com/remind101/empire/events/stdout"
	"github.com/remind101/empire/extractor"
	"github.com/remind101/empire/logs"
	"github.com/remind101/empire/pkg/dockerauth"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/pkg/troposphere"
	"github.com/remind101/empire/scheduler/cloudformation"
	"github.com/remind101/empire/scheduler/docker"
	"github.com/remind101/empire/stats"
	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/config"
)

// DB ===================================

func newDB(c *Context) (*empire.DB, error) {
	db, err := empire.OpenDB(c.String(FlagDB))
	if err != nil {
		return nil, err
	}

	db.Schema = &empire.Schema{
		InstancePortPool: &empire.InstancePortPool{
			Start: uint(c.Int(FlagInstancePortPoolStart)),
			End:   uint(c.Int(FlagInstancePortPoolEnd)),
		},
	}

	return db, nil
}

// Empire ===============================

func newEmpire(db *empire.DB, c *Context) (*empire.Empire, error) {
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
	e.EventStream = empire.AsyncEvents(streams)
	e.ProcfileExtractor = extractor.PullAndExtract(docker)
	e.Environment = c.String(FlagEnvironment)
	e.RunRecorder = runRecorder
	e.MessagesRequired = c.Bool(FlagMessagesRequired)

	switch c.String(FlagAllowedCommands) {
	case "procfile":
		e.AllowedCommands = empire.AllowCommandProcfile
	default:
	}

	if logs != nil {
		e.LogsStreamer = logs
	}

	return e, nil
}

// Scheduler ============================

func newScheduler(db *empire.DB, c *Context) (empire.Scheduler, error) {
	var (
		s   empire.Scheduler
		err error
	)

	switch c.String(FlagScheduler) {
	case "cloudformation":
		s, err = newCloudFormationScheduler(db, c)
	default:
		return nil, fmt.Errorf("unknown scheduler: %s", c.String(FlagScheduler))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize %s scheduler: %v", c.String(FlagScheduler), err)
	}

	// If ECS tasks support being attached to with a TTY + stdin, let the
	// CloudFormation backend run attached processes.
	if c.Bool(FlagECSAttachedEnabled) {
		return s, nil
	}

	d, err := newDockerClient(c)
	if err != nil {
		return nil, err
	}

	a := docker.RunAttachedWithDocker(s, d)
	a.ShowAttached = c.Bool(FlagXShowAttached)
	return a, nil
}

func newCloudFormationScheduler(db *empire.DB, c *Context) (*cloudformation.Scheduler, error) {
	logDriver := c.String(FlagECSLogDriver)
	logOpts := c.StringSlice(FlagECSLogOpts)
	logConfiguration := newLogConfiguration(logDriver, logOpts)

	zoneID := c.String(FlagRoute53InternalZoneID)
	zone, err := cloudformation.HostedZone(c, zoneID)
	if err != nil {
		return nil, err
	}

	t := &cloudformation.EmpireTemplate{
		VpcId:                   c.String(FlagELBVpcId),
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

	s := cloudformation.NewScheduler(db.DB.DB(), c)
	s.Cluster = c.String(FlagECSCluster)
	s.Template = t
	if v := c.String(FlagCloudFormationStackNameTemplate); v != "" {
		s.StackNameTemplate = stackNameTemplate(v)
	} else {
		s.StackNameTemplate = prefixedStackName(c.String(FlagEnvironment))
	}
	s.Bucket = c.String(FlagS3TemplateBucket)
	s.Tags = tags
	s.NewDockerClient = func(ec2Instance *ec2.Instance) (cloudformation.DockerClient, error) {
		certPath := c.String(FlagECSDockerCert)
		host := ec2Instance.PrivateIpAddress
		if host == nil {
			return nil, fmt.Errorf("instance %s does not have a private ip address", aws.StringValue(ec2Instance.InstanceId))
		}
		port := "2376"
		if certPath == "" {
			port = "2375"
		}
		c, err := dockerutil.NewDockerClient(fmt.Sprintf("tcp://%s:%s", *host, port), certPath)
		if err != nil {
			return c, err
		}
		// Ping the host, just to make sure we can connect.
		return c, c.Ping()
	}

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

func newLogConfiguration(logDriver string, logOpts []string) *ecs.LogConfiguration {
	if logDriver == "" {
		// Default to the docker daemon default logging driver.
		return nil
	}

	logOptions := make(map[string]*string)

	for _, opt := range logOpts {
		logOpt := strings.SplitN(opt, "=", 2)
		if len(logOpt) == 2 {
			logOptions[logOpt[0]] = &logOpt[1]
		}
	}

	return &ecs.LogConfiguration{
		LogDriver: aws.String(logDriver),
		Options:   logOptions,
	}
}

// prefixedStackName returns a text/template that prefixes the stack name with
// the given prefix, if it's set.
func prefixedStackName(prefix string) *template.Template {
	t := `{{ if "` + prefix + `" }}{{"` + prefix + `"}}-{{ end }}{{.Name}}`
	return stackNameTemplate(t)
}

func stackNameTemplate(t string) *template.Template {
	return template.Must(template.New("stack_name").Parse(t))
}

// DockerClient ========================

func newDockerClient(c *Context) (*dockerutil.Client, error) {
	host := c.String(FlagDockerHost)
	certPath := c.String(FlagDockerCert)
	authProvider, err := newAuthProvider(c)
	if err != nil {
		return nil, err
	}

	return dockerutil.NewClient(authProvider, host, certPath)
}

// LogStreamer =========================

func newLogsStreamer(c *Context) (empire.LogsStreamer, error) {
	switch c.String(FlagLogsStreamer) {
	case "kinesis":
		return newKinesisLogsStreamer(c)
	default:
		log.Println("Streaming logs are disabled")
		return nil, nil
	}
}

func newKinesisLogsStreamer(c *Context) (empire.LogsStreamer, error) {
	log.Println("Using Kinesis backend for log streaming")
	return logs.NewKinesisLogsStreamer(), nil
}

// Events ==============================

func newEventStreams(c *Context) (empire.MultiEventStream, error) {
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

func newAppEventStream(c *Context) (empire.EventStream, error) {
	e := app.NewEventStream(c)
	log.Println("Using App (Kinesis) events backend")
	return e, nil
}

func newSNSEventStream(c *Context) (empire.EventStream, error) {
	e := sns.NewEventStream(c)
	e.TopicARN = c.String(FlagSNSTopic)

	log.Println("Using SNS events backend with the following configuration:")
	log.Println(fmt.Sprintf("  TopicARN: %s", e.TopicARN))

	return e, nil
}

func newStdoutEventStream(c *Context) (empire.EventStream, error) {
	e := stdout.NewEventStream(c)
	log.Println("Using Stdout events backend")
	return e, nil
}

// RunRecorder =========================

func newRunRecorder(c *Context) (empire.RunRecorder, error) {
	backend := c.String(FlagRunLogsBackend)
	switch backend {
	case "cloudwatch":
		group := c.String(FlagCloudWatchLogGroup)

		log.Println("Using CloudWatch run logs backend with the following configuration:")
		log.Println(fmt.Sprintf("  LogGroup: %s", group))

		return logs.RecordToCloudWatch(group, c), nil
	case "stdout":
		log.Println("Using Stdout run logs backend")
		return logs.RecordTo(os.Stdout), nil
	default:
		panic(fmt.Sprintf("unknown run logs backend: %v", backend))
	}
}

// Logger ==============================

func newLogger(c *Context) (log15.Logger, error) {
	lvl := c.String(FlagLogLevel)
	l := log15.New()
	log.Println(fmt.Sprintf("Using log level %s", lvl))
	v, err := log15.LvlFromString(lvl)
	if err != nil {
		return l, err
	}
	h := log15.LvlFilterHandler(v, log15.StreamHandler(os.Stdout, log15.LogfmtFormat()))
	if lvl == "debug" {
		h = log15.CallerFileHandler(h)
	}
	l.SetHandler(log15.LazyHandler(h))
	return l, err
}

// Reporter ============================

func newReporter(c *Context) (reporter.Reporter, error) {
	rep, err := config.NewReporterFromUrls(c.StringSlice(FlagReporter))
	if err != nil {
		panic(fmt.Errorf("couldn't create reporter: %#v", err))
	}
	return rep, nil
}

// Stats =======================

func newStats(c *Context) (stats.Stats, error) {
	u := c.String(FlagStats)
	if u == "" {
		return stats.Null, nil
	}

	uri, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	switch uri.Scheme {
	case "statsd":
		return newStatsdStats(uri.Host)
	case "dogstatsd":
		return newDogstatsdStats(uri.Host)
	default:
		return stats.Null, nil
	}
}

func newStatsdStats(addr string) (stats.Stats, error) {
	return stats.NewStatsd(addr, "empire")
}

func newDogstatsdStats(addr string) (stats.Stats, error) {
	s, err := stats.NewDogstatsd(addr)
	if err != nil {
		return nil, err
	}
	s.Namespace = "empire."
	s.Tags = []string{
		fmt.Sprintf("empire_version:%s", empire.Version),
	}
	return s, nil
}

// Auth provider =======================

func newAuthProvider(c *Context) (dockerauth.AuthProvider, error) {
	provider := dockerauth.NewMultiAuthProvider()
	provider.AddProvider(dockerauth.NewECRAuthProvider(func(region string) dockerauth.ECR {
		return ecr.New(c, &aws.Config{Region: aws.String(region)})
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

func newConfigProvider(c *Context) client.ConfigProvider {
	stats := c.Stats()
	config := aws.NewConfig()

	if c.Bool(FlagAWSDebug) {
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

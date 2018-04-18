package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/inconshreveable/log15"
	"github.com/remind101/empire"
	"github.com/remind101/empire/events/app"
	"github.com/remind101/empire/events/sns"
	"github.com/remind101/empire/events/stdout"
	"github.com/remind101/empire/internal/ghinstallation"
	"github.com/remind101/empire/logs"
	"github.com/remind101/empire/pkg/dockerauth"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/registry"
	"github.com/remind101/empire/stats"
	"github.com/remind101/empire/storage/github"
	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/config"
)

// Empire ===============================

func newEmpire(c *Context) (*empire.Empire, error) {
	storage, err := newStorage(c)
	if err != nil {
		return nil, err
	}

	docker, err := newDockerClient(c)
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

	reg, err := newRegistry(docker, c)
	if err != nil {
		return nil, err
	}

	e := empire.New(storage)
	e.EventStream = empire.AsyncEvents(streams)
	e.ImageRegistry = reg
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

// Storage ==============================

func newStorage(c *Context) (empire.Storage, error) {
	return newGitHubStorage(c)
}

func newGitHubStorage(c *Context) (*github.Storage, error) {
	githubAppID := c.Int(FlagStorageGitHubAppID)
	githubInstallationID := c.Int(FlagStorageGitHubInstallationID)
	githubPrivateKey, err := base64.StdEncoding.DecodeString(c.String(FlagStorageGitHubPrivateKey))
	if err != nil {
		return nil, err
	}
	itr, err := ghinstallation.New(http.DefaultTransport, githubAppID, githubInstallationID, githubPrivateKey)
	if err != nil {
		return nil, err
	}

	s := github.NewStorage(&http.Client{Transport: itr})
	parts := strings.SplitN(c.String(FlagStorageGitHubRepo), "/", 2)
	s.Owner = parts[0]
	s.Repo = parts[1]
	s.BasePath = c.String(FlagStorageGitHubBasePath)
	s.Ref = c.String(FlagStorageGitHubRef)

	return s, nil
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

func newRegistry(client *dockerutil.Client, c *Context) (empire.ImageRegistry, error) {
	r := registry.DockerDaemon(client)

	digests := c.String(FlagDockerDigests)
	switch digests {
	case "prefer":
		r.Digests = registry.DigestsPrefer
	case "enforce":
		log.Println("Image digests are enforced")
		r.Digests = registry.DigestsOnly
	case "disable":
		log.Println("Image digests are disabled")
		r.Digests = registry.DigestsDisable
	default:
		return nil, fmt.Errorf("invalid value for %s: %s", FlagDockerDigests, digests)
	}

	return r, nil
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

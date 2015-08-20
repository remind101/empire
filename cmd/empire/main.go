package main

import (
	"fmt"
	"net/url"
	"os"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/codegangsta/cli"
	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire"
	"github.com/remind101/empire/server/github"
	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/hb"
)

const (
	FlagPort        = "port"
	FlagAutoMigrate = "automigrate"

	FlagGithubClient       = "github.client.id"
	FlagGithubClientSecret = "github.client.secret"
	FlagGithubOrg          = "github.organization"
	FlagGithubApiURL       = "github.api.url"

	FlagGithubWebhooksSecret           = "github.webhooks.secret"
	FlagGithubDeploymentsEnvironment   = "github.deployments.environment"
	FlagGithubDeploymentsImageTemplate = "github.deployments.template"
	FlagGithubDeploymentsTugboatURL    = "github.deployments.tugboat.url"

	FlagDBPath = "path"
	FlagDB     = "db"

	FlagDockerSocket = "docker.socket"
	FlagDockerCert   = "docker.cert"
	FlagDockerAuth   = "docker.auth"

	FlagAWSDebug       = "aws.debug"
	FlagECSCluster     = "ecs.cluster"
	FlagECSServiceRole = "ecs.service.role"

	FlagELBSGPrivate = "elb.sg.private"
	FlagELBSGPublic  = "elb.sg.public"

	FlagEC2SubnetsPrivate = "ec2.subnets.private"
	FlagEC2SubnetsPublic  = "ec2.subnets.public"

	FlagRoute53InternalZoneID = "route53.zoneid.internal"

	FlagSecret   = "secret"
	FlagReporter = "reporter"
	FlagRunner   = "runner"
)

// Commands are the subcommands that are available.
var Commands = []cli.Command{
	{
		Name:      "server",
		ShortName: "s",
		Usage:     "Run the empire HTTP api",
		Flags: append([]cli.Flag{
			cli.StringFlag{
				Name:   FlagPort,
				Value:  "8080",
				Usage:  "The port to run the server on",
				EnvVar: "EMPIRE_PORT",
			},
			cli.BoolFlag{
				Name:  FlagAutoMigrate,
				Usage: "Whether to run the migrations at startup or not",
			},
			cli.StringFlag{
				Name:   FlagGithubClient,
				Value:  "",
				Usage:  "The client id for the GitHub OAuth application",
				EnvVar: "EMPIRE_GITHUB_CLIENT_ID",
			},
			cli.StringFlag{
				Name:   FlagGithubClientSecret,
				Value:  "",
				Usage:  "The client secret for the GitHub OAuth application",
				EnvVar: "EMPIRE_GITHUB_CLIENT_SECRET",
			},
			cli.StringFlag{
				Name:   FlagGithubOrg,
				Value:  "",
				Usage:  "The organization to allow access to",
				EnvVar: "EMPIRE_GITHUB_ORGANIZATION",
			},
			cli.StringFlag{
				Name:   FlagGithubApiURL,
				Value:  "",
				Usage:  "The URL to use when talking to GitHub.",
				EnvVar: "EMPIRE_GITHUB_API_URL",
			},
			cli.StringFlag{
				Name:   FlagGithubWebhooksSecret,
				Value:  "",
				Usage:  "Shared secret between GitHub and Empire for signing webhooks.",
				EnvVar: "EMPIRE_GITHUB_WEBHOOKS_SECRET",
			},
			cli.StringFlag{
				Name:   FlagGithubDeploymentsEnvironment,
				Value:  "",
				Usage:  "If provided, only github deployments to the specified environment will be handled.",
				EnvVar: "EMPIRE_GITHUB_DEPLOYMENTS_ENVIRONMENT",
			},
			cli.StringFlag{
				Name:   FlagGithubDeploymentsImageTemplate,
				Value:  github.DefaultTemplate,
				Usage:  "A Go text/template that will be used to determine the docker image to deploy.",
				EnvVar: "EMPIRE_GITHUB_DEPLOYMENTS_IMAGE_TEMPLATE",
			},
			cli.StringFlag{
				Name:   FlagGithubDeploymentsTugboatURL,
				Value:  "",
				Usage:  "If provided, logs from deployments triggered via GitHub deployments will be sent to this tugboat instance.",
				EnvVar: "EMPIRE_TUGBOAT_URL",
			},
		}, append(EmpireFlags, DBFlags...)...),
		Action: runServer,
	},
	{
		Name:   "migrate",
		Usage:  "Migrate the database",
		Flags:  DBFlags,
		Action: runMigrate,
	},
}

var DBFlags = []cli.Flag{
	cli.StringFlag{
		Name:  FlagDBPath,
		Value: "./migrations",
		Usage: "Path to database migrations",
	},
	cli.StringFlag{
		Name:   FlagDB,
		Value:  "postgres://localhost/empire?sslmode=disable",
		Usage:  "SQL connection string for the database",
		EnvVar: "EMPIRE_DATABASE_URL",
	},
}

var EmpireFlags = []cli.Flag{
	cli.StringFlag{
		Name:   FlagDockerSocket,
		Value:  "unix:///var/run/docker.sock",
		Usage:  "The location of the docker api",
		EnvVar: "DOCKER_HOST",
	},
	cli.StringFlag{
		Name:   FlagDockerCert,
		Value:  "",
		Usage:  "If using TLS, a path to a certificate to use",
		EnvVar: "DOCKER_CERT_PATH",
	},
	cli.StringFlag{
		Name:   FlagDockerAuth,
		Value:  path.Join(os.Getenv("HOME"), ".dockercfg"),
		Usage:  "Path to a docker registry auth file (~/.dockercfg)",
		EnvVar: "DOCKER_AUTH_PATH",
	},
	cli.BoolFlag{
		Name:   FlagAWSDebug,
		Usage:  "Enable verbose debug output for AWS integration.",
		EnvVar: "EMPIRE_AWS_DEBUG",
	},
	cli.StringFlag{
		Name:   FlagECSCluster,
		Value:  "default",
		Usage:  "The ECS cluster to create services within",
		EnvVar: "EMPIRE_ECS_CLUSTER",
	},
	cli.StringFlag{
		Name:   FlagECSServiceRole,
		Value:  "ecsServiceRole",
		Usage:  "The IAM Role to use for managing ECS",
		EnvVar: "EMPIRE_ECS_SERVICE_ROLE",
	},
	cli.StringFlag{
		Name:   FlagELBSGPrivate,
		Value:  "",
		Usage:  "The ELB security group to assign private load balancers",
		EnvVar: "EMPIRE_ELB_SG_PRIVATE",
	},
	cli.StringFlag{
		Name:   FlagELBSGPublic,
		Value:  "",
		Usage:  "The ELB security group to assign public load balancers",
		EnvVar: "EMPIRE_ELB_SG_PUBLIC",
	},
	cli.StringSliceFlag{
		Name:   FlagEC2SubnetsPrivate,
		Value:  &cli.StringSlice{},
		Usage:  "The comma separated private subnet ids",
		EnvVar: "EMPIRE_EC2_SUBNETS_PRIVATE",
	},
	cli.StringSliceFlag{
		Name:   FlagEC2SubnetsPublic,
		Value:  &cli.StringSlice{},
		Usage:  "The comma separated public subnet ids",
		EnvVar: "EMPIRE_EC2_SUBNETS_PUBLIC",
	},
	cli.StringFlag{
		Name:   FlagSecret,
		Value:  "<change this>",
		Usage:  "The secret used to sign access tokens",
		EnvVar: "EMPIRE_TOKEN_SECRET",
	},
	cli.StringFlag{
		Name:   FlagReporter,
		Value:  "",
		Usage:  "The error reporter to use. (e.g. hb://api.honeybadger.io?key=<apikey>&environment=production)",
		EnvVar: "EMPIRE_REPORTER",
	},
	cli.StringFlag{
		Name:   FlagRunner,
		Value:  "",
		Usage:  "The location of the container runner api",
		EnvVar: "EMPIRE_RUNNER",
	},
	cli.StringFlag{
		Name:   FlagRoute53InternalZoneID,
		Value:  "",
		Usage:  "The route53 zone ID of the internal 'empire.' zone.",
		EnvVar: "EMPIRE_ROUTE53_INTERNAL_ZONE_ID",
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "empire"
	app.Usage = "Platform as a Binary"
	app.Version = Version
	app.Commands = Commands

	app.Run(os.Args)
}

func newEmpire(c *cli.Context) (*empire.Empire, error) {
	opts := empire.Options{}

	opts.Docker.Socket = c.String(FlagDockerSocket)
	opts.Docker.CertPath = c.String(FlagDockerCert)
	opts.AWSConfig = aws.NewConfig()
	if c.Bool(FlagAWSDebug) {
		opts.AWSConfig.WithLogLevel(1)
	}
	opts.ECS.Cluster = c.String(FlagECSCluster)
	opts.ECS.ServiceRole = c.String(FlagECSServiceRole)
	opts.ELB.InternalSecurityGroupID = c.String(FlagELBSGPrivate)
	opts.ELB.ExternalSecurityGroupID = c.String(FlagELBSGPublic)
	opts.ELB.InternalSubnetIDs = c.StringSlice(FlagEC2SubnetsPrivate)
	opts.ELB.ExternalSubnetIDs = c.StringSlice(FlagEC2SubnetsPublic)
	opts.ELB.InternalZoneID = c.String(FlagRoute53InternalZoneID)
	opts.DB = c.String(FlagDB)
	opts.Secret = c.String(FlagSecret)

	auth, err := dockerAuth(c.String(FlagDockerAuth))
	if err != nil {
		return nil, err
	}

	opts.Docker.Auth = auth

	e, err := empire.New(opts)
	if err != nil {
		return e, err
	}

	reporter, err := newReporter(c.String(FlagReporter))
	if err != nil {
		return e, err
	}

	e.Reporter = reporter

	return e, nil
}

func newReporter(u string) (reporter.Reporter, error) {
	if u == "" {
		return empire.DefaultReporter, nil
	}

	uri, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	switch uri.Scheme {
	case "hb":
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

func dockerAuth(path string) (*docker.AuthConfigurations, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return docker.NewAuthConfigurations(f)
}

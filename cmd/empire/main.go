package main

import (
	"os"
	"path"

	"github.com/codegangsta/cli"
	"github.com/remind101/empire"
	"github.com/remind101/empire/server/github"
)

const (
	FlagPort             = "port"
	FlagAutoMigrate      = "automigrate"
	FlagScheduler        = "scheduler"
	FlagEventsBackend    = "events.backend"
	FlagRunLogsBackend   = "runlogs.backend"
	FlagMessagesRequired = "messages.required"
	FlagLogLevel         = "log.level"

	FlagDogStatsd = "dogstatsd"

	FlagGithubClient       = "github.client.id"
	FlagGithubClientSecret = "github.client.secret"
	FlagGithubOrg          = "github.organization"
	FlagGithubApiURL       = "github.api.url"
	FlagGithubTeam         = "github.team.id"

	FlagGithubWebhooksSecret           = "github.webhooks.secret"
	FlagGithubDeploymentsEnvironments  = "github.deployments.environment"
	FlagGithubDeploymentsImageBuilder  = "github.deployments.image_builder"
	FlagGithubDeploymentsImageTemplate = "github.deployments.template"
	FlagGithubDeploymentsTugboatURL    = "github.deployments.tugboat.url"

	FlagConveyorURL = "conveyor.url"

	FlagDB = "db"

	FlagDockerSocket = "docker.socket"
	FlagDockerCert   = "docker.cert"
	FlagDockerAuth   = "docker.auth"

	FlagAWSDebug             = "aws.debug"
	FlagS3TemplateBucket     = "s3.templatebucket"
	FlagCustomResourcesTopic = "customresources.topic"
	FlagCustomResourcesQueue = "customresources.queue"
	FlagECSCluster           = "ecs.cluster"
	FlagECSServiceRole       = "ecs.service.role"
	FlagECSLogDriver         = "ecs.logdriver"
	FlagECSLogOpts           = "ecs.logopt"

	FlagELBSGPrivate = "elb.sg.private"
	FlagELBSGPublic  = "elb.sg.public"

	FlagEC2SubnetsPrivate = "ec2.subnets.private"
	FlagEC2SubnetsPublic  = "ec2.subnets.public"

	FlagRoute53InternalZoneID = "route53.zoneid.internal"

	FlagSNSTopic           = "sns.topic"
	FlagCloudWatchLogGroup = "cloudwatch.loggroup"

	FlagSecret       = "secret"
	FlagReporter     = "reporter"
	FlagRunner       = "runner"
	FlagLogsStreamer = "logs.streamer"

	FlagEnvironment = "environment"

	// Expiremental flags.
	FlagXShowAttached = "x.showattached"
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
				Name:   FlagScheduler,
				Value:  "ecs",
				Usage:  "The scheduling backend to use. Current options are `ecs`, `cloudformation-migration`, and `cloudformation`.",
				EnvVar: "EMPIRE_SCHEDULER",
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
				Name:   FlagGithubTeam,
				Value:  "",
				Usage:  "The ID of the github team to allow access to",
				EnvVar: "EMPIRE_GITHUB_TEAM_ID",
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
				Name:   FlagGithubDeploymentsEnvironments,
				Value:  "",
				Usage:  "If provided, only github deployments to the specified environments will be handled.",
				EnvVar: "EMPIRE_GITHUB_DEPLOYMENTS_ENVIRONMENT",
			},
			cli.StringFlag{
				Name:   FlagGithubDeploymentsImageBuilder,
				Value:  "template",
				Usage:  "Determines how the Docker image to deploy is determined when a GitHub Deployment event is received. Possible options are `template` and `conveyor`.",
				EnvVar: "EMPIRE_GITHUB_DEPLOYMENTS_IMAGE_BUILDER",
			},
			cli.StringFlag{
				Name:   FlagGithubDeploymentsImageTemplate,
				Value:  github.DefaultTemplate,
				Usage:  "A Go text/template that will be used to determine the docker image to deploy. This flag is only used when `--" + FlagGithubDeploymentsImageBuilder + "` is set to `template`.",
				EnvVar: "EMPIRE_GITHUB_DEPLOYMENTS_IMAGE_TEMPLATE",
			},
			cli.StringFlag{
				Name:   FlagGithubDeploymentsTugboatURL,
				Value:  "",
				Usage:  "If provided, logs from deployments triggered via GitHub deployments will be sent to this tugboat instance.",
				EnvVar: "EMPIRE_TUGBOAT_URL",
			},
			cli.StringFlag{
				Name:   FlagConveyorURL,
				Value:  "",
				Usage:  "When combined with the `--" + FlagGithubDeploymentsImageBuilder + "` flag when set to `conveyor`, this determines where the location of a Conveyor instance is to perform Docker image builds.",
				EnvVar: "EMPIRE_CONVEYOR_URL",
			},
		}, append(CommonFlags, append(EmpireFlags, DBFlags...)...)...),
		Action: runServer,
	},
	{
		Name:   "migrate",
		Usage:  "Migrate the database",
		Flags:  append(CommonFlags, DBFlags...),
		Action: runMigrate,
	},
}

var CommonFlags = []cli.Flag{
	cli.StringFlag{
		Name:   FlagLogLevel,
		Value:  "info",
		Usage:  "Specify the log level for the empire server. You can use this to enable debug logs by specifying `debug`.",
		EnvVar: "EMPIRE_LOG_LEVEL",
	},
	cli.StringFlag{
		Name:   FlagDogStatsd,
		Value:  "",
		Usage:  "The address of a dogstatsd instance to send metrics to.",
		EnvVar: "EMPIRE_DOGSTATSD",
	},
	cli.StringFlag{
		Name:   FlagReporter,
		Value:  "",
		Usage:  "The error reporter to use. (e.g. hb://api.honeybadger.io?key=<apikey>&environment=production)",
		EnvVar: "EMPIRE_REPORTER",
	},
}

var DBFlags = []cli.Flag{
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
		Name:   FlagS3TemplateBucket,
		Usage:  "When using the cloudformation backend, this is the bucket where templates will be stored",
		EnvVar: "EMPIRE_S3_TEMPLATE_BUCKET",
	},
	cli.StringFlag{
		Name:   FlagCustomResourcesTopic,
		Usage:  "The ARN of the SNS topic used to create custom resources when using the CloudFormation backend.",
		EnvVar: "EMPIRE_CUSTOM_RESOURCES_TOPIC",
	},
	cli.StringFlag{
		Name:   FlagCustomResourcesQueue,
		Usage:  "The queue url of the SQS queue to pull CloudFormation Custom Resource requests from.",
		EnvVar: "EMPIRE_CUSTOM_RESOURCES_QUEUE",
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
		Name:   FlagECSLogDriver,
		Value:  "",
		Usage:  "Log driver to use when running containers. Maps to the --log-driver docker cli arg",
		EnvVar: "EMPIRE_ECS_LOG_DRIVER",
	},
	cli.StringSliceFlag{
		Name:   FlagECSLogOpts,
		Value:  &cli.StringSlice{},
		Usage:  "Log driver to options. Maps to the --log-opt docker cli arg",
		EnvVar: "EMPIRE_ECS_LOG_OPT",
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
	cli.StringFlag{
		Name:   FlagLogsStreamer,
		Value:  "",
		Usage:  "The location of the logs to stream",
		EnvVar: "EMPIRE_LOGS_STREAMER",
	},
	cli.StringFlag{
		Name:   FlagEventsBackend,
		Value:  "",
		Usage:  "The backend implementation to use to send event notifactions",
		EnvVar: "EMPIRE_EVENTS_BACKEND",
	},
	cli.StringFlag{
		Name:   FlagRunLogsBackend,
		Value:  "stdout",
		Usage:  "The backend implementation to use to record the logs from interactive runs. Current supports `cloudwatch` and `stdout`",
		EnvVar: "EMPIRE_RUN_LOGS_BACKEND",
	},
	cli.StringFlag{
		Name:   FlagSNSTopic,
		Value:  "",
		Usage:  "When using the SNS events backend, this is the SNS topic that gets published to",
		EnvVar: "EMPIRE_SNS_TOPIC",
	},
	cli.StringFlag{
		Name:   FlagEnvironment,
		Value:  "",
		Usage:  "Used to distinguish the environment this Empire is used to manage. Used for tagging of resources and annotating events.",
		EnvVar: "EMPIRE_ENVIRONMENT",
	},
	cli.StringFlag{
		Name:   FlagCloudWatchLogGroup,
		Value:  "",
		Usage:  "When using the `cloudwatch` backend with the `--" + FlagRunLogsBackend + "` flag , this is the log group that CloudWatch log streams will be created in.",
		EnvVar: "EMPIRE_CLOUDWATCH_LOG_GROUP",
	},
	cli.BoolFlag{
		Name:   FlagMessagesRequired,
		Usage:  "If true, messages will be required for empire actions that emit events.",
		EnvVar: "EMPIRE_MESSAGES_REQUIRED",
	},
	cli.BoolFlag{
		Name:   FlagXShowAttached,
		Usage:  "If true, attached runs will be shown in `emp ps` output.",
		EnvVar: "EMPIRE_X_SHOW_ATTACHED",
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "empire"
	app.Usage = "Platform as a Binary"
	app.Version = empire.Version
	app.Commands = Commands

	app.Run(os.Args)
}

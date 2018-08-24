package main

import (
	"fmt"
	"os"
	"path"

	"github.com/remind101/empire"
	"github.com/remind101/empire/server/github"
	"github.com/urfave/cli"
)

const hbExampleURL = "hb://api.honeybadger.io?key=<key>&environment=<environment>"
const rollbarExampleURL = "rollbar://api.rollbar.com?key=<key>&environment=<environment>"

const (
	FlagURL            = "url"
	FlagPort           = "port"
	FlagScheduler      = "scheduler"
	FlagEventsBackend  = "events.backend"
	FlagRunLogsBackend = "runlogs.backend"
	FlagLogLevel       = "log.level"

	FlagMessagesRequired = "messages.required"
	FlagAllowedCommands  = "commands.allowed"

	FlagStats = "stats"

	FlagServerAuth              = "server.auth"
	FlagServerSessionExpiration = "server.session.expiration"
	FlagServerRealIp            = "server.realip"

	FlagStorageGitHubAppID          = "storage.github.app_id"
	FlagStorageGitHubInstallationID = "storage.github.installation_id"
	FlagStorageGitHubPrivateKey     = "storage.github.private_key"
	FlagStorageGitHubRepo           = "storage.github.repo"
	FlagStorageGitHubBasePath       = "storage.github.base_path"
	FlagStorageGitHubRef            = "storage.github.ref"

	FlagSAMLMetadata       = "saml.metadata"
	FlagSAMLKey            = "saml.key"
	FlagSAMLCert           = "saml.cert"
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

	FlagDockerHost    = "docker.socket"
	FlagDockerCert    = "docker.cert"
	FlagDockerAuth    = "docker.auth"
	FlagDockerDigests = "docker.digests"

	FlagAWSDebug = "aws.debug"

	FlagSNSTopic           = "sns.topic"
	FlagCloudWatchLogGroup = "cloudwatch.loggroup"

	FlagSecret       = "secret"
	FlagReporter     = "reporter"
	FlagRunner       = "runner"
	FlagLogsStreamer = "logs.streamer"

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
				Name:   FlagURL,
				Value:  "",
				Usage:  "That base URL where this Empire instance runs",
				EnvVar: "EMPIRE_URL",
			},
			cli.StringFlag{
				Name:   FlagPort,
				Value:  "8080",
				Usage:  "The port to run the server on",
				EnvVar: "EMPIRE_PORT",
			},
			cli.StringFlag{
				Name:   FlagScheduler,
				Value:  "cloudformation",
				Usage:  "The scheduling backend to use. Current options are `cloudformation`.",
				EnvVar: "EMPIRE_SCHEDULER",
			},
			cli.StringFlag{
				Name:   FlagServerAuth,
				Value:  "",
				Usage:  "The authentication backend to use to authenticate requests to the API. Can be `fake`, `github`, or `saml`.",
				EnvVar: "EMPIRE_SERVER_AUTH",
			},
			cli.DurationFlag{
				Name:   FlagServerSessionExpiration,
				Value:  0,
				Usage:  "The maximum amount of time that sessions and access tokens exist for. For security, it's a good idea to set this to a low value, like `24h`. Refer to the ParseDuration docs for details on acceptable values (https://golang.org/pkg/time/#ParseDuration). By default, sessions do not expire.",
				EnvVar: "EMPIRE_SERVER_SESSION_EXPIRATION",
			},
			cli.StringSliceFlag{
				Name:   FlagServerRealIp,
				Value:  &cli.StringSlice{},
				Usage:  "Determines the headers that can be trusted to determine the real ip. By default, no headers are trusted and the ip is extracted from the remote address. If you're using ELB, you should set this to X-Forwarded-For. If you're using something like nginx + real_ip module, you can set this to X-Real-Ip.",
				EnvVar: "EMPIRE_SERVER_REAL_IP",
			},
			cli.IntFlag{
				Name:   FlagStorageGitHubAppID,
				Usage:  "The GitHub app ID that will be used to make automated commits.",
				EnvVar: "EMPIRE_STORAGE_GITHUB_APP_ID",
			},
			cli.IntFlag{
				Name:   FlagStorageGitHubInstallationID,
				Usage:  "The GitHub app installation ID that will be used to make automated commits.",
				EnvVar: "EMPIRE_STORAGE_GITHUB_INSTALLATION_ID",
			},
			cli.StringFlag{
				Name:   FlagStorageGitHubPrivateKey,
				Usage:  "Base64'd PEM encoded private key for the GitHub app.",
				EnvVar: "EMPIRE_STORAGE_GITHUB_PRIVATE_KEY",
			},
			cli.StringFlag{
				Name:   FlagStorageGitHubRepo,
				Usage:  "The name of the GitHub repo where app configuration will be stored.",
				EnvVar: "EMPIRE_STORAGE_GITHUB_REPO",
			},
			cli.StringFlag{
				Name:   FlagStorageGitHubBasePath,
				Usage:  "The base path for where app configuration will be stored.",
				EnvVar: "EMPIRE_STORAGE_GITHUB_BASE_PATH",
			},
			cli.StringFlag{
				Name:   FlagStorageGitHubRef,
				Value:  "refs/heads/master",
				Usage:  "The target ref where changes made by Empire will be merged.",
				EnvVar: "EMPIRE_STORAGE_GITHUB_REF",
			},
			cli.StringFlag{
				Name:   FlagSAMLMetadata,
				Value:  "",
				Usage:  "The location of the SAML metadata XML. This can be a url, path to a file, or the raw xml content. (e.g. https://app.onelogin.com/saml/metadata/1234, file:///etc/empire/saml_metadata.xml)",
				EnvVar: "EMPIRE_SAML_METADATA",
			},
			cli.StringFlag{
				Name:   FlagSAMLKey,
				Value:  "",
				Usage:  "The location of the RSA key used to sign requests. (e.g. file:///etc/empire/saml.key)",
				EnvVar: "EMPIRE_SAML_KEY",
			},
			cli.StringFlag{
				Name:   FlagSAMLCert,
				Value:  "",
				Usage:  "The location of the public key for this service provider. (e.g. file:///etc/empire/saml.cert)",
				EnvVar: "EMPIRE_SAML_CERT",
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
		}, append(CommonFlags, EmpireFlags...)...),
		Action: runServer,
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
		Name:   FlagStats,
		Value:  "",
		Usage:  "The stats backend to use. (e.g. statsd://localhost:8125)",
		EnvVar: "EMPIRE_STATS",
	},
	cli.StringSliceFlag{
		Name:  FlagReporter,
		Value: &cli.StringSlice{},
		Usage: fmt.Sprintf("The reporter to use to report errors. Available options are `%s` or `%s`",
			hbExampleURL, rollbarExampleURL),
		EnvVar: "EMPIRE_REPORTER",
	},
}

var EmpireFlags = []cli.Flag{
	cli.StringFlag{
		Name:   FlagDockerHost,
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
	cli.StringFlag{
		Name:   FlagDockerDigests,
		Value:  "prefer",
		Usage:  "Determines how Empire stores Docker image references. By default, Empire will try to resolve a mutable reference (e.g. remind101/acme-inc:master) to an immutable reference using the images content adressable digest (e.g. remind101/acme-inc@sha256:c6f77d2098bc0e32aef3102e71b51831a9083dd9356a0ccadca860596a1e9007) if the Docker daemon supports it. This can be disabled by setting to \"disable\" or enforce digests by setting to \"enforce\".",
		EnvVar: "DOCKER_DIGESTS",
	},
	cli.BoolFlag{
		Name:   FlagAWSDebug,
		Usage:  "Enable verbose debug output for AWS integration.",
		EnvVar: "EMPIRE_AWS_DEBUG",
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
	cli.StringFlag{
		Name:   FlagAllowedCommands,
		Value:  "any",
		Usage:  "Specifies what commands are allowed when using `emp run`. Can be `any`, or `procfile`.",
		EnvVar: "EMPIRE_ALLOWED_COMMANDS",
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

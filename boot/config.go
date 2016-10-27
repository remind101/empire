package boot

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config represents a config object that's provided to build an Empire
// instance.
type Config struct {
	// Environment is used to signify what "environment" this Empire
	// instance serves (e.g. production, staging, etc).
	//
	// This is used internally to:
	//
	// 1. Prefix CloudFormation stacks to prevent conflicts across different
	// Empire instances in the same AWS account.
	// 2. Add an `environment` tag to all AWS provisioned resources.
	Environment *string `toml:"environment,omitempty"`

	// If true, messages will be required for empire actions that emit
	// events.
	MessagesRequired *bool `toml:"messages_required,omitempty"`

	// Specifies what commands are allowed when using `emp run`. Can be
	// `any`, or `procfile`.
	AllowedCommands *string `toml:"allowed_commands,omitempty"`

	// If true, attached runs will be shown in `emp ps` output.
	ShowAttachedProcesses *bool `toml:"show_attached_processes,omitempty"`

	// Options for configuring the database.
	DB struct {
		// Postgres connection string to use to connect to Postgres.
		URL *string `toml:"url,omitempty"`

		// When true, the database will be automatically migrated to the
		// latest schema before starting Empire.
		NoAutoMigrate *bool `toml:"auto_migrate,omitempty"`
	} `toml:"db"`

	// Options for configuring the HTTP server.
	Server struct {
		URL *string `toml:"url,omitempty"`

		// The port to run the server on. If not provided, defaults to
		// 8080.
		Port *string `toml:"port,omitempty"`

		// Options for configuring the backend used for authenticating
		// users.
		Auth struct {
			// The name of the authentication backend to use.
			// Available options are: `fake`, `github`, or `saml`.
			Backend *string `toml:"backend,omitempty"`

			// The secret used to sign access tokens.
			Secret *string `toml:"secret,omitempty"`

			// Options for configuring the SAML authentication
			// backend.
			SAML struct {
				// Metadata is the Service Provider metadata
				// provided by the Identity Provider. This can
				// either be the raw XML content, an HTTP url to
				// download from, or the path to a file on disk.
				Metadata *string `toml:"metadata,omitempty"`

				// An RSA key to use to sign SAML requests.
				// This can either be the raw content, or the
				// path to a file on disk.
				Key *string `toml:"key,omitempty"`

				// The public cert for the private key.
				// Currently unused.
				Cert *string `toml:"cert,omitempty"`
			} `toml:"saml"`

			// Options for configuring the GitHub authentication
			// backend.
			GitHub struct {
				APIURL *string `toml:"api_url,omitempty"`

				// A GitHub OAuth client id.
				ClientID *string `toml:"client_id,omitempty"`

				// A GitHub OAuth client secret.
				ClientSecret *string `toml:"client_secret,omitempty"`

				// When provided, users will be checked to
				// ensure that they're a member of this
				// organization after authenticating.
				Organization *string `toml:"organization,omitempty"`

				// When provided, users will be checked to
				// ensure that they're a member of this team
				// after authenticating.
				TeamID *string `toml:"team_id,omitempty"`
			} `toml:"github"`
		} `toml:"auth"`

		// Options for configuring the GitHub webhooks integration.
		GitHub struct {
			// Shared secret between GitHub and Empire to validate
			// requests.
			Secret *string `toml:"secret,omitempty"`

			// Options for the GitHub Deployments integration.
			Deployments struct {
				// If provided, only github deployments to the
				// specified environments will be handled.
				Environments *[]string `toml:"environments,omitempty"`

				// Determines how the Docker image to deploy is
				// determined when a GitHub Deployment event is
				// received. Possible options are `template` and
				// `conveyor`.
				ImageBuilder *string `toml:"image_builder,omitempty"`

				// A Go text/template that will be used to
				// determine the docker image to deploy. This
				// flag is only used ImageBuilder is set to
				// `template`.
				ImageTemplate *string `toml:"image_template,omitempty"`

				// If enabled, logs from deployments triggered
				// via GitHub deployments will be sent to this
				// tugboat instance.
				Tugboat *bool `toml:"tugboat,omitempty"`
			} `toml:"deployments"`
		} `toml:"github"`
	} `toml:"server"`

	// Options for configuring the scheduler.
	Scheduler struct {
		// The backend to use to run applications. Currently, the only
		// supported value is `cloudformation`.
		Backend *string `toml:"backend"`

		// Options for the CloudFormation backend.
		CloudFormation struct {
			// The ID of the VPC that apps will run in. This is
			// required when ALB's are used.
			VpcID *string `toml:"vpc_id,omitempty"`

			// The ECS cluster to create ECS services within.
			ECSCluster *string `toml:"ecs_cluster,omitempty"`

			// The name of an IAM role that gives AWS access to
			// register instances with ELB.
			ECSServiceRole *string `toml:"ecs_service_role,omitempty"`

			// An ELB security group to assign to private load
			// balancers.
			ELBPrivateSecurityGroup *string `toml:"elb_private_security_group,omitempty"`

			// EC2 subnets to assign to private load balancers.
			EC2PrivateSubnets *[]string `toml:"ec2_private_subnets,omitempty"`

			// EC2 subnets to assign to public load balancers.
			EC2PublicSubnets *[]string `toml:"ec2_public_subnets,omitempty"`

			// The Zone ID of a Route53 internal hosted zone to
			// create CNAME and ALIAS records for private load
			// balancers.
			Route53InternalHostedZoneID *string `toml:"route53_internal_hosted_zone_id,omitempty"`

			// An ELB security group to assign to public load
			// balancers.
			ELBPublicSecurityGroup *string `toml:"elb_public_security_group,omitempty"`

			// The name of an S3 bucket where CloudFormation
			// templates will be uploaded.
			TemplateBucket *string `toml:"template_bucket,omitempty"`

			ECSLogDriver *string            `toml:"ecs_log_driver,omitempty"`
			ECSLogOpts   *map[string]string `toml:"ecs_log_opts,omitempty"`
		} `toml:"cloudformation"`
	} `toml:"scheduler"`

	// Options configuring the CloudFormation custom resource provisioner.
	CloudFormationCustomResources struct {
		// The ARN of an SNS topic used to create custom
		// resources.
		Topic *string `toml:"topic,omitempty"`

		// The Queue URL of an SQS queue that us subscribed to
		// the CustomResourcesTopic. The Custom Resource
		// provisioner will poll this queue for custom resources
		// to provision.
		Queue *string `toml:"queue,omitempty"`
	} `toml:"cloudformation_customresources"`

	// Options for configuring the Docker Daemon that Empire connects to.
	Docker struct {
		// Equivalent to the DOCKER_HOST env var, this is the endpoint
		// for the Docker client to connect to.
		Host *string `toml:"host,omitempty"`

		// Path to a directory containing TLS certificates to use when
		// connecting to the Docker Daemon over TLS. If provided, TLS
		// will be used.
		CertPath *string `toml:"certpath,omitempty"`

		// Path to a .dockercfg formatted file container Docker
		// credentials for pulling images from private repositories.
		DockerCfg *string `toml:"auth,omitempty"`
	} `toml:"docker"`

	// Options for Event forwarding.
	Events struct {
		// The backend implementation to use to send event notifactions.
		Backend *string `toml:"backend,omitempty"`

		// Options for configuring the SNS backend.
		SNS struct {
			// The SNS topic to publish events to.
			Topic *string `toml:"topic,omitempty"`
		} `toml:"sns"`
	} `toml:"events"`

	// Options for configuring Run logs.
	RunLogs struct {
		// The backend implementation to use to record the logs from
		// interactive runs. Currently supports `cloudwatch` and
		// `stdout`. Defaults to stdout.
		Backend *string `toml:"backend,omitempty"`

		// Options for the CloudWatch backend.
		CloudWatch struct {
			// This is the log group that CloudWatch log streams
			// will be created in.
			LogGroup *string `toml:"cloudwatch"`
		} `toml:"cloudwatch"`
	} `toml:"run_logs"`

	// Options for configuring where internal Empire stats and metrics go.
	Stats struct {
		// The backend to use to forward stats. Can be `statsd` or
		// `dogstatsd`.
		Backend *string `toml:"backend,omitempty"`

		// Options for the statsd backend.
		Statsd struct {
			Addr *string `toml:"addr,omitempty"`
		} `toml:"statsd"`

		// Options for the dogstatsd backend.
		DogStatsd struct {
			Addr *string `toml:"addr,omitempty"`
		} `toml:"dogstatsd"`
	} `toml:"stats"`

	// Options for configuring where errors are reported.
	ErrorReporter struct {
		// The error reporter to use. Can be `honeybadger`. If left
		// blank, errors will be logged.
		Backend *string `toml:"backend,omitempty"`

		// Options for configuring the Honeybadger backend.
		Honeybadger struct {
			// Honeybadger API key.
			ApiKey *string `toml:"api_key,omitempty"`

			// The Honeybadger environment to report this as.
			Environment *string `toml:"environment,omitempty"`
		} `toml:"honeybadger"`
	} `toml:"reporter"`

	Tugboat struct {
		URL *string `toml:"url,omitempty"`
	} `toml:"tugboat"`

	Conveyor struct {
		URL *string `toml:"url,omitempty"`
	} `toml:"conveyor"`

	AWS struct {
		Debug *bool `toml:"debug"`
	} `toml:"aws"`
}

// ParseConfig parses a Config object from the io.Reader.
func ParseConfig(r io.Reader) (*Config, error) {
	var config Config
	if _, err := toml.DecodeReader(r, &config); err != nil {
		return nil, fmt.Errorf("unable to parse config: %v", err)
	}
	return &config, nil
}

// ValidateProductionConfig sanity checks the config, and returns nil when the
// config is valid, otherwise it returns a ValidationResult with information
// about what is invalid.
func ValidateProductionConfig(c *Config) *ValidationResult {
	r := newValidationResult(c)

	r.Requires("server.auth.secret")

	// Validate server auth backend configuration.
	if backend := c.Server.Auth.Backend; backend != nil {
		switch *backend {
		case "fake":
		case "github":
			r.Requires("server.auth.github.client_id")
			r.Requires("server.auth.github.client_secret")
		case "saml":
			r.Requires("server.url")
			r.Requires("server.auth.saml.metadata")
			r.Requires("server.auth.saml.key")
			r.Requires("server.auth.saml.cert")
		default:
			r.UnknownValue("server.auth.backend", []string{"fake", "github", "saml"})
		}
	}

	// Validate scheduler config
	if backend := c.Scheduler.Backend; backend != nil {
		switch *backend {
		case "cloudformation":
			r.Requires("cloudformation_customresources.topic")
			r.Requires("scheduler.cloudformation.vpc_id")
			r.Requires("scheduler.cloudformation.template_bucket")
			r.Requires("scheduler.cloudformation.elb_private_security_group")
			r.Requires("scheduler.cloudformation.ec2_private_subnets")
			r.Requires("scheduler.cloudformation.elb_public_security_group")
			r.Requires("scheduler.cloudformation.ec2_public_subnets")
			r.Requires("scheduler.cloudformation.ecs_cluster")
			r.Requires("scheduler.cloudformation.ecs_service_role")
			r.Requires("scheduler.cloudformation.route53_internal_hosted_zone_id")
		default:
			r.UnknownValue("scheduler.backend", []string{"cloudformation"})
		}
	}

	// If there are no errors, return nil.
	if len(r.Errors) == 0 {
		return nil
	}

	return r
}

// ValidationResult is returned when validation the Config.
type ValidationResult struct {
	Errors []error

	// Maps a toml key to a value
	fields map[string]reflect.Value
}

func newValidationResult(config *Config) *ValidationResult {
	return &ValidationResult{fields: compileTomlFieldValues(config)}
}

func (r *ValidationResult) Error() string {
	var errors []string
	for _, err := range r.Errors {
		errors = append(errors, err.Error())
	}
	return fmt.Sprintf("%d errors:\n%s", len(r.Errors), strings.Join(errors, "\n"))
}

func (r *ValidationResult) String() string {
	return r.Error()
}

func (r *ValidationResult) Requires(path string) {
	v, ok := r.fields[path]
	if !ok {
		panic(fmt.Errorf("unknown path: %s", path))
	}
	if v.IsNil() {
		r.ValueRequired(path)
	}
}

func (r *ValidationResult) ValueRequired(path string) ValueError {
	return r.valueError(path, errors.New("missing"))
}

func (r *ValidationResult) UnknownValue(path string, available []string) ValueError {
	return r.valueError(path, fmt.Errorf("valid values are: %v", available))
}

func (r *ValidationResult) AddError(err error) {
	r.Errors = append(r.Errors, err)
}

func (r *ValidationResult) valueError(path string, err error) ValueError {
	err2 := valueError(path, err)
	r.AddError(err2)
	return err2
}

// ValueError is returned when the a required value is missing from the
// config.
type ValueError struct {
	// Name of the config option.
	Name string

	// Path to the config option (e.g. server.auth.saml)
	Path string

	Err error
}

func valueError(path string, err error) ValueError {
	parts := strings.Split(path, ".")
	return ValueError{
		Name: parts[len(parts)-1],
		Path: strings.Join(parts[:len(parts)-1], "."),
		Err:  err,
	}
}

// Error implements the error interface.
func (e ValueError) Error() string {
	return fmt.Sprintf("%s.%s: %v", e.Path, e.Name, e.Err)
}

func String(v string) *string {
	return &v
}

func StringSlice(v []string) *[]string {
	return &v
}

func compileTomlFieldValues(config *Config) map[string]reflect.Value {
	fields := make(map[string]reflect.Value)
	var f func(v reflect.Value, path []string)
	f = func(v reflect.Value, path []string) {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v.Kind() != reflect.Struct {
			panic(fmt.Sprintf("unreachable: was %s", v.Kind()))
		}

		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			tag := strings.Split(t.Field(i).Tag.Get("toml"), ",")[0]
			if field.Kind() == reflect.Struct {
				f(field, append(path, tag))
			} else {
				fields[strings.Join(append(path, tag), ".")] = field
			}
		}
	}
	f(reflect.ValueOf(config), []string{})
	return fields
}

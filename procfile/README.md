# Procfile

This is a Go library for parsing the Procfile format.

## Formats

### Standard

The standard Procfile format is what you're probably most familiar with, which maps a process name to the command to run. An example of a standard Procfile might look like:

```yaml
web: ./bin/web
worker: ./bin/worker
```

The standard Procfile format is specified in https://devcenter.heroku.com/articles/procfile.

### Extended

The extended Procfile format is Empire specific and implements a subset of the attributes defined in the [docker-compose.yml](https://docs.docker.com/compose/yml/) format. The extended Procfile format gives you more control, and allows you to configure additional settings. An example of an extended Procfile might look like:

```yaml
web:
  command: ./bin/web
scheduled-job:
  command: ./bin/scheduled-job
  cron: '0/2 * * * ? *'
```

#### Attributes

**Command**

Specifies the command that should be run when executing this process.

```yaml
command: ./bin/web
```

**Cron**

When provided, signifies that the process is a scheduled process. The value should be a valid cron expression.

```yaml
cron: * * * * * * // Run once every minute
```

**Noservice**

When provided, signifies that the process is an "operational" one off command. These processes will not get any AWS resources attached to them.

This can be used to alias a common command, or by enforcing whitelisting of commands for `emp run`.

```yaml
noservice: true
```

**Ports**

This allows you to define what ports to expose, and what protocol to expose them with. This works similarly to the `ports:` attribute in docker-compose.yml.

```yaml
ports:
  # Map port 80 on the container, as port 80 on the load balancer, using the default protocol.
  - "80"
  # Map port 8080 on the container, as port 80 on the load balancer, using the default protocol.
  - "80:8080"
  # Map port 5678 on the container, as port 5678 on the load balancer, using the tcp protocol.
  - "5678":
      protocol: "tcp"
```

**Environment**

This allows you to set process specific environment variables. If these are set with `emp set`, the value within the Procfile will take precendence.

```yaml
environment:
  EMPIRE_X_LOAD_BALANCER_TYPE: "alb"
```

Supported environment variables that can either be set via `emp set` for the whole application or
inside the `Procfile` for a specific process.

Name | Default value | Available options | Description
-----|---------------|-------------------|------------
`EMPIRE_X_LOAD_BALANCER_TYPE` | `elb` | `alb`, `elb`| Determines whether you will use an ALB or ELB
`EMPIRE_X_EXPOSURE` | `private` | `private`, `public` | Sets whether your ALB or ELB will be public (internet-facing) or private (internal), the default is private, however if you have used the deprecated `domain-add` command then the load balancer will become public. **If you change this setting, the load balancer will be recreated as soon as you deploy**
`EMPIRE_X_TASK_DEFINITION_TYPE` | not set | `custom` | Determines whether we use the Custom::ECSTaskDefinition (better explanation needed)
`EMPIRE_X_TASK_ROLE_ARN` | not set | any IAM role ARN | Sets the IAM role for that app/process. **Your ECS cluster MUST have Task Role support enabled before this can work!**

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

**Run**

When provided, signifies that the process is an "operational" one off command. These processes will not get any AWS resources attached to them.

This can be used to alias a common command, or by enforcing whitelisting of commands for `emp run`.

```yaml
run: true
```

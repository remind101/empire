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

The extended Procfile format is Empire specific and implements a subset of the attributes defined in the [docker-compose.yml](https://docs.docker.com/compose/yml/) format. The extended Procfile format gives you more control, and allows you to configure things like health checks for the individual processes. An example of an extended Procfile might look like:

```yaml
web:
  command: ./bin/web
  health_checks:
    - type: tcp
worker:
  command: ./bin/worker
```

#### Attributes

**Command**

Specifies the command that should be run when executing this process 

```yaml
command: ./bin/web
```

**Health Checks**

Allows you to specify an array of health checks that should be performed against the process.

```yaml
health_checks:
  - type: tcp
  - type: http
    path: /health
    interval: 30
    timeout: 30
```

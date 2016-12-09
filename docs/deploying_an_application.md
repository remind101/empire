# Deploying an Application

Before using Empire successfully it's important to understand its design and architecture. It's suggested you first read the [Features & Design Goals document](http://empire.readthedocs.org/en/latest/features_and_design_goals/).

There are a few key things which your applications, and overall architecture, will need to keep in mind in order to be successful with Empire.  Below is an exhaustive set of criteria your applications must meet in order to work with Empire.  It's not that long and by no means locks you in to Empire.  Structuring your application in this way provides many benefits, only one of which is being able to be managed by Empire.

For the impatient, there is one absolute:

- You must create a `Procfile` which defines how to run your application

To see an example application, you can look at [remind101/acme-inc].

## Procfile

Empire was modeled after the Heroku API. As such, many concepts and commands are similar. One concept which has bled over and which is a requirement is that of a [Procfile][procfile]. For your application to actually work, you must include a Procfile in the WORKDIR for your application.

Empire supports two Procfile formats; `standard` and `extended`. The `standard` format is probably what you're used to if you've used Heroku before and simply maps a process name to a command to run:

```yaml
worker: celery -A tasks worker --loglevel=info
```

The `extended` format is Empire specific and allows you to define additional configuration for processes, like exposure settings (http/https/tcp/ssl), scheduled tasks, and more. An example of the Procfile above in the `extended` format would simply be:

```yaml
worker:
  command: celery -A tasks worker --loglevel=info
```

The extended Procfile format is documented [here][extended-procfile].

Whichever format you use, the file would be named `Procfile`, and live at the directory root for your application.

```console
$ tree .
.
├── Dockerfile
├── Procfile
├── myapplication/

```

## Empire application types

Empire treats "web" and "non-web" services different.  Here, a "web" process is defined as anything which needs to expose a port.  The way to differentiate the two types of processes is easy.

### Non-web processes

Let's start with non-web processes first since they're much simpler.  If your service doesn't expose a port it doesn't need to be discovered by any other system.  With that, all you need to do it deploy your application and let Empire manage it. The act of discovering *other* services is up to you (database, caches, etc.)

### Web processes

#### Extended Procfile

When using the extended Procfile format, any process defined in the Procfile can list a set of ports it listens on, and get a load balancer attached to it. Ports can be defined using the `ports` key, similar to docker-compose.yml:

```yaml
nginx:
  command: nginx
  ports:
    - "80:8080":
        protocol: "http"
```

Here, "80:8080" means "expose port 8080 of the container, as port 80 on the load balancer". The application would then bind to port 8080.

When ports are defined in the process, Empire does a few things:

* Empire creates an ALIAS record for `<process>.<app>` inside the internal route53 hosted zone. This ALIAS record targets the ELB (or ALB) for the process.
* Empire creates and ELB (or ALB) for the process. As you scale the process up, instances running your process are placed into the ELB by ECS. Likewise, as you scale your process down, instances are removed from the ELB.
* When using ELB, Empire will automatically create and manage an "instance port" which maps a port on the EC2 instance to a port in the container. When using ALB, dynamic port mapping is used.

In the example above, if our application was named "router", then routing within the VPC would be handled like this:

```
     http://nginx.router/
              +
              |
              |
              v
             ELB: port 80
              +
              |
              |
              v
            Minion: port [9000-10000]
              +
              |
              |
              v
          Container: port 8080
```

#### Standard Procfile

When using the standard Procfile, you cannot define ports like you can with the extended Procfile. Instead, Empire treats processes called `web` specially. If a `web` process is defined, it is essentially equivalent to the following extended Procfile:

```yaml
web:
  command: ./bin/web
  environment:
    PORT: "8080"
  ports:
    - "80:8080":
        protocol: "http"
```

If an SSL cert is provided on the application, then an https listener is also added:

```yaml
web:
  command: ./bin/web
  environment:
    PORT: "8080"
  ports:
    - "80:8080":
        protocol: "http"
    - "443:8080":
        protocol: "https"
```

For backwards compatibility reasons, Empire will also create a CNAME for `<app>.<zone>` that points to the ELB/ALB. It's recommended that you use the more specific `<process>.<app>.<zone>` ALIAS record instead.

It's also recommended that your application binds to the `$PORT` environment variable, rather than specifically to port 8080.

#### ELB vs ALB

**NOTE:** This feature is currently experimental, and requires the CloudFormation backend.

By default, processes that have ports defined will get an ELB attached. If you'd rather use an ALB (Application Load Balancer) you can set the `EMPIRE_X_LOAD_BALANCER_TYPE` environment variable to `alb`:

```
emp set EMPIRE_X_LOAD_BALANCER_TYPE=alb
```

When using an ALB, ECS can run multiple instances of a "web" process on the same host, thanks to dynamic port mapping. This is currently not possible when using ELB.

ALB has a number of advantages and disadvantages detailed below:

Feature | ELB | ALB
--------|-----|-----
http/2 | no | yes (only from `client -> ALB`. `ALB -> backend` is downgraded to http/1.1)
websockets | no | yes
tcp load balancing | yes | no
tcp+ssl load balancing | yes | no
dynamic port mapping | no | yes

### Scheduled processes

**NOTE:** This feature is currently experimental, and requires the CloudFormation backend.

Processes that should run at scheduled times can be configured in the extended procfile by setting a `cron` expression. The builtin support for running scheduled tasks has multiple advantages over using a process that runs cron inside the container:

1. Scheduled tasks will not be killed during `emp deploy` or `emp restart`. If you have a long running task (e.g. a couple hours), those tasks will continue to run until completion.
2. Memory and CPU constraints can be controlled on a per process basis, just like any other process in the Procfile.
3. Scheduled tasks will show up in `emp ps` while they're running.

To use scheduled processes, simply add a `cron: ` key when using the extended Procfile format:

```yaml
web:
  command: ./bin/web
scheduled-job:
  command: ./bin/scheduled-job
  cron: '0/2 * * * ? *' # Run once every 2 minutes
```

Like other non-web processes, scheduled processes are disabled by default. To enable a scheduled job, simply scale it up:

```console
$ emp scale scheduled-job=1
```

If you want to run more than 1 instance of the process when the cron expression triggers, you can scale it to a value greater than 1:

```console
$ emp scale scheduled-job=5 # Run 5 of these every minute
```

To disable a scheduled job, simply scale it back down to 0:

```console
$ emp scale scheduled-job=0
```

When the cron expression triggers, and a task is started, you'll be able to see it when using `emp ps`:

```console
$ emp ps
v54.scheduled-job.9b649d34-b4f5-4fb7-bfe2-889d80dbd3c9  1X  RUNNING  11s  "./bin/scheduled-job"
v54.web.fd130482-675f-4611-a599-eb0da1879a10            1X  RUNNING   9m  "./bin/web"
```

Refer to http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/ScheduledEvents.html for details on the cron expression syntax.

## Run only processes

When using `emp run`, if the command you provide matches a process within the Procfile, it will invoke the command defined inside the process. For example, you might define a `migrate` process inside the Procfile, which users would use to run migrations:

```console
#!/bin/bash

dir=${1:-up}
exec bundle exec rake db:migrate:$dir
```

```yaml
migrate:
  command: ./bin/migrate
```

Now, users can invoke the `migrate` process like so:

```console
$ emp run migrate up
$ emp run migrate down
```

You can add the `noservice: true` flag to tell Empire to not create any AWS resources for the process, for extra re-assurance that the process won't be scaled up.

```yaml
migrate:
  command: ./bin/migrate
  noservice: true
```

## Environment variables

TODO

[procfile]: https://devcenter.heroku.com/articles/procfile
[extended-procfile]: https://github.com/remind101/empire/tree/master/procfile
[remind101/acme-inc]: https://github.com/remind101/acme-inc

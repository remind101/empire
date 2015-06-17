# Empire :: Overview

1. [Overview](./index.md)
2. [Installing](./installing.md)
3. [Using](./using.md)
4. [Administering](./administering.md) **TODO**
5. [Troubleshooting](./troubleshooting.md) **TODO**
6. [Roadmap](./roadmap.md) **TODO**

Empire is an open source, self-hosted PaaS that makes deployment and management
of [dockerized][dockerized] [12-factor apps][12factor] easy. Empire's goals are
described below.

### Ease of Use

We're big fans of Twelve-Factor apps and Heroku. When we built Empire we strived
to make it as similar to the Heroku experience as we could.

### Simple Design

We want to make managing Empire as easy as possible. This means keeping its
dependencies to a minimum. Of those dependencies, as many as possible are
available as managed services.

As of now, Empire's two major dependencies are:

1. [Amazon EC2 Container Service ][ecs]
2. A PostgreSQL database. We use [Amazon RDS][amazonrds]

Empire itself is also very simple. Once an application has been handed to ECS,
Empire takes a hands off approach and lets ECS manage it entirely. It doesn't
attempt to modify the application unless someone asks it to (i.e., scaling up
or down, modifying environment variables, or deploying new releases).

### Failure Resilient

We want to make sure that in the case that we lose a container, a host, or even
multiple hosts the system will recover. By using Amazon services and features,
we are able to achieve this.

* A container dies? No problem; ECS will bring it back up. In the meantime
  traffic for that app is routed to other containers (provided you have scaled
  your app to more than one process) via [ELB][elb].

* A host dies? No problem; ECS will reschedule containers while
  [Auto Scaling][autoscaling] brings up a new host in its place. Again, ELB
  will make sure that the app stays up, provided there are multiple copies of it
  running.

* Multiple hosts dies? See above - though it may take a longer to recover.

### Better Security Controls

We have full of control over the hosts that our containers run on. That means we
can control things like filesystem encryption. We also have direct access to
[Amazon Identity and Access Management (IAM)][iam].

### Better Visibility

Since we control everything down to the instance OS, we can gather stats at
every layer in the stack. We can tell if we have a 'noisy neighbor' by watching
for stolen CPU time. We can install any type of monitoring, logging, or metrics
software we want on the host side.

# Who should use Empire?

Empire isn't for everyone. While it aims to be simple to run, there are still
some operational costs involved in managing the service. Empire is well suited
for companies that have a microservices/SOA type architecture, are looking for
the ease of use of a Heroku like system, but need additional control over
security and the systems their applications run on.

# What does Empire not do?

There are a few things Empire does not do currently - largely due to the fact
that these add quite a bit of complexity to Empire. We feel that different users
will want the flexibility to come up with their own solutions.

In the future, as we iterate on how we handle these things at Remind, we'll
share how we handle them, or open source them.

## Logging and Metrics

Internally at Remind we use a combination of [logspout][logspout], [Heka][heka],
and [Sumo Logic][sumologic] to aggregate logs from both containers and the
container host. We use [collectd][collectd] and [Librato][librato] to gather
stats from both the containers and the container host as well.

This solution works for us, but we don't feel that it's sufficiently generic
or simple enough to make it a part of the core Empire project itself.

## Creating and Serving Docker Images

Empire deploys docker images, so you'll need some place to host those images and
something to build them.

At Remind we host our docker images in a private repository on the official
[Docker Hub Registry][dockerhub]. We build images via CircleCI on each push into
each repository.

## Attached Resources

[dockerized]: https://docs.docker.com/userguide/dockerizing/
[12factor]: http://12factor.net/
[amazonrds]: http://aws.amazon.com/rds/postgresql/
[ecs]: http://aws.amazon.com/ecs/
[elb]: http://aws.amazon.com/elasticloadbalancing/
[autoscaling]: http://aws.amazon.com/autoscaling/
[iam]: http://aws.amazon.com/iam/
[logspout]: https://github.com/gliderlabs/logspout
[heka]: http://hekad.readthedocs.org/en/latest/
[sumologic]: https://www.sumologic.com/
[collectd]: https://collectd.org/
[librato]: https://www.librato.com/
[dockerhub]: https://registry.hub.docker.com/
[circleci]: https://circleci.com/
# Empire :: Overview

1. [Overview](./README.md)
2. [Installing](./installing.md)
3. [Using](./using.md)
4. [Administering](./administering.md) **TODO**
5. [Troubleshooting](./troubleshooting.md) **TODO**
6. [Roadmap](./roadmap.md) **TODO**

Empire is an open source, self-hosted PaaS that intends to make the deployment and
management of 12-factor apps easy. It's goals are:

### Ease of Use

We're big fans of Twelve-Factor apps and Heroku, and so when we built Empire we sought
to try and make it as similar to Heroku as we could.

### Simple in Design

We wanted to make the management of Empire itself as easy as possible. This meant
keeping it's dependencies to a small number, and trying to make as many of those be
managed services.

As of now, the two major dependencies are:

- Amazon's EC2 Container Service (ECS)
- A postgres database (which we tend to use RDS for, simplifying management even more)

As well, Empire itself is meant to be simple. Once an application has been handed to
ECS, Empire lets ECS manage it entirely, taking a hands off approach. It doesn't
attempt to modify the application unless someone asks it to (ie: scaling up, modifying
environment variables, or deploying a new release).

### Failure Resiliency

We wanted to make sure that in the case that we lost a container, a host, or even
multiple hosts the system would recover. Using Amazon services & features, we're able
to achieve this.

- A container dies? No problem, ECS will bring it back up. In the mean time traffic for
  that app will be routed to other containers (provided you are running multiple) via
  ELB.
- A host dies? No problem, ECS will reschedule containers while Autoscaling will
  bring up a new host in it's place. Again ELB will make sure that the app stays up,
  provided there are multiple copies of it running.
- Multiple hosts dies? See above - though obviously it might take a little longer to
  recover.

### Better security controls

Because we control the hosts that our containers run on, we can control things like
having the filesystem be encrypted. We also have direct access to things like
Amazon Security Groups & Identity and Access Management (IAM).

### Better visibility

Again, because we control everything down to the instance OS, we can gather stats
all the way down. We can tell if we have a 'noisy neighbor' by watching for things
like stolen CPU time. Also we can implement just about any piece of software we want
on the host side.

# Who should use Empire?

Empire isn't for everyone. While it aims to be operationally simple to run, there are
still some operational costs involved in managing the service. Mostly Empire is
meant for companies who run many services in a microservices/SOA type architecture,
are looking for the ease of use of a Heroku like system, but need additional control
over things like security and the systems their applications run on.


# What doesn't Empire do?

So what doesn't Empire do for you that Heroku does? There are a few things - largely
due to the fact that these add quite a bit of complexity to Empire, and we felt that
different users would want the flexibility to come up with their own solutions.

In the future, as we iterate on how we handle these things at Remind, we'll share how
we handle them, or even just open source those projects as well.

## Logging & Metrics

Internally @ Remind we use a combination of logspout, heka, and sumologic to aggregate
the logs from both containers and the host itself. We use collectd and librato to
gather stats from both the containers and the host as well.

In general this solution works for us, but we don't feel that it's sufficiently generic
or simple enough to make it a part of the core Empire project itself.

## Creating & Serving Docker Images

Empire deploys docker images, so you'll need some place to host those images and
something to build them.

At Remind we host our docker images in a private repository on the main Docker Hub
site. We build images via circle-ci on each push into each repository.

## Attached Resources

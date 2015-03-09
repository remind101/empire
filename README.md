# Empire [![Build Status](https://magnum.travis-ci.com/remind101/empire.svg?token=Uca1q7og621AjpUDJUEC&branch=master)](https://magnum.travis-ci.com/remind101/empire) [![Docker Repository on Quay.io](https://quay.io/repository/remind/empire/status?token=8ea56dcf-fc6f-405c-a281-1896994ef8c2 "Docker Repository on Quay.io")](https://quay.io/repository/remind/empire)

Empire is Remind's next generation PaaS, which we will eventually use to migrate
away from Heroku.

## Usage

Running the server:

```console
$ empire server -h
NAME:
   server - Run the empire HTTP api

USAGE:
   command server [command options] [arguments...]

OPTIONS:
   --port '8080'                                        The port to run the server on
   --github.client.id                                   The client id for the GitHub OAuth application [$GITHUB_CLIENT_ID]
   --github.client.secret                               The client secret for the GitHub OAuth application [$GITHUB_CLIENT_SECRET]
   --github.organization                                The organization to allow access to [$GITHUB_ORGANIZATION]
   --docker.socket 'unix:///var/run/docker.sock'        The location of the docker api [$DOCKER_HOST]
   --docker.cert                                        If using TLS, a path to a certificate to use [$DOCKER_CERT_PATH]
   --docker.auth '/Users/$HOME/.dockercfg'              Path to a docker registry auth file (~/.dockercfg) [$DOCKER_AUTH_PATH]
   --fleet.api                                          The location of the fleet api [$FLEET_URL]
   --secret '<change this>'                             The secret used to sign access tokens
   --db 'postgres://localhost/empire?sslmode=disable'   SQL connection string for the database

```

## Heroku API compatibility

We are aiming to be compatible with heroku's API

You can use the `hk` cli with empire like this:

```console
HEROKU_API_URL=<empire_url> hk ...
```

### Supported commands

```console
hk apps
hk create <name>
hk env
hk get
hk set
hk scale
hk dynos
hk rollback
```

## Quickstart

```console
$ make install
$ vagrant up
# Wait for vagrant image to boot...
$ emp deploy quay.io/ejholmes/acme-inc:ec238137726b58285f8951802aed0184f915323668487b4919aff2671c0f9a02
$ emp apps
$ emp releases -a acme-inc
$ emp env -a acme-inc
$ emp rollback V1 -a acme-inc
$ emp releases -a acme-inc
```

## Components

**DISCLAIMER**: Empire is incredibly young and a lot of things will most likely
change as we try to productionize it.

Empire is a distributed system for deploying and running
[12factor][12factor] [Docker][docker] based
applications in a compute cluster. The following components are employed:

**[Quay][quay]** Quay is used to automatically build docker images when we push commits to GitHub.

**[Quayd][quayd]** Quayd is used to handle webhook events from Quay and create GitHub Commit Statuses as well as tag the resulting images with the git sha.

**[Etcd][etcd]** Used for service discovery and a general key/val store.

**[Fleet][fleet]** Used for process scheduling.

**[Postgres][postgres]** Used as a backend for empire app data.

**[Registrator][registrator]** Used to automatically register services with consul.

**[Shipr][shipr]** Shipr is used to handle GitHub Deployments and forward them to Empire.

**[hubot-deploy][hubotdeploy]** Hubot and the hubot-deploy script is used as our abstraction around deploying.

## Development

To get started, run:

```console
$ make bootstrap
```

To run the tests:

```console
$ godep go test ./...
```

**Caveats**

1. `emp login` won't work by default if you're running on a non-standard port.
   Once you emp login, you'll need to change the appropriate `machine` entry in
   your `~/.netrc` to include to port.

   ```
   machine 0.0.0.0:8080
   ```

## Tests

Unit tests live alongside each go file as `_test.go`.

There is also a `tests` directory that contains
integration and functional tests that tests the system
using the
**[heroku-go](https://github.com/bgentry/heroku-go)**
client and the **[hk
command](https://github.com/heroku/hk)**.

## How do I deploy to Empire?

The same way you would with Heroku, but easier:

1. Create a github repo.
2. Add a [Dockerfile](https://docs.docker.com/reference/builder/) to run your app. Include a line to copy the Procfile to the root of the container:

   ```
   ADD ./Procfile /
   ```

3. Deploy your service with marvin:

   ```
   marvin deploy my-service to staging
   ```

## Can I deploy with Git?

No.

## Architecture

Empire is heavily influenced by Heroku and the philosophies described in [The Twelve-Factor App][12factor], as well as similar projects such as [flynn][flynn] and [deis][deis]

### Phases

There are three phases during deployment:

1. **Build**: This phase happens after a git push to GitHub, which triggers a docker build. Once the image is built, it gets tagged with the git sha that triggered the build. This is in contrast to systems like heroku, where the build phase always happens during the deployment process. The primary advantage behind Empire's philosophy, is that once a git sha has been built, deployment is nearly instant.
2. **Release**: This phase happens when a developer triggers a deploy for a git sha via marvin. The git sha is resolved to a docker image, empire creates a "slug", then combines the slug and the latest config into a "release", which is then sent to the process manager to run on the cluster.
3. **Run**: The run phase happens inside the compute cluster. The init system will bring up the desired instance count inside the cluster.

[legion]: https://github.com/remind101/legion
[quay]: https://quay.io
[quayd]: https://github.com/remind101/quayd
[consul]: https://github.com/hashicorp/consul
[registrator]: https://github.com/progrium/registrator
[shipr]: https://github.com/remidn101/shipr
[hubotdeploy]: https://github.com/remidn101/hubot-deploy
[12factor]: http://12factor.net/
[docker]: https://www.docker.com/
[flynn]: https://flynn.io/
[deis]: http://deis.io/
[fleet]: https://github.com/coreos/fleet
[postgres]: http://www.postgresql.org/
[etcd]: https://github.com/coreos/etcd

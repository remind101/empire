# Empire [![Build Status](https://magnum.travis-ci.com/remind101/empire.svg?token=Uca1q7og621AjpUDJUEC&branch=master)](https://magnum.travis-ci.com/remind101/empire)

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
   --docker.socket 'unix:///var/run/docker.sock'        The location of the docker api [$DOCKER_HOST]
   --docker.registry                                    The docker registry to pull container images from [$DOCKER_HOST]
   --docker.cert                                        If using TLS, a path to a certificate to use [$DOCKER_CERT_PATH]
   --fleet.api 'http://127.0.0.1:49153'                 The location of the fleet api
   --db 'postgres://localhost/empire?sslmode=disable'   SQL connection string for the database

```

## Components

**DISCLAIMER**: Empire is incredibly young and a lot of things will most likely
change as we try to productionize it.

Empire is a distributed system for deploying and running
[12factor][12factor] [Docker][docker] based
applications in a compute cluster. The following components are employed:

**[Quay][quay]** Quay is used to automatically build docker images when we push commits to GitHub.

**[Quayd][quayd]** Quayd is used to handle webhook events from Quay and create GitHub Commit Statuses as well as tag the resulting images with the git sha.

**[Consul][consul]** Used for service discovery and a general key/val store.

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

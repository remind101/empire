# Empire

[![Install](https://s3.amazonaws.com/cloudformation-examples/cloudformation-launch-stack.png)](https://console.aws.amazon.com/cloudformation/home?region=us-east-1#cstack=sn%7Eempire%7Cturl%7Ehttps://s3.amazonaws.com/empirepaas/cloudformation.json)

[![readthedocs badge](https://readthedocs.org/projects/pip/badge/?version=latest)](http://empire.readthedocs.org/en/latest/) [![Circle CI](https://circleci.com/gh/remind101/empire.svg?style=svg)](https://circleci.com/gh/remind101/empire)

Empire is a control layer on top of [Amazon EC2 Container Service (ECS)][ecs] that provides a Heroku like workflow. It conforms to a subset of the [Heroku Platform API][heroku-api], which means you can use the same tools and processes that you use with Heroku, but with all the power of EC2 and [Docker][docker].

Empire is targeted at small to medium sized startups that are running a large number of microservices and need more flexibility than what Heroku provides. You can read the original blog post about why we built empire on the [Remind engineering blog](http://engineering.remind.com/introducing-empire/).

## Quickstart

To use Empire, you'll need to have an ECS cluster running. See the [quickstart guide][guide] for more information.

## Architecture

Empire aims to make it trivially easy to deploy a container based microservices architecture, without all of the complexities of managing systems like Mesos or Kubernetes. ECS takes care of much of that work, but Empire attempts to enhance the interface to ECS for deploying and maintaining applications, allowing you to deploy Docker images as easily as:

```console
$ emp deploy remind101/acme-inc:latest
```

### Heroku API compatibility

Empire supports a subset of the [Heroku Platform API][heroku-api], which means any tool that uses the Heroku API can probably be used with Empire, if the endpoint is supported.

As an example, you can use the `hk` CLI with Empire like this:

```console
$ HEROKU_API_URL=<empire_url> hk ...
```

However, the best user experience will be by using the [emp](https://github.com/remind101/emp) command, which is a fork of `hk` with Empire specific features.

### Routing

Empire's routing layer is backed by internal ELBs. Any application that specifies a web process will get an internal ELB attached to its associated ECS Service. When a new version of the app is deployed, ECS manages spinning up the new versions of the process, waiting for old connections to drain, then killing the old release.

When a new internal ELB is created, an associated CNAME record will be created in Route53 under the internal TLD, which means you can use DNS for service discovery. If we deploy an app named `feed` then it will be available at `http://feed` within the ECS cluster.

Apps default to only being exposed internally, unless you add a custom domain to them. Adding a custom domain will create a new external ELB for the ECS service.

### Deploying

Any tagged Docker image can be deployed to Empire as an app. Empire doesn't enforce how you tag your Docker images, but we recommend tagging the image with the git sha that it was built from, and deploying that. We have a tool for performing deployments called [Tugboat][tugboat] that supports deploying Docker images to Empire.

When you deploy a Docker image to Empire, it will extract a `Procfile` from the WORKDIR. Like Heroku, you can specify different process types that compose your service (e.g. `web` and `worker`), and scale them individually. Each process type in the Procfile maps directly to an ECS Service.

**Caveats**

Because `docker run` does not exec commands within a shell, commands specified within the Procfile will also not be exec'd within a shell. This means that you cannot specify environment variables in the Procfile. The following is not valid:

```
web: acme-inc server -port=$PORT
```

If you need to specify environment variables as part of the command, we recommend splitting out your Procfile commands into small bash shims instead:

```
web: ./bin/web
```

```bash
#!/bin/bash

set -e

exec acme-inc server -port=$PORT
```

## Tests

Unit tests live alongside each go file as `_test.go`.

There is also a `tests` directory that contains integration and functional tests that tests the system using the [heroku-go][heroku-go] client and the [emp][emp] command.

To get started, run:

```console
$ make bootstrap
```

The bootstrap command assumes you have a running postgres server. It will create a database called `empire`
using the postgres client connection defaults.

To run the tests:

```console
$ godep go test ./...
```

## Development

If you want to contribute to Empire, you may end up wanting to run a local instance against an ECS cluster. Doing this is relatively easy:

1. Ensure that you have the AWS CLI installed and configured.
2. Ensure that you accepted the terms and conditions for the official ECS AMI:

   https://aws.amazon.com/marketplace/ordering?productId=4ce33fd9-63ff-4f35-8d3a-939b641f1931&ref_=dtl_psb_continue&region=us-east-1

   Also check that the offical ECS AMI ID for US East matches with the one in [cloudformation.json](./docs/cloudformation.json): https://github.com/remind101/empire/blob/master/docs/cloudformation.json#L20

3. Run docker-machine and export the environment variables so Empire can connect:

   ```console
   $ docker-machine start default
   $ eval "$(docker-machine env default)"
   ```
4. Run the bootstrap script, which will create a cloudformation stack, ecs cluster and populate a .env file:

   ```console
   $ DEMOMODE=0 ./bin/bootstrap
   ```
5. Run Empire with [docker-compose](https://docs.docker.com/compose/):

   ```console
   $ docker-compose up
   ```

   **NOTE**: You might need to run this twice the first time you start it up, to give the postgres container time to initialize.
6. Install the emp CLI.

   ```console
   $ go get -u github.com/remind101/emp
   ```

Empire will be available at `http://$(docker-machine ip default):8080` and you can point the CLI there.

```console
$ export EMPIRE_API_URL=http://$(docker-machine ip default):8080
$ emp deploy remind101/acme-inc
```

## Community

We have a google group, [empire-dev][empire-dev], where you can ask questions and engage with the Empire community.

[ecs]: http://aws.amazon.com/ecs/
[docker]: https://github.com/docker/docker
[heroku-api]: https://devcenter.heroku.com/articles/platform-api-reference
[tugboat]: https://github.com/remind101/tugboat
[heroku-go]: https://github.com/bgentry/heroku-go
[hk]: https://github.com/heroku/hk
[emp]: https://github.com/remind101/emp
[guide]: http://empire.readthedocs.org/en/latest/
[empire-dev]: https://groups.google.com/forum/#!forum/empire-dev

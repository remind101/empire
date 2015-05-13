# Empire

Empire is a control layer on top of [Amazon Elastic Container Service (ECS)][ecs] that provides a Heroku like workflow. It conforms to a subset of the [Heroku Platform API][heroku-api], which means you can use the same tools and processes that you use with Heroku, but with all the power of EC2 and [Docker][docker].

Empire is targeted at small to medium sized startups that are running a large number of microservices and need more flexibility than what Heroku provides.

[![](https://s3.amazonaws.com/ejholmes.github.com/eyiq4.png)](https://www.youtube.com/watch?v=myGiBYTfn08&feature=youtu.be&VQ=HD720)

## Quickstart

To use Empire, you'll need to have an ECS cluster running. See the [quickstart guide][guide] for more information.

## Architecture

Empire aims to make it trivially easy to deploy a container based microservices architecture, without all of the complexities of managing systems like Mesos or Kubernetes. ECS takes care of much of that work, but lacks a convenient interface for performing common administrative tasks like deploying new versions, scaling and updating configuration. This is where Empire comes in, allowing you to deploy Docker images as easily as:

```console
$ emp deploy remind101/acme-inc:latest
```

### Heroku API compatibility

Empire supports a subset of the [Heroku Platform API][heroku-api], which means any tool that uses the Heroku API can probably be used with Empire, if the endpoint is supported.

As an example, you can use the `hk` CLI with Empire like this:

```console
$ HEROKU_API_URL=<empire_url> hk ...
```

In fact, the Empire CLI itself (`emp`), is just a wrapper around `hk`.

The following `hk` commands are supported: `apps`, `create`, `env`, `get`, `set`, `scale`, `dynos`, `rollback`, `releases`, `domains`.

Currently, Empire doesn't support setting the size of the dyno (e.g. `hk scale web=1:PX`), but something that we're planning to add in the future.

### Routing

Empire's routing layer is backed by internal ELB's. Any application that specifies a web process will get an internal ELB attached to it's associated ECS Service. When a new version of the app is deployed, ECS manages spinning up the new versions of the process, waiting for old connections to drain, then killing the old release.

When a new internal ELB is created, an associated CNAME record will be created in Route53 under the internal TLD, which means you can use DNS for service discovery. If we deploy an app named `feed` then it will be available at `http://feed` within the ECS cluster.

Apps default to only being exposed internally, unless you add a custom domain to them. Adding a custom domain will create a new external ELB for the ECS service.

### Deploying

Any tagged Docker image can be deployed to Empire as an app. Empire doesn't enforce how you tag your Docker images, but we recommend tagging the image with the git sha that it was built from, and deploying that. We have a tool for performing deployments called [Tugboat][tugboat] that supports deploying Docker images to empire.

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

1. `emp login` won't work by default if you're running on a non-standard port. Once you `emp login`, you'll need to change the appropriate `machine` entry in your `~/.netrc` to include the port:

   ```
   machine 0.0.0.0:8080
   ```

## Tests

Unit tests live alongside each go file as `_test.go`.

There is also a `tests` directory that contains integration and functional tests that tests the system using the [heroku-go][heroku-go] client and the [hk][hk] command.

## Development

If you want to contribute to empire, you maybe end up wanting to run a local instance against an ECS cluster. Doing this is relatively easy:

1. Ensure that you have the AWS CLI installed and configured.
2. Run boot2docker and export the environment variables so empire can connect:
   
   ```console
   $ boot2docker start
   $ $(boot2docker shellinit)
   ```
3. Run the bootstrap script, which will create a cloudformation stack, ecs cluster and populate a .env file:

   ```console
   $ ./bin/bootstrap
   ```
4. Build the empire binary and run the server:
   
   ```console
   $ cd empire
   $ make cmd
   $ export $(cat .env)
   $ ./build/empire server
   ```

Empire will be available at http://0.0.0.0:8080 and you can point the CLI there.

[ecs]: http://aws.amazon.com/ecs/
[docker]: https://github.com/docker/docker
[heroku-api]: https://devcenter.heroku.com/articles/platform-api-reference
[tugboat]: https://github.com/remind101/tugboat
[heroku-go]: https://github.com/bgentry/heroku-go
[hk]: https://github.com/heroku/hk
[guide]: ./docs/guide

# Empire

[![readthedocs badge](https://readthedocs.org/projects/pip/badge/?version=latest)](http://empire.readthedocs.org/en/latest/)
[![Circle CI](https://circleci.com/gh/remind101/empire.svg?style=shield)](https://circleci.com/gh/remind101/empire)
[![Slack Status](https://empire-slack.herokuapp.com/badge.svg)](https://empire-slack.herokuapp.com)

![Empire](empire.png)

Empire is a control layer on top of [Amazon EC2 Container Service (ECS)][ecs] that provides a Heroku like workflow. It conforms to a subset of the [Heroku Platform API][heroku-api], which means you can use the same tools and processes that you use with Heroku, but with all the power of EC2 and [Docker][docker].

Empire is targeted at small to medium sized startups that are running a large number of microservices and need more flexibility than what Heroku provides. You can read the original blog post about why we built empire on the [Remind engineering blog](http://engineering.remind.com/introducing-empire/).

## Quickstart

[![Install](https://s3.amazonaws.com/cloudformation-examples/cloudformation-launch-stack.png)](https://console.aws.amazon.com/cloudformation/home?region=us-east-1#cstack=sn%7Eempire%7Cturl%7Ehttps://s3.amazonaws.com/empirepaas/cloudformation.json)

To use Empire, you'll need to have an ECS cluster running. See the [quickstart guide][guide] for more information.

## Architecture

Empire aims to make it trivially easy to deploy a container based microservices architecture, without all of the complexities of managing systems like Mesos or Kubernetes. ECS takes care of much of that work, but Empire attempts to enhance the interface to ECS for deploying and maintaining applications, allowing you to deploy Docker images as easily as:

```console
$ emp deploy remind101/acme-inc:master
```

### Heroku API compatibility

Empire supports a subset of the [Heroku Platform API][heroku-api], which means any tool that uses the Heroku API can probably be used with Empire, if the endpoint is supported.

As an example, you can use the `hk` CLI with Empire like this:

```console
$ HEROKU_API_URL=<empire_url> hk ...
```

However, the best user experience will be by using the [emp](https://github.com/remind101/empire/tree/master/cmd/emp) command, which is a fork of `hk` with Empire specific features.

### Routing

Empire's routing layer is backed by internal ELBs. Any application that specifies a web process will get an internal ELB attached to its associated ECS Service. When a new version of the app is deployed, ECS manages spinning up the new versions of the process, waiting for old connections to drain, then killing the old release.

When a new internal ELB is created, an associated CNAME record will be created in Route53 under the internal TLD, which means you can use DNS for service discovery. If we deploy an app named `feed` then it will be available at `http://feed` within the ECS cluster.

Apps default to only being exposed internally, unless you add a custom domain to them. Adding a custom domain will create a new external ELB for the ECS service.

### Deploying

Any tagged Docker image can be deployed to Empire as an app. Empire doesn't enforce how you tag your Docker images, but we recommend tagging the image with the git sha that it was built from (any any immutable identifier), and deploying that.

When you deploy a Docker image to Empire, it will extract a `Procfile` from the WORKDIR. Like Heroku, you can specify different process types that compose your service (e.g. `web` and `worker`), and scale them individually. Each process type in the Procfile maps directly to an ECS Service.

## Contributing

Pull requests are more than welcome! For help with setting up a development environment, see [CONTRIBUTING.md](./CONTRIBUTING.md)

## Community

We have a google group, [empire-dev][empire-dev], where you can ask questions and engage with the Empire community.

You can also [join](https://empire-slack.herokuapp.com/) our Slack team for discussions and support.

[ecs]: http://aws.amazon.com/ecs/
[docker]: https://github.com/docker/docker
[heroku-api]: https://devcenter.heroku.com/articles/platform-api-reference
[tugboat]: https://github.com/remind101/tugboat
[heroku-go]: https://github.com/bgentry/heroku-go
[hk]: https://github.com/heroku/hk
[emp]: https://github.com/remind101/emp
[guide]: http://empire.readthedocs.org/en/latest/
[empire-dev]: https://groups.google.com/forum/#!forum/empire-dev

## Auth Flow

The current authentication model used by `emp login` relies on a [deprecated](https://developer.github.com/changes/2020-02-14-deprecating-oauth-auth-endpoint/) GitHub endpoint that is scheduled to be deactivated in November 2020.  Therefore both the client and the server need to be updated to support the [web authentication flow](https://developer.github.com/apps/building-oauth-apps/authorizing-oauth-apps/#web-application-flow)

The web flow works like this

1. The user runs a command like `emp web-login`
1. The client starts up a HTTP listener on a free local port
1. The client opens a browser window on the local machine to `$EMPIRE_API_URL/oauth/start?port=?????`
    * The port parameter specifies where the client is listening
1. The browser executes a GET against the URL
1. The Empire server sees the request and constructs an OAuth request URL that will hit the GitHub OAuth endpoint and returns it as a redirect
1. The browser makes the request to the GitHub auth endpoint, which shows the UI a request to authorize the application
    * If they've previously authorized it will just immediately grant the request
1. GitHub redirects the browser back to the redirect URL specified in the configuration, meaning back to the Empire server
1. The Empire server receives the browser request and can now perform the [code exchange](https://developer.github.com/apps/building-oauth-apps/authorizing-oauth-apps/#2-users-are-redirected-back-to-your-site-by-github) to turn the provided code into an actual authentication token
    * This is just like it would have received from the old endpoint.  However, it's not usable yet because it still isn't in the possession of the client, only the browser
1. The Empire server now redirects the browser back to `localhost` on the original port provided by the client
1. The client receives the token, but can't use it directly.  The Empire server expects it to be wrapped in a JSON Web Token that only the server can create.
1. The client can now make a request directly to the Empire server (its first in this sequence) providing the token and requesting a JSON Web Token in response
1. The client stores the received token just as it would have with the response to an `emp login` command
1. The client is authenticated    

In theory the Empire server could construct the JWT directly after the code exchange and push that directly to the client, but the abstraction doesn't really seem to easily support that flow

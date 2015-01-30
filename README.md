# Empire

Empire is Remind's next generation PaaS, which we will eventually use to migrate
away from Heroku.

## Components

**DISCLAIMER**: Empire is incredibly young and a lot of things will most likely
change as we try to productionize it.

Empire is a distributed system for deploying and running
[12factor][12factor] [Docker][docker] based
applications in a compute cluster. The following components are employed:

**[Centurion][centurion]** The controller API for deploying, scaling, running 1 off processes, etc.

**[Celsus][celsus]** Manages application configuration and versioning.

**[Sluggy][sluggy]** Generates "slugs" by extracting runnable process types from containers.

**[Phalanx][phalanx]** Manages the desired instance counts for runnable processes.

**[Ignitr][ignitr]** Our distributed init system.

**[Pantheon][pantheon]** Stores and exposes releases.

**[Quay][quay]** Quay is used to automatically build docker images when we push commits to GitHub.

**[Quayd][quayd]** Quayd is used to handle webhook events from Quay and create GitHub Commit Statuses as well as tag the resulting images with the git sha.

**[Consul][consul]** Used for service discovery and a general key/val store.

**[Registrator][registrator]** Used to automatically register services with consul.

**[Shipr][shipr]** Shipr is used to handle GitHub Deployments and forward them to Empire.

**[hubot-deploy][hubotdeploy]** Hubot and the hubot-deploy script is used as our abstraction around deploying.

## Development

This will improve over time, but currently, you need to clone all the repos into
your GOPATH then:

```console
$ fig up
```

## How to I deploy to Empire?

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

## Architecture

Empire is heavily influenced by Heroku and the philosophies described in [The Twelve-Factor App][12factor], as well as similar projects such as [flynn][flynn] and [deis][deis]

### Phases

There are three phases during deployment:

1. **Build**: This phase happens after a git push to GitHub, which triggers a docker build. Once the image is built, it gets tagged with the git sha that triggered the build.
2. **Release**: This phase happens when a developer triggers a deploy for a git sha via marvin. The git sha is resolved to a docker image, empire creates a "slug", then combines the slug and the latest config into a "release", which is then sent to the process manager to run on the cluster.
3. **Run**: The run phase happens inside the compute cluster. The init system will bring up the desired instance count inside the cluster.

[centurion]: https://github.com/remind101/centurion
[celsus]: https://github.com/remind101/celsus
[sluggy]: https://github.com/remind101/sluggy
[phalanx]: https://github.com/remind101/phalanx
[ignitr]: https://github.com/remind101/ignitr
[pantheon]: https://github.com/remind101/pantheon
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

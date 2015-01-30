# Empire

Empire is Remind's next generation PaaS, which we will eventually use to migrate
away from Heroku.

## Components

**DISCLAIMER**: Empire is incredibly young and a lot of things will most likely
change as we try to productionize it.

Empire is a distributed system for deploying and running
[12factor](http://12factor.net/) [Docker](https://www.docker.com/) based
applications in a compute cluster. The following components are employed:

**[Centurion][centurion]** This is controller API for deploying, scaling, running 1 off processes, etc.

**[Celsus][celsus]** Manages application configuration and versioning.

**[Sluggy][sluggy]** Generates "slugs" by extracting runnable process types from containers.

**[Phalanx][phalanx]** Manages the desired instance counts for runnable processes.

**[Ignitr][ignitr]** Our distributed init system.

**[Rectory][rectory]** You don't want to know.

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

[centurion]: https://github.com/remind101/centurion
[celsus]: https://github.com/remind101/celsus
[sluggy]: https://github.com/remind101/sluggy
[phalanx]: https://github.com/remind101/phalanx
[ignitr]: https://github.com/remind101/ignitr
[rectory]: https://github.com/remind101/rectory
[quay]: https://quay.io
[quayd]: https://github.com/remind101/quayd
[consul]: https://github.com/hashicorp/consul
[registrator]: https://github.com/progrium/registrator
[shipr]: https://github.com/remidn101/shipr
[hubotdeploy]: https://github.com/remidn101/hubot-deploy

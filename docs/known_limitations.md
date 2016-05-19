# Empire :: Known Limitations

A good place to see a list of known issues and limitations is the
[github issue list](https://github.com/remind101/empire/issues). Here's a
list of well known issues as well.

## Only http(s) services

Right now Empire can only serve http & https services.

## Only one exposed process per app

You can only have a single exposed process per app, the `web` app.

## Only one web process per app per minion

This is actually a limitation of Elastic Load Balancers. An ELB can only
use a single port for all the backend processes it is sending traffic to.
Because of that, when exposing an ECS task via ELB (as we do for all
web processes) all of those tasks must share the same port. This means that
you can only have one of those processes per container instance. This does
not affect non-exposed/non-elb-attached processes (in Empire - non-web 
processes).

A side effect of this comes into play when upgrading. Because of the way
ECS does rolling upgrades, you need to have at least N+1 minion hosts for
any given exposed process. This is because when an upgrade happens for a
a process, ECS must first bring up a new process then ensure that it is
healthy before tearing down an old process. ECS will try to parallelize
this as much as it can, but this is limited by the # of free minion
hosts for a given web process.

Again, it's worth noting: This only affects exposed/web processes, but
not processes that do not expose a port.

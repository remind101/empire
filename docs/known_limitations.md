# Empire :: Known Limitations

A good place to see a list of known issues and limitations is the
[github issue list](https://github.com/remind101/empire/issues). Here's a
list of well known issues as well.

## Only http(s) services

Right now Empire can only serve http & https services.

## Only one exposed process per app

You can only have a single exposed process per app, the `web` app.

## Unable to update ELB for an app once it is deployed

Due to a [bug](https://github.com/remind101/empire/issues/498) in
the way that ELBs are setup, it's not possible to modify an ELB once
it is created.

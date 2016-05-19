# Exposing an app publicly

To expose an app publicly you need to add a domain to an app.

Here is how you would expose an app named `acme-inc` with a docker image on the
docker registry at `mycompany/acme-inc` with a tag `latest` publicly:

```console
emp create acme-inc
emp domain-add -a acme-inc acme-inc.com
emp deploy mycompany/acme-inc:latest
```

## Downtime

When adding a domain, CloudFormation will need to destroy the existing internal
load balancer, and create a new internet-facing load balancer. This means that
you're app will experience downtime when making it external, so plan accordingly.

# Exposing an app publicly

To expose an app publicly you need to add a domain to an app. Currently this is
only possible before your app has been deployed, as we don't allow changes to
the ELB associated with apps.

Here is how you would expose an app named `acme-inc` with a docker image on the
docker registry at `mycompany/acme-inc` with a tag `latest` publicly:

```console
emp create acme-inc
emp domain-add -a acme-inc acme-inc.com
emp deploy mycompany/acme-inc:latest
```

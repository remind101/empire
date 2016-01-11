# Empire :: SSL Certs

Empire allows you to attach IAM server certificates to an applications web process. Follow these steps to attach an SSL certificate.

Upload the certificate using the command below.
The .pem file should contain a full chain of the public certificate and any root certificates.

```
emp ssl-cert-add -a <app> cert.pem cert.key
```

**Caveat**: Currently, attaching SSL certificates must happen before you deploy anything to the application (e.g. `emp create` then `emp ssl-cert-add` immediately after).

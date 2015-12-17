# Empire :: SSL Certs

Empire allows you to attach IAM server certificates to an applications web process. Follow these steps to attach an SSL certificate.

First, [upload a certificate to IAM](http://docs.aws.amazon.com/cli/latest/reference/iam/upload-server-certificate.html):

```console
$ aws iam upload-server-certificate --server-certificate-name myServerCertificate --certificate-body file://public_key_cert_file.pem --private-key file://my_private_key.pem --certificate-chain file://my_certificate_chain_file.pem
```

Then attach it to the application:

```console
$ emp certs-attach myServerCertificate -a <app>
```

**Caveat**: Currently, attaching SSL certificates must happen before you deploy anything to the application (e.g. `emp create` then `emp certs-attach` immediately after).

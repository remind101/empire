# Empire :: SSL Certs

Empire allows you to attach IAM and ACM server certificates to the ELB used by an application's web process. Follow these steps to attach an SSL certificate.


## IAM Certificates
First, [upload a certificate to IAM](http://docs.aws.amazon.com/cli/latest/reference/iam/upload-server-certificate.html):

```console
$ aws iam upload-server-certificate --server-certificate-name myServerCertificate --certificate-body file://public_key_cert_file.pem --private-key file://my_private_key.pem --certificate-chain file://my_certificate_chain_file.pem
```

Then attach it to the application using the name you chose for your certificate:

```console
$ emp cert-attach myServerCertificate -a <app>
```

or alternatively, using it's ARN:

```console
$ emp cert-attach arn:aws:iam::<aws_account_id>:server-certificate/<certificate_object_guid> -a <app>
```

## ACM certificates

You can create certificates like any other resource in AWS. ACM certificates are currently free and only available in `us-east-1`. 

Once you create a certificate for your domain, you can use its ARN to attach it to the load balancer: 

```console
$ emp cert-attach  arn:aws:acm:us-east-1:<aws_account_id>:certificate/<certificate_object_guid> -a <app>
```

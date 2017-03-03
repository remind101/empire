package main

import "github.com/remind101/empire/pkg/heroku"

var (
	process string
)

var cmdCertAttach = &Command{
	Run:      runCertAttach,
	Usage:    "cert-attach <aws_cert_arn>",
	NeedsApp: true,
	Category: "certs",
	NumArgs:  1,
	Short:    "attach a certificate to an app",
	Long: `
Attaches an SSL certificate to an applications web process. When using the ECS backend, this will attach an IAM server certificate to the applications ELB.

Before running this command, you should upload your SSL certificate and key to IAM using the AWS CLI.

Examples:

    $ aws iam upload-server-certificate --server-certificate-name myServerCertificate --certificate-body file://public_key_cert_file.pem --private-key file://my_private_key.pem --certificate-chain file://my_certificate_chain_file.pem
    # ^^ The above command will return the ARN of the certificate, you'll need that for the command below
    # Say it returns the arn arn:aws:iam::123456789012:server-certificate/myServerCertificate, you'd use that like this:
    $ emp cert-attach arn:aws:iam::123456789012:server-certificate/myServerCertificate -a myapp
    # By default, this will attach the certifcate to a process named "web". You can override that with the -p flag:
    $ emp cert-attach myServerCertificate -p http -a myapp
`,
}

func init() {
	cmdCertAttach.Flag.StringVarP(&process, "process", "p", "", "process name")
}

func runCertAttach(cmd *Command, args []string) {
	cmd.AssertNumArgsCorrect(args)

	cert := args[0]

	opts := &heroku.CertsAttachOpts{
		Cert: &cert,
	}
	if process != "" {
		opts.Process = &process
	}
	err := client.CertsAttach(mustApp(), opts)
	must(err)
}

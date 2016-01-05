package main

import "errors"

func sslRemoved(cmd *Command, args []string) {
	must(errors.New("the ssl commands have been replaced with `cert-attach`"))
}

var cmdSSL = &Command{
	Run:      sslRemoved,
	Hidden:   true,
	Usage:    "ssl",
	NeedsApp: true,
	Category: "ssl",
	Short:    "show ssl endpoint info",
	Long:     `Show SSL endpoint and certificate information.`,
}

var cmdSSLCertAdd = &Command{
	Run:      sslRemoved,
	Hidden:   true,
	Usage:    "ssl-cert-add [-s] <certfile> <keyfile>",
	NeedsApp: true,
	Category: "ssl",
	Short:    "add a new ssl cert",
	Long: `
Add a new SSL certificate to an app. An SSL endpoint will be
created if the app doesn't yet have one. Otherwise, its cert will
be updated.
Options:
    -s  skip SSL cert optimization and pre-processing
Examples:
    $ emp ssl-cert-add cert.pem key.pem
    hobby-dev        $0/mo
`,
}

var cmdSSLDestroy = &Command{
	Run:      sslRemoved,
	Hidden:   true,
	Usage:    "ssl-destroy",
	NeedsApp: true,
	Category: "ssl",
	Short:    "destroy ssl endpoint",
	Long: `
Removes the SSL endpoints from an app along with all SSL
certificates. If your app's DNS is still configured to point at
the SSL endpoint, this may take your app offline. The command
will prompt for confirmation, or accept confirmation via stdin.
Examples:
    $ emp ssl-destroy
    warning: This will destroy the SSL endpoint on myapp. Please type "myapp" to continue:
    > myapp
    Destroyed SSL endpoint on myapp.
    $ echo myapp | emp ssl-destroy
    Destroyed SSL endpoint on myapp.
`,
}

var cmdSSLCertRollback = &Command{
	Run:      sslRemoved,
	Hidden:   true,
	Usage:    "ssl-cert-rollback",
	NeedsApp: true,
	Category: "ssl",
	Short:    "add a new ssl cert",
	Long: `
Rolls back an SSL endpoint's certificate to the previous version.
Examples:
    $ emp ssl-cert-rollback
    Rolled back cert for myapp.
`,
}

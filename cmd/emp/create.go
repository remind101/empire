package main

import (
	"log"

	"github.com/remind101/empire/pkg/heroku"
)

var cmdCreate = &Command{
	Run:             runCreate,
	Usage:           "create [-r <region>] [-o <org>] [--http-git] [<name>]",
	OptionalMessage: true,
	Category:        "app",
	Short:           "create an app",
	Long: `
Create creates a new Heroku app. If <name> is not specified, the
app is created with a random haiku name.

Options:

    -r <region>  Heroku region to create app in
    -o <org>     name of Heroku organization to create app in
    <name>       optional name for the app

Examples:

    $ emp create
    Created dodging-samurai-42.

    $ emp create -r eu myapp
    Created myapp.
`,
}

var flagRegion string
var flagOrgName string
var flagHTTPGit bool

func init() {
	cmdCreate.Flag.StringVarP(&flagRegion, "region", "r", "", "region name")
	cmdCreate.Flag.StringVarP(&flagOrgName, "org", "o", "", "organization name")
	cmdCreate.Flag.BoolVar(&flagHTTPGit, "http-git", false, "use http git remote")
}

func runCreate(cmd *Command, args []string) {
	appname := ""
	if len(args) > 0 {
		appname = args[0]
	}
	message := getMessage()

	var opts heroku.OrganizationAppCreateOpts
	if appname != "" {
		opts.Name = &appname
	}
	if flagOrgName == "personal" { // "personal" means "no org"
		personal := true
		opts.Personal = &personal
	} else if flagOrgName != "" {
		opts.Organization = &flagOrgName
	}
	if flagRegion != "" {
		opts.Region = &flagRegion
	}

	app, err := client.OrganizationAppCreate(&opts, message)
	must(err)

	addGitRemote(app, flagHTTPGit)

	if app.Organization != nil {
		log.Printf("Created %s in the %s org.", app.Name, app.Organization.Name)
	} else {
		log.Printf("Created %s.", app.Name)
	}
}

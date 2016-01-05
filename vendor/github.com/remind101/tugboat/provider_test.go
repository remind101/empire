package tugboat_test

import (
	"io"
	"log"

	"github.com/remind101/tugboat"
	"golang.org/x/net/context"
)

func deploy(ctx context.Context, d *tugboat.Deployment, w io.Writer) error {
	// Write some log lines
	io.WriteString(w, `My
Log
Lines`)

	// Successful deployment!
	return nil
}

func Example() {
	// First, you'll want to create a tugboat.Client and point it at a
	// tugboat server.
	c := tugboat.NewClient(nil)

	// Options should be parsed from the github webhook.
	opts := tugboat.DeployOpts{
		ID:          1,
		Sha:         "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57",
		Ref:         "master",
		Environment: "staging",
		Description: "Deploying to staging",
		Repo:        "remind101/acme-inc",
	}

	// Calling Deploy will perform the deployment and record the
	// logs.
	if _, err := c.Deploy(context.TODO(), opts, tugboat.ProviderFunc(deploy)); err != nil {
		log.Fatal(err)
	}
}

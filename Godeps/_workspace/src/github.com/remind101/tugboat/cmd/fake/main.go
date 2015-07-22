package main

import (
	"bufio"
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/remind101/tugboat"
	"golang.org/x/net/context"
)

var (
	payload = flag.String("payload", "tests/api/test-fixtures/deployment.json", "")
	secret  = flag.String("secret", "", "")
	url     = flag.String("url", "http://eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJQcm92aWRlciI6ImZha2UifQ.zmy2Wq7Zbol7N1X-R7WX5R4E2i7uH_Arv7FRR2UwnDE:@localhost:8080", "")
	fail    = flag.Bool("fail", false, "Whether the deployment should fail with an error")
)

func main() {
	flag.Parse()

	if err := deploy(); err != nil {
		log.Fatal(err)
	}
}

func deploy() error {
	raw, err := ioutil.ReadFile(*payload)
	if err != nil {
		return err
	}

	c := tugboat.NewClient(nil)
	c.URL = *url

	opts, err := tugboat.NewDeployOptsFromReader(bytes.NewReader(raw))
	if err != nil {
		return err
	}

	c.Deploy(context.Background(), opts, tugboat.ProviderFunc(perform))

	return nil
}

func perform(ctx context.Context, d *tugboat.Deployment, w io.Writer) error {
	if _, err := io.Copy(w, bufio.NewReader(os.Stdin)); err != nil {
		return err
	}

	if *fail {
		return tugboat.ErrFailed
	}

	return nil
}

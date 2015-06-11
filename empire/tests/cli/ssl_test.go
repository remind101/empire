package cli_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestSSLCertAdd(t *testing.T) {
	crt := mustCreateTempfile(t, "server.crt")
	key := mustCreateTempfile(t, "server.key")

	defer os.Remove(crt.Name())
	defer os.Remove(key.Name())

	run(t, []Command{
		{
			"deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
			`Pulling repository remind101/acme-inc
345c7524bc96: Pulling image (9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2) from remind101/acme-inc
345c7524bc96: Pulling image (9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2) from remind101/acme-inc, endpoint: https://registry-1.docker.io/v1/
345c7524bc96: Pulling dependent layers
a1dd7097a8e8: Download complete
Status: Image is up to date for remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2
Status: Created new release v1 for acme-inc`,
		},
		{
			fmt.Sprintf("ssl-cert-add -a acme-inc %s %s", crt.Name(), key.Name()),
			`Added cert for acme-inc at .`,
		},
		{
			`set FOO=bar -a acme-inc`, // Trigger a release
			`Set env vars and restarted acme-inc.`,
		},
	})
}

func mustCreateTempfile(t *testing.T, name string) *os.File {
	f, err := ioutil.TempFile("", name)
	if err != nil {
		t.Fatal(err)
	}

	return f
}

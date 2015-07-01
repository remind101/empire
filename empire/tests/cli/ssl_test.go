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
		DeployCommand("latest", "v1"),
		{
			fmt.Sprintf("ssl-cert-add -a acme-inc %s %s", crt.Name(), key.Name()),
			`Added cert for acme-inc.`,
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

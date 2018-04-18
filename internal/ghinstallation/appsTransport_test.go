package ghinstallation

import (
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

func TestNewAppsTransportKeyFromFile(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(key); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = NewAppsTransportKeyFromFile(&http.Transport{}, integrationID, tmpfile.Name())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
}

package hkclient

import (
	"os"
	"reflect"
	"testing"
)

func TestLoadNetRc(t *testing.T) {
	os.Setenv("NETRC_PATH", "/fake/net/rc")

	nrc, err := LoadNetRc()
	if err != nil {
		t.Fatal(err)
	}
	if nrc == nil {
		t.Fatal("expected an empty NetRc, got nil")
	}
	if !reflect.DeepEqual(*nrc, NetRc{}) {
		t.Errorf("expected an empty NetRc, got %v", *nrc)
	}

	os.Setenv("NETRC_PATH", "")
}

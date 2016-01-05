package main

import (
	"log"
	"os"
	"path/filepath"
	"testing"
)

func setupFakeNetrc() {
	nrc = nil
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	err = os.Setenv("NETRC_PATH", filepath.Join(wd, "fakenetrc"))
	if err != nil {
		log.Fatal(err)
	}
}

func cleanupNetrc() {
	nrc = nil
	os.Setenv("NETRC_PATH", "")
}

func TestGetCreds(t *testing.T) {
	setupFakeNetrc()

	u, p := getCreds("https://omg:wtf@api.heroku.com")
	if u != "omg" {
		t.Errorf("expected user=omg, got %s", u)
	}
	if p != "wtf" {
		t.Errorf("expected password=wtf, got %s", p)
	}
	u, p = getCreds("https://api.heroku.com")
	if u != "user@test.com" {
		t.Errorf("expected user=user@test.com, got %s", u)
	}
	if p != "faketestpassword" {
		t.Errorf("expected password=faketestpassword, got %s", p)
	}

	// test with a nil machine
	u, p = getCreds("https://someotherapi.heroku.com")
	if u != "" || p != "" {
		t.Errorf("expected empty user and pass, got u=%q p=%q", u, p)
	}

	cleanupNetrc()
}

func TestNetrcPath(t *testing.T) {
	fakepath := "/fake/net/rc"
	os.Setenv("NETRC_PATH", fakepath)
	if p := netrcPath(); p != fakepath {
		t.Errorf("NETRC_PATH override expected %q, got %q", fakepath, p)
	}
	os.Setenv("NETRC_PATH", "")
}

func TestLoadNetrc(t *testing.T) {
	setupFakeNetrc()

	loadNetrc()
	m := nrc.FindMachine("api.heroku.com")
	if m == nil {
		t.Errorf("machine api.heroku.com not found")
	} else if m.Login != "user@test.com" {
		t.Errorf("expected user=user@test.com, got %s", m.Login)
	}

	nrc = nil
	fakepath := "/fake/net/rc"
	os.Setenv("NETRC_PATH", fakepath)

	loadNetrc()
	if nrc == nil {
		t.Fatalf("expected non-nil netrc")
	}
	m = nrc.FindMachine("api.heroku.com")
	if m != nil {
		t.Errorf("unexpected machine api.heroku.com found")
	}

	cleanupNetrc()
}

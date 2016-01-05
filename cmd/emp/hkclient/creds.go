package hkclient

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/bgentry/go-netrc/netrc"
)

type NetRc struct {
	netrc.Netrc
}

func netRcPath() string {
	if s := os.Getenv("NETRC_PATH"); s != "" {
		return s
	}

	return filepath.Join(HomePath(), netrcFilename)
}

func LoadNetRc() (nrc *NetRc, err error) {
	onrc, err := netrc.ParseFile(netRcPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &NetRc{}, nil
		}
		return nil, err
	}

	return &NetRc{*onrc}, nil
}

func (nrc *NetRc) GetCreds(apiURL *url.URL) (user, pass string, err error) {
	if err != nil {
		return "", "", fmt.Errorf("invalid API URL: %s", err)
	}
	if apiURL.Host == "" {
		return "", "", fmt.Errorf("missing API host: %s", apiURL)
	}
	if apiURL.User != nil {
		pw, _ := apiURL.User.Password()
		return apiURL.User.Username(), pw, nil
	}

	m := nrc.FindMachine(apiURL.Host)
	if m == nil {
		return "", "", nil
	}
	return m.Login, m.Password, nil
}

func (nrc *NetRc) SaveCreds(address, user, pass string) error {
	m := nrc.FindMachine(address)
	if m == nil || m.IsDefault() {
		m = nrc.NewMachine(address, user, pass, "")
	}
	m.UpdateLogin(user)
	m.UpdatePassword(pass)

	body, err := nrc.MarshalText()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(netRcPath(), body, 0600)
}

func (nrc *NetRc) RemoveCreds(host string) error {
	nrc.RemoveMachine(host)

	body, err := nrc.MarshalText()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(netRcPath(), body, 0600)
}

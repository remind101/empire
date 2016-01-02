// +build windows

package hkclient

import (
	"log"
	"os/user"
)

const netrcFilename = "_netrc"

func homePath() string {
	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return u.HomeDir
}

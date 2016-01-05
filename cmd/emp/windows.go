// +build windows

package main

import (
	"log"
	"os"
	"os/exec"
	"os/user"
)

const (
	netrcFilename           = "_netrc"
	acceptPasswordFromStdin = false
)

func sysExec(path string, args []string, env []string) error {
	cmd := exec.Command(path, args...)
	cmd.Env = env
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		return err
	}
	os.Exit(0)
	return nil
}

func homePath() string {
	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return u.HomeDir
}

// +build darwin freebsd linux netbsd openbsd

package main

import "syscall"

const (
	netrcFilename           = ".netrc"
	acceptPasswordFromStdin = true
)

func sysExec(path string, args []string, env []string) error {
	return syscall.Exec(path, args, env)
}

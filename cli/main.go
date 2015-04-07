package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"

	"github.com/codegangsta/cli"
)

var commands = []cli.Command{
	cmdListAPIs,
	cmdAddAPI,
	cmdSetAPI,
	cmdDeleteAPI,
}

func hk(args ...string) {
	cmd := exec.Command("hk", args...)
	output, _ := cmd.CombinedOutput()
	if len(output) > 0 {
		fmt.Printf("%s", string(output))
	}
}

func emp() {
	app := cli.NewApp()
	app.Name = "emp"
	app.Commands = commands
	app.Run(os.Args)
}

func main() {
	args := os.Args[1:]
	empCmd, _ := regexp.Compile("^api.+")
	setEnv()

	if len(args) == 0 {
		hk(args...)
	} else if empCmd.MatchString(args[0]) {
		emp()
	} else {
		hk(args...)
	}
}

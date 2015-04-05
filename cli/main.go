package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/codegangsta/cli"
)

var commands = []cli.Command{
	cmdListAPIs,
	cmdAddAPI,
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
	setAPI()

	if len(args) == 0 {
		hk(args...)
	} else if args[0] == "apis" || args[0] == "api-add" || args[0] == "api-set" {
		emp()
	} else {
		hk(args...)
	}
}

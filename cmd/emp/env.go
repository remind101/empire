package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"github.com/joho/godotenv"
)

var cmdEnv = &Command{
	Run:      runEnv,
	Usage:    "env",
	NeedsApp: true,
	Category: "config",
	Short:    "list env vars",
	Long:     `Show all env vars.`,
}

func runEnv(cmd *Command, args []string) {
	if len(args) != 0 {
		cmd.PrintUsage()
		os.Exit(2)
	}
	config, err := client.ConfigVarInfo(mustApp())
	must(err)
	var configKeys []string
	for k := range config {
		configKeys = append(configKeys, k)
	}
	sort.Strings(configKeys)
	for _, k := range configKeys {
		fmt.Printf("%s=%s\n", k, config[k])
	}
}

var cmdGet = &Command{
	Run:      runGet,
	Usage:    "get <name>",
	NeedsApp: true,
	Category: "config",
	Short:    "get env var" + extra,
	Long: `
Get the value of an env var.

Example:

    $ emp get BUILDPACK_URL
    http://github.com/kr/heroku-buildpack-inline.git
`,
}

func runGet(cmd *Command, args []string) {
	if len(args) != 1 {
		cmd.PrintUsage()
		os.Exit(2)
	}
	config, err := client.ConfigVarInfo(mustApp())
	must(err)
	value, found := config[args[0]]
	if !found {
		printFatal("No such key as '%s'", args[0])
	}
	fmt.Println(value)
}

var cmdSet = &Command{
	Run:             maybeMessage(runSet),
	Usage:           "set <name>=<value>...",
	NeedsApp:        true,
	OptionalMessage: true,
	Category:        "config",
	Short:           "set env var",
	Long: `
Set the value of an env var.

Example:

    $ emp set BUILDPACK_URL=http://github.com/kr/heroku-buildpack-inline.git
    Set env vars and restarted myapp.
`,
}

func runSet(cmd *Command, args []string) {
	appname := mustApp()
	message := getMessage()
	if len(args) == 0 {
		cmd.PrintUsage()
		os.Exit(2)
	}
	config := make(map[string]*string)
	for _, arg := range args {
		i := strings.Index(arg, "=")
		if i < 0 {
			printFatal("bad format: %#q. See 'emp help set'", arg)
		}
		val := arg[i+1:]
		config[arg[:i]] = &val
	}
	_, err := client.ConfigVarUpdate(appname, config, message)
	must(err)
	log.Printf("Set env vars and restarted " + appname + ".")
}

var cmdUnset = &Command{
	Run:             maybeMessage(runUnset),
	Usage:           "unset <name>...",
	NeedsApp:        true,
	OptionalMessage: true,
	Category:        "config",
	Short:           "unset env var",
	Long: `
Unset an env var.

Example:

    $ emp unset BUILDPACK_URL
    Unset env vars and restarted myapp.
`,
}

func runUnset(cmd *Command, args []string) {
	appname := mustApp()
	message := getMessage()
	if len(args) == 0 {
		cmd.PrintUsage()
		os.Exit(2)
	}
	config := make(map[string]*string)
	for _, key := range args {
		config[key] = nil
	}
	_, err := client.ConfigVarUpdate(appname, config, message)
	must(err)
	log.Printf("Unset env vars and restarted %s.", appname)
}

var cmdEnvLoad = &Command{
	Run:             maybeMessage(runEnvLoad),
	Usage:           "env-load <file>",
	NeedsApp:        true,
	OptionalMessage: true,
	Category:        "config",
	Short:           "load env file",
	Long: `
Loads environment variables from a file.

Example:

    $ emp env-load app.env
    Set env vars from app.env and restarted myapp.
`,
}

func runEnvLoad(cmd *Command, args []string) {
	appname := mustApp()
	message := getMessage()
	if len(args) != 1 {
		cmd.PrintUsage()
		os.Exit(2)
	}

	parsedVars, err := godotenv.Read(args[0])
	must(err)

	config := make(map[string]*string)
	for key, value := range parsedVars {
		config[key] = &value
	}

	_, err = client.ConfigVarUpdate(appname, config, message)
	must(err)
	log.Printf("Updated env vars from %s and restarted %s.", args[0], appname)
}

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
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

	parsedVars, err := ParseEnvFile(args[0])
	must(err)

	config := make(map[string]*string)
	for _, value := range parsedVars {
		kv := strings.SplitN(value, "=", 2)
		if len(kv) == 1 {
			config[kv[0]] = new(string)
		} else {
			config[kv[0]] = &kv[1]
		}
	}

	_, err = client.ConfigVarUpdate(appname, config, message)
	must(err)
	log.Printf("Updated env vars from %s and restarted %s.", args[0], appname)
}

// Stripped from https://github.com/docker/docker/blob/3d13fddd2bc4d679f0eaa68b0be877e5a816ad53/runconfig/opts/envfile.go
//
// ParseEnvFile reads a file with environment variables enumerated by lines
//
// ``Environment variable names used by the utilities in the Shell and
// Utilities volume of IEEE Std 1003.1-2001 consist solely of uppercase
// letters, digits, and the '_' (underscore) from the characters defined in
// Portable Character Set and do not begin with a digit. *But*, other
// characters may be permitted by an implementation; applications shall
// tolerate the presence of such names.''
// -- http://pubs.opengroup.org/onlinepubs/009695399/basedefs/xbd_chap08.html
//
// As of #16585, it's up to application inside docker to validate or not
// environment variables, that's why we just strip leading whitespace and
// nothing more.
func ParseEnvFile(filename string) ([]string, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return []string{}, err
	}
	defer fh.Close()

	lines := []string{}
	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		// trim the line from all leading whitespace first
		line := strings.TrimLeft(scanner.Text(), whiteSpaces)
		// line is not empty, and not starting with '#'
		if len(line) > 0 && !strings.HasPrefix(line, "#") {
			data := strings.SplitN(line, "=", 2)

			// trim the front of a variable, but nothing else
			variable := strings.TrimLeft(data[0], whiteSpaces)
			if strings.ContainsAny(variable, whiteSpaces) {
				return []string{}, ErrBadEnvVariable{fmt.Sprintf("variable '%s' has white spaces", variable)}
			}

			if len(data) > 1 {

				// pass the value through, no trimming
				lines = append(lines, fmt.Sprintf("%s=%s", variable, data[1]))
			} else {
				// if only a pass-through variable is given, clean it up.
				lines = append(lines, fmt.Sprintf("%s=%s", strings.TrimSpace(line), os.Getenv(line)))
			}
		}
	}
	return lines, scanner.Err()
}

var whiteSpaces = " \t"

// ErrBadEnvVariable typed error for bad environment variable
type ErrBadEnvVariable struct {
	msg string
}

func (e ErrBadEnvVariable) Error() string {
	return fmt.Sprintf("poorly formatted environment: %s", e.msg)
}

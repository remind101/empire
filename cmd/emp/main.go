package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	flag "github.com/bgentry/pflag"
	"github.com/docker/docker/pkg/term"
	"github.com/mgutz/ansi"
	"github.com/remind101/empire/cmd/emp/hkclient"
	"github.com/remind101/empire/pkg/heroku"
)

var (
	apiURL = "http://localhost:8080"
	stdin  = bufio.NewReader(os.Stdin)

	inFd, outFd                 uintptr
	isTerminalIn, isTerminalOut bool
)

type Command struct {
	// args does not include the command name
	Run         func(cmd *Command, args []string)
	Flag        flag.FlagSet
	NeedsApp    bool
	OptionalApp bool
	NoClient    bool

	Usage    string // first word is the command name
	Category string // i.e. "App", "Account", etc.
	Short    string // `emp help` output
	Long     string // `emp help cmd` output
	Alias    string // Optional alias for the command.
	Hidden   bool   // Set to true to hide a command from usage information.
}

func (c *Command) PrintUsage() {
	if c.Runnable() {
		fmt.Fprintf(os.Stderr, "Usage: emp %s\n", c.FullUsage())
	}
	fmt.Fprintf(os.Stderr, "Use 'emp help %s' for more information.\n", c.Name())
}

func (c *Command) PrintLongUsage() {
	if c.Runnable() {
		fmt.Printf("Usage: emp %s\n\n", c.FullUsage())
	}
	fmt.Println(strings.Trim(c.Long, "\n"))
}

func (c *Command) FullUsage() string {
	if c.NeedsApp || c.OptionalApp {
		return c.Name() + " [-a <app or remote>]" + strings.TrimPrefix(c.Usage, c.Name())
	}
	return c.Usage
}

func (c *Command) Name() string {
	name := c.Usage
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

func (c *Command) Runnable() bool {
	return c.Run != nil
}

func (c *Command) Visible() bool {
	return c.Runnable() && !c.Hidden
}

const extra = " (extra)"

func (c *Command) List() bool {
	return c.Short != "" && !strings.HasSuffix(c.Short, extra)
}

func (c *Command) ListAsExtra() bool {
	return c.Short != "" && strings.HasSuffix(c.Short, extra)
}

func (c *Command) ShortExtra() string {
	return c.Short[:len(c.Short)-len(extra)]
}

// Running `emp help` will list commands in this order.
var commands = []*Command{
	cmdCreate,
	cmdApps,
	cmdDynos,
	cmdReleases,
	cmdReleaseInfo,
	cmdRollback,
	cmdScale,
	cmdRestart,
	cmdEnvLoad,
	cmdSet,
	cmdUnset,
	cmdEnv,
	cmdRun,
	cmdLog,
	cmdInfo,
	cmdRename,
	cmdDestroy,
	cmdDomains,
	cmdDomainAdd,
	cmdDomainRemove,
	cmdCertAttach,
	cmdDeploy,
	cmdVersion,
	cmdHelp,

	helpCommands,
	helpEnviron,
	helpMore,
	helpAbout,

	// listed by emp help more
	cmdAPI,
	cmdAuthorize,
	cmdCreds,
	cmdGet,
	cmdLogin,
	cmdLogout,
	cmdSSL,
	cmdSSLCertAdd,
	cmdSSLCertRollback,
	cmdSSLDestroy,
	cmdURL,
	cmdWhichApp,
}

var (
	flagApp   string
	client    *heroku.Client
	hkAgent   = "hk/" + Version + " (" + runtime.GOOS + "; " + runtime.GOARCH + ")"
	userAgent = hkAgent + " " + heroku.DefaultUserAgent
)

func initClients() {
	loadNetrc()
	suite, err := hkclient.New(nrc, hkAgent)
	if err != nil {
		printFatal(err.Error())
	}

	client = suite.Client
	apiURL = suite.ApiURL
}

func main() {
	log.SetFlags(0)

	// make sure command is specified, disallow global args
	args := os.Args[1:]
	if len(args) < 1 || strings.IndexRune(args[0], '-') == 0 {
		printUsageTo(os.Stderr)
		os.Exit(2)
	}

	inFd, isTerminalIn = term.GetFdInfo(os.Stdin)
	outFd, isTerminalOut = term.GetFdInfo(os.Stdout)

	if !isTerminalOut {
		ansi.DisableColors(true)
	}

	for _, cmd := range commands {
		if matchesCommand(cmd, args[0]) && cmd.Run != nil {
			defer recoverPanic()

			if !cmd.NoClient {
				initClients()
			}

			cmd.Flag.SetDisableDuplicates(true) // disallow duplicate flag options
			if !gitConfigBool("hk.strict-flag-ordering") {
				cmd.Flag.SetInterspersed(true) // allow flags & non-flag args to mix
			}
			cmd.Flag.Usage = func() {
				cmd.PrintUsage()
			}
			if cmd.NeedsApp || cmd.OptionalApp {
				cmd.Flag.StringVarP(&flagApp, "app", "a", "", "app name")
			}
			if err := cmd.Flag.Parse(args[1:]); err == flag.ErrHelp {
				cmdHelp.Run(cmdHelp, args[:1])
				return
			} else if err != nil {
				printError(err.Error())
				os.Exit(2)
			}
			if flagApp != "" {
				if gitRemoteApp, err := appFromGitRemote(flagApp); err == nil {
					flagApp = gitRemoteApp
				}
			}
			if cmd.NeedsApp || cmd.OptionalApp {
				a, err := app()
				switch {
				case err == errMultipleHerokuRemotes && cmd.NeedsApp, err == nil && a == "" && cmd.NeedsApp:
					msg := "no app specified"
					if err != nil {
						msg = err.Error()
					}
					printError(msg)
					cmd.PrintUsage()
					os.Exit(2)
				case err != nil && cmd.NeedsApp:
					printFatal(err.Error())
				}
			}
			cmd.Run(cmd, cmd.Flag.Args())
			return
		}
	}

	fmt.Fprintf(os.Stderr, "Unknown command: %s\n", args[0])
	if g := suggest(args[0]); len(g) > 0 {
		fmt.Fprintf(os.Stderr, "Possible alternatives: %v\n", strings.Join(g, " "))
	}
	fmt.Fprintf(os.Stderr, "Run 'emp help' for usage.\n")
	os.Exit(2)
}

func recoverPanic() {
	if Version != "dev" {
		if rec := recover(); rec != nil {
			printFatal("emp encountered and reported an internal client error")
		}
	}
}

func app() (string, error) {
	if flagApp != "" {
		return flagApp, nil
	}

	if app := os.Getenv("EMPAPP"); app != "" {
		return app, nil
	}

	return appFromGitRemote(remoteFromGitConfig())
}

func mustApp() string {
	name, err := app()
	if err != nil {
		printFatal(err.Error())
	}
	return name
}

// matchesCommand checks if the Command matches the command that we want to run.
func matchesCommand(cmd *Command, want string) bool {
	if cmd.Alias != "" && cmd.Alias == want {
		return true
	}

	return cmd.Name() == want
}

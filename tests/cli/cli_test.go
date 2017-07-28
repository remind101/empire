package cli_test

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/remind101/empire"
	"github.com/remind101/empire/empiretest"
	"github.com/remind101/empire/empiretest/cli"
	"github.com/remind101/empire/pkg/timex"
	"github.com/remind101/empire/server/auth"
	"github.com/remind101/empire/server/heroku"
)

// empPath can be used to change what binary is used to run the tests.
const empPath = "../../build/emp"

var fakeUser = &empire.User{
	Name:        "fake",
	GitHubToken: "token",
}

func DeployCommand(tag, version string) Command {
	return Command{
		fmt.Sprintf("deploy remind101/acme-inc:%s", tag),
		`Pulling repository remind101/acme-inc
345c7524bc96: Pulling image (` + tag + `) from remind101/acme-inc
345c7524bc96: Pulling image (` + tag + `) from remind101/acme-inc, endpoint: https://registry-1.docker.io/v1/
345c7524bc96: Pulling dependent layers
a1dd7097a8e8: Download complete
Status: Image is up to date for remind101/acme-inc:` + tag + `
Status: Created new release ` + version + ` for acme-inc
Status: Finished processing events for release ` + version + ` of acme-inc`,
	}
}

// Run the tests with empiretest.Run, which will lock access to the database
// since it can't be shared by parallel tests.
func TestMain(m *testing.M) {
	empiretest.Run(m)
}

var fakeNow = time.Date(2015, time.January, 1, 1, 1, 1, 1, time.UTC)

// Stubs out time.Now in empire.
func init() {
	now(fakeNow)
}

// now stubs out empire.Now.
func now(t time.Time) {
	timex.Now = func() time.Time {
		return t
	}
}

func resetNow() {
	now(fakeNow)
}

type Command struct {
	// Command represents a cli command to run.
	Command string

	// Output is the output we expect to see.
	Output interface{}
}

// run is a simple helper that builds a new CLI instance, and runs the given
// commands against it.
func run(t testing.TB, commands []Command) {
	cli := newCLI(t)
	defer cli.Close()
	cli.Run(t, commands)
}

// CLI wraps an empire instance, a server and a CLI as one unit, which can be
// used to execute emp commands.
type CLI struct {
	*empiretest.Server
	*cli.CLI
	started bool // holds whether server has been started or not.
}

// newCLI returns a new CLI instance.
func newCLI(t testing.TB) *CLI {
	e := empiretest.NewEmpire(t)
	s := empiretest.NewServer(t, e)
	return newCLIWithServer(t, s)
}

func newCLIWithServer(t testing.TB, s *empiretest.Server) *CLI {
	path, err := filepath.Abs(empPath)
	if err != nil {
		t.Fatal(err)
	}

	u, err := url.Parse(s.URL())
	if err != nil {
		t.Fatal(err)
	}

	cli, err := cli.New(path, u)
	if err != nil {
		t.Fatal(err)
	}

	return &CLI{
		CLI:    cli,
		Server: s,
	}
}

// Close closes the CLI, and the test server.
func (c *CLI) Close() {
	if err := c.CLI.Close(); err != nil {
		panic(err)
	}

	c.Server.Close()
}

// Run authenticates the CLI and runs the commands.
func (c *CLI) Run(t testing.TB, commands []Command) {
	c.Auth(t)
	c.RunCommands(t, commands)
}

// Auth creates a new access token, and authorizes the CLI using it.
func (c *CLI) Auth(t testing.TB) {
	token, err := c.Server.Heroku.AccessTokensCreate(&heroku.AccessToken{
		User: fakeUser,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Authorize(fakeUser.Name, token.Token); err != nil {
		t.Fatal(err)
	}
}

// Start starts the underlying Empire HTTP server if it hasn't already been
// started.
func (c *CLI) Start() {
	if !c.started {
		c.Server.Start()
		c.started = true
	}
}

// RunCommands runs all of the given commands and verifies their output.
func (c *CLI) RunCommands(t testing.TB, commands []Command) {
	c.Start() // Ensure server is started.

	for _, cmd := range commands {
		args := strings.Split(cmd.Command, " ")

		b, err := c.Command(args...).CombinedOutput()
		got := string(b)
		t.Log(fmt.Sprintf("\n$ %s\n%s", cmd.Command, got))
		if expectedErr, ok := cmd.Output.(error); ok {
			expectedErrString := fmt.Sprintf("%v\n", expectedErr)
			if got != expectedErrString {
				t.Fatalf("Expected %q, got %q", expectedErr, got)
			}
		} else if err != nil {
			t.Fatal(err)
		}

		if want, ok := cmd.Output.(string); ok {
			if want != "" {
				want = want + "\n"
			}

			if got != want {
				t.Fatalf("%q != %q", got, want)
			}
		} else if regex, ok := cmd.Output.(*regexp.Regexp); ok {
			if !regex.MatchString(got) {
				t.Fatalf("%q != %q", got, regex.String())
			}
		}
	}
}

func newAuth(authenticator auth.Authenticator) *auth.Auth {
	return &auth.Auth{
		Strategies: auth.Strategies{
			{
				Name:          auth.StrategyUsernamePassword,
				Authenticator: authenticator,
			},
		},
	}
}

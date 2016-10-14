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
	"github.com/remind101/empire/server/heroku"
	"github.com/remind101/pkg/timex"
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

func run(t testing.TB, commands []Command) {
	runWithPre(t, nil, commands)
}

func runWithCLI(t testing.TB, commands []Command, cli *CLI) {
	token, err := cli.Server.Heroku.AccessTokensCreate(&heroku.AccessToken{
		User: fakeUser,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := cli.Authorize(fakeUser.Name, token.Token); err != nil {
		t.Fatal(err)
	}

	for _, cmd := range commands {
		args := strings.Split(cmd.Command, " ")

		b, err := cli.Command(args...).CombinedOutput()
		got := string(b)
		t.Log(fmt.Sprintf("\n$ %s\n%s", cmd.Command, got))
		if expectedErr, ok := cmd.Output.(error); ok {
			expectedErrString := fmt.Sprintf("%v\n", expectedErr)
			if got != expectedErrString {
				t.Fatalf("Expected %s, got %v", expectedErr, got)
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

func runWithPre(t testing.TB, pre func(*CLI), commands []Command) {
	cli := newCLI(t)
	defer cli.Close()
	if pre != nil {
		pre(cli)
	}
	runWithCLI(t, commands, cli)
}

// CLI wraps an empire instance, a server and a CLI as one unit, which can be
// used to execute emp commands.
type CLI struct {
	*empiretest.Server
	*cli.CLI
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

func (c *CLI) Close() {
	if err := c.CLI.Close(); err != nil {
		panic(err)
	}

	c.Server.Close()
}

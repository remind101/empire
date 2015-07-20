package cli_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/remind101/empire"
	"github.com/remind101/empire/empiretest"
	"github.com/remind101/pkg/timex"
)

func DeployCommand(tag, version string) Command {
	return Command{
		fmt.Sprintf("deploy remind101/acme-inc:%s", tag),
		`Pulling repository remind101/acme-inc
345c7524bc96: Pulling image (` + tag + `) from remind101/acme-inc
345c7524bc96: Pulling image (` + tag + `) from remind101/acme-inc, endpoint: https://registry-1.docker.io/v1/
345c7524bc96: Pulling dependent layers
a1dd7097a8e8: Download complete
Status: Image is up to date for remind101/acme-inc:` + tag + `
Status: Created new release ` + version + ` for acme-inc`,
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

// cli runs an cli command against a server.
func cli(t testing.TB, token, url, command string) string {
	cmd := NewCmd(url, command)
	cmd.Authorize(token)

	b, err := cmd.CombinedOutput()
	t.Log(fmt.Sprintf("\n$ %s\n%s", command, string(b)))
	if err != nil {
		t.Fatal(err)
	}

	return string(b)
}

// Cmd represents an cli command.
type Cmd struct {
	*exec.Cmd

	// The Heroku API URL.
	URL string
}

func NewCmd(url, command string) *Cmd {
	args := strings.Split(command, " ")

	p, err := filepath.Abs("../../build/emp")
	if err != nil {
		return nil
	}

	cmd := exec.Command(p, args...)
	cmd.Env = []string{
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		"TERM=screen-256color",
		fmt.Sprintf("EMPIRE_API_URL=%s", url),
	}

	return &Cmd{
		Cmd: cmd,
		URL: url,
	}
}

func (c *Cmd) Authorize(token string) {
	netrc, err := writeNetrc(token, c.URL)
	if err != nil {
		panic(err)
	}

	c.Cmd.Env = append(c.Cmd.Env, fmt.Sprintf("NETRC_PATH=%s", netrc.Name()))
}

func writeNetrc(token, uri string) (*os.File, error) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return f, err
	}
	defer f.Close()

	u, err := url.Parse(uri)
	if err != nil {
		return f, err
	}

	if _, err := io.WriteString(f, `machine `+u.Host+`
  login foo@example.com
  password `+token); err != nil {
		return f, err
	}

	return f, nil
}

func deleteNetrc() error {
	err := os.Remove(".netrc")
	if err != nil {
		return err
	}
	return nil
}

type Command struct {
	// Command represents a cli command to run.
	Command string

	// Output is the output we expect to see.
	Output string
}

func run(t testing.TB, commands []Command) {
	e := empiretest.NewEmpire(t)
	s := empiretest.NewServer(t, e)
	defer s.Close()

	if err := deleteNetrc(); err != nil {
		if err.Error() != "remove .netrc: no such file or directory" {
			t.Fatal(err)
		}
	}

	token, err := e.AccessTokensCreate(&empire.AccessToken{
		User: &empire.User{Name: "fake", GitHubToken: "token"},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, cmd := range commands {
		got := cli(t, token.Token, s.URL, cmd.Command)

		want := cmd.Output
		if want != "" {
			want = want + "\n"
		}

		if got != want {
			t.Fatalf("%q != %q", got, want)
		}
	}
}

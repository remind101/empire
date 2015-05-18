package hk_test

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

	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/empiretest"
	"github.com/remind101/pkg/timex"
)

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

// hk runs an hk command against a server.
func hk(t testing.TB, token, url, command string) string {
	cmd := NewHKCmd(url, command)
	cmd.Authorize(token)

	b, err := cmd.CombinedOutput()
	t.Log(fmt.Sprintf("\n$ %s\n%s", command, string(b)))
	if err != nil {
		t.Fatal(err)
	}

	return string(b)
}

// HKCmd represents an hk command.
type HKCmd struct {
	*exec.Cmd

	// The Heroku API URL.
	URL string
}

func NewHKCmd(url, command string) *HKCmd {
	args := strings.Split(command, " ")

	p, err := filepath.Abs("../../build/hk")
	if err != nil {
		return nil
	}

	cmd := exec.Command(p, args...)
	cmd.Env = []string{
		fmt.Sprintf("PATH=../../../cli/build/:%s", os.Getenv("PATH")),
		"HKPATH=../../../cli/hk-plugins",
		fmt.Sprintf("HEROKU_API_URL=%s", url),
	}

	return &HKCmd{
		Cmd: cmd,
		URL: url,
	}
}

func (c *HKCmd) Authorize(token string) {
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

type Command struct {
	// Command represents an hk command to run.
	Command string

	// Output is the output we expect to see.
	Output string
}

func run(t testing.TB, commands []Command) {
	e := empiretest.NewEmpire(t)
	s := empiretest.NewServer(t, e)
	defer s.Close()

	token, err := e.AccessTokensCreate(&empire.AccessToken{
		User: &empire.User{Name: "fake", GitHubToken: "token"},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, cmd := range commands {
		got := hk(t, token.Token, s.URL, cmd.Command)

		want := cmd.Output
		if want != "" {
			want = want + "\n"
		}

		if got != want {
			t.Fatalf("%q != %q", got, want)
		}
	}
}

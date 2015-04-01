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
	"github.com/remind101/empire/empiretest"
)

func TestLogin(t *testing.T) {
	e := empiretest.NewEmpire(t)
	s := empiretest.NewServer(t, e)
	defer s.Close()

	input := "fake\nbar\n"

	cmd := NewHKCmd(s.URL, "login")
	cmd.Stdin = strings.NewReader(input)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(out), "Enter email: Logged in.\n"; got != want {
		t.Fatalf("%q", got)
	}
}

func TestLoginUnauthorized(t *testing.T) {
	e := empiretest.NewEmpire(t)
	s := empiretest.NewServer(t, e)
	defer s.Close()

	input := "foo\nbar\n"

	cmd := NewHKCmd(s.URL, "login")
	cmd.Stdin = strings.NewReader(input)

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("Expected an error")
	}

	if got, want := string(out), "Enter email: error: Request not authenticated, API token is missing, invalid or expired Log in with `hk login`.\n"; got != want {
		t.Fatalf("%q", got)
	}
}

func TestLoginTwoFactor(t *testing.T) {
	e := empiretest.NewEmpire(t)
	s := empiretest.NewServer(t, e)
	defer s.Close()

	input := "twofactor\nbar\ncode\n"

	cmd := NewHKCmd(s.URL, "login")
	cmd.Stdin = strings.NewReader(input)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(out), "Enter email: Enter two-factor auth code: Logged in.\n"; got != want {
		t.Fatalf("%q", got)
	}
}

func TestCreate(t *testing.T) {
	run(t, []Command{
		{
			"apps",
			"",
		},
		{
			"create acme-inc",
			"Created acme-inc.",
		},
	})
}

func TestApps(t *testing.T) {
	run(t, []Command{
		{
			"create acme-inc",
			"Created acme-inc.",
		},
		{
			"apps",
			"acme-inc      Dec 31 17:01",
		},
	})
}

func TestConfig(t *testing.T) {
	run(t, []Command{
		{
			"create acme-inc",
			"Created acme-inc.",
		},
		{
			"set RAILS_ENV=production -a acme-inc",
			"Set env vars and restarted acme-inc.",
		},
		{
			"env -a acme-inc",
			"RAILS_ENV=production",
		},
		{
			"set DATABASE_URL=postgres://localhost AUTH=foo -a acme-inc",
			"Set env vars and restarted acme-inc.",
		},
		{
			"unset RAILS_ENV -a acme-inc",
			"Unset env vars and restarted acme-inc.",
		},
		{
			"env -a acme-inc",
			"AUTH=foo\nDATABASE_URL=postgres://localhost",
		},
	})
}

func TestUpdateConfigNewReleaseSameFormation(t *testing.T) {
	now(time.Now().AddDate(0, 0, -5))
	defer resetNow()

	run(t, []Command{
		{
			"deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
			"Deployed remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
		},
		{
			"dynos -a acme-inc",
			"acme-inc.1.web.1    unknown   5d  \"./bin/web\"",
		},
		{
			"scale web=2 -a acme-inc",
			"Scaled acme-inc to web=2:1X.",
		},
		{
			"dynos -a acme-inc",
			`acme-inc.1.web.1    unknown   5d  "./bin/web"
acme-inc.1.web.2    unknown   5d  "./bin/web"`,
		},
		{
			"set DATABASE_URL=postgres://localhost AUTH=foo -a acme-inc",
			"Set env vars and restarted acme-inc.",
		},
		{
			"dynos -a acme-inc",
			`acme-inc.2.web.1    unknown   5d  "./bin/web"
acme-inc.2.web.2    unknown   5d  "./bin/web"`,
		},
	})
}

func TestDomains(t *testing.T) {
	run(t, []Command{
		{
			"create acme-inc",
			"Created acme-inc.",
		},
		{
			"domains -a acme-inc",
			"",
		},
		{
			"domain-add example.com -a acme-inc",
			"Added example.com to acme-inc.",
		},
		{
			"domains -a acme-inc",
			"example.com",
		},
		{
			"domain-remove example.com -a acme-inc",
			"Removed example.com from acme-inc.",
		},
		{
			"domains -a acme-inc",
			"",
		},
	})

}

func TestDeploy(t *testing.T) {
	run(t, []Command{
		{
			"deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
			"Deployed remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
		},
		{
			"releases -a acme-inc",
			"v1    Dec 31 17:01  Deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
		},
		{
			"deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
			"Deployed remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
		},
		{
			"releases -a acme-inc",
			"v1    Dec 31 17:01  Deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2\nv2    Dec 31 17:01  Deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
		},
	})
}

func TestScale(t *testing.T) {
	now(time.Now().AddDate(0, 0, -5))
	defer resetNow()

	run(t, []Command{
		{
			"deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
			"Deployed remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
		},
		{
			"scale web=2 -a acme-inc",
			"Scaled acme-inc to web=2:1X.",
		},
		{
			"dynos -a acme-inc",
			`acme-inc.1.web.1    unknown   5d  "./bin/web"
acme-inc.1.web.2    unknown   5d  "./bin/web"`,
		},

		{
			"scale web=1 -a acme-inc",
			"Scaled acme-inc to web=1:1X.",
		},
		{
			"dynos -a acme-inc",
			"acme-inc.1.web.1    unknown   5d  \"./bin/web\"",
		},
	})
}

func TestRollback(t *testing.T) {
	run(t, []Command{
		{
			"deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
			"Deployed remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
		},
		{
			"deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
			"Deployed remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
		},
		{
			"rollback v1 -a acme-inc",
			"Rolled back acme-inc to v1 as v3.",
		},
		{
			"releases -a acme-inc",
			`v1    Dec 31 17:01  Deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2
v2    Dec 31 17:01  Deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2
v3    Dec 31 17:01  Rollback to v1`,
		},
	})
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
	empire.Now = func() time.Time {
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
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		"HKPATH=../../../hk-plugins",
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

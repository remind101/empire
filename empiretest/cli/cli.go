// Package cli provides test helpers for testing the emp CLI against an Empire
// environment.
package cli

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"text/template"
)

// CLI represents an `emp` CLI.
type CLI struct {
	url   *url.URL
	path  string
	netrc *os.File
}

// New returns a new CLI instance, which can be used to run emp commands.
func New(path string, url *url.URL) (*CLI, error) {
	netrc, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, fmt.Errorf("error creating .netrc: %v", err)
	}

	return &CLI{
		path:  path,
		url:   url,
		netrc: netrc,
	}, nil
}

// Command creates a new exec.Cmd instance that will run an emp command.
func (c *CLI) Command(arg ...string) *exec.Cmd {
	cmd := exec.Command(c.path, arg...)
	cmd.Env = []string{
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		"TERM=screen-256color",
		"TZ=America/Los_Angeles",
		fmt.Sprintf("EMPIRE_API_URL=%s", c.url.String()),
		fmt.Sprintf("NETRC_PATH=%s", c.netrc.Name()),
	}
	return cmd
}

// Authorize writes the access token to the .netrc file.
func (c *CLI) Authorize(user, token string) error {
	entry := struct {
		Host     string
		User     string
		Password string
	}{
		Host:     c.url.Host,
		User:     user,
		Password: token,
	}

	t := template.Must(template.New("netrc").Parse(netrcEntry))
	if err := t.Execute(c.netrc, entry); err != nil {
		return fmt.Errorf("error writing .netrc: %v", err)
	}

	return nil
}

// Close closes the file descriptor for the netrc.
func (c *CLI) Close() error {
	if err := c.netrc.Close(); err != nil {
		return fmt.Errorf("error closing .netrc: %v", err)
	}

	return nil
}

const netrcEntry = `
machine {{.Host}}
  login {{.User}}
  password {{.Password}}`

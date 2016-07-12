// Package cli provides test helpers for testing the emp CLI against an Empire
// environment.
package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
)

// CLI represents an `emp` CLI.
type CLI struct {
	url   *url.URL
	path  string
	netrc *os.File
}

// NewCLI returns a new CLI instance, which can be used to run emp commands.
func NewCLI(path string, url *url.URL) (*CLI, error) {
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

// Exec creates a new exec.Cmd instance that will run an emp command.
func (c *CLI) Exec(arg ...string) *exec.Cmd {
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
func (c *CLI) Authorize(token string) error {
	if _, err := io.WriteString(c.netrc, `machine `+c.url.Host+`
  login foo@example.com
  password `+token); err != nil {
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

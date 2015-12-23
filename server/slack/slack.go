// Package slack provides a Slack slash command control layer for Empire.
package slack

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ejholmes/slash"
	"github.com/mattn/go-shellwords"
	"github.com/remind101/empire"
	"github.com/remind101/empire/cli"
	"github.com/remind101/pkg/httpx"
	"golang.org/x/net/context"
)

// NewServer returns a new httpx.Handler that serves the slack slash commands.
func NewServer(e *empire.Empire) httpx.Handler {
	h := NewHandler(e)
	return slash.NewServer(h)
}

// NewHandler returns a new slash.Handler that serves the Empire public API over
// slack slash commands.
func NewHandler(e *empire.Empire) slash.Handler {
	cli := cli.NewCLI(e)
	return NewCLIHandler(cli)
}

// NewCLIHandler returns a new slash.Handler that adapts the empire CLI to be
// served over slack slash commands.
func NewCLIHandler(cli *cli.CLI) slash.Handler {
	return &CLIHandler{CLI: cli}
}

// Interface for running a CLI command.
type CLI interface {
	Run(ctx context.Context, stdout io.Writer, args []string) error
}

// CLIHandler is a slash.Handler that serves a CLI over slack slash commands.
type CLIHandler struct {
	CLI
}

func (h *CLIHandler) ServeCommand(ctx context.Context, r slash.Responder, c slash.Command) error {
	// TODO: Wrap with middleware that authenticates the Slack user.
	ctx = empire.WithUser(ctx, &empire.User{Name: "Slack"})

	args, err := shellwords.Parse(c.Text)
	if err != nil {
		return err
	}

	w := new(bytes.Buffer)
	if err := h.Run(ctx, w, append([]string{""}, args...)); err != nil {
		return err
	}

	// Only respond if there was output.
	if w.Len() > 0 {
		// Wrap the response in a code block for formatting.
		return r.Respond(slash.Say(fmt.Sprintf("%s %s:\n```%s```", c.Command, c.Text, w.String())))
	}

	return nil
}

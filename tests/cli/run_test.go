package cli_test

import (
	"errors"
	"testing"

	"github.com/remind101/empire"
	"golang.org/x/net/context"
)

func TestRunDetached(t *testing.T) {
	run(t, []Command{
		DeployCommand("latest", "v1"),
		{
			"run -d migration -a acme-inc",
			"Ran `migration` on acme-inc as run, detached.",
		},
	})
}

func TestRunAttached(t *testing.T) {
	run(t, []Command{
		DeployCommand("latest", "v1"),
		{
			"run migration -a acme-inc",
			"Fake output for `[migration]` on acme-inc",
		},
	})
}

func TestRunAttached_WithConfirmation_Failed(t *testing.T) {
	pre := func(cli *CLI) {
		cli.Empire.ConfirmActions = map[empire.Action]empire.ActionConfirmer{
			empire.ActionRun: empire.ActionConfirmerFunc(func(ctx context.Context, user *empire.User, action empire.Action, params map[string]string) (bool, error) {
				return false, nil
			}),
		}
	}

	runWithPre(t, pre, []Command{
		DeployCommand("latest", "v1"),
		{
			"run bash -a acme-inc",
			"request to Run was denied\r",
		},
	})
}

func TestRunAttached_WithConfirmation_Error(t *testing.T) {
	pre := func(cli *CLI) {
		cli.Empire.ConfirmActions = map[empire.Action]empire.ActionConfirmer{
			empire.ActionRun: empire.ActionConfirmerFunc(func(ctx context.Context, user *empire.User, action empire.Action, params map[string]string) (bool, error) {
				return false, errors.New("duo api error")
			}),
		}
	}

	runWithPre(t, pre, []Command{
		DeployCommand("latest", "v1"),
		{
			"run bash -a acme-inc",
			"duo api error\r",
		},
	})
}

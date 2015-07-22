package fake

import (
	"errors"
	"fmt"
	"io"

	"github.com/remind101/tugboat"
	"golang.org/x/net/context"
)

var (
	_ tugboat.Provider = &Provider{}

	// DefaultScenarios are the default set of scenarios that will be played
	// back.
	DefaultScenarios = map[string]Scenario{
		"Ok": Scenario{
			Logs: `-----> Fetching custom git buildpack... done
-----> Go app detected
-----> Using go1.3
-----> Running: godep go install -tags heroku ./...
-----> Discovering process types
       Procfile declares types -> web

-----> Compressing... done, 1.6MB
-----> Launching... done, v6
       https://acme-inc.herokuapp.com/ deployed to Heroku
`,
		},
		"Failure": Scenario{
			Error: errors.New("boom"),
		},
	}
)

// Provider is a fake provider.
type Provider struct {
	Scenarios map[string]Scenario
}

func NewProvider() *Provider {
	return &Provider{
		Scenarios: DefaultScenarios,
	}
}

func (p *Provider) Deploy(ctx context.Context, d *tugboat.Deployment, w io.Writer) error {
	s, ok := p.Scenarios[d.Description]
	if !ok {
		return fmt.Errorf("no scenario for %s", d.Repo)
	}

	if _, err := io.WriteString(w, s.Logs); err != nil {
		return err
	}

	return nil
}

func (p *Provider) Name() string {
	return "fake"
}

type Scenario struct {
	Logs  string
	Error error
}

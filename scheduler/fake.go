package scheduler

import (
	"fmt"
	"io"

	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
)

type FakeManager struct {
	apps map[string]*App
}

func NewFakeManager() *FakeManager {
	return &FakeManager{
		apps: make(map[string]*App),
	}
}

func (m *FakeManager) Submit(ctx context.Context, app *App) error {
	m.apps[app.ID] = app
	return nil
}

func (m *FakeManager) Scale(ctx context.Context, app string, ptype string, instances uint) error {
	if a, ok := m.apps[app]; ok {
		var process *Process
		for _, p := range a.Processes {
			if p.Type == ptype {
				process = p
			}
		}

		if process != nil {
			process.Instances = instances
		}
	}
	return nil
}

func (m *FakeManager) Remove(ctx context.Context, appID string) error {
	delete(m.apps, appID)
	return nil
}

func (m *FakeManager) Instances(ctx context.Context, appID string) ([]*Instance, error) {
	var instances []*Instance
	if a, ok := m.apps[appID]; ok {
		for _, p := range a.Processes {
			for i := uint(1); i <= p.Instances; i++ {
				instances = append(instances, &Instance{
					ID:        fmt.Sprintf("%d", i),
					State:     "running",
					Process:   p,
					UpdatedAt: timex.Now(),
				})
			}
		}
	}
	return instances, nil
}

func (m *FakeManager) Stop(ctx context.Context, instanceID string) error {
	return nil
}

func (m *FakeManager) Run(ctx context.Context, app *App, p *Process, in io.Reader, out io.Writer) error {
	if out != nil {
		fmt.Fprintf(out, "Fake output for `%s` on %s\n", p.Command, app.Name)
	}
	return nil
}

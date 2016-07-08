package scheduler

import (
	"fmt"
	"io"

	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
)

type FakeScheduler struct {
	apps map[string]*App
}

func NewFakeScheduler() *FakeScheduler {
	return &FakeScheduler{
		apps: make(map[string]*App),
	}
}

func (m *FakeScheduler) Submit(ctx context.Context, app *App, ss StatusStream) error {
	m.apps[app.ID] = app
	if ss != nil {
		ss.Done(nil)
	}
	return nil
}

func (m *FakeScheduler) Scale(ctx context.Context, app string, ptype string, instances uint) error {
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

func (m *FakeScheduler) Remove(ctx context.Context, appID string) error {
	delete(m.apps, appID)
	return nil
}

func (m *FakeScheduler) Instances(ctx context.Context, appID string) ([]*Instance, error) {
	var instances []*Instance
	if a, ok := m.apps[appID]; ok {
		for _, p := range a.Processes {
			pp := *p
			pp.Env = Env(a, p)
			for i := uint(1); i <= p.Instances; i++ {
				instances = append(instances, &Instance{
					ID:        fmt.Sprintf("%d", i),
					State:     "running",
					Process:   &pp,
					UpdatedAt: timex.Now(),
				})
			}
		}
	}
	return instances, nil
}

func (m *FakeScheduler) Stop(ctx context.Context, instanceID string) error {
	return nil
}

func (m *FakeScheduler) Run(ctx context.Context, app *App, p *Process, in io.Reader, out io.Writer) error {
	if out != nil {
		fmt.Fprintf(out, "Fake output for `%s` on %s\n", p.Command, app.Name)
	}
	return nil
}

package scheduler

import (
	"fmt"
	"io"

	"github.com/remind101/empire/12factor"
	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
)

type FakeScheduler struct {
	apps    map[string]twelvefactor.App
	running map[string]map[string]uint
}

func NewFakeScheduler() *FakeScheduler {
	return &FakeScheduler{
		apps:    make(map[string]twelvefactor.App),
		running: make(map[string]map[string]uint),
	}
}

func (m *FakeScheduler) Submit(ctx context.Context, app twelvefactor.App) error {
	m.apps[app.ID] = app
	for _, p := range app.Processes {
		if m.running[app.ID] == nil {
			m.running[app.ID] = make(map[string]uint)
		}
		m.running[app.ID][p.Type] = p.Instances
	}
	return nil
}

func (m *FakeScheduler) Scale(ctx context.Context, app string, ptype string, instances uint) error {
	m.running[app][ptype] = instances
	return nil
}

func (m *FakeScheduler) Remove(ctx context.Context, appID string) error {
	delete(m.apps, appID)
	delete(m.running, appID)
	return nil
}

func (m *FakeScheduler) Instances(ctx context.Context, appID string) ([]Instance, error) {
	var instances []Instance
	if a, ok := m.apps[appID]; ok {
		for _, p := range a.Processes {
			for i := uint(1); i <= m.running[a.ID][p.Type]; i++ {
				p.Env = twelvefactor.ProcessEnv(a, p)
				instances = append(instances, Instance{
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

func (m *FakeScheduler) Stop(ctx context.Context, instanceID string) error {
	return nil
}

func (m *FakeScheduler) Run(ctx context.Context, app twelvefactor.App, p twelvefactor.Process, in io.Reader, out io.Writer) error {
	if out != nil {
		fmt.Fprintf(out, "Fake output for `%s` on %s\n", p.Command, app.Name)
	}
	return nil
}

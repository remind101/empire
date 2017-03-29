package twelvefactor

import (
	"fmt"
	"io"
	"sync"

	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
)

type FakeScheduler struct {
	sync.Mutex
	apps map[string]*Manifest
}

func NewFakeScheduler() *FakeScheduler {
	return &FakeScheduler{
		apps: make(map[string]*Manifest),
	}
}

func (m *FakeScheduler) Submit(ctx context.Context, app *Manifest, ss StatusStream) error {
	m.Lock()
	defer m.Unlock()
	m.apps[app.AppID] = app
	return nil
}

func (m *FakeScheduler) Restart(ctx context.Context, app *Manifest, ss StatusStream) error {
	return m.Submit(ctx, app, ss)
}

func (m *FakeScheduler) Remove(ctx context.Context, appID string) error {
	delete(m.apps, appID)
	return nil
}

func (m *FakeScheduler) Tasks(ctx context.Context, appID string) ([]*Task, error) {
	var instances []*Task
	if a, ok := m.apps[appID]; ok {
		for _, p := range a.Processes {
			pp := *p
			pp.Env = Env(a, p)
			for i := 1; i <= p.Quantity; i++ {
				instances = append(instances, &Task{
					ID:        fmt.Sprintf("%d", i),
					Host:      Host{ID: "i-aa111aa1"},
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

func (m *FakeScheduler) Run(ctx context.Context, app *Manifest, p *Process, in io.Reader, out io.Writer) error {
	if out != nil {
		fmt.Fprintf(out, "Fake output for `%s` on %s\n", p.Command, app.Name)
	}
	return nil
}

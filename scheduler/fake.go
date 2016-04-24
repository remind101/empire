package scheduler

import (
	"fmt"
	"io"

	"github.com/remind101/empire/12factor"
	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
)

type FakeScheduler struct {
	manifests map[string]twelvefactor.Manifest
	running   map[string]map[string]uint
}

func NewFakeScheduler() *FakeScheduler {
	return &FakeScheduler{
		manifests: make(map[string]twelvefactor.Manifest),
		running:   make(map[string]map[string]uint),
	}
}

func (m *FakeScheduler) Submit(ctx context.Context, manifest twelvefactor.Manifest) error {
	m.manifests[manifest.ID] = manifest
	for _, p := range manifest.Processes {
		if m.running[manifest.ID] == nil {
			m.running[manifest.ID] = make(map[string]uint)
		}
		m.running[manifest.ID][p.Type] = p.Instances
	}
	return nil
}

func (m *FakeScheduler) Scale(ctx context.Context, appID string, ptype string, instances uint) error {
	m.running[appID][ptype] = instances
	return nil
}

func (m *FakeScheduler) Remove(ctx context.Context, appID string) error {
	delete(m.manifests, appID)
	delete(m.running, appID)
	return nil
}

func (m *FakeScheduler) Instances(ctx context.Context, appID string) ([]Instance, error) {
	var instances []Instance
	if manifest, ok := m.manifests[appID]; ok {
		for _, p := range manifest.Processes {
			for i := uint(1); i <= m.running[manifest.App.ID][p.Type]; i++ {
				p.Env = twelvefactor.ProcessEnv(manifest.App, p)
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

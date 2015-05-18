package service

import (
	"fmt"

	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/trace"
	"golang.org/x/net/context"
)

// LoggedManager is a Manager implementation that wraps another Manager to log
// using pkg/logger.
type LoggedManager struct {
	Prefix string
	Manager
}

func WithLogging(m Manager) *LoggedManager {
	return &LoggedManager{
		Manager: m,
	}
}

func (m *LoggedManager) Submit(ctx context.Context, app *App) (err error) {
	ctx, done := trace.Trace(ctx)
	defer func() { done(err, "submitting app", "app", app.ID) }()
	err = m.Manager.Submit(ctx, app)
	return err
}

func (m *LoggedManager) Scale(ctx context.Context, app string, process string, instances uint) (err error) {
	ctx, done := trace.Trace(ctx)
	defer func() { done(err, "scaling process", "app", app, "process", process, "instances", instances) }()
	err = m.Manager.Scale(ctx, app, process, instances)
	return err
}

func (m *LoggedManager) Remove(ctx context.Context, app string) error {
	err := m.Manager.Remove(ctx, app)
	logger.Info(ctx, m.msg("Remove"), "err", err, "app", app)
	return err
}

func (m *LoggedManager) Instances(ctx context.Context, app string) ([]*Instance, error) {
	instances, err := m.Manager.Instances(ctx, app)
	logger.Info(ctx, m.msg("Instances"), "err", err, "app", app, "instances", len(instances))
	return instances, err
}

func (m *LoggedManager) Stop(ctx context.Context, instanceID string) error {
	err := m.Manager.Stop(ctx, instanceID)
	logger.Info(ctx, m.msg("Stop"), "err", err, "instanceID", instanceID)
	return err
}

func (m *LoggedManager) msg(msg string) string {
	return fmt.Sprintf("%s.%s", m.Prefix, msg)
}

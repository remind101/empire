package service

import (
	"github.com/remind101/pkg/logger"
	"golang.org/x/net/context"
)

// LoggedManager is a Manager implementation that wraps another Manager to log
// using pkg/logger.
type LoggedManager struct {
	Manager
}

func Log(m Manager) *LoggedManager {
	return &LoggedManager{
		Manager: m,
	}
}

func (m *LoggedManager) Submit(ctx context.Context, app *App) error {
	err := m.Manager.Submit(ctx, app)
	logger.Log(ctx, "at", "manager.submit", "err", err, "app", app.Name)
	return err
}

func (m *LoggedManager) Scale(ctx context.Context, app string, process string, instances uint) error {
	err := m.Manager.Scale(ctx, app, process, instances)
	logger.Log(ctx, "at", "manager.scale", "err", err, "app", app, "process", process, "instances", instances)
	return err
}

func (m *LoggedManager) Remove(ctx context.Context, app string) error {
	err := m.Manager.Remove(ctx, app)
	logger.Log(ctx, "at", "manager.remove", "err", err, "app", app)
	return err
}

func (m *LoggedManager) Instances(ctx context.Context, app string) ([]*Instance, error) {
	instances, err := m.Manager.Instances(ctx, app)
	logger.Log(ctx, "at", "manager.instances", "err", err, "app", app, "instances", len(instances))
	return instances, err
}

func (m *LoggedManager) Stop(ctx context.Context, instanceID string) error {
	err := m.Manager.Stop(ctx, instanceID)
	logger.Log(ctx, "at", "manager.stop", "err", err, "instanceID", instanceID)
	return err
}

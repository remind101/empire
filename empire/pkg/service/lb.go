package service

import (
	"github.com/remind101/empire/empire/pkg/lb"
	"github.com/remind101/pkg/logger"
	"golang.org/x/net/context"
)

// LBProcessManager is an implementation of the ProcessManager interface that creates
// LoadBalancers when a Process is created.
type LBProcessManager struct {
	ProcessManager
	lb lb.Manager
}

// CreateProcess ensures that there is a load balancer for the process, then
// creates it. It uses the following algorithm:
//
// * Attempt to find existing load balancer.
// * If the load balancer exists, check that the exposure is appropriate for the process.
// * If the load balancer's External attribute doesn't match what we want. Delete the process, also deleting the load balancer.
// * Create the load balancer
// * Attach it to the process.
func (m *LBProcessManager) CreateProcess(ctx context.Context, app *App, p *Process) error {
	if p.Exposure > ExposeNone {
		// Attempt to find an existing load balancer for this app.
		l, err := m.findLoadBalancer(ctx, app.Name, p.Type)
		if err != nil {
			return err
		}

		if l != nil {
			// If the load balancer doesn't match the exposure that we
			// wan't, we'll destroy the process, which will also destroy the
			// existing load balancer, then let a new load balancer get
			// created.
			if !lbOk(p, l) {
				logger.Info(ctx, "existing load balancer not suitable", "err", err, "external", l.External, "exposure", p.Exposure.String())

				if err := m.RemoveProcess(ctx, app.Name, p.Type); err != nil {
					if !noService(err) {
						return err
					}
				}

				// We set l to nil so that a new load balancer will get
				// created below.
				l = nil
			}
		}

		// If this app doesn't have a load balancer yet, create one.
		if l == nil {
			l, err = m.lb.CreateLoadBalancer(ctx, lb.CreateLoadBalancerOpts{
				Name:         app.Name,
				InstancePort: *p.Ports[0].Host, // TODO: Check that the process has ports.
				External:     p.Exposure == ExposePublic,
				Tags:         lbTags(app.Name, p.Type),
			})
			if err != nil {
				return err
			}
		}

		// Attach the name of the load balancer to the process so it can be used
		// downstream.
		p.LoadBalancer = l.Name
	}

	return m.ProcessManager.CreateProcess(ctx, app, p)
}

// RemoveProcess removes the process then removes the associated LoadBalancer.
func (m *LBProcessManager) RemoveProcess(ctx context.Context, app string, p string) error {
	if err := m.ProcessManager.RemoveProcess(ctx, app, p); err != nil {
		return err
	}

	l, err := m.findLoadBalancer(ctx, app, p)
	if err != nil {
		// TODO: Maybe we shouldn't care here.
		return err
	}

	if l != nil {
		if err := m.lb.DestroyLoadBalancer(ctx, l.Name); err != nil {
			// TODO: Maybe we shouldn't care here.
			return err
		}
	}

	return nil
}

// findLoadBalancer attempts to find an existing load balancer for the app.
func (m *LBProcessManager) findLoadBalancer(ctx context.Context, app string, process string) (*lb.LoadBalancer, error) {
	lbs, err := m.lb.LoadBalancers(ctx, lbTags(app, process))
	if err != nil || len(lbs) == 0 {
		return nil, err
	}

	return lbs[0], nil
}

// lbTags returns the tags that should be attached to the load balancer so that
// we can find it later.
func lbTags(app string, process string) map[string]string {
	return map[string]string{
		"AppName":     app,
		"ProcessType": process,
	}
}

// lbOk checks if the load balancer is suitable for the process.
func lbOk(p *Process, lb *lb.LoadBalancer) bool {
	if p.Exposure == ExposePublic && !lb.External {
		return false
	}

	if p.Exposure == ExposePrivate && lb.External {
		return false
	}

	if *p.Ports[0].Host != lb.InstancePort {
		return false
	}

	return true
}

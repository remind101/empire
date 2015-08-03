package service

import (
	"fmt"

	"github.com/remind101/empire/pkg/lb"
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
		l, err := m.findLoadBalancer(ctx, app.ID, p.Type)
		if err != nil {
			return err
		}

		// If the load balancer doesn't match the exposure that we
		// want, we'll return an error. Users should manually destroy
		// the app and re-create it with the proper exposure.
		if l != nil {
			if err = lbOk(p, l); err != nil {
				return err
			}
		}

		// If this app doesn't have a load balancer yet, create one.
		if l == nil {
			tags := lbTags(app.ID, p.Type)

			// Add "App" tag so that a CNAME can be created.
			tags[lb.AppTag] = app.Name

			l, err = m.lb.CreateLoadBalancer(ctx, lb.CreateLoadBalancerOpts{
				InstancePort: *p.Ports[0].Host, // TODO: Check that the process has ports.
				External:     p.Exposure == ExposePublic,
				SSLCert:      p.SSLCert,
				Tags:         tags,
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
		if err := m.lb.DestroyLoadBalancer(ctx, l); err != nil {
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
		"AppID":       app,
		"ProcessType": process,
	}
}

// LoadBalancerExposureError is returned when the exposure of the process in the data store does not match the exposure of the ELB
type LoadBalancerExposureError struct {
	proc *Process
	lb   *lb.LoadBalancer
}

func (e *LoadBalancerExposureError) Error() string {
	var lbExposure string
	if !e.lb.External {
		lbExposure = "private"
	} else {
		lbExposure = "public"
	}

	return fmt.Sprintf("Process %s is %s, but load balancer is %s.", e.proc.Type, e.proc.Exposure, lbExposure)
}

// LoadBalancerPortMismatchError is returned when the port stored in the data store does not match the ELB instance port
type LoadBalancerPortMismatchError struct {
	proc *Process
	lb   *lb.LoadBalancer
}

func (e *LoadBalancerPortMismatchError) Error() string {
	return fmt.Sprintf("Process %s instance port is %d, but load balancer instance port is %d.", e.proc.Type, e.proc.Ports[0].Host, e.lb.InstancePort)
}

// SslCertMismatchError is returned when the ssl cert in the data store does not match the ssl cert on the ELB
type SslCertMismatchError struct {
	proc *Process
	lb   *lb.LoadBalancer
}

func (e *SslCertMismatchError) Error() string {
	return fmt.Sprintf("Process ssl certificate (%s) does not match load balancer ssl certificate (%s).", e.proc.SSLCert, e.lb.SSLCert)
}

// lbOk checks if the load balancer is suitable for the process.
func lbOk(p *Process, lb *lb.LoadBalancer) error {
	if p.Exposure == ExposePublic && !lb.External {
		return &LoadBalancerExposureError{p, lb}
	}

	if p.Exposure == ExposePrivate && lb.External {
		return &LoadBalancerExposureError{p, lb}
	}

	if *p.Ports[0].Host != lb.InstancePort {
		return &LoadBalancerPortMismatchError{p, lb}
	}

	if p.SSLCert != lb.SSLCert {
		return &SslCertMismatchError{p, lb}
	}

	return nil
}

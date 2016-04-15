package lb

import (
	"github.com/remind101/pkg/logger"
	"golang.org/x/net/context"
)

var _ Manager = &LoggedManager{}

// LoggedManager is an implementation of the Manager interface that logs when
// LoadBalancers are created and destroyed.
type LoggedManager struct {
	Manager
}

// WithLogging wraps the manager with logging.
func WithLogging(m Manager) *LoggedManager {
	return &LoggedManager{m}
}

func (m *LoggedManager) CreateLoadBalancer(ctx context.Context, o CreateLoadBalancerOpts) (*LoadBalancer, error) {
	var dnsName, name string
	lb, err := m.Manager.CreateLoadBalancer(ctx, o)
	if err == nil && lb != nil {
		name = lb.Name
		dnsName = lb.DNSName
	}

	logger.Info(ctx, "creating load balancer",
		"err", err,
		"name", name,
		"external", o.External,
		"dns-name", dnsName,
		"cert", o.SSLCert,
	)
	return lb, err
}

func (m *LoggedManager) DestroyLoadBalancer(ctx context.Context, lb *LoadBalancer) error {
	err := m.Manager.DestroyLoadBalancer(ctx, lb)
	logger.Info(ctx, "destroying load balancer", "err", err, "name", lb.Name)
	return err
}

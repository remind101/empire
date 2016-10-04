// package lb provides an abstraction around creating load balancers.
package lb

import "context"

const AppTag = "App"

// CreateLoadBalancerOpts are options that can be provided when creating a
// LoadBalancer.
type CreateLoadBalancerOpts struct {
	// An arbitrary list of tags to assign to the load balancer.
	Tags map[string]string

	// True if the load balancer should be publicy exposed.
	External bool

	// The SSL Certificate
	SSLCert string
}

// UpdateLoadBalancerOpts are options that can be provided when updating an
// existing load balancer.
type UpdateLoadBalancerOpts struct {
	// The name of the load balancer.
	Name string

	// The SSL Certificate
	SSLCert *string
}

// LoadBalancer represents a load balancer.
type LoadBalancer struct {
	// The name of the load balancer.
	Name string

	// DNSName is the DNS name for the load balancer. CNAME records can be
	// created that point to this location.
	DNSName string

	// True if the load balancer is exposed externally.
	External bool

	// The SSL Certificate to associate with the load balancer.
	SSLCert string

	// InstancePort is the port that this load balancer forwards requests to
	// on the host.
	InstancePort int64

	// Tags contain the tags attached to the LoadBalancer
	Tags map[string]string
}

// Manager is our API interface for interacting with LoadBalancers.
type Manager interface {
	// CreateLoadBalancer creates a new LoadBalancer with the given options.
	CreateLoadBalancer(context.Context, CreateLoadBalancerOpts) (*LoadBalancer, error)

	// UpdateLoadBalancer updates an existing load balancer.
	UpdateLoadBalancer(context.Context, UpdateLoadBalancerOpts) error

	// DestroyLoadBalancer destroys a load balancer by name.
	DestroyLoadBalancer(ctx context.Context, lb *LoadBalancer) error

	// LoadBalancers returns a list of LoadBalancers, optionally provide
	// tags to filter by.
	LoadBalancers(ctx context.Context, tags map[string]string) ([]*LoadBalancer, error)
}

// WithCNAME wraps a Manager to create CNAME records for the LoadBalancer
// using a Nameserver.
func WithCNAME(m Manager, n Nameserver) *CNAMEManager {
	return &CNAMEManager{
		Manager:    m,
		Nameserver: n,
	}
}

// CNAMEManager is an implementation of the Manager interface that creates CNAME
// records for the LoadBalancer after its created.
type CNAMEManager struct {
	Manager
	Nameserver
}

// CreateLoadBalancer will create the LoadBalancer using the underlying manager,
// then create a CNAME record pointed at the LoadBalancers DNSName. The CNAME
// will be pulled from the `Service` tag if provided.
func (m *CNAMEManager) CreateLoadBalancer(ctx context.Context, opts CreateLoadBalancerOpts) (*LoadBalancer, error) {
	lb, err := m.Manager.CreateLoadBalancer(ctx, opts)
	if err != nil {
		return lb, err
	}

	if n, ok := opts.Tags[AppTag]; ok {
		return lb, m.CreateCNAME(n, lb.DNSName)
	}

	return lb, nil
}

// DestroyLoadBalancer destroys an ELB, then removes any CNAMEs that were
// pointed at that ELB.
func (m *CNAMEManager) DestroyLoadBalancer(ctx context.Context, lb *LoadBalancer) error {
	err := m.Manager.DestroyLoadBalancer(ctx, lb)
	if err != nil {
		return err
	}

	return m.RemoveCNAME(ctx, lb)
}

func (m *CNAMEManager) RemoveCNAME(ctx context.Context, lb *LoadBalancer) error {
	if n, ok := lb.Tags[AppTag]; ok {
		return m.DeleteCNAME(n, lb.DNSName)
	}

	return nil
}

func (m *CNAMEManager) RemoveCNAMEs(ctx context.Context, tags map[string]string) error {
	lbs, err := m.LoadBalancers(ctx, tags)
	if err != nil {
		return err
	}

	for _, lb := range lbs {
		if err := m.RemoveCNAME(ctx, lb); err != nil {
			return err
		}
	}

	return nil
}

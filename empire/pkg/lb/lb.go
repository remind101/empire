// package lb provides an abstraction around creating load balancers.
package lb

import (
	"strings"

	"golang.org/x/net/context"
)

// CreateLoadBalancerOpts are options that can be provided when creating a
// LoadBalancer.
type CreateLoadBalancerOpts struct {
	// The name of the load balancer.
	Name string

	// The port to route requests to on the hosts.
	InstancePort int64

	// An arbitrary list of tags to assign to the load balancer.
	Tags map[string]string

	// True if the load balancer should be publicy exposed.
	External bool
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

	// InstancePort is the port that this load balancer forwards requests to
	// on the host.
	InstancePort int64
}

// Manager is our API interface for interacting with LoadBalancers.
type Manager interface {
	// CreateLoadBalancer creates a new LoadBalancer with the given options.
	CreateLoadBalancer(context.Context, CreateLoadBalancerOpts) (*LoadBalancer, error)

	// DestroyLoadBalancer destroys a load balancer by name.
	DestroyLoadBalancer(ctx context.Context, name string) error

	// LoadBalancers returns a list of LoadBalancers, optionally provide
	// tags to filter by.
	LoadBalancers(ctx context.Context, tags map[string]string) ([]*LoadBalancer, error)
}

// WithCNAME wraps a Manager to create CNAME records for the LoadBalancer
// using a Nameserver.
func WithCNAME(m Manager, n Nameserver) Manager {
	return &cnameManager{
		Manager:    m,
		Nameserver: n,
	}
}

// cnameManager is an implementation of the Manager interface that creates CNAME
// records for the LoadBalancer after it's created.
type cnameManager struct {
	Manager
	Nameserver
}

// CreateLoadBalancer will create the LoadBalancer using the underlying manager,
// then create a CNAME record pointed at the LoadBalancers DNSName.
func (m *cnameManager) CreateLoadBalancer(ctx context.Context, opts CreateLoadBalancerOpts) (*LoadBalancer, error) {
	lb, err := m.Manager.CreateLoadBalancer(ctx, opts)
	if err != nil {
		return lb, err
	}

	return lb, m.CNAME(lb.Name, lb.DNSName)
}

// SanitizeUUIDs wraps a Manager implementation to strip `-` from the
// LoadBalancer during creation. This is useful with the ELBManager
// implementation when the provided names are UUID's, since it'll truncate the
// UUID's to 32 characters, which is the maximum allowed length for ELB names.
func SanitizeUUIDs(m Manager) Manager {
	return &sanitizeManager{m}
}

// sanitizeManager is a Manager implementation that sanitizes uuids to fit
// within ELB naming constraints.
type sanitizeManager struct {
	Manager
}

func (m *sanitizeManager) CreateLoadBalancer(ctx context.Context, o CreateLoadBalancerOpts) (*LoadBalancer, error) {
	o.Name = strings.Replace(o.Name, "-", "", -1)
	return m.Manager.CreateLoadBalancer(ctx, o)
}

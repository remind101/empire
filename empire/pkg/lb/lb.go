// package lb provides an abstraction around creating load balancers.
package lb

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
}

// Manager is our API interface for interacting with LoadBalancers.
type Manager interface {
	// CreateLoadBalancer creates a new LoadBalancer with the given options.
	CreateLoadBalancer(CreateLoadBalancerOpts) (*LoadBalancer, error)

	// DestroyLoadBalancer destroys a load balancer by name.
	DestroyLoadBalancer(name string) error

	// LoadBalancers returns a list of LoadBalancers, optionally provide
	// tags to filter by.
	LoadBalancers(tags map[string]string) ([]*LoadBalancer, error)
}

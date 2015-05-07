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

	// DNSName is the DNS name for the load balancer. CNAME records can be
	// created that point to this location.
	DNSName string
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

// CreateCNAMEs wraps a Manager to create CNAME records for the LoadBalancer
// using a Nameserver.
func CreateCNAMEs(m Manager, n Nameserver) Manager {
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
func (m *cnameManager) CreateLoadBalancer(opts CreateLoadBalancerOpts) (*LoadBalancer, error) {
	lb, err := m.Manager.CreateLoadBalancer(opts)
	if err != nil {
		return lb, err
	}

	return lb, m.CNAME(lb.Name, lb.DNSName)
}

// Ensure NullManager conforms to Manager interface.
var _ Manager = &NullManager{}

// NullManager is an implementation of the Manager interface that does nothing.
type NullManager struct{}

func (m *NullManager) CreateLoadBalancer(CreateLoadBalancerOpts) (*LoadBalancer, error) {
	return &LoadBalancer{
		Name: "fake",
	}, nil
}

func (m *NullManager) DestroyLoadBalancer(name string) error {
	return nil
}

func (m *NullManager) LoadBalancers(tags map[string]string) ([]*LoadBalancer, error) {
	return nil, nil
}

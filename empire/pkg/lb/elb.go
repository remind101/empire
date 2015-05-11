package lb

import (
	"strings"

	"code.google.com/p/go-uuid/uuid"
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/awslabs/aws-sdk-go/service/elb"
	"golang.org/x/net/context"
)

const (
	schemeInternal = "internal"
	schemeExternal = "internet-facing"
)

var _ Manager = &ELBManager{}

// ELBManager is an implementation of the Manager interface that creates Elastic
// Load Balancers.
type ELBManager struct {
	// The ID of the security group to assign to internal load balancers.
	InternalSecurityGroupID string
	// The ID of the security group to assign to external load balancers.
	ExternalSecurityGroupID string

	// SubnetFinder is used to determine what subnets to attach the ELB to.
	SubnetFinder

	elb *elb.ELB

	newName func() string
}

// NewELBManager returns a new ELBManager backed by the aws config.
func NewELBManager(c *aws.Config) *ELBManager {
	return &ELBManager{
		elb:     elb.New(c),
		newName: newName,
	}
}

// NewVPCELBManager returns a new ELBManager that will use a VPCSubnetFinder to
// determine what subnets to attach to the ELB.
func NewVPCELBManager(vpc string, c *aws.Config) *ELBManager {
	f := NewVPCSubnetFinder(c)
	f.VPC = vpc

	m := NewELBManager(c)
	m.SubnetFinder = f

	return m
}

// CreateLoadBalancer creates a new ELB:
//
// * The ELB is created and connection draining is enabled.
// * An internal DNS CNAME record is created, pointing the the DNSName of the ELB.
func (m *ELBManager) CreateLoadBalancer(ctx context.Context, o CreateLoadBalancerOpts) (*LoadBalancer, error) {
	subnets, err := m.subnets()
	if err != nil {
		return nil, err
	}

	scheme := schemeInternal
	sg := m.InternalSecurityGroupID

	if o.External {
		scheme = schemeExternal
		sg = m.ExternalSecurityGroupID
	}

	input := &elb.CreateLoadBalancerInput{
		Listeners:        elbListeners(o.InstancePort, o.SSLCert),
		LoadBalancerName: aws.String(m.newName()),
		Scheme:           aws.String(scheme),
		SecurityGroups:   []*string{aws.String(sg)},
		Subnets:          subnets,
		Tags:             elbTags(o.Tags),
	}

	// Create the ELB.
	out, err := m.elb.CreateLoadBalancer(input)
	if err != nil {
		return nil, err
	}

	// Add connection draining to the LoadBalancer.
	if _, err := m.elb.ModifyLoadBalancerAttributes(&elb.ModifyLoadBalancerAttributesInput{
		LoadBalancerAttributes: &elb.LoadBalancerAttributes{
			ConnectionDraining: &elb.ConnectionDraining{
				Enabled: aws.Boolean(true),
				Timeout: aws.Long(300), // TODO: Configurable?
			},
		},
		LoadBalancerName: input.LoadBalancerName,
	}); err != nil {
		return nil, err
	}

	return &LoadBalancer{
		Name:         *input.LoadBalancerName,
		DNSName:      *out.DNSName,
		External:     o.External,
		SSLCert:      o.SSLCert,
		InstancePort: o.InstancePort,
	}, nil
}

// DestroyLoadBalancer destroys an ELB.
func (m *ELBManager) DestroyLoadBalancer(ctx context.Context, name string) error {
	_, err := m.elb.DeleteLoadBalancer(&elb.DeleteLoadBalancerInput{
		LoadBalancerName: aws.String(name),
	})
	return err
}

// LoadBalancers returns all load balancers. If tags are provided, then the
// resulting load balancers will be filtered to only those containing the
// provided tags.
func (m *ELBManager) LoadBalancers(ctx context.Context, tags map[string]string) ([]*LoadBalancer, error) {
	var (
		nextMarker *string
		lbs        []*LoadBalancer
	)

	for {
		out, err := m.elb.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
			Marker: nextMarker,
		})
		if err != nil {
			return nil, err
		}

		if len(out.LoadBalancerDescriptions) == 0 {
			continue
		}

		// Create a names slice and descriptions map.
		names := make([]*string, len(out.LoadBalancerDescriptions))
		descs := map[string]*elb.LoadBalancerDescription{}

		for i, d := range out.LoadBalancerDescriptions {
			names[i] = d.LoadBalancerName
			descs[*d.LoadBalancerName] = d
		}

		// Find all the tags for this batch of load balancers.
		out2, err := m.elb.DescribeTags(&elb.DescribeTagsInput{LoadBalancerNames: names})
		if err != nil {
			return lbs, err
		}

		// Append matching load balancers to our result set.
		for _, d := range out2.TagDescriptions {
			if containsTags(tags, d.Tags) {
				elb := descs[*d.LoadBalancerName]
				var instancePort int64
				var sslCert string

				if len(elb.ListenerDescriptions) > 0 {
					instancePort = *elb.ListenerDescriptions[0].Listener.InstancePort
					for _, ld := range elb.ListenerDescriptions {
						if ld.Listener.SSLCertificateID != nil {
							sslCert = *ld.Listener.SSLCertificateID
						}
					}
				}

				lbs = append(lbs, &LoadBalancer{
					Name:         *elb.LoadBalancerName,
					DNSName:      *elb.DNSName,
					External:     *elb.Scheme == schemeExternal,
					SSLCert:      sslCert,
					InstancePort: instancePort,
				})
			}
		}

		nextMarker = out.NextMarker
		if nextMarker == nil || *nextMarker == "" {
			// No more items
			break
		}
	}

	return lbs, nil
}

func (m *ELBManager) subnets() ([]*string, error) {
	subnets, err := m.Subnets()
	if err != nil {
		return nil, err
	}

	var p []*string
	for _, s := range subnets {
		ss := s
		p = append(p, &ss)
	}
	return p, nil
}

// SubnetFinder is an interface that can return a list of subnets.
type SubnetFinder interface {
	Subnets() ([]string, error)
}

// SubnetFinderFunc is a function definition that satisfies the SubnetFinder
// interface.
type SubnetFinderFunc func() ([]string, error)

func (f SubnetFinderFunc) Subnets() ([]string, error) {
	return f()
}

// StaticSubnets returns a SubnetFinder that returns the given subnets.
func StaticSubnets(subnets []string) SubnetFinder {
	return SubnetFinderFunc(func() ([]string, error) {
		return subnets, nil
	})
}

// VPCSubnetFinder is an implementation of the SubnetFinder interface that
// queries for subnets in a VPC.
type VPCSubnetFinder struct {
	VPC string
	ec2 *ec2.EC2
}

// NewVPCSubnetFinder returns a new VPCSubnetFinder instance with a configured
// ec2 client.
func NewVPCSubnetFinder(c *aws.Config) *VPCSubnetFinder {
	return &VPCSubnetFinder{
		ec2: ec2.New(c),
	}
}

// subnets queryies for subnets within the vpc.
func (f *VPCSubnetFinder) Subnets() ([]string, error) {
	var subnets []string

	out, err := f.ec2.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(f.VPC)}},
		},
	})
	if err != nil {
		return nil, err
	}

	for _, s := range out.Subnets {
		subnets = append(subnets, *s.SubnetID)
	}

	return subnets, nil
}

// newName returns a string that's suitable as a load balancer name for elb.
func newName() string {
	return strings.Replace(uuid.New(), "-", "", -1)
}

// elbListeners returns a suitable list of listeners. We listen on post 80 by default.
// If certID is not empty an SSL listener will be added to the list. certID should be
// the Amazon Resource Name (ARN) of the server certificate.
func elbListeners(port int64, certID string) []*elb.Listener {
	listeners := []*elb.Listener{
		{
			InstancePort:     aws.Long(port),
			LoadBalancerPort: aws.Long(80),
			Protocol:         aws.String("http"),
			InstanceProtocol: aws.String("http"),
		},
	}

	if certID != "" {
		listeners = append(listeners, &elb.Listener{
			InstancePort:     aws.Long(port),
			LoadBalancerPort: aws.Long(443),
			SSLCertificateID: aws.String(certID),
			Protocol:         aws.String("https"),
			InstanceProtocol: aws.String("http"),
		})
	}
	return listeners
}

// elbTags takes a map[string]string and converts it to the elb.Tag format.
func elbTags(tags map[string]string) []*elb.Tag {
	var e []*elb.Tag

	for k, v := range tags {
		e = append(e, elbTag(k, v))
	}

	return e
}

func elbTag(k, v string) *elb.Tag {
	return &elb.Tag{
		Key:   aws.String(k),
		Value: aws.String(v),
	}
}

// containsTags ensures that b contains all of the tags in a.
func containsTags(a map[string]string, b []*elb.Tag) bool {
	for k, v := range a {
		t := elbTag(k, v)
		if !containsTag(t, b) {
			return false
		}
	}
	return true
}

func containsTag(t *elb.Tag, tags []*elb.Tag) bool {
	for _, t2 := range tags {
		if *t.Key == *t2.Key && *t.Value == *t2.Value {
			return true
		}
	}
	return false
}

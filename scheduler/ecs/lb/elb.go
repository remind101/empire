package lb

import (
	"strings"

	"code.google.com/p/go-uuid/uuid"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/elb"
	"golang.org/x/net/context"
)

const (
	schemeInternal = "internal"
	schemeExternal = "internet-facing"
)

var defaultConnectionDrainingTimeout int64 = 30

var _ Manager = &ELBManager{}

// elbClient describes the interface from elb.ELB that we use.
type elbClient interface {
	CreateLoadBalancer(input *elb.CreateLoadBalancerInput) (*elb.CreateLoadBalancerOutput, error)
	ModifyLoadBalancerAttributes(input *elb.ModifyLoadBalancerAttributesInput) (*elb.ModifyLoadBalancerAttributesOutput, error)
	SetLoadBalancerListenerSSLCertificate(input *elb.SetLoadBalancerListenerSSLCertificateInput) (*elb.SetLoadBalancerListenerSSLCertificateOutput, error)
	DeleteLoadBalancer(input *elb.DeleteLoadBalancerInput) (*elb.DeleteLoadBalancerOutput, error)
	DescribeLoadBalancers(input *elb.DescribeLoadBalancersInput) (*elb.DescribeLoadBalancersOutput, error)
	DescribeTags(input *elb.DescribeTagsInput) (*elb.DescribeTagsOutput, error)
}

// ELBManager is an implementation of the Manager interface that creates Elastic
// Load Balancers.
type ELBManager struct {
	// The ID of the security group to assign to internal load balancers.
	InternalSecurityGroupID string

	// The ID of the security group to assign to external load balancers.
	ExternalSecurityGroupID string

	// The Subnet IDs to assign when creating internal load balancers.
	InternalSubnetIDs []string

	// The Subnet IDs to assign when creating external load balancers.
	ExternalSubnetIDs []string

	elb elbClient

	// Ports is the PortAllocator used to allocate ports to new load
	// balancers.
	Ports PortAllocator

	newName func() string
}

// NewELBManager returns a new ELBManager backed by the aws config.
func NewELBManager(p client.ConfigProvider) *ELBManager {
	return &ELBManager{
		elb:     elb.New(p),
		newName: newName,
	}
}

// CreateLoadBalancer creates a new ELB:
//
// * The ELB is created and connection draining is enabled.
// * An internal DNS CNAME record is created, pointing the the DNSName of the ELB.
func (m *ELBManager) CreateLoadBalancer(ctx context.Context, o CreateLoadBalancerOpts) (*LoadBalancer, error) {
	scheme := schemeInternal
	sg := m.InternalSecurityGroupID
	subnets := m.internalSubnets()

	if o.External {
		scheme = schemeExternal
		sg = m.ExternalSecurityGroupID
		subnets = m.externalSubnets()
	}

	// Allocate a new instance port for this load balancer.
	port, err := m.Ports.Get()
	if err != nil {
		return nil, err
	}

	input := &elb.CreateLoadBalancerInput{
		Listeners:        elbListeners(port, o.SSLCert),
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
				Enabled: aws.Bool(true),
				Timeout: aws.Int64(defaultConnectionDrainingTimeout),
			},
			CrossZoneLoadBalancing: &elb.CrossZoneLoadBalancing{
				Enabled: aws.Bool(true),
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
		InstancePort: port,
	}, nil
}

func (m *ELBManager) UpdateLoadBalancer(ctx context.Context, opts UpdateLoadBalancerOpts) error {
	if opts.SSLCert != nil {
		if err := m.updateSSLCert(ctx, opts.Name, *opts.SSLCert); err != nil {
			return err
		}
	}

	return nil
}

func (m *ELBManager) updateSSLCert(ctx context.Context, name, certID string) error {
	_, err := m.elb.SetLoadBalancerListenerSSLCertificate(&elb.SetLoadBalancerListenerSSLCertificateInput{
		LoadBalancerName: aws.String(name),
		LoadBalancerPort: aws.Int64(443),
		SSLCertificateId: aws.String(certID),
	})
	return err
}

// DestroyLoadBalancer destroys an ELB.
func (m *ELBManager) DestroyLoadBalancer(ctx context.Context, lb *LoadBalancer) error {
	if err := m.releasePorts(ctx, lb.Name); err != nil {
		return nil
	}

	_, err := m.elb.DeleteLoadBalancer(&elb.DeleteLoadBalancerInput{
		LoadBalancerName: aws.String(lb.Name),
	})

	return err
}

// releases any allocated ports for this load balancer.
func (m *ELBManager) releasePorts(ctx context.Context, loadBalancer string) error {
	resp, err := m.elb.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{aws.String(loadBalancer)},
	})
	if err != nil {
		return err
	}

	lb := resp.LoadBalancerDescriptions[0]

	for _, ld := range lb.ListenerDescriptions {
		if err := m.Ports.Put(*ld.Listener.InstancePort); err != nil {
			return err
		}
	}

	return nil
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
			Marker:   nextMarker,
			PageSize: aws.Int64(20), // Set this to 20, because DescribeTags has a limit of 20 on the LoadBalancerNames attribute.
		})
		if err != nil {
			return nil, err
		}

		if len(out.LoadBalancerDescriptions) == 0 {
			break
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
				lb := loadBalancerFromDescription(descs[*d.LoadBalancerName])
				lb.Tags = mapTags(d.Tags)

				lbs = append(lbs, lb)
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

func (m *ELBManager) internalSubnets() []*string {
	return awsStringSlice(m.InternalSubnetIDs)
}

func (m *ELBManager) externalSubnets() []*string {
	return awsStringSlice(m.ExternalSubnetIDs)
}

func loadBalancerFromDescription(elb *elb.LoadBalancerDescription) *LoadBalancer {
	var instancePort int64
	var sslCert string

	if len(elb.ListenerDescriptions) > 0 {
		instancePort = *elb.ListenerDescriptions[0].Listener.InstancePort
		for _, ld := range elb.ListenerDescriptions {
			if ld.Listener.SSLCertificateId != nil {
				sslCert = *ld.Listener.SSLCertificateId
			}
		}
	}

	return &LoadBalancer{
		Name:         *elb.LoadBalancerName,
		DNSName:      *elb.DNSName,
		External:     *elb.Scheme == schemeExternal,
		SSLCert:      sslCert,
		InstancePort: instancePort,
	}
}

func awsStringSlice(ss []string) []*string {
	as := make([]*string, len(ss))
	for i, s := range ss {
		as[i] = aws.String(s)
	}
	return as
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
			InstancePort:     aws.Int64(port),
			LoadBalancerPort: aws.Int64(80),
			Protocol:         aws.String("http"),
			InstanceProtocol: aws.String("http"),
		},
	}

	if certID != "" {
		listeners = append(listeners, &elb.Listener{
			InstancePort:     aws.Int64(port),
			LoadBalancerPort: aws.Int64(443),
			SSLCertificateId: aws.String(certID),
			Protocol:         aws.String("https"),
			InstanceProtocol: aws.String("http"),
		})
	}
	return listeners
}

// mapTags takes a list of []*elb.Tag's and converts them into a map[string]string
func mapTags(tags []*elb.Tag) map[string]string {
	tagMap := make(map[string]string)
	for _, t := range tags {
		tagMap[*t.Key] = *t.Value
	}

	return tagMap
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

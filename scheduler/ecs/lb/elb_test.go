package lb

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

func TestELB_CreateLoadBalancer(t *testing.T) {
	c := new(mockELBClient)
	m := newTestELBManager()
	m.elb = c

	c.On("CreateLoadBalancer", &elb.CreateLoadBalancerInput{
		LoadBalancerName: aws.String("acme-inc"),
		Scheme:           aws.String("internet-facing"),
		SecurityGroups:   []*string{aws.String("")},
		Subnets:          []*string{aws.String("public-subnet")},
		Listeners: []*elb.Listener{
			{
				InstancePort:     aws.Int64(9000),
				InstanceProtocol: aws.String("http"),
				LoadBalancerPort: aws.Int64(80),
				Protocol:         aws.String("http"),
			},
		},
	}).Return(&elb.CreateLoadBalancerOutput{
		DNSName: aws.String("acme-inc.us-east-1.elb.amazonaws.com"),
	}, nil)

	c.On("ModifyLoadBalancerAttributes", &elb.ModifyLoadBalancerAttributesInput{
		LoadBalancerName: aws.String("acme-inc"),
		LoadBalancerAttributes: &elb.LoadBalancerAttributes{
			ConnectionDraining: &elb.ConnectionDraining{
				Enabled: aws.Bool(true),
				Timeout: aws.Int64(30),
			},
			CrossZoneLoadBalancing: &elb.CrossZoneLoadBalancing{
				Enabled: aws.Bool(true),
			},
		},
	}).Return(&elb.ModifyLoadBalancerAttributesOutput{}, nil)

	lb, err := m.CreateLoadBalancer(context.Background(), CreateLoadBalancerOpts{
		External: true,
	})
	assert.NoError(t, err)
	assert.NotNil(t, lb)

	c.AssertExpectations(t)
}

func TestELB_UpdateLoadBalancer(t *testing.T) {
	c := new(mockELBClient)
	m := newTestELBManager()
	m.elb = c

	c.On("SetLoadBalancerListenerSSLCertificate", &elb.SetLoadBalancerListenerSSLCertificateInput{
		LoadBalancerName: aws.String("acme-inc"),
		LoadBalancerPort: aws.Int64(443),
		SSLCertificateId: aws.String("newcert"),
	}).Return(&elb.SetLoadBalancerListenerSSLCertificateOutput{}, nil)

	err := m.UpdateLoadBalancer(context.Background(), UpdateLoadBalancerOpts{
		Name:    "acme-inc",
		SSLCert: aws.String("newcert"),
	})
	assert.NoError(t, err)

	c.AssertExpectations(t)
}

func TestELB_DestroyLoadBalancer(t *testing.T) {
	c := new(mockELBClient)
	m := newTestELBManager()
	m.elb = c

	lb := &LoadBalancer{
		Name:         "acme-inc",
		DNSName:      "acme-inc.us-east-1.elb.amazonaws.com",
		InstancePort: 9000,
		External:     true,
		Tags:         map[string]string{AppTag: "acme-inc"},
	}

	c.On("DescribeLoadBalancers", &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{aws.String("acme-inc")},
	}).Return(&elb.DescribeLoadBalancersOutput{
		LoadBalancerDescriptions: []*elb.LoadBalancerDescription{
			{
				ListenerDescriptions: []*elb.ListenerDescription{
					{
						Listener: &elb.Listener{
							InstancePort: aws.Int64(9000),
						},
					},
				},
			},
		},
	}, nil)

	c.On("DeleteLoadBalancer", &elb.DeleteLoadBalancerInput{
		LoadBalancerName: aws.String("acme-inc"),
	}).Return(&elb.DeleteLoadBalancerOutput{}, nil)

	err := m.DestroyLoadBalancer(context.Background(), lb)
	assert.NoError(t, err)

	c.AssertExpectations(t)
}

func TestELBwDNS_DestroyLoadBalancer(t *testing.T) {
	c := new(mockELBClient)
	m := newTestELBManager()
	m.elb = c

	ns := newTestNameserver("FAKEZONE")

	m2 := WithCNAME(m, ns)

	lb := &LoadBalancer{
		Name:         "acme-inc",
		DNSName:      "acme-inc.us-east-1.elb.amazonaws.com",
		InstancePort: 9000,
		External:     true,
		Tags:         map[string]string{AppTag: "acme-inc"},
	}

	c.On("DescribeLoadBalancers", &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{aws.String("acme-inc")},
	}).Return(&elb.DescribeLoadBalancersOutput{
		LoadBalancerDescriptions: []*elb.LoadBalancerDescription{
			{
				ListenerDescriptions: []*elb.ListenerDescription{
					{
						Listener: &elb.Listener{
							InstancePort: aws.Int64(9000),
						},
					},
				},
			},
		},
	}, nil)

	c.On("DeleteLoadBalancer", &elb.DeleteLoadBalancerInput{
		LoadBalancerName: aws.String("acme-inc"),
	}).Return(&elb.DeleteLoadBalancerOutput{}, nil)

	err := m2.DestroyLoadBalancer(context.Background(), lb)
	assert.NoError(t, err)
	assert.True(t, ns.DeleteCNAMECalled)

	c.AssertExpectations(t)
}

func TestELB_LoadBalancers(t *testing.T) {
	c := new(mockELBClient)
	m := newTestELBManager()
	m.elb = c

	c.On("DescribeLoadBalancers", &elb.DescribeLoadBalancersInput{
		PageSize: aws.Int64(20),
	}).Return(&elb.DescribeLoadBalancersOutput{
		NextMarker: aws.String("abcd"),
		LoadBalancerDescriptions: []*elb.LoadBalancerDescription{
			{
				LoadBalancerName:  aws.String("foo"),
				DNSName:           aws.String("foo.us-east-1.elb.amazonaws.com"),
				VPCId:             aws.String("vpc-1"),
				SecurityGroups:    []*string{aws.String("sg-1")},
				AvailabilityZones: []*string{aws.String("us-east-1a")},
				Scheme:            aws.String("internal"),
				Subnets:           []*string{aws.String("subnet-1a")},
				ListenerDescriptions: []*elb.ListenerDescription{
					{
						Listener: &elb.Listener{
							InstancePort:     aws.Int64(9000),
							InstanceProtocol: aws.String("http"),
							LoadBalancerPort: aws.Int64(80),
							Protocol:         aws.String("http"),
						},
					},
				},
			},
		},
	}, nil)

	c.On("DescribeTags", &elb.DescribeTagsInput{
		LoadBalancerNames: []*string{aws.String("foo")},
	}).Return(&elb.DescribeTagsOutput{
		TagDescriptions: []*elb.TagDescription{
			{
				LoadBalancerName: aws.String("foo"),
				Tags: []*elb.Tag{
					{Key: aws.String("AppName"), Value: aws.String("foo")},
					{Key: aws.String("ProcessType"), Value: aws.String("web")},
				},
			},
		},
	}, nil)

	c.On("DescribeLoadBalancers", &elb.DescribeLoadBalancersInput{
		Marker:   aws.String("abcd"),
		PageSize: aws.Int64(20),
	}).Return(&elb.DescribeLoadBalancersOutput{
		LoadBalancerDescriptions: []*elb.LoadBalancerDescription{
			{
				LoadBalancerName:  aws.String("bar"),
				DNSName:           aws.String("bar.us-east-1.elb.amazonaws.com"),
				VPCId:             aws.String("vpc-1"),
				SecurityGroups:    []*string{aws.String("sg-1")},
				AvailabilityZones: []*string{aws.String("us-east-1a")},
				Scheme:            aws.String("internet-facing"),
				Subnets:           []*string{aws.String("subnet-1a")},
				ListenerDescriptions: []*elb.ListenerDescription{
					{
						Listener: &elb.Listener{
							InstancePort:     aws.Int64(9001),
							InstanceProtocol: aws.String("http"),
							LoadBalancerPort: aws.Int64(80),
							Protocol:         aws.String("http"),
						},
					},
				},
			},
		},
	}, nil)

	c.On("DescribeTags", &elb.DescribeTagsInput{
		LoadBalancerNames: []*string{aws.String("bar")},
	}).Return(&elb.DescribeTagsOutput{
		TagDescriptions: []*elb.TagDescription{
			{
				LoadBalancerName: aws.String("bar"),
				Tags: []*elb.Tag{
					{Key: aws.String("AppName"), Value: aws.String("bar")},
					{Key: aws.String("ProcessType"), Value: aws.String("web")},
				},
			},
		},
	}, nil)

	lbs, err := m.LoadBalancers(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(lbs))

	expected := []*LoadBalancer{
		{Name: "foo", DNSName: "foo.us-east-1.elb.amazonaws.com", InstancePort: 9000, Tags: map[string]string{"AppName": "foo", "ProcessType": "web"}},
		{Name: "bar", DNSName: "bar.us-east-1.elb.amazonaws.com", External: true, InstancePort: 9001, Tags: map[string]string{"AppName": "bar", "ProcessType": "web"}},
	}

	for i := range expected {
		assert.Equal(t, expected[i], lbs[i])
	}

	c.AssertExpectations(t)
}

func TestELB_EmptyLoadBalancers(t *testing.T) {
	c := new(mockELBClient)
	m := newTestELBManager()
	m.elb = c

	c.On("DescribeLoadBalancers", &elb.DescribeLoadBalancersInput{
		PageSize: aws.Int64(20),
	}).Return(&elb.DescribeLoadBalancersOutput{
		LoadBalancerDescriptions: []*elb.LoadBalancerDescription{},
	}, nil)

	lbs, err := m.LoadBalancers(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(lbs))

	c.AssertExpectations(t)
}

func newTestELBManager() *ELBManager {
	return &ELBManager{
		InternalSubnetIDs: []string{"private-subnet"},
		ExternalSubnetIDs: []string{"public-subnet"},
		Ports:             newPortAllocator(9000, 1),
		newName: func() string {
			return "acme-inc"
		},
	}
}

// fakeNameserver is a fake implementation of the Nameserver interface.
type fakeNameserver struct {
	ZoneID string

	CNAMECalled       bool
	DeleteCNAMECalled bool
}

func (n *fakeNameserver) CreateCNAME(cname, record string) error {
	n.CNAMECalled = true
	return nil
}

func (n *fakeNameserver) DeleteCNAME(cname, record string) error {
	n.DeleteCNAMECalled = true
	return nil
}

func newTestNameserver(zoneID string) *fakeNameserver {
	return &fakeNameserver{
		ZoneID:            zoneID,
		CNAMECalled:       false,
		DeleteCNAMECalled: false,
	}
}

type mockELBClient struct {
	elbClient
	mock.Mock
}

func (m *mockELBClient) CreateLoadBalancer(input *elb.CreateLoadBalancerInput) (*elb.CreateLoadBalancerOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*elb.CreateLoadBalancerOutput), args.Error(1)
}

func (m *mockELBClient) ModifyLoadBalancerAttributes(input *elb.ModifyLoadBalancerAttributesInput) (*elb.ModifyLoadBalancerAttributesOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*elb.ModifyLoadBalancerAttributesOutput), args.Error(1)
}

func (m *mockELBClient) SetLoadBalancerListenerSSLCertificate(input *elb.SetLoadBalancerListenerSSLCertificateInput) (*elb.SetLoadBalancerListenerSSLCertificateOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*elb.SetLoadBalancerListenerSSLCertificateOutput), args.Error(1)
}

func (m *mockELBClient) DescribeLoadBalancers(input *elb.DescribeLoadBalancersInput) (*elb.DescribeLoadBalancersOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*elb.DescribeLoadBalancersOutput), args.Error(1)
}

func (m *mockELBClient) DeleteLoadBalancer(input *elb.DeleteLoadBalancerInput) (*elb.DeleteLoadBalancerOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*elb.DeleteLoadBalancerOutput), args.Error(1)
}

func (m *mockELBClient) DescribeTags(input *elb.DescribeTagsInput) (*elb.DescribeTagsOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*elb.DescribeTagsOutput), args.Error(1)
}

type portAllocator struct {
	ports []int64
	taken map[int64]bool
}

func newPortAllocator(start int64, count int64) *portAllocator {
	var ports []int64
	for i := int64(0); i < count; i++ {
		ports = append(ports, i+start)
	}
	taken := make(map[int64]bool)
	for _, port := range ports {
		taken[port] = false
	}
	return &portAllocator{
		ports: ports,
		taken: taken,
	}
}

func (a *portAllocator) Get() (int64, error) {
	for _, port := range a.ports {
		if !a.taken[port] {
			a.taken[port] = true
			return port, nil
		}
	}

	panic("All ports taken")
}

func (a *portAllocator) Put(port int64) error {
	a.taken[port] = false
	return nil
}

package lb

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/remind101/empire/pkg/awsutil"
	"golang.org/x/net/context"
)

func TestELB_CreateLoadBalancer(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=CreateLoadBalancer&Listeners.member.1.InstancePort=9000&Listeners.member.1.InstanceProtocol=http&Listeners.member.1.LoadBalancerPort=80&Listeners.member.1.Protocol=http&LoadBalancerName=acme-inc&Scheme=internet-facing&SecurityGroups.member.1=&Subnets.member.1=public-subnet&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<?xml version="1.0"?>
<CreateLoadBalancerResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
	<DNSName>acme-inc.us-east-1.elb.amazonaws.com</DNSName>
</CreateLoadBalancerResponse>`,
			},
		},
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=ModifyLoadBalancerAttributes&LoadBalancerAttributes.ConnectionDraining.Enabled=true&LoadBalancerAttributes.ConnectionDraining.Timeout=30&LoadBalancerAttributes.CrossZoneLoadBalancing.Enabled=true&LoadBalancerName=acme-inc&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<?xml version="1.0"?>
<ModifyLoadBalancerAttributesResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
</ModifyLoadBalancerAttributesResponse>`,
			},
		},
	})
	m, s := newTestELBManager(h)
	defer s.Close()

	lb, err := m.CreateLoadBalancer(context.Background(), CreateLoadBalancerOpts{
		External: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := &LoadBalancer{
		Name:         "acme-inc",
		DNSName:      "acme-inc.us-east-1.elb.amazonaws.com",
		InstancePort: 9000,
		External:     true,
	}

	if got, want := lb, expected; !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadBalancer => %v; want %v", got, want)
	}
}

func TestELB_UpdateLoadBalancer(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=SetLoadBalancerListenerSSLCertificate&LoadBalancerName=loadbalancer&LoadBalancerPort=443&SSLCertificateId=newcert&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<?xml version="1.0"?>
<UpdateLoadBalancerResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
</UpdateLoadBalancerResponse>`,
			},
		},
	})
	m, s := newTestELBManager(h)
	defer s.Close()

	err := m.UpdateLoadBalancer(context.Background(), UpdateLoadBalancerOpts{
		Name:    "loadbalancer",
		SSLCert: aws.String("newcert"),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func buildLoadBalancerForDestroy() (*ELBManager, *httptest.Server, *LoadBalancer) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeLoadBalancers&PageSize=20&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<DescribeLoadBalancersResponse xmlns="http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/">
	  <DescribeLoadBalancersResult>
	    <NextMarker>
	      abcd
	    </NextMarker>
	    <LoadBalancerDescriptions>
	      <member>
	        <SecurityGroups>
	          <member>sg-1</member>
	        </SecurityGroups>
	        <LoadBalancerName>foo</LoadBalancerName>
		<DNSName>foo.us-east-1.elb.amazonaws.com</DNSName>
	        <VPCId>vpc-1</VPCId>
	        <ListenerDescriptions>
	          <member>
	            <PolicyNames/>
	            <Listener>
	              <Protocol>HTTP</Protocol>
	              <LoadBalancerPort>80</LoadBalancerPort>
	              <InstanceProtocol>HTTP</InstanceProtocol>
	              <InstancePort>9000</InstancePort>
	            </Listener>
	          </member>
	        </ListenerDescriptions>
	        <AvailabilityZones>
	          <member>us-east-1a</member>
	        </AvailabilityZones>
	        <Scheme>internal</Scheme>
	        <Subnets>
	          <member>subnet-1a</member>
	        </Subnets>
	      </member>
	    </LoadBalancerDescriptions>
	  </DescribeLoadBalancersResult>
	</DescribeLoadBalancersResponse>`,
			},
		},

		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DeleteLoadBalancer&LoadBalancerName=acme-inc&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<?xml version="1.0"?>
<DeleteLoadBalancerResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
</DeleteLoadBalancerResponse>`,
			},
		},
	})
	m, s := newTestELBManager(h)

	lb := &LoadBalancer{
		Name:         "acme-inc",
		DNSName:      "acme-inc.us-east-1.elb.amazonaws.com",
		InstancePort: 9000,
		External:     true,
		Tags:         map[string]string{AppTag: "acme-inc"},
	}
	return m, s, lb
}

func TestELB_DestroyLoadBalancer(t *testing.T) {
	m, s, lb := buildLoadBalancerForDestroy()
	defer s.Close()

	if err := m.DestroyLoadBalancer(context.Background(), lb); err != nil {
		t.Fatal(err)
	}
}

func TestELB_LoadBalancers(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeLoadBalancers&PageSize=20&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<DescribeLoadBalancersResponse xmlns="http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/">
	  <DescribeLoadBalancersResult>
	    <NextMarker>
	      abcd
	    </NextMarker>
	    <LoadBalancerDescriptions>
	      <member>
	        <SecurityGroups>
	          <member>sg-1</member>
	        </SecurityGroups>
	        <LoadBalancerName>foo</LoadBalancerName>
		<DNSName>foo.us-east-1.elb.amazonaws.com</DNSName>
	        <VPCId>vpc-1</VPCId>
	        <ListenerDescriptions>
	          <member>
	            <PolicyNames/>
	            <Listener>
	              <Protocol>HTTP</Protocol>
	              <LoadBalancerPort>80</LoadBalancerPort>
	              <InstanceProtocol>HTTP</InstanceProtocol>
	              <InstancePort>9000</InstancePort>
	            </Listener>
	          </member>
	        </ListenerDescriptions>
	        <AvailabilityZones>
	          <member>us-east-1a</member>
	        </AvailabilityZones>
	        <Scheme>internal</Scheme>
	        <Subnets>
	          <member>subnet-1a</member>
	        </Subnets>
	      </member>
	    </LoadBalancerDescriptions>
	  </DescribeLoadBalancersResult>
	</DescribeLoadBalancersResponse>`,
			},
		},
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeTags&LoadBalancerNames.member.1=foo&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<DescribeTagsResponse xmlns="http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/">
	  <DescribeTagsResult>
	    <TagDescriptions>
	      <member>
	        <Tags>
	          <member>
	            <Key>AppName</Key>
	            <Value>foo</Value>
	          </member>
	          <member>
	            <Key>ProcessType</Key>
	            <Value>web</Value>
	          </member>
	        </Tags>
	        <LoadBalancerName>foo</LoadBalancerName>
	      </member>
	    </TagDescriptions>
	  </DescribeTagsResult>
	</DescribeTagsResponse>`,
			},
		},
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeLoadBalancers&Marker=%0A%09++++++abcd%0A%09++++&PageSize=20&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<DescribeLoadBalancersResponse xmlns="http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/">
	  <DescribeLoadBalancersResult>
	    <NextMarker></NextMarker>
	    <LoadBalancerDescriptions>
	      <member>
	        <SecurityGroups>
	          <member>sg-1</member>
	        </SecurityGroups>
	        <LoadBalancerName>bar</LoadBalancerName>
		<DNSName>bar.us-east-1.elb.amazonaws.com</DNSName>
	        <VPCId>vpc-1</VPCId>
	        <ListenerDescriptions>
	          <member>
	            <PolicyNames/>
	            <Listener>
	              <Protocol>HTTP</Protocol>
	              <LoadBalancerPort>80</LoadBalancerPort>
	              <InstanceProtocol>HTTP</InstanceProtocol>
	              <InstancePort>9001</InstancePort>
	            </Listener>
	          </member>
	        </ListenerDescriptions>
	        <AvailabilityZones>
	          <member>us-east-1a</member>
	        </AvailabilityZones>
	        <Scheme>internet-facing</Scheme>
	        <Subnets>
	          <member>subnet-1a</member>
	        </Subnets>
	      </member>
	    </LoadBalancerDescriptions>
	  </DescribeLoadBalancersResult>
	</DescribeLoadBalancersResponse>`,
			},
		},
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeTags&LoadBalancerNames.member.1=bar&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<DescribeTagsResponse xmlns="http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/">
	  <DescribeTagsResult>
	    <TagDescriptions>
	      <member>
	        <Tags>
	          <member>
	            <Key>AppName</Key>
	            <Value>bar</Value>
	          </member>
	          <member>
	            <Key>ProcessType</Key>
	            <Value>web</Value>
	          </member>
	        </Tags>
	        <LoadBalancerName>bar</LoadBalancerName>
	      </member>
	    </TagDescriptions>
	  </DescribeTagsResult>
	</DescribeTagsResponse>`,
			},
		},
	})
	m, s := newTestELBManager(h)
	defer s.Close()

	lbs, err := m.LoadBalancers(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(lbs), 2; got != want {
		t.Fatalf("%v load balancers; want %v", got, want)
	}

	expected := []*LoadBalancer{
		{Name: "foo", DNSName: "foo.us-east-1.elb.amazonaws.com", InstancePort: 9000, Tags: map[string]string{"AppName": "foo", "ProcessType": "web"}},
		{Name: "bar", DNSName: "bar.us-east-1.elb.amazonaws.com", External: true, InstancePort: 9001, Tags: map[string]string{"AppName": "bar", "ProcessType": "web"}},
	}

	if got, want := lbs, expected; !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadBalancers => %v; want %v", got, want)
	}
}

func TestELB_EmptyLoadBalancers(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeLoadBalancers&PageSize=20&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<DescribeLoadBalancersResponse xmlns="http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/">
	  <DescribeLoadBalancersResult>
	    <LoadBalancerDescriptions/>
	  </DescribeLoadBalancersResult>
	</DescribeLoadBalancersResponse>`,
			},
		},
	})
	m, s := newTestELBManager(h)
	defer s.Close()

	lbs, err := m.LoadBalancers(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(lbs), 0; got != want {
		t.Fatalf("%v load balancers; want %v", got, want)
	}
}

func TestELBwDNS_DestroyLoadBalancer(t *testing.T) {
	m, s, lb := buildLoadBalancerForDestroy()
	defer s.Close()
	ns := newTestNameserver("FAKEZONE")

	m2 := WithCNAME(m, ns)

	if err := m2.DestroyLoadBalancer(context.Background(), lb); err != nil {
		t.Fatal(err)
	}

	if ok := ns.DeleteCNAMECalled; !ok {
		t.Fatal("DeleteCNAME was not called.")
	}

}

func newTestELBManager(h http.Handler) (*ELBManager, *httptest.Server) {
	s := httptest.NewServer(h)

	config := &aws.Config{
		Credentials: credentials.NewStaticCredentials(" ", " ", " "),
		Endpoint:    aws.String(s.URL),
		Region:      aws.String("localhost"),
	}
	config.WithLogLevel(0)

	m := NewELBManager(session.New(config))
	m.newName = func() string {
		return "acme-inc"
	}
	m.InternalSubnetIDs = []string{"private-subnet"}
	m.ExternalSubnetIDs = []string{"public-subnet"}
	m.Ports = newPortAllocator(9000, 1)

	return m, s
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

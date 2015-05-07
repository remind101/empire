package lb

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/remind101/empire/empire/pkg/awsutil"
	"golang.org/x/net/context"
)

func TestELB_CreateLoadBalancer(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=CreateLoadBalancer&Listeners.member.1.InstancePort=0&Listeners.member.1.InstanceProtocol=http&Listeners.member.1.LoadBalancerPort=80&Listeners.member.1.Protocol=http&LoadBalancerName=acme-inc&Scheme=internet-facing&SecurityGroups.member.1=&Subnets.member.1=subnet&Version=2012-06-01`,
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
				Body:       `Action=ModifyLoadBalancerAttributes&LoadBalancerAttributes.ConnectionDraining.Enabled=true&LoadBalancerAttributes.ConnectionDraining.Timeout=300&LoadBalancerName=acme-inc&Version=2012-06-01`,
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
		Name:     "acme-inc",
		External: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := &LoadBalancer{
		Name:     "acme-inc",
		DNSName:  "acme-inc.us-east-1.elb.amazonaws.com",
		External: true,
	}

	if got, want := lb, expected; !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadBalancer => %v; want %v", got, want)
	}
}

func TestELB_DestroyLoadBalancer(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
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
	defer s.Close()

	if err := m.DestroyLoadBalancer(context.Background(), "acme-inc"); err != nil {
		t.Fatal(err)
	}
}

func TestELB_LoadBalancers(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeLoadBalancers&Version=2012-06-01`,
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
	              <InstancePort>8080</InstancePort>
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
				Body:       `Action=DescribeLoadBalancers&Marker=%0A%09++++++abcd%0A%09++++&Version=2012-06-01`,
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
	              <InstancePort>8080</InstancePort>
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
		{Name: "foo", DNSName: "foo.us-east-1.elb.amazonaws.com"},
		{Name: "bar", DNSName: "bar.us-east-1.elb.amazonaws.com", External: true},
	}

	if got, want := lbs, expected; !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadBalancers => %v; want %v", got, want)
	}
}

func TestVPCSubnetFinder(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeSubnets&Filter.1.Name=vpc-id&Filter.1.Value.1=&Version=2014-10-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<?xml version="1.0" encoding="UTF-8"?>
	<DescribeSubnetsResponse xmlns="http://ec2.amazonaws.com/doc/2015-03-01/">
	    <requestId>fd72c284-0fb5-45c1-a149-dbe7ed8e034a</requestId>
	    <subnetSet>
	        <item>
	            <subnetId>subnet-a</subnetId>
	            <state>available</state>
	            <vpcId>vpc-1</vpcId>
	            <cidrBlock>10.0.0.0/24</cidrBlock>
	            <availableIpAddressCount>249</availableIpAddressCount>
	            <availabilityZone>us-east-1a</availabilityZone>
	            <defaultForAz>false</defaultForAz>
	            <mapPublicIpOnLaunch>false</mapPublicIpOnLaunch>
	        </item>
	    </subnetSet>
	</DescribeSubnetsResponse>`,
			},
		},
	})
	f, s := newTestVPCSubnetFinder(h)
	defer s.Close()

	subnets, err := f.Subnets()
	if err != nil {
		t.Fatal(err)
	}

	if got, want := subnets, []string{"subnet-a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Subnets => %v; want %v", got, want)
	}
}

func newTestVPCSubnetFinder(h http.Handler) (*VPCSubnetFinder, *httptest.Server) {
	s := httptest.NewServer(h)

	f := NewVPCSubnetFinder(
		aws.DefaultConfig.Merge(&aws.Config{
			Credentials: aws.Creds("", "", ""),
			Endpoint:    s.URL,
			Region:      "localhost",
			LogLevel:    0,
		}),
	)

	return f, s
}

func newTestELBManager(h http.Handler) (*ELBManager, *httptest.Server) {
	s := httptest.NewServer(h)

	m := NewELBManager(
		aws.DefaultConfig.Merge(&aws.Config{
			Credentials: aws.Creds("", "", ""),
			Endpoint:    s.URL,
			Region:      "localhost",
			LogLevel:    0,
		}),
	)
	m.SubnetFinder = StaticSubnets([]string{"subnet"})

	return m, s
}

// fakeNameserver is a fake implementation of the Nameserver interface.
type fakeNameserver struct{}

func (n *fakeNameserver) CNAME(cname, record string) error {
	return nil
}

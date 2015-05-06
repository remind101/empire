package lb

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/remind101/empire/empire/pkg/awsutil"
)

func TestELB_CreateLoadBalancer(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=CreateLoadBalancer&Listeners.member.1.InstancePort=0&Listeners.member.1.InstanceProtocol=http&Listeners.member.1.LoadBalancerPort=80&Listeners.member.1.Protocol=http&LoadBalancerName=acme-inc&Scheme=internal&SecurityGroups.member.1=&Subnets.member.1=10.0.0.0%2F24&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<?xml version="1.0"?>
<CreateLoadBalancerResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
	<DNSName>acme-inc.us-east-1.elb.amazonaws.com</DNSName>
</CreateLoadBalancerResponse>`,
			},
		},
	})
	m, s := newTestELBManager(h)
	defer s.Close()

	_, err := m.CreateLoadBalancer(CreateLoadBalancerOpts{
		Name: "acme-inc",
	})
	if err != nil {
		t.Fatal(err)
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

	if err := m.DestroyLoadBalancer("acme-inc"); err != nil {
		t.Fatal(err)
	}
}

func TestELB_LoadBalancers(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeLoadBalancers&Marker=&Version=2012-06-01`,
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

	_, err := m.LoadBalancers(nil)
	if err != nil {
		t.Fatal(err)
	}
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
	m.SubnetFinder = StaticSubnets([]string{"10.0.0.0/24"})

	return m, s
}

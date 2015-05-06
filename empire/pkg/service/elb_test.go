package service

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/remind101/empire/empire/pkg/awsutil"
	"github.com/remind101/pkg/logger"
	"golang.org/x/net/context"
)

// A new service, needs new load balancer.
func TestECSWithELBManager_Submit_Create(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeSubnets&Filter.1.Name=vpc-id&Filter.1.Value.1=vpc-1&Version=2014-10-01`,
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
	            <cidrBlock>10.0.1.0/24</cidrBlock>
	            <availableIpAddressCount>249</availableIpAddressCount>
	            <availabilityZone>us-east-1a</availabilityZone>
	            <defaultForAz>false</defaultForAz>
	            <mapPublicIpOnLaunch>false</mapPublicIpOnLaunch>
	        </item>
	    </subnetSet>
	</DescribeSubnetsResponse>`,
			},
		},

		// No existing load balancers
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeLoadBalancers&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<DescribeLoadBalancersResponse xmlns="http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/">
	  <DescribeLoadBalancersResult>
	    <LoadBalancerDescriptions></LoadBalancerDescriptions>
	  </DescribeLoadBalancersResult>
	</DescribeLoadBalancersResponse>`,
			},
		},

		// Scale previous service to 0 (in this case there is no previous process)
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.UpdateService",
				Body:       `{"cluster":"","desiredCount":0,"service":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 400,
				Body:       `{"__type":"ClientException","message":"Service not found."}`,
			},
		},

		// Create new load balancer
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=CreateLoadBalancer&Listeners.member.1.InstancePort=8080&Listeners.member.1.InstanceProtocol=HTTP&Listeners.member.1.LoadBalancerPort=80&Listeners.member.1.Protocol=HTTP&LoadBalancerName=foo--web&Scheme=internal&SecurityGroups.member.1=internal-sg&Subnets.member.1=subnet-a&Tags.member.1.Key=AppName&Tags.member.1.Value=foo&Tags.member.2.Key=ProcessType&Tags.member.2.Value=web&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<?xml version="1.0"?>
<CreateLoadBalancerResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <DNSName>foo--web.us-east-1.elb.amazonaws.com</DNSName>
</CreateLoadBalancerResponse>`,
			},
		},

		// Update zone records
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: `/2013-04-01/hostedzonesbyname?dnsname=empire.`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<?xml version="1.0"?>
<ListHostedZonesByNameResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <HostedZones>
    <HostedZone>
      <Id>/hostedzone/ABCD</Id>
      <Name>empire.</Name>
    </HostedZone>
  </HostedZones>
  <DNSName>empire</DNSName>
</ListHostedZonesByNameResponse>`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: `/2013-04-01/hostedzone/ABCD/rrset`,
				// Body:       `<ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><ChangeBatch><Changes><Change><Action>UPSERT</Action><ResourceRecordSet><Name>foo.empire.</Name><ResourceRecords><ResourceRecord><Value>foo--web.us-east-1.elb.amazonaws.com</Value></ResourceRecord></ResourceRecords><TTL>60</TTL><Type>CNAME</Type></ResourceRecordSet></Change></Changes><Comment>Update for empire foo app</Comment></ChangeBatch></ChangeResourceRecordSetsRequest>`,
				Body: `ignore`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       ``,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.ListServices",
				Body:       `{"cluster":""}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"serviceArns":[]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition",
				Body:       `{"containerDefinitions":[{"cpu":128,"command":["acme-inc","web"],"environment":[{"name":"USER","value":"foo"}],"essential":true,"image":"remind101/acme-inc:latest","memory":128,"name":"web","portMappings":[{"containerPort":8080,"hostPort":8080}]}],"family":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       "",
			},
		},

		// We try to update first, if that fails with service not found, we try to create.
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.UpdateService",
				Body:       `{"cluster":"","desiredCount":0,"service":"foo--web","taskDefinition":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 400,
				Body:       `{"__type":"ClientException","message":"Service not found."}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.CreateService",
				Body:       `{"cluster":"","desiredCount":0,"loadBalancers":[{"containerName":"web","containerPort":8080,"loadBalancerName":"foo--web"}],"role":"ecsServiceRole","serviceName":"foo--web","taskDefinition":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       "",
			},
		},
	})

	m, s := newTestECSWithELBManager(h)
	defer s.Close()

	if err := m.Submit(newTestContext(), fakeApp); err != nil {
		t.Fatal(err)
	}
}

// A previous load balancer exists and needs to be recreated.
func TestECSWithELBManager_Submit_Recreate(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeSubnets&Filter.1.Name=vpc-id&Filter.1.Value.1=vpc-1&Version=2014-10-01`,
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
	            <cidrBlock>10.0.1.0/24</cidrBlock>
	            <availableIpAddressCount>249</availableIpAddressCount>
	            <availabilityZone>us-east-1a</availabilityZone>
	            <defaultForAz>false</defaultForAz>
	            <mapPublicIpOnLaunch>false</mapPublicIpOnLaunch>
	        </item>
	    </subnetSet>
	</DescribeSubnetsResponse>`,
			},
		},

		// Existing load balancer, publicly exposed
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeLoadBalancers&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<DescribeLoadBalancersResponse xmlns="http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/">
	  <DescribeLoadBalancersResult>
	    <LoadBalancerDescriptions>
	      <member>
	        <SecurityGroups>
	          <member>sg-1</member>
	        </SecurityGroups>
	        <LoadBalancerName>foo--web</LoadBalancerName>
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

		// Tags for previous load balancer
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeTags&LoadBalancerNames.member.1=foo--web&Version=2012-06-01`,
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
	        <LoadBalancerName>foo--web</LoadBalancerName>
	      </member>
	    </TagDescriptions>
	  </DescribeTagsResult>
	</DescribeTagsResponse>`,
			},
		},

		// Scale previous service to 0
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.UpdateService",
				Body:       `{"cluster":"","desiredCount":0,"service":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"Service":{}}`,
			},
		},

		// Delete the service
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DeleteService",
				Body:       `{"cluster":"","service":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"Service":{}}`,
			},
		},

		// Delete old load balancer
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DeleteLoadBalancer&LoadBalancerName=foo--web&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{}`,
			},
		},

		// Create new load balancer
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=CreateLoadBalancer&Listeners.member.1.InstancePort=8080&Listeners.member.1.InstanceProtocol=HTTP&Listeners.member.1.LoadBalancerPort=80&Listeners.member.1.Protocol=HTTP&LoadBalancerName=foo--web&Scheme=internal&SecurityGroups.member.1=internal-sg&Subnets.member.1=subnet-a&Tags.member.1.Key=AppName&Tags.member.1.Value=foo&Tags.member.2.Key=ProcessType&Tags.member.2.Value=web&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<?xml version="1.0"?>
<CreateLoadBalancerResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <DNSName>foo--web.us-east-1.elb.amazonaws.com</DNSName>
</CreateLoadBalancerResponse>`,
			},
		},

		// Update zone records
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: `/2013-04-01/hostedzonesbyname?dnsname=empire.`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<?xml version="1.0"?>
<ListHostedZonesByNameResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <HostedZones>
    <HostedZone>
      <Id>/hostedzone/ABCD</Id>
      <Name>empire.</Name>
    </HostedZone>
  </HostedZones>
  <DNSName>empire</DNSName>
</ListHostedZonesByNameResponse>`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: `/2013-04-01/hostedzone/ABCD/rrset`,
				// Body:       `<ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><ChangeBatch><Changes><Change><Action>UPSERT</Action><ResourceRecordSet><Name>foo.empire.</Name><ResourceRecords><ResourceRecord><Value>foo--web.us-east-1.elb.amazonaws.com</Value></ResourceRecord></ResourceRecords><TTL>60</TTL><Type>CNAME</Type></ResourceRecordSet></Change></Changes><Comment>Update for empire foo app</Comment></ChangeBatch></ChangeResourceRecordSetsRequest>`,
				Body: `ignore`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       ``,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.ListServices",
				Body:       `{"cluster":""}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"serviceArns":[]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition",
				Body:       `{"containerDefinitions":[{"cpu":128,"command":["acme-inc","web"],"environment":[{"name":"USER","value":"foo"}],"essential":true,"image":"remind101/acme-inc:latest","memory":128,"name":"web","portMappings":[{"containerPort":8080,"hostPort":8080}]}],"family":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       "",
			},
		},

		// We try to update first, if that fails with service not found, we try to create.
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.UpdateService",
				Body:       `{"cluster":"","desiredCount":0,"service":"foo--web","taskDefinition":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 400,
				Body:       `{"__type":"ClientException","message":"Service not found."}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.CreateService",
				Body:       `{"cluster":"","desiredCount":0,"loadBalancers":[{"containerName":"web","containerPort":8080,"loadBalancerName":"foo--web"}],"role":"ecsServiceRole","serviceName":"foo--web","taskDefinition":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       "",
			},
		},
	})

	m, s := newTestECSWithELBManager(h)
	defer s.Close()

	if err := m.Submit(newTestContext(), fakeApp); err != nil {
		t.Fatal(err)
	}
}

// A previous load balancer exists but does not need to be recreated.
func TestECSWithELBManager_Submit_No_Recreate(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeSubnets&Filter.1.Name=vpc-id&Filter.1.Value.1=vpc-1&Version=2014-10-01`,
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
            <cidrBlock>10.0.1.0/24</cidrBlock>
            <availableIpAddressCount>249</availableIpAddressCount>
            <availabilityZone>us-east-1a</availabilityZone>
            <defaultForAz>false</defaultForAz>
            <mapPublicIpOnLaunch>false</mapPublicIpOnLaunch>
        </item>
    </subnetSet>
</DescribeSubnetsResponse>`,
			},
		},

		// Existing load balancer.
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeLoadBalancers&Version=2012-06-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: `<DescribeLoadBalancersResponse xmlns="http://elasticloadbalancing.amazonaws.com/doc/2012-06-01/">
  <DescribeLoadBalancersResult>
    <LoadBalancerDescriptions>
      <member>
        <SecurityGroups>
          <member>internal-sg</member>
        </SecurityGroups>
        <LoadBalancerName>foo--web</LoadBalancerName>
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
          <member>subnet-a</member>
        </Subnets>
      </member>
    </LoadBalancerDescriptions>
  </DescribeLoadBalancersResult>
</DescribeLoadBalancersResponse>`,
			},
		},

		// Tags for previous load balancer
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Body:       `Action=DescribeTags&LoadBalancerNames.member.1=foo--web&Version=2012-06-01`,
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
        <LoadBalancerName>foo--web</LoadBalancerName>
      </member>
    </TagDescriptions>
  </DescribeTagsResult>
</DescribeTagsResponse>`,
			},
		},

		// Previous service exists
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.ListServices",
				Body:       `{"cluster":""}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"serviceArns":["arn:aws:ecs:us-east-1a:249285743859:service/foo--web"]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeServices",
				Body:       `{"cluster":"","services":["arn:aws:ecs:us-east-1a:249285743859:service/foo--web"]}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"services":[{"desiredCount":1,"taskDefinition":"foo--web"}]}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition",
				Body:       `{"taskDefinition":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskDefinition":{"containerDefinitions":[{"cpu":128,"command":["acme-inc","web"],"environment":[{"name":"USER","value":"foo"}],"essential":true,"image":"remind101/acme-inc:latest","memory":128,"name":"web"}]}}`,
			},
		},

		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition",
				Body:       `{"containerDefinitions":[{"cpu":128,"command":["acme-inc","web"],"environment":[{"name":"USER","value":"foo"}],"essential":true,"image":"remind101/acme-inc:latest","memory":128,"name":"web","portMappings":[{"containerPort":8080,"hostPort":8080}]}],"family":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       "",
			},
		},

		// We try to update first, if that fails with service not found, we try to create.
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.UpdateService",
				Body:       `{"cluster":"","desiredCount":1,"service":"foo--web","taskDefinition":"foo--web"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"service":{"desiredCount":1,"taskDefinition":"foo--web"}}`,
			},
		},
	})

	m, s := newTestECSWithELBManager(h)
	defer s.Close()

	fakeApp.Processes[0].Instances = 1

	if err := m.Submit(newTestContext(), fakeApp); err != nil {
		t.Fatal(err)
	}
}

func newTestContext() context.Context {
	c := context.Background()
	c = logger.WithLogger(c, logger.New(log.New(os.Stdout, "", 0)))
	return c
}

func newTestECSWithELBManager(h http.Handler) (*ECSWithELBManager, *httptest.Server) {
	s := httptest.NewServer(h)

	m := NewECSWithELBManager(
		aws.DefaultConfig.Merge(&aws.Config{
			Credentials: aws.Creds("", "", ""),
			Endpoint:    s.URL,
			Region:      "localhost",
			LogLevel:    0,
		}),
	)
	m.VPCID = "vpc-1"
	m.InternalSecurityGroupID = "internal-sg"
	m.ExternalSecurityGroupID = "external-sg"

	return m, s
}

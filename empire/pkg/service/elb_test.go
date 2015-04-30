package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/remind101/empire/empire/pkg/awsutil"
	"golang.org/x/net/context"
)

func TestECSWithELBManager_Submit_Create(t *testing.T) {
	h := awsutil.NewHandler(map[awsutil.Request]awsutil.Response{
		awsutil.Request{
			Body: `Action=DescribeSubnets&Filter.1.Name=vpc-id&Filter.1.Value.1=ECSVPC&Version=2014-10-01`,
		}: awsutil.Response{
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

		// No existing load balancers
		awsutil.Request{
			Body: `Action=DescribeLoadBalancers&Version=2012-06-01`,
		}: awsutil.Response{
			StatusCode: 200,
			Body:       `{"LoadBalancerDescriptions":[]}`,
		},

		// Create new load balancer
		awsutil.Request{
			Body: `Action=CreateLoadBalancer&AvailabilityZones.member.1=us-east-1a&Listeners.member.1.InstancePort=8080&Listeners.member.1.InstanceProtocol=http&Listeners.member.1.LoadBalancerPort=80&Listeners.member.1.Protocol=http&LoadBalancerName=foo--web&Scheme=internal&SecurityGroups.member.1=ECSLoadBalancerSG&Subnets.member.1=subnet-a&Tags.member.1.Key=AppName&Tags.member.1.Value=foo&Tags.member.2.Key=ProcessType&Tags.member.2.Value=web&Version=2012-06-01`,
		}: awsutil.Response{
			StatusCode: 200,
			Body:       `{"DNSName": "foo--web.us-east-1.elb.amazonaws.com"}`,
		},

		// Scale previous service to 0 (in this case there is no previous process)
		awsutil.Request{
			Operation: "AmazonEC2ContainerServiceV20141113.UpdateService",
			Body:      `{"cluster":"","desiredCount":0,"service":"foo--web"}`,
		}: awsutil.Response{
			StatusCode: 400,
			Body:       `{"__type":"ClientException","message":"Service not found."}`,
		},

		awsutil.Request{
			Operation: "AmazonEC2ContainerServiceV20141113.ListServices",
			Body:      `{"cluster":""}`,
		}: awsutil.Response{
			StatusCode: 200,
			Body:       `{"serviceArns":[]}`,
		},

		awsutil.Request{
			Operation: "AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition",
			Body:      `{"containerDefinitions":[{"cpu":128,"command":["acme-inc","web"],"environment":[{"name":"USER","value":"foo"}],"essential":true,"image":"remind101/acme-inc:latest","memory":128,"name":"web","portMappings":[{"containerPort":8080,"hostPort":8080}]}],"family":"foo--web"}`,
		}: awsutil.Response{
			StatusCode: 200,
			Body:       "",
		},

		// We try to update first, if that fails with service not found, we try to create.
		awsutil.Request{
			Operation: "AmazonEC2ContainerServiceV20141113.UpdateService",
			Body:      `{"cluster":"","desiredCount":0,"service":"foo--web","taskDefinition":"foo--web"}`,
		}: awsutil.Response{
			StatusCode: 400,
			Body:       `{"__type":"ClientException","message":"Service not found."}`,
		},

		awsutil.Request{
			Operation: "AmazonEC2ContainerServiceV20141113.CreateService",
			Body:      `{"cluster":"","desiredCount":0,"loadBalancers":[{"containerName":"web","containerPort":8080,"loadBalancerName":"foo--web"}],"serviceName":"foo--web","taskDefinition":"foo--web"}`,
		}: awsutil.Response{
			StatusCode: 200,
			Body:       "",
		},
	})

	m, s := newTestECSWithELBManager(h)
	defer s.Close()

	if err := m.Submit(context.Background(), fakeApp); err != nil {
		t.Fatal(err)
	}

}
func newTestECSWithELBManager(h http.Handler) (*ECSWithELBManager, *httptest.Server) {
	s := httptest.NewServer(h)

	return NewECSWithELBManager(
		aws.DefaultConfig.Merge(&aws.Config{
			Credentials: aws.Creds("", "", ""),
			Endpoint:    s.URL,
			Region:      "localhost",
		}),
		&ELBConfig{
			VPCID:           "ECSVPC",
			SecurityGroupID: "ECSLoadBalancerSG",
		},
	), s
}

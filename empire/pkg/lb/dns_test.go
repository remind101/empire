package lb

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/remind101/empire/empire/pkg/awsutil"
)

func TestRoute53_CNAME(t *testing.T) {
	h := awsutil.NewHandler([]awsutil.Cycle{
		{
			Request: awsutil.Request{
				RequestURI: "/2013-04-01/hostedzonesbyname?dnsname=empire.",
				Body:       ``,
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
		{
			Request: awsutil.Request{
				RequestURI: `/2013-04-01/hostedzone/ABCD/rrset`,
				Body:       `ignore`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       ``,
			},
		},
	})
	n, s := newTestRoute53Nameserver(h)
	defer s.Close()

	if err := n.CNAME("acme-inc", "123456789.us-east-1.elb.amazonaws.com"); err != nil {
		t.Fatal(err)
	}
}

func newTestRoute53Nameserver(h http.Handler) (*Route53Nameserver, *httptest.Server) {
	s := httptest.NewServer(h)

	n := NewRoute53Nameserver(
		aws.DefaultConfig.Merge(&aws.Config{
			Credentials: credentials.NewStaticCredentials(" ", " ", " "),
			Endpoint:    s.URL,
			Region:      "localhost",
			LogLevel:    0,
		}),
	)
	n.Zone = "empire."

	return n, s
}

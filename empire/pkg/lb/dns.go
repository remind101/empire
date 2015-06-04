package lb

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

// errHostedZone is returned when the hosted zone is not found.
var errHostedZone = errors.New("hosted zone not found, unable to update records")

// Nameserver represents a service for creating dns records.
type Nameserver interface {
	// CNAME creates a cname record pointed at record.
	CNAME(cname, record string) error
}

// Route53Nameserver is an implementation of the nameserver interface backed by
// route53.
type Route53Nameserver struct {
	// The Hosted Zone that records will be created under.
	Zone   string
	ZoneID string

	route53 *route53.Route53
}

// NewRoute53Nameserver returns a Route53Nameserver instance with a configured
// route53 client.
func NewRoute53Nameserver(c *aws.Config) *Route53Nameserver {
	return &Route53Nameserver{
		route53: route53.New(c),
	}
}

// CNAME creates a CNAME record under the HostedZone specified by Zone.
func (n *Route53Nameserver) CNAME(cname, record string) error {
	zone, err := n.zone()
	if err != nil {
		return err
	}

	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				&route53.Change{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(fmt.Sprintf("%s.%s", cname, *zone.Name)),
						Type: aws.String("CNAME"),
						ResourceRecords: []*route53.ResourceRecord{
							&route53.ResourceRecord{
								Value: aws.String(record),
							},
						},
						TTL: aws.Long(60),
					},
				},
			},
		},
		HostedZoneID: zone.ID,
	}
	_, err = n.route53.ChangeResourceRecordSets(input)
	return err
}

func fixHostedZoneIDPrefix(zoneID string) *string {
	prefix := "/hostedzone/"
	s := zoneID
	if ok := strings.HasPrefix(zoneID, prefix); !ok {
		s = strings.Join([]string{prefix, zoneID}, "")
	}
	return &s
}

// zone returns the HostedZone for the Zone.
// TODO: Deal w/ pagination of results from AWS API, shouldn't have to worry for most cases since using ByName returns the list with that name first.
func (n *Route53Nameserver) zone() (*route53.HostedZone, error) {
	zid := fixHostedZoneIDPrefix(n.ZoneID)
	out, err := n.route53.GetHostedZone(&route53.GetHostedZoneInput{ID: zid})
	if err != nil {
		return nil, err
	}

	return out.HostedZone, nil
}

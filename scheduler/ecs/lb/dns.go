package lb

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/route53"
)

// Nameserver represents a service for creating dns records.
type Nameserver interface {
	// CNAME creates a cname record pointed at record.
	CreateCNAME(cname, record string) error
	DeleteCNAME(cname, record string) error
}

// Route53Nameserver is an implementation of the nameserver interface backed by
// route53.
type Route53Nameserver struct {
	// The Hosted Zone ID that records will be created under.
	ZoneID string

	route53 *route53.Route53
}

// NewRoute53Nameserver returns a Route53Nameserver instance with a configured
// route53 client.
func NewRoute53Nameserver(p client.ConfigProvider) *Route53Nameserver {
	return &Route53Nameserver{
		route53: route53.New(p),
	}
}

// CreateCNAME creates a CNAME record under the HostedZone specified by ZoneID.
func (n *Route53Nameserver) CreateCNAME(cname, record string) error {
	zone, err := n.zone()
	if err != nil {
		return err
	}

	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				&route53.Change{
					Action:            aws.String("UPSERT"),
					ResourceRecordSet: newCNAMERecordSet(fmt.Sprintf("%s.%s", cname, *zone.Name), record, 60),
				},
			},
		},
		HostedZoneId: zone.Id,
	}
	_, err = n.route53.ChangeResourceRecordSets(input)
	return err
}

// DeleteCNAME deletes the CNAME of an ELB from the internal zone
func (n *Route53Nameserver) DeleteCNAME(cname, record string) error {
	zone, err := n.zone()
	if err != nil {
		return err
	}

	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				&route53.Change{
					Action:            aws.String("DELETE"),
					ResourceRecordSet: newCNAMERecordSet(fmt.Sprintf("%s.%s", cname, *zone.Name), record, 60),
				},
			},
		},
		HostedZoneId: zone.Id,
	}
	_, err = n.route53.ChangeResourceRecordSets(input)
	return err
}

func newCNAMERecordSet(cname string, target string, ttl int64) *route53.ResourceRecordSet {
	return &route53.ResourceRecordSet{
		Name: aws.String(cname),
		Type: aws.String("CNAME"),
		ResourceRecords: []*route53.ResourceRecord{
			&route53.ResourceRecord{
				Value: aws.String(target),
			},
		},
		TTL: aws.Int64(ttl),
	}
}

func fixHostedZoneIDPrefix(zoneID string) *string {
	prefix := "/hostedzone/"
	s := zoneID
	if ok := strings.HasPrefix(zoneID, prefix); !ok {
		s = strings.Join([]string{prefix, zoneID}, "")
	}
	return &s
}

// zone returns the HostedZone for the ZoneID.
func (n *Route53Nameserver) zone() (*route53.HostedZone, error) {
	zid := fixHostedZoneIDPrefix(n.ZoneID)
	out, err := n.route53.GetHostedZone(&route53.GetHostedZoneInput{Id: zid})
	if err != nil {
		return nil, err
	}

	return out.HostedZone, nil
}

package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/awslabs/aws-sdk-go/service/ecs"
	"github.com/awslabs/aws-sdk-go/service/elb"
	"github.com/awslabs/aws-sdk-go/service/route53"
	"github.com/remind101/pkg/logger"
	"golang.org/x/net/context"
)

const (
	ProtocolHTTP  = "HTTP"
	ProtocolHTTPS = "HTTPS"
)

// The role to assign ECS load balancers.
var ECSServiceRole = "ecsServiceRole"

// The default zone to create CNAME records for internal services.
var DefaultZone = "empire."

// ECSWithELBManager wraps ECSManager and manages load
// balancing for the service with ELB.
type ECSWithELBManager struct {
	*ECSManager

	ec2   *ec2.EC2
	VPCID string

	elb                     *elb.ELB
	InternalSecurityGroupID string
	ExternalSecurityGroupID string

	route53 *route53.Route53
	Zone    string
}

func NewECSWithELBManager(c *aws.Config) *ECSWithELBManager {
	m := &ECSWithELBManager{
		ECSManager: NewECSManager(c),
		elb:        elb.New(c),
		ec2:        ec2.New(c),
		route53:    route53.New(c),
	}

	m.Zone = DefaultZone

	return m
}

// Submit will create an internal load balancer if the app contains a web process. It will
// also create a CNAME record named after the app that points to the load balancer.
//
// If the app has domains associated with it, the load balancer and service
// will be recreated, and the load balancer will be made public.
func (m *ECSWithELBManager) Submit(ctx context.Context, app *App) error {
	for _, p := range app.Processes {
		if p.Exposure > ExposeNone {
			logger.Info(ctx, "process exposure greater than none, updating load balancer", "app", app.Name, "process", p.Type)

			err := m.updateLoadBalancer(ctx, app, p)
			if err != nil {
				return err
			}
		}
	}

	return m.ECSManager.Submit(ctx, app)
}

// Remove removes any ECS services that belong to this app, along with any associated load balancers.
func (m *ECSWithELBManager) Remove(ctx context.Context, app string) error {
	processes, err := m.listProcesses(app)
	if err != nil {
		return err
	}

	for t, _ := range processTypes(processes) {
		if err := m.removeProcess(ctx, app, t); err != nil {
			return err
		}

		lb, err := m.findLoadBalancer(app, t)
		if err != nil {
			return err
		}

		if _, err := m.elb.DeleteLoadBalancer(&elb.DeleteLoadBalancerInput{LoadBalancerName: lb.LoadBalancerName}); err != nil {
			return err
		}

	}

	return nil
}

// updateLoadBalancer determines if the app process needs a new load balancer, creates one, and decorates
// the process with the load balancer information. If a previous load balancer exists, it will be
// removed along with existing process.
func (m *ECSWithELBManager) updateLoadBalancer(ctx context.Context, app *App, process *Process) error {
	recreate := true

	// Build input for load balancer
	input, err := m.loadBalancerInputFromApp(ctx, app, process)
	if err != nil {
		return err
	}

	// Look for existing load balancer
	prev, err := m.findLoadBalancer(app.Name, process.Type)
	logger.Info(ctx, "looking for existing load balancer", "err", err, "app", app.Name, "process", process.Type)
	if err != nil {
		return err
	}

	// If one exists, build input from previous load balancer and compare to current input.
	// If they differ, we need to recreate the load balancer and service.
	if prev != nil {
		prevInput := m.loadBalancerInputFromDesc(prev, m.loadBalancerTags(app.Name, process.Type))

		if reflect.DeepEqual(input, prevInput) {
			logger.Info(ctx, "previous load balancer exists, and is up to date.", "app", app.Name, "process", process.Type)
			recreate = false
		} else {
			jsonInput, _ := json.Marshal(input)
			jsonPrevInput, _ := json.Marshal(prevInput)
			logger.Info(ctx, "previous load balancer is stale, recreating", "app", app.Name, "process", process.Type, "input", string(jsonInput), "prevInput", string(jsonPrevInput))
		}
	}

	// If we need to recreate, first create the new load balancer, then destroy the old load balancer and process.
	if recreate {
		// Remove process
		err := m.removeProcess(ctx, app.Name, process.Type)
		logger.Info(ctx, "removing previous process", "err", err, "app", app.Name, "process", process.Type)
		if err != nil {
			return err
		}

		// Remove previous load balancer
		if prev != nil {
			_, err := m.elb.DeleteLoadBalancer(&elb.DeleteLoadBalancerInput{LoadBalancerName: prev.LoadBalancerName})
			logger.Info(ctx, "removing previous load balancer", "err", err, "app", app.Name, "process", process.Type)

			if err != nil {
				return err
			}
		}

		// Create new load balancer
		out, err := m.elb.CreateLoadBalancer(input)
		logger.Info(ctx, "creating new load balancer", "err", err, "app", app.Name, "process", process.Type)
		if err != nil {
			return err
		}

		// Update record set
		err = m.updateRecordSet(app.Name, out.DNSName)
		logger.Info(ctx, "updating zone records", "err", err, "app", app.Name, "process", process.Type, "lb", input.LoadBalancerName)
		if err != nil {
			return err
		}
	}

	if process.Attributes == nil {
		process.Attributes = make(map[string]interface{})
	}

	process.Attributes["Role"] = aws.String(ECSServiceRole)
	process.Attributes["LoadBalancers"] = []*ecs.LoadBalancer{
		{
			ContainerName:    aws.String(process.Type),
			ContainerPort:    process.Ports[0].Container,
			LoadBalancerName: input.LoadBalancerName,
		},
	}

	return nil
}

// loadBalancerInputFromApp returns a CreateLoadBalanerInput based on an App and Process.
func (m *ECSWithELBManager) loadBalancerInputFromApp(ctx context.Context, app *App, process *Process) (*elb.CreateLoadBalancerInput, error) {
	name := app.Name + "--" + process.Type

	subnets, err := m.subnets()
	if err != nil {
		return nil, err
	}

	scheme := ""
	sg := m.ExternalSecurityGroupID
	if process.Exposure == ExposePrivate {
		scheme = "internal"
		sg = m.InternalSecurityGroupID
	}

	listeners := []*elb.Listener{
		{
			InstancePort:     aws.Long(*process.Ports[0].Host),
			LoadBalancerPort: aws.Long(80),
			Protocol:         aws.String(ProtocolHTTP),
			InstanceProtocol: aws.String(ProtocolHTTP),
		},
	}

	input := &elb.CreateLoadBalancerInput{
		Listeners:        listeners,
		LoadBalancerName: aws.String(name),
		Scheme:           aws.String(scheme),
		SecurityGroups:   []*string{aws.String(sg)},
		Subnets:          subnets,
		Tags:             m.loadBalancerTags(app.Name, process.Type),
	}

	return input, nil
}

// loadBalancerInputFromDesc returns a CreateLoadBalancerInput based on a LoadBalancerDescription.
func (m *ECSWithELBManager) loadBalancerInputFromDesc(desc *elb.LoadBalancerDescription, tags []*elb.Tag) *elb.CreateLoadBalancerInput {
	listeners := make([]*elb.Listener, len(desc.ListenerDescriptions))
	for i, l := range desc.ListenerDescriptions {
		listeners[i] = l.Listener
	}

	return &elb.CreateLoadBalancerInput{
		Listeners:        listeners,
		LoadBalancerName: desc.LoadBalancerName,
		Scheme:           desc.Scheme,
		SecurityGroups:   desc.SecurityGroups,
		Subnets:          desc.Subnets,
		Tags:             tags,
	}
}

// findLoadBalancer returns the load balancer tagged with the given app name and process type.
func (m *ECSWithELBManager) findLoadBalancer(app string, process string) (*elb.LoadBalancerDescription, error) {
	lbs, err := m.findLoadBalancersByTags(m.loadBalancerTags(app, process))
	if err != nil {
		return nil, err
	}

	if len(lbs) == 0 {
		return nil, nil
	}

	return lbs[0], nil
}

// findLoadBalancersByTags returns a list of load balancers tagged with the given tag list.
func (m *ECSWithELBManager) findLoadBalancersByTags(tags []*elb.Tag) ([]*elb.LoadBalancerDescription, error) {
	lbs := make([]*elb.LoadBalancerDescription, 0)
	nextMarker := aws.String("")

	// Iterate through all the load balancers.
	for i := 0; i == 0 || *nextMarker != ""; i++ {
		out, err := m.elb.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{})
		if err != nil {
			return lbs, err
		}

		if len(out.LoadBalancerDescriptions) == 0 {
			continue
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
				lbs = append(lbs, descs[*d.LoadBalancerName])
			}
		}

		nextMarker = out.NextMarker
		if nextMarker == nil {
			nextMarker = aws.String("")
		}
	}

	return lbs, nil
}

// subnets returns a list of subnets within the given VPC.
func (m *ECSWithELBManager) subnets() (subnets []*string, err error) {
	subnetout, err := m.ec2.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(m.VPCID)}},
		},
	})
	if err != nil {
		return
	}

	for _, s := range subnetout.Subnets {
		subnets = append(subnets, s.SubnetID)
	}

	return
}

func (m *ECSWithELBManager) loadBalancerTags(app string, process string) []*elb.Tag {
	return []*elb.Tag{
		{
			Key:   aws.String("AppName"),
			Value: aws.String(app),
		},
		{
			Key:   aws.String("ProcessType"),
			Value: aws.String(process),
		},
	}
}

// updateRecordSet updates the internal zone to include a CNAME for the app,
// pointed at its load balancer.
func (m *ECSWithELBManager) updateRecordSet(app string, dnsName *string) error {
	var zone *route53.HostedZone
	out, err := m.route53.ListHostedZonesByName(&route53.ListHostedZonesByNameInput{DNSName: aws.String(m.Zone)})
	if err != nil {
		return err
	}

	for _, z := range out.HostedZones {
		if *z.Name == m.Zone {
			zone = z
		}
	}

	if zone == nil {
		return errors.New("hosted zone not found, unable to update records")
	}

	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				&route53.Change{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(fmt.Sprintf("%s.%s", app, m.Zone)),
						Type: aws.String("CNAME"),
						ResourceRecords: []*route53.ResourceRecord{
							&route53.ResourceRecord{
								Value: dnsName,
							},
						},
						TTL: aws.Long(60),
					},
				},
			},
			Comment: aws.String(fmt.Sprintf("Update for empire %s app", app)),
		},
		HostedZoneID: zone.ID,
	}
	_, err = m.route53.ChangeResourceRecordSets(input)
	return err
}

func containsTags(a []*elb.Tag, b []*elb.Tag) bool {
	for _, t := range a {
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

func inspectTags(tags []*elb.Tag) string {
	s := make([]string, len(tags))
	for i, t := range tags {
		s[i] = inspectTag(t)
	}
	return strings.Join(s, ", ")
}

func inspectTag(t *elb.Tag) string {
	return fmt.Sprintf("<Key: %s, Value: %s>", *t.Key, *t.Value)
}

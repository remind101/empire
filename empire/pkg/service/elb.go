package service

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/awslabs/aws-sdk-go/service/ecs"
	"github.com/awslabs/aws-sdk-go/service/elb"
	"golang.org/x/net/context"
)

var ECSServiceRole = "ecsServiceRole"

// ECSWithELBManager wraps ECSManager and manages load
// balancing for the service with ELB.
type ECSWithELBManager struct {
	*ECSManager
	elb                     *elb.ELB
	ec2                     *ec2.EC2
	VPCID                   string
	InternalSecurityGroupID string
	ExternalSecurityGroupID string
}

func NewECSWithELBManager(c *aws.Config) *ECSWithELBManager {
	return &ECSWithELBManager{
		ECSManager: NewECSManager(c),
		elb:        elb.New(c),
		ec2:        ec2.New(c),
	}
}

// Submit will create an internal ELB if the app contains a web process. It will
// also create a CNAME named after the app that points to the load balancer.
//
// If the app has domains associated with it, the load balancer and service
// will be recreated, and the load balancer will be made public.
func (m *ECSWithELBManager) Submit(ctx context.Context, app *App) error {
	for _, p := range app.Processes {
		if p.Exposure > ExposeNone {
			err := m.updateLoadBalancer(ctx, app, p)
			if err != nil {
				return err
			}
		}
	}

	return m.ECSManager.Submit(ctx, app)
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
	prev, err := m.findLoadBalancer(app, process)
	if err != nil {
		return err
	}

	// If one exists, build input from previous load balancer and compare to current input.
	// If they differ, we need to recreate the load balancer and service.
	if prev != nil {
		prevInput := m.loadBalancerInputFromDesc(prev, m.loadBalancerTags(app, process))
		if reflect.DeepEqual(input, prevInput) {
			recreate = false
		}
	}

	// If we need to recreate, first create the new load balancer, then destroy the old load balancer and process.
	if recreate {
		// Remove process
		if err := m.removeProcess(ctx, app.Name, process.Type); err != nil {
			return err
		}

		// Remove previous load balancer
		if prev != nil {
			if _, err := m.elb.DeleteLoadBalancer(&elb.DeleteLoadBalancerInput{LoadBalancerName: prev.LoadBalancerName}); err != nil {
				return err
			}
		}

		// Create new load balancer
		if _, err := m.elb.CreateLoadBalancer(input); err != nil {
			return err
		}
	}

	if process.Attributes == nil {
		process.Attributes = make(map[string]interface{})
	}

	process.Attributes["Role"] = ECSServiceRole
	process.Attributes["LoadBalancers"] = []*ecs.LoadBalancer{
		{
			ContainerName:    aws.String(process.Type),
			ContainerPort:    process.Ports[0].Host,
			LoadBalancerName: input.LoadBalancerName,
		},
	}

	return nil
}

// loadBalancerInputFromApp returns a CreateLoadBalanerInput based on an App and Process.
func (m *ECSWithELBManager) loadBalancerInputFromApp(ctx context.Context, app *App, process *Process) (*elb.CreateLoadBalancerInput, error) {
	name := app.Name + "--" + process.Type

	subnets, zones, err := m.subnetsAndZones()
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
			Protocol:         aws.String("http"),
			InstanceProtocol: aws.String("http"),
		},
	}

	input := &elb.CreateLoadBalancerInput{
		Listeners:         listeners,
		LoadBalancerName:  aws.String(name),
		AvailabilityZones: zones,
		Scheme:            aws.String(scheme),
		SecurityGroups:    []*string{aws.String(sg)},
		Subnets:           subnets,
		Tags:              m.loadBalancerTags(app, process),
	}

	return input, nil
}

func (m *ECSWithELBManager) loadBalancerInputFromDesc(desc *elb.LoadBalancerDescription, tags []*elb.Tag) *elb.CreateLoadBalancerInput {
	listeners := make([]*elb.Listener, len(desc.ListenerDescriptions))
	for i, l := range desc.ListenerDescriptions {
		listeners[i] = l.Listener
	}

	return &elb.CreateLoadBalancerInput{
		Listeners:         listeners,
		LoadBalancerName:  desc.LoadBalancerName,
		AvailabilityZones: desc.AvailabilityZones,
		Scheme:            desc.Scheme,
		SecurityGroups:    desc.SecurityGroups,
		Subnets:           desc.Subnets,
		Tags:              tags,
	}
}

func (m *ECSWithELBManager) findLoadBalancer(app *App, process *Process) (*elb.LoadBalancerDescription, error) {
	lbs, err := m.findLoadBalancersByTags(m.loadBalancerTags(app, process))
	if err != nil {
		return nil, err
	}

	if len(lbs) == 0 {
		return nil, nil
	}

	return lbs[0], nil
}

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

func (m *ECSWithELBManager) subnetsAndZones() (subnets []*string, zones []*string, err error) {
	subnetout, err := m.ec2.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(m.VPCID)}},
		},
	})
	if err != nil {
		return
	}

	for _, s := range subnetout.Subnets {
		zones = append(zones, s.AvailabilityZone)
		subnets = append(subnets, s.SubnetID)
	}

	return
}

func (m *ECSWithELBManager) loadBalancerTags(app *App, process *Process) []*elb.Tag {
	return []*elb.Tag{
		{
			Key:   aws.String("AppName"),
			Value: aws.String(app.Name),
		},
		{
			Key:   aws.String("ProcessType"),
			Value: aws.String(process.Type),
		},
	}
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

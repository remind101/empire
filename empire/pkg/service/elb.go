package service

import (
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
	elb             *elb.ELB
	ec2             *ec2.EC2
	VPCID           string
	SecurityGroupID string
}

type ELBConfig struct {
	// The Amazon VPC ID.
	VPCID string

	// The Security Group ID to assign when creating new load balancers.
	SecurityGroupID string
}

func NewECSWithELBManager(c *aws.Config, ec *ELBConfig) *ECSWithELBManager {
	return &ECSWithELBManager{
		ECSManager:      NewECSManager(c),
		elb:             elb.New(c),
		ec2:             ec2.New(c),
		VPCID:           ec.VPCID,
		SecurityGroupID: ec.SecurityGroupID,
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
	// prev, err := m.findLoadBalancer(app, process)
	name, err := m.createLoadbalancer(ctx, app, process)
	if err != nil {
		return err
	}

	if process.Attributes == nil {
		process.Attributes = make(map[string]interface{})
	}
	process.Attributes["Role"] = ECSServiceRole
	process.Attributes["LoadBalancers"] = []*ecs.LoadBalancer{
		&ecs.LoadBalancer{
			ContainerName:    aws.String(process.Type),
			ContainerPort:    process.Ports[0].Host,
			LoadBalancerName: aws.String(name),
		},
	}

	return nil
}

func (m *ECSWithELBManager) createLoadbalancer(ctx context.Context, app *App, process *Process) (string, error) {
	input, err := m.loadBalancerInputFromApp(ctx, app, process)
	if err != nil {
		return "", err
	}

	if _, err := m.elb.CreateLoadBalancer(input); err != nil {
		return "", err
	}

	return *input.LoadBalancerName, nil
}

// loadBalancerInputFromApp returns a CreateLoadBalanerInput based on an App and Process.
func (m *ECSWithELBManager) loadBalancerInputFromApp(ctx context.Context, app *App, process *Process) (*elb.CreateLoadBalancerInput, error) {
	name := app.Name + "--" + process.Type
	subnets := []*string{}
	zones := []*string{}

	subnetout, err := m.ec2.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{aws.String(m.VPCID)}},
		},
	})
	if err != nil {
		return nil, err
	}

	for _, s := range subnetout.Subnets {
		zones = append(zones, s.AvailabilityZone)
		subnets = append(subnets, s.SubnetID)
	}

	scheme := ""
	if process.Exposure == ExposePrivate {
		scheme = "internal"
	}

	input := &elb.CreateLoadBalancerInput{
		Listeners: []*elb.Listener{
			{
				InstancePort:     aws.Long(*process.Ports[0].Host),
				LoadBalancerPort: aws.Long(80),
				Protocol:         aws.String("http"),
				InstanceProtocol: aws.String("http"),
			},
		},
		LoadBalancerName:  aws.String(name),
		AvailabilityZones: zones,
		Scheme:            aws.String(scheme),
		SecurityGroups: []*string{
			aws.String(m.SecurityGroupID),
		},
		Subnets: subnets,
		Tags:    m.loadBalancerTags(app, process),
	}

	return input, nil
}

func (m *ECSWithELBManager) loadBalancerInputFromDesc(*elb.LoadBalancerDescription) (*elb.CreateLoadBalancerInput, error) {
	return nil, nil
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

		// Create a names slice and descriptions map.
		names := make([]*string, len(out.LoadBalancerDescriptions))
		descs := map[*string]*elb.LoadBalancerDescription{}

		for i, d := range out.LoadBalancerDescriptions {
			names[i] = d.LoadBalancerName
			descs[d.LoadBalancerName] = d
		}

		// Find all the tags for this batch of load balancers.
		out2, err := m.elb.DescribeTags(&elb.DescribeTagsInput{LoadBalancerNames: names})
		if err != nil {
			return lbs, err
		}

		// Append matching load balancers to our result set.
		for _, d := range out2.TagDescriptions {
			if containsTags(tags, d.Tags) {
				lbs = append(lbs, descs[d.LoadBalancerName])
			}
		}

		nextMarker = out.NextMarker
	}

	return lbs, nil
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
		if t.Key == t2.Key && t.Value == t2.Value {
			return true
		}
	}
	return false
}

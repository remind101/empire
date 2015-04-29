package service

import (
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/awslabs/aws-sdk-go/service/elb"
	"golang.org/x/net/context"
)

var ECSServiceRole = "ecsServiceRole"

// ECSWithELBManager wraps ECSManager and manages load
// balancing for the service with ELB.
type ECSWithELBManager struct {
	*ECSManager
	elb *elb.ELB
	ec2 *ec2.EC2

	// The Amazon VPC ID.
	VPCID string

	// The Security Group ID to assign when creating new load balancers.
	SecurityGroupID string
}

func NewECSWithELBManager(config *aws.Config, vpc string, sg string) *ECSWithELBManager {
	return &ECSWithELBManager{
		ECSManager:      NewECSManager(config),
		elb:             elb.New(config),
		ec2:             ec2.New(config),
		VPCID:           vpc,
		SecurityGroupID: sg,
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
			m.ensureLoadBalancer(ctx, app, p)
		}
	}

	return m.ECSManager.Submit(ctx, app)
}

func (m *ECSWithELBManager) Remove(ctx context.Context, app string) error {
	return nil
}

func (m *ECSWithELBManager) ensureLoadBalancer(ctx context.Context, app *App, process *Process) error {
	name, err := m.createLoadbalancer(ctx, app, process)
	if err != nil {
		return err
	}

	process.Attributes["Role"] = ECSServiceRole
	process.Attributes["LoadBalancers"] = []*ecs.LoadBalancers{
		&ecs.LoadBalancer{
			ContainerName:    aws.String(process.Type),
			ContainerPort:    aws.String(process.Ports[0].Host),
			LoadBalancerName: aws.String(name),
		},
	}

	return nil
}

func (m *ECSWithELBManager) createLoadbalancer(ctx context.Context, app *App, process *Process) (string, error) {
	name := app.Name + "--" + process.Type
	subnets := []*string{}
	zones := []*string{}

	subnetout, err := m.ec2.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{Name: aws.String("vpc-id"), Values: []*string{aws.String(m.VPCID)}},
		},
	})
	if err != nil {
		return name, err
	}

	for _, s := range subnetout.Subnets {
		zones = append(zones, s.AvailabilityZone)
		subnets = append(subnets, s.SubnetID)
	}

	scheme := ""
	if process.Exposure == ExposePrivate {
		scheme = "internal"
	}

	params := &elb.CreateLoadBalancerInput{
		Listeners: []*elb.Listener{
			&elb.Listener{
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
		Tags: []*elb.Tag{
			&elb.Tag{
				Key:   aws.String("AppName"),
				Value: aws.String(app.Name),
			},
			&elb.Tag{
				Key:   aws.String("ProcessType"),
				Value: aws.String(process.Type),
			},
			&elb.Tag{
				Key:   aws.String("ECSServiceName"),
				Value: aws.String(name),
			},
		},
	}

	if _, err := m.elb.CreateLoadBalancer(params); err != nil {
		return name, err
	}

	return name, nil
}

func (m *ECSWithELBManager) findLoadBalancer(app *App, process Process) (*elb.LoadBalancerDescription, error) {
	return nil, nil
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

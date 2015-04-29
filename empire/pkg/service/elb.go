package service

import (
	"fmt"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/elb"
	"golang.org/x/net/context"
)

var ECSServiceRole = "ecsServiceRole"

// ECSWithELBManager wraps ECSManager and manages load
// balancing for the service with ELB.
type ECSWithELBManager struct {
	*ECSManager
	elb *elb.ELB
}

func NewECSWithELBManager(config *aws.Config) *ECSWithELBManager {
	return &ECSWithELBManager{
		ECSManager: NewECSManager(config),
		elb:        elb.New(config),
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
	process.Attributes["Role"] = ECSServiceRole
	return nil
}

func (m *ECSWithELBManager) createLoadbalancer(ctx context.Context, app *App, process *Process) error {
	// zones:= DescribeAvailabilityZones()
	zones := []*string{
		aws.String("AvailabilityZone"), // Required
		// More values...
	}

	// subnets := DescribeSubnets()
	subnets := []*string{
		aws.String("SubnetId"), // Required
		// More values...
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
		LoadBalancerName:  aws.String(app.Name + "--" + process.Type),
		AvailabilityZones: zones,
		Scheme:            aws.String(scheme),
		SecurityGroups: []*string{
			aws.String("SecurityGroupId"),
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
		},
	}

	_, err := m.elb.CreateLoadBalancer(params)
	if awserr := aws.Error(err); awserr != nil {
		// A service error occurred.
		fmt.Println("Error:", awserr.Code, awserr.Message)
		return err
	} else if err != nil {
		// A non-service error occurred.
		panic(err)
	}

	return nil
}

// findLoadBalancerForService(appname string, processtype string) *elb.LoadBalancerDescription
// findLoadBalancersByTags([]elb.Tag) []*elb.LoadBalancerDescription

func (m *ECSWithELBManager) findLoadBalancersByTags(tags []*elb.Tag) ([]*elb.LoadBalancerDescription, error) {
	lbs := make([]*elb.LoadBalancerDescription, 0)
	nextMarker = ""

	// Iterate through all the load balancers.
	for i := 0; i == 0 || nextMarker != ""; i++ {
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
		out2, err := m.elb.DescribeTags(elb.DescribeTagsInput{LoadBalancerNames: names})
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
	for t := range a {
		if !containsTag(t, b) {
			return false
		}
	}
	return true
}

func containsTag(t elb.Tag, tags []*elb.Tag) bool {
	for _, t2 := range tags {
		if t.Key == t2.Key && t.Value == t2.Value {
			return true
		}
	}
	return false
}

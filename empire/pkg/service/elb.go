package service

import (
	"fmt"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/elb"
	"golang.org/x/net/context"
)

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
func (s *ECSWithELBManager) Submit(ctx context.Context, app *App) error {
	for _, p := range app.Processes {
		if p.Exposure > ExposeNone {
			s.ensureLoadBalancer(ctx, app, p)
		}
	}

	return s.ECSManager.Submit(ctx, app)
}

func (s *ECSWithELBManager) Remove(ctx context.Context, app string) error {
	return nil
}

func (s *ECSWithELBManager) ensureLoadBalancer(ctx context.Context, app *App, process *Process) error {
	return nil
}

func (s *ECSWithELBManager) createLoadbalancer(ctx context.Context, app *App, process *Process) error {
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
		LoadBalancerName:  aws.String(app.Name + "-" + process.Type),
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

	_, err := s.elb.CreateLoadBalancer(params)
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

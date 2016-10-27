package boot

import (
	"context"
	"testing"

	"github.com/remind101/empire"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestBoot(t *testing.T) {
	config := new(Config)
	e, err := Boot(config)
	assert.NoError(t, err)
	assert.Nil(t, e.Empire.Scheduler)
	assert.Nil(t, e.Empire.LogsStreamer)
	assert.Nil(t, e.Empire.RunRecorder)
	assert.NotNil(t, e.Empire.EventStream)
	assert.NotNil(t, e.Empire.ProcfileExtractor)
	assert.Equal(t, empire.AllowCommandAny, e.Empire.AllowedCommands)
}

// TODO: Mock call to get route53 hosted zone id.
func testBoot_CloudFormationScheduler(t *testing.T) {
	config := new(Config)
	config.Scheduler.Backend = String("cloudformation")
	config.Scheduler.CloudFormation.VpcID = String("vpc-d315edb4")
	config.Scheduler.CloudFormation.Route53InternalHostedZoneID = String("Z185KSIEQC21FF")
	config.Scheduler.CloudFormation.TemplateBucket = String("empire-77792028-templatebucket-195oucd149ybu")
	config.Scheduler.CloudFormation.ELBPrivateSecurityGroup = String("sg-f33ef988")
	config.Scheduler.CloudFormation.EC2PrivateSubnets = StringSlice([]string{"subnet-d280dfa4", "subnet-89402ad1"})
	config.Scheduler.CloudFormation.EC2PublicSubnets = StringSlice([]string{"subnet-d280dfa4", "subnet-89402ad1"})
	config.Scheduler.CloudFormation.ELBPrivateSecurityGroup = String("sg-fa3ef981")
	config.Scheduler.CloudFormation.ECSCluster = String("empire-77792028-Cluster-1CU7HL67LPPHO")
	config.Scheduler.CloudFormation.ECSServiceRole = String("empire-77792028-ServiceRole-1SD04YIKP9AIP")
	config.CloudFormationCustomResources.Topic = String("arn:aws:sns:us-east-1:066251891493:empire-77792028-CustomResourcesTopic-SEBA731MZZU5")

	_, err := Boot(config)
	assert.NoError(t, err)
}

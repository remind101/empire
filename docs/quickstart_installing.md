# Empire :: Installing

**IMPORTANT:** Use the following document as a quick way to try empire. This
method is not suggested for a production environment, and should not be
considered secure. There will be further docs describing best practices for
production environments.

### Prerequisites

This guide assumes that you have the following installed:

* **Amazon EC2 Key Pair**: Make sure you've created an Amazon EC2 Key Pair. See
  [creating or importing a keypair][keypair] for more information.

* **Docker Hub Credentials** (optional): If you want to deploy an image from a
  private repository, you'll need to provide your [Docker Hub][dockerhub]
  email address, username, and password to the script in step 3. Empire uses
  these credentials to authenticate with Docker Hub to pull images for private
  repositories.

## Step 1 - ECS AMI

Before doing any of the following, log in to your AWS account and
[accept the terms and conditions for the official ECS AMI][amiterms].

If you don't do this, no EC2 instances will be started by the auto scaling
group that our CloudFormation stack creates.

Also, check that the offical ECS AMI ID for US East matches with the one in
[cloudformation.json][democloud].

## Step 2 - Create CloudFormation stack

[![Install](https://s3.amazonaws.com/cloudformation-examples/cloudformation-launch-stack.png)](https://console.aws.amazon.com/cloudformation/home?region=us-east-1#cstack=sn%7Eempire%7Cturl%7Ehttps://s3.amazonaws.com/empirepaas/cloudformation.json)

This is a very simple stack that will:

* Create a VPC with 2 subnets.
* Create an EC2 Instance Profile with the required permissions for the
  [ECS agent][ecsagent].
* Create a Launch Configuration and Auto Scaling Group that will use the
  official ECS AMI.
* Create an ECS Cluster and Service for Empire.
* Configure the instances to be able to pull from a private registry. (If
  docker credentials were provided).

## Step 3 - Get the emp client

The last thing you need to do is download the Empire client, **emp**. Refer to the [README][empclient] for instructions on how to install it.

[awscli]: http://aws.amazon.com/cli/
[keypair]: http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html
[dockerhub]: https://hub.docker.com/
[amiterms]: https://aws.amazon.com/marketplace/ordering?productId=4ce33fd9-63ff-4f35-8d3a-939b641f1931&ref_=dtl_psb_continue&region=us-east-1
[democloud]: https://github.com/remind101/empire/blob/master/docs/cloudformation.json#L15
[ecsagent]: https://github.com/aws/amazon-ecs-agent
[empclient]: https://github.com/remind101/empire/tree/master/cmd/emp#installation

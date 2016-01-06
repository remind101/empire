# Empire :: Installing

**IMPORTANT:** Use the following document as a quick way to try empire. This
method is not suggested for a production environment, and should not be
considered secure. There will be further docs describing best practices for
production environments.

### Prerequisites

This guide assumes that you have the following installed:

* **AWS CLI**: You can find the instructions at
  [http://aws.amazon.com/cli/][cli]. You'll need a fairly recent version of the
  CLI that has support for ECS.

```console
$ pip install --upgrade awscli
```

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

## Step 2 - Clone the empire repo

In order to run the script and cloudformation template for this guide, you'll
need to clone this repository.

```console
$ git clone https://github.com/remind101/empire.git
$ cd empire
```

## Step 3 - Create CloudFormation stack

Create a new CloudFormation stack using the [bootstrap](../bin/bootstrap)
script.

```console
$ ./bin/bootstrap
AWS SSH KeyName: default
Do you have a docker account & want to use it for private repo access? [y/N] n
==> Launching empire in AZs: us-east-1a us-east-1b, Cloudformation Stack empire-1a96c6f3
==> Waiting for stack to complete
==> Status: CREATE_IN_PROGRESS
==> Stack empire-1a96c6f3 complete. Now run the following commands - when asked for a username, enter 'fake'. The password is blank:
$ export EMPIRE_API_URL=http://empire-1a-LoadBala-EC3V01X8GHOO-1318261069.us-east-1.elb.amazonaws.com/
$ emp login
```

This is a very simple stack that will:

* Create a VPC with 2 subnets.
* Create an EC2 Instance Profile with the required permissions for the
  [ECS agent][ecsagent].
* Create a a Launch Configuration and Auto Scaling Group that will use the
  official ECS AMI.
* Create an ECS Cluster and Service for Empire.
* Configure the instances to be able to pull from a private registry. (If
  docker credentials were provided).

## Step 4 - Get the emp client

The last thing you need to do is download the Empire client, **emp**. Refer to the [README][empclient] for instructions on how to install it.

[awscli]: http://aws.amazon.com/cli/
[keypair]: http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html
[dockerhub]: https://hub.docker.com/
[amiterms]: https://aws.amazon.com/marketplace/ordering?productId=4ce33fd9-63ff-4f35-8d3a-939b641f1931&ref_=dtl_psb_continue&region=us-east-1
[democloud]: https://github.com/remind101/empire/blob/master/docs/cloudformation.json#L15
[ecsagent]: https://github.com/aws/amazon-ecs-agent
[empclient]: https://github.com/remind101/empire/tree/master/cmd/emp#installation

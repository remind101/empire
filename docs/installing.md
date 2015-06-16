# Empire :: Installing

The following is meant to be used as a quick way to try empire. It is not secure and is not suitable for production use.

### Prerequisites

This guide assumes that you have the following installed:

* **AWS CLI**: If you haven't already done so, you can find the instructions at http://aws.amazon.com/cli/. You'll need a fairly recent version of the CLI, which has support for ECS.

  ```console
  sudo pip install --upgrade awscli
  ```

* **EC2 SSH KeyPair**: You'll need to make sure that you've created an EC2 SSH KeyPair for the hosts that you are launching in the demo environment. See [creating or importing a keypair](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html) for more information

* **DockerHub Credentials**: If you need to get access to images in a private [hub.docker.com](https://hub.docker.com/) repository, you'll need to provide your username, password and email address for an account with access to that repository when you run the setup. This is totally optional, and you'll be able to skip it as well.

## Step 1 - ECS AMI

Before doing any of the following, log in to your AWS account and accept the terms and conditions for the official ECS AMI:

https://aws.amazon.com/marketplace/ordering?productId=4ce33fd9-63ff-4f35-8d3a-939b641f1931&ref_=dtl_psb_continue&region=us-east-1

If you don't do this, no EC2 instances will be started by the auto scaling group that our CloudFormation stack will create.

Also check that the offical ECS AMI ID for US East matches with the one in [cloudformation.json](./cloudformation.json): https://github.com/remind101/empire/blob/master/docs/guide/cloudformation.json#L20

## Step 2 - Check out the code for empire

In order to get access to the script & cloudformation template, you need to check out a copy of the empire source control:

  ```console
  git clone https://github.com/remind101/empire.git
  cd empire
  ```

## Step 2 - CloudFormation

Create a new CloudFormation stack using the [launch\_demo](../bin/launch_demo) script.


  ```console
  # ./bin/launch_demo
  AWS KeyName: default
  Do you have a docker account & want to use it for private repo access? [y/N] n
  ==> Launching empire in AZs: us-east-1b us-east-1c, Cloudformation Stack empire-33f2adf2
  ==> Waiting for stack to complete
  ==> Status: CREATE_IN_PROGRESS
  ==> Stack empire-33f2adf2 complete. EMPIRE_API_URL=http://http://empire-20-LoadBala-13W8IZP5QE8B7-1586184748.us-east-1.elb.amazonaws.com/
  ```

This is a very simple stack that will:

* Create a VPC with 2 subnets.
* Create an EC2 Instance Profile with the required permissions for the [ECS agent](https://github.com/aws/amazon-ecs-agent).
* Create a a Launch Configuration and Auto Scaling Group that will use the official ECS AMI.
* Create an ECS Cluster and Service for Empire.
* Configure the instances to be able to pull from a private registry. (If you provide docker credentials when it asks)

## Step 3 - Deploy something

Now once Empire is running and has registered itself with ELB, you can use the `emp` CLI to deploy apps:

```console
$ export EMPIRE_API_URL=http://$(stack-output $STACK ELBDNSName)
$ emp login # username is fake, password is blank
$ emp deploy remind101/acme-inc:latest
```

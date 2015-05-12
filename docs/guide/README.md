# Empire Quickstart

The following is meant to be used as a quick way to try empire. It is not secure and is not suitable for production use.

### Prerequisites

This guide assumes that you have the following installed:

* **AWS CLI**: If you haven't already done so, you can find the instructions at http://aws.amazon.com/cli/. You'll need a fairly recent version of the CLI, which has support for ECS.

  ```console
  sudo pip install --upgrade awscli
  ```

## Step 1 - ECS AMI

Before doing any of the following, log in to your AWS account and accept the terms and conditions for the official ECS AMI:

https://aws.amazon.com/marketplace/ordering?productId=4ce33fd9-63ff-4f35-8d3a-939b641f1931&ref_=dtl_psb_continue&region=us-east-1

If you don't do this, no EC2 instances will be started by the auto scaling group that our CloudFormation stack will create.

## Step 2 - ECS cluster

Before we provision resources with CloudFormation, let's create an ECS stack for Empire to schedule containers into.

```console
$ aws ecs create-cluster --cluster-name default
```

**NOTE**: In a production setup, you would probably want to isolate the Empire controller in it's own Autoscaling Group & ECS cluster, and the minions in a separate Autoscaling Group & ECS cluster.

## Step 3 - CloudFormation

Create a new CloudFormation stack using the [cloudformation.json](./cloudformation.json) file within this directory. This is a very simple stack that will:

* Create a VPC with 2 subnets.
* Create an EC2 Instance Profile with the required permissions for the [ECS agent](https://github.com/aws/amazon-ecs-agent).
* Create a a Launch Configuration and Auto Scaling Group that will use the official ECS AMI.
* Configure the instances to be able to pull from a private registry.

If you haven't already, you'll need to [create or import a keypair](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html) first. Also, have your docker registry credentials ready (email, username, password).

## Step 4 - Empire Service

The next step is to run Empire itself on ECS.

**Create the Task Definition**

First, replace the values of `<<VPC>>`, `<<InternalELBSG>>`, `<<ExternalELBSG>>`, with the values from the stack output:

```console
$ export STACK=empire # Change this if you didn't use `empire` as the CloudFormation Stack name.
$ function stack-outputs() { aws cloudformation describe-stacks --stack-name $1 --query 'Stacks[0].Outputs[*].[OutputKey,OutputValue]' --output text; }
$ function stack-output() { stack-output $1 $2 | grep $2 | cut -f 2; }
$ stack-outputs $STACK
```

Then create the task definition:

```console
$ aws ecs register-task-definition --family empire --cli-input-json file://$PWD/docs/guide/empire.ecs.json
```

**Create the ECS Service Role**

Refer to the ECS documentation to create the `ecsServiceRole` role: http://docs.aws.amazon.com/AmazonECS/latest/developerguide/IAM_policies.html#service_IAM_role

**Create the Service**

```console
$ aws ecs create-service --cluster default --service-name empire --task-definition empire \
  --desired-count 1 --role ecsServiceRole \
  --load-balancers loadBalancerName=$(stack-output $STACK ELBName),containerName=empire,containerPort=8080
```

## Step 5 - Deploy something

Now once Empire is running and has registered itself with ELB, you can use the `emp` CLI to deploy apps:

```console
$ export EMPIRE_URL=http://$(stack-output $STACK ELBDNSName)
$ emp login # username is fake, password is blank
$ emp deploy remind101/acme-inc:latest
```

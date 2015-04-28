# Empire Quickstart

The following is meant to be used as a quick way to test empire. It is not secure and is not suitable for production use.

## Step 1 - ECS AMI

Before doing any of the following, log in to your AWS account and accept the terms and conditions for the official ECS AMI:

https://aws.amazon.com/marketplace/ordering?productId=4ce33fd9-63ff-4f35-8d3a-939b641f1931&ref_=dtl_psb_continue&region=us-east-1

If you don't do this, no EC2 instances will be started by the auto scaling group that our CloudFormation stack will create.

## Step 2 - ECS cluster

Before we provision resources with CloudFormation, let's create an ECS stack for Empire to schedule containers into.

```console
$ aws ecs create-cluster --cluster-name default
```

**NOTE**: In a production setup, you would probably want to isolate the Empire controller within it's own VPC and ECS Cluster.

## Step 3 - CloudFormation

Create a new CloudFormation stack using the [cloudformation.json](./cloudformation.json) file within this directory.

## Step 4 - Empire Service

The next step is to run Empire itself on ECS.

**Create the Task Definition**

```console
$ aws ecs register-task-definition --family empire --cli-input-json file://$PWD/guide/empire.ecs.json
```

**Create the Service**

```console
$ export ELB=$(aws cloudformation describe-stacks --stack-name empire | jq '.Stacks[0].Outputs | .[] | {(.OutputKey): .OutputValue} | .ELB')
$ export SERVICE_ROLE=$(aws cloudformation describe-stacks --stack-name empire | jq '.Stacks[0].Outputs | .[] | {(.OutputKey): .OutputValue} | .ServiceRole')
$ aws ecs create-service --cluster default --service-name empire --task-definition empire \
  --desired-count 1 --role $SERVICE_ROLE \
  --load-balancers loadBalancerName=$ELB,containerName=empire,containerPort=8080
```

## Step 5 - Deploy something

Now once Empire is running and has registered itself with ELB, you can use the `emp` CLI to deploy apps:

```console
$ export EMPIRE_URL=$(aws cloudformation describe-stacks --stack-name empire | jq '.Stacks[0].Outputs | .[] | {(.OutputKey): .OutputValue} | .ELBDNSName')
$ emp login # username is fake, password is blank
$ emp deploy remind101/acme-inc:latest
```

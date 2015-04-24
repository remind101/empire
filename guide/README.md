# Empire Quickstart

The following is meant to be used as a quick way to test empire. It is not secure and is not suitable for production use.

## Step 1 - ECS AMI

Before doing any of the following, log in to your AWS account and accept the terms and conditions for the official ECS AMI:

https://aws.amazon.com/marketplace/ordering?productId=4ce33fd9-63ff-4f35-8d3a-939b641f1931&ref_=dtl_psb_continue&region=us-east-1

If you don't do this, no EC2 instances will be started by the auto scaling group that our CloudFormation stack will create.

## Step 2 - ECS cluster

Before we provision resources with CloudFormation, let's create an ECS stack for Empire to schedule containers into.

1. Within the AWS Console, go to `EC2 Container Service`.
2. If you don't already have a cluster, then you'll be asked to go through the getting started guide. You can simply cancel out of this.
3. Select `Create Cluster`. Name it `default`.

**NOTE**: In a production setup, you would probably want to isolate the Empire controller within it's own VPC and ECS Cluster.

## Step 3 - CloudFormation

Create a new CloudFormation stack using the [cloudformation.json](./cloudformation.json) file within this directory.

## Step 4 - Empire Service

The next step is to run Empire itself on ECS.

**Create the Task Definition**

1. Within the AWS Console, go to `EC2 Container Service`.
2. Click `Task Definitions`, then `Create new Task Definition`.
3. Click the `JSON` tab and copy the contents of the [empire.ecs.json](./empire.ecs.json) file.

**Create the Service**

1. Click `Create Service`.
2. Use the following parameters to create the service:

   | Field           | Value    |
   |-----------------|----------|
   | Task Definition | empire:1 |
   | Service name    | empire   |
   | Number of tasks | 1        |

3. Associate the service with the Empire ELB. When asked for the `Container Name : Host Port`, select `empire:8080`.
4. Associate an IAM role with the service by selecting `Manage AIM Role` and `Allow`.

## Step 5 - Deploy something

Now once Empire is running and has registered itself with ELB, you can use the `emp` CLI to deploy apps:

```console
$ export EMPIRE_URL=<ELB DNS Name>
$ emp login # username is fake, password is blank
$ emp deploy remind101/acme-inc:latest
```

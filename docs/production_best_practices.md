# Empire :: Production Best Practices - WORK IN PROGRESS

This document is meant to explain what we believe are some best practices & guidelines around running Empire in a production or production-like environment. We hope to provide more instruction for this as well as tools to make this easier in the future.

1. Separate Minion & Controller hosts
2. Private VPC for all hosts
3. Securing the API
3. Access to the Empire API via VPN/specific IP
4. Separate, secure database for Empire
5. Router application

## Clusters

In general it is recommended that you have two pools of servers. Neither pool should be able to communicate with the other directly (no ssh, etc).

### Empire Controllers

These are the servers where you will run the Empire daemon. We keep this separate from where you run actual applications (the minions, see below) so that if there is an issue with an application you deploy, it doesn't affect the control plane. They are also kept separate in order to provide additional security - if someone manages to break out of one of your applications, they should not be able to reach the Controllers, which at least stops them from spinning up more resources via Empire.

These hosts should sit on a private subnet in your VPC, and only expose the API via an elastic load balancer.

### Empire Minions

The minion hosts are where applications deployed via Empire will run. Again, these should have no access to the controllers.

They should sit on a private subnet, and shouldn't be exposed to the internet. When you expose an app via a domain in Empire, it will create a load balancer for that app that will handle the exposure for you. These load balancers are all part of the same security group, and have the same security group rules.

## Private VPC

All hosts should run in a private VPC and should not be directly exposed to the internet. If you need ssh access to the hosts, we suggest either using a VPN or a set of tightly controlled & secured bastion hosts.

If you go the bastion host route, we suggest that you limit the access to those hosts - via security group rules (only allowing your office IP to connect, for example) and any other means available.

Also, it might be worth looking into something like [ssh-ca](https://github.com/cloudtools/ssh-ca) for managing ssh keys, allowing you to grant access to hosts on a case by case basis without putting any single person's ssh keys on the host.


## Securing the API

It's very important that you keep the Empire API secure. There are a few best practices for doing so:

### HTTPS

Since you will be passing authentication information to the Empire API, you should ensure that the load balancer that it uses is setup to only accept connections via HTTPS

### Github Authentication

Refer to the docs on [configuring the GitHub authentication backend](./configuration).

### Limiting API Access by IP

A final option that can be useful is to only allow access to the Empire API loadbalancer via VPN, or from specific IP addresses (such as your office IP).

## Separate, secure database for Empire API

Empire requires a postgres database in order to function. It is suggested that you use a database host specifically for this purpose, rather than sharing with other applications. This allows you to lock access down to only the Empire API.

## Router Application

If you plan to expose services, rather than having Empire expose them (via adding a domain) you can instead deploy a 'router' application, and expose that. Remind uses a single router app that is exposed to the internet via Empire (through an ELB) that has rules to route to other services based on the hostname.

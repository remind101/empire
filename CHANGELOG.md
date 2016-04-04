# Changelog

## HEAD

**Features**

* Empire now supports deploying Docker images from the EC2 Container Registry [#730](https://github.com/remind101/empire/pull/730).
* The Docker logging driver that the ECS backend uses is now configurable via the `--ecs.logdriver` flag [#731](https://github.com/remind101/empire/pull/731).
* It's now possible to lock down the GitHub authorization to a specific team via the `--github.team.id` flag [#745](https://github.com/remind101/empire/pull/745).
* Empire can now integrate with Conveyor to build Docker images on demand when using the GitHub Deployments integration [#747](https://github.com/remind101/empire/pull/747).
* Stdout and Stdin from interactive run sessions can now be sent to CloudWatch Logs for longterm storage and auditing [#757](https://github.com/remind101/empire/pull/757).
* Add `Environment` and `Release` to Deploy Events. `--environment` will likely be used for tagging resources later. [#758](https://github.com/remind101/empire/pull/758)
* Add constraint changes to scale events [#773](https://github.com/remind101/empire/pull/773)

**Bugs**

* `emp run` now works with unofficial Docker registries [#740](https://github.com/remind101/empire/pull/740).
* `emp scale -l` now lists configured scale, not the running processes [#769](https://github.com/remind101/empire/pull/769)

**Security**

* Empire is now built with Go 1.5.3 to address [CVE-2015-8618](https://groups.google.com/forum/#!topic/golang-announce/MEATuOi_ei4) [#737](https://github.com/remind101/empire/pull/737).

## 0.10.0 (2016-01-13)

**Features**

* `emp ps` now shows the correct uptime of the process thanks to ECS support [#683](https://github.com/remind101/empire/pull/683).
* `emp run` now supports the `-d` flag for detached processes [#695](https://github.com/remind101/empire/pull/695).
* You can now deploy images from unofficial Docker registries, such as Quay.io [#692](https://github.com/remind101/empire/pull/692).
* Empire now allows you to "attach" existing IAM certificates. This replaces the old `ssl-*` commands in the `emp` CLI [#701](https://github.com/remind101/empire/pull/701).
* You can now have Empire publish events to an SNS topic [#698](https://github.com/remind101/empire/pull/698).
* Empire now supports environement aliases for Github Deployments [#681](https://github.com/remind101/empire/pull/681)

**Bugs**

* Allow floating point numbers to be provided when scaling the memory on a process [#694](https://github.com/remind101/empire/pull/694).
* Empire will now update the SSL certificate on the associated ELB if it changes from `emp cert-attach` [#700](https://github.com/remind101/empire/pull/700).
* The Tugboat integration now updates the deployment status with any errors that occurred [#709](https://github.com/remind101/empire/pull/709).
* Deploying a non-existent docker image to Empire will no longer create an app [#713](https://github.com/remind101/empire/pull/713).
* It's no longer necessary to re-deploy an application when scaling a process with new CPU or memory constraints [#713](https://github.com/remind101/empire/pull/713).

**Security**

* GitHub Organization membership is now checked on every request, not just at access token creation time [#687](https://github.com/remind101/empire/pull/687).

**Internal**

* The `emp` CLI has been moved to the primary [remind101/empire](https://github.com/remind101/empire/tree/master/cmd/emp) repo [#712](https://github.com/remind101/empire/pull/712).

## 0.9.2 (2015-10-27)

**Documentation**

* Added doc on enabling log tailing #671.
* Added doc on deploying an application #642.
* Added doc on exposing an app publicly #668.
* Added doc on known limitations #672.

**Features**

* Added log tailing from Kinesis #651.
* Added AWS API errors exposition when deploying #628.
* Added CrossZoneLoadBalancing to ELBs #641.
* Added the process type in the get processes endpoint #649.
* Reversed process and version in SOURCE environment variable #652.
* Set empire.* labels on containers #679.

**Bugs**

* Added more specific load balancer error messages #629
* Update aws-sdk-go to v0.9.15. Fixed ThrottlingExceptions during restart #645.
* Fixed pagination when listing processes (tasks) #648.
* Fixed release description for config updates (`set` and `unset` env variables) #678.

## 0.9.1 (2015-07-31)

**Documentation**

* Updated demo to support private registries other than the official registry #528.
* General updates to documentation.
* Changed ELB health check thresholds in example CloudFormation stack to follow AWS defaults.

**Features**

* Implemented support for attached one-off commands #568.
* Added support for GitHub Deployments #602.
* Added support for deploying a docker image to a specific app #622.
* Added support for `emp info` command #619.
* Added pagination support for `/apps/{app}/releases` endpoint #591.

**Bugs**

* Fixed a bug that caused containers launched by one-off tasks to stay around if the client disconnected. #589.
* Fixed an issue where deploying an app to an AWS account with no ELB's would cause an infinite loop #623.
* Fixed a bug that prevented scaling a processes memory to more than 1GB #593.

## 0.9.0 (2015-06-16)

Initial public release

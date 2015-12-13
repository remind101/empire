# Changelog

## Head

**Features**

* `emp ps` now shows the correct uptime of the process thanks to ECS support [#683](https://github.com/remind101/empire/pull/683).

**Bugs**

* Allow floating point numbers to be provided when scaling the memory on a process [#694](https://github.com/remind101/empire/pull/694).

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

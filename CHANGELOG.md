# Changelog

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

* Fixes a bug that caused containers launched by one-off tasks to stay around if the client disconnected. #589.
* Fixed an issue where deploying an app to an AWS account with no ELB's would cause an infinite loop #623.
* Fixed a bug that prevented scaling a processes memory to more than 1GB #593.

## 0.9.0 (2015-06-16)

Initial public release

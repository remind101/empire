# Changelog

## HEAD

**Documentation**

* Updated demo to support private registries other than the official registry #528.
* General updates to documentation.
* Changed ELB health check thresholds in example CloudFormation stack to follow AWS defaults.

**Features**

* Implemented support for attached one-off commands #568
* Added support for GitHub Deployments #602

**Bugs**

* Fixes a bug that caused containers launched by one-off tasks to stay around if the client disconnected. #589

## 0.9.0 (2015-06-16)

Initial public release

# Changelog

## HEAD

**Features**

* Empire now includes experimental support for showing attached runs in `emp ps`. This can be enabled with the `--x.showattached` flag, or `EMPIRE_X_SHOW_ATTACHED` [#911](https://github.com/remind101/empire/pull/911)
* Empire now includes experimental support for scheduled tasks [#919](https://github.com/remind101/empire/pull/919)
* Empire now supports streaming status updates from the scheduler while deploying [#888](https://github.com/remind101/empire/issues/888)
* Empire now supports sending internal metrics to statsd or dogstatsd [#953](https://github.com/remind101/empire/pull/953)
* Attached and detached runs now have an `empire.user` label attached to them [#965](https://github.com/remind101/empire/pull/965)

**Improvements**

* The Custom::ECSService custom resource now waits for newly created ECS services to stabilize [#878](https://github.com/remind101/empire/pull/878)
* The CloudFormation backend now uses the Custom::ECSService resource instead of AWS::ECS::Service, by default [#877](https://github.com/remind101/empire/pull/877)
* The database schema version is now checked at boot, as well as in the http health checks. [#893](https://github.com/remind101/empire/pull/893)
* The log level within empire can now be configured when starting the service. [#929](https://github.com/remind101/empire/issues/929)
* The CloudFormation backend now has experimental support for a `Custom::ECSTaskDefinition` resource that greatly reduces the size of generated templates. [#935](https://github.com/remind101/empire/pull/935)
* The Scheduler now has a `Restart` method which will trigger a restart of all the processes within an app. Previously, a "Restart" just re-released the app. Now schedulers like the cloudformation backend can optimize how the restart is handled. [#697](https://github.com/remind101/empire/issues/697)
* `emp run` now publishes an event when it is ran [#954](https://github.com/remind101/empire/pull/954)

**Bugs**

* Fixed a bug where multiple duplicate ECS services could be created by the CloudFormation backend, when using the `Custom::ECSService` resource [#884](https://github.com/remind101/empire/pull/884).
* Fixed a bug where the lock obtained during stack operations was not always unlocked. [#892](https://github.com/remind101/empire/pull/892)
* Fixed an issue where Procfile's would not be extracted when Docker 1.12+ was used. [#915](https://github.com/remind101/empire/pull/915)
* Fixed a bug where the failed creation of a custom resources could cause a CloudFormation stack to fail to rollback. [#938](https://github.com/remind101/empire/pull/938)
* Fixed a bug where waiting for a deploy to stabilize was failing if you had more than 10 services. [#944](https://github.com/remind101/empire/issues/944)
* Fixed an issue in the Tugboat integration where the log stream to a Tugboat instance could be closed. [#950](https://github.com/remind101/empire/pull/950)
* Fixed an issue where typing commit message does not allow user to use arrow keys, etc. [#958](https://github.com/remind101/empire/pull/958)

**Performance**

* Performance of creating/updating/deleting custom resources in the CloudFormation backend has been improved. [#942](https://github.com/remind101/empire/pull/942)
* ECS Task Definitions are now cached in memory for improved `emp ps` performance. [#902](https://github.com/remind101/empire/pull/902)

**Security**

## 0.10.1 (2016-06-14)

**Features**

* Empire now contains expiremental support for using CloudFormation to provision resources for applications [#814](https://github.com/remind101/empire/pull/814), [#803](https://github.com/remind101/empire/pull/803).
* Empire now supports requiring commit messages for all actions that emit an event via `--messages.required`. If a commit message is required for an action, emp will gracefully handle it and ask the user to input a value [#767](https://github.com/remind101/empire/issues/767).
* You can now supply a commit message to any event that is published by Empire [#767](https://github.com/remind101/empire/issues/767).
* Empire now supports deploying Docker images from the EC2 Container Registry [#730](https://github.com/remind101/empire/pull/730).
* The Docker logging driver that the ECS backend uses is now configurable via the `--ecs.logdriver` flag [#731](https://github.com/remind101/empire/pull/731).
* It's now possible to lock down the GitHub authorization to a specific team via the `--github.team.id` flag [#745](https://github.com/remind101/empire/pull/745).
* Empire can now integrate with Conveyor to build Docker images on demand when using the GitHub Deployments integration [#747](https://github.com/remind101/empire/pull/747).
* Stdout and Stdin from interactive run sessions can now be sent to CloudWatch Logs for longterm storage and auditing [#757](https://github.com/remind101/empire/pull/757).
* Add `Environment` and `Release` to Deploy Events. `--environment` will likely be used for tagging resources later. [#758](https://github.com/remind101/empire/pull/758)
* Add constraint changes to scale events [#773](https://github.com/remind101/empire/pull/773)
* You can now specify the CPU and memory constraints for attached one-off tasks with the `-s` flag to `emp run` [#809](https://github.com/remind101/empire/pull/809)
* You can now provide a duration to `emp log` with the `-d` flag to start streaming logs from a specific point in time ie (5m, 10m, 1h) [#829](https://github.com/remind101/empire/issues/829)
* If log streaming is enabled, Empire will attempt to write events to the kinesis stream for the application [#832](https://github.com/remind101/empire/issues/832)
* Added Stdout event stream [#874](https://github.com/remind101/empire/issues/874)

**Bugs**

* `emp run` now works with unofficial Docker registries [#740](https://github.com/remind101/empire/pull/740).
* `emp scale -l` now lists configured scale, not the running processes [#769](https://github.com/remind101/empire/pull/769)
* Fixed a bug where it was previously possible to create a signed access token with an empty username [#780](https://github.com/remind101/empire/pull/780)
* ECR authentication now supports multiple regions, and works independently of ECS region [#784](https://github.com/remind101/empire/pull/784)
* Provisioned ELB's are only destroyed when the entire app is removed [#801](https://github.com/remind101/empire/pull/801)
* Docker containers started by attached runs now have labels, cpu and memory constraints applied to them [#809](https://github.com/remind101/empire/pull/809)
* Fixed a bug where interactive `emp run` would get stuck attempting to read bytes after an error from the initial request [#795](https://github.com/remind101/empire/issues/795)

**Performance**

* `emp ps` should be significantly faster for services running a lot of processes [#781](https://github.com/remind101/empire/pull/781)
* Scaling multiple processes within the Cloudformation scheduler results in 1 update now instead of N [#844](https://github.com/remind101/empire/pull/844)

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

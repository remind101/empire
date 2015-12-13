# 12factor

12factor is a Go library for describing and running [12factor](http://12factor.net/) applications.

## Packages

* **[scheduler](./scheduler)**: Provides various implementations of the Scheduler interface for running 12factor apps. Implementations include Docker, ECS, Kubernetes and Nomad.
* **[procfile](./procfile)**: Provides methods for parsing the Procfile manifest format.

## Terminology

### App

An App describes a common environment and root filesystem, which is generally specified as a Docker container.

### Process

A Process represents a named command that can be scaled horizontally.

### Manifest

A manifest is the composition of an App and its Processes.

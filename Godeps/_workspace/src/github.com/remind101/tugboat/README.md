# [Tugboat](https://github.com/ejholmes/tugboat) [![Build Status](https://travis-ci.org/remind101/tugboat.svg?branch=master)](https://travis-ci.org/remind101/tugboat)

Tugboat is an API and AngularJS client for aggregating deployments of GitHub repos.

![](https://s3.amazonaws.com/ejholmes.github.com/ioiPx.png)

## Providers

Tugboat by itself isn't all that exciting; it won't perform deployments for you, but it does provide an API for deployment providers to hook into.

Writing your own providers is really simple and you can write them in any language that you want.

### Provider API

Tugboat exposes an API for registering deployments, add logs, and updating the status. For an example of how to create an external provider with Go, see [provider_test.go](./provider_test.go).

---

#### Authorization

The API expects the `user` part of a basic auth `Authorization` header to be a provider auth token. You can generate a provider auth token using the following:

```console
$ tugboat tokens create <provider>
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJQcm92aWRlciI6ImZvbyJ9.UYMrZD7cgBdeEXLf11nwEiZpUI2DuOdRsGOZyG2SluU
```

#### Create Deployment

Creates a new Deployment within tugboat. In general, this would include a post body extracted from a GitHub `deployment` event webhook payload.

The response from this endpoint will be a `Deployment` resource.

```
POST /deployments
```

**Example Request**

```json
{
  "ID": 1234,
  "Sha": "abcd...xyz",
  "Ref": "master"
}
```

**Example Response**

```json
{
  "ID": "01234567-89ab-cdef-0123-456789abcdef",
  "Repo": "remind101/r101-api",
  "Token": "01234567-89ab-cdef-0123-456789abcdef"
}
```

---

#### Add Log Lines

This adds lines of logs to the deployment. You can simply stream your logs and they will be added as they come in. Logs show up automatically in the UI via pusher events.

```
POST /deployments/:id/logs
```

**Example Request**

```
Authorization: dXNlcjo=\n

Deploying to production
Deployed
```

---

#### Update Status

Updates the status of the deployment. The `status` field should be one of `succeeded`, `failed` or `errored`. If the `status` is `errored` then you can provide an `error` field with details about the error. This will also update the status of the deployment within GitHub itself.

```
POST /deployments/:id/status
```

**Example Request**

```json
{
  "status": "succeeded"
}
```

## Setup

**TODO**

## Roadmap

* [Librato annotations notifier](https://github.com/ejholmes/tugboat/issues/7).
* [.tugboat.yml configuration file](https://github.com/ejholmes/tugboat/issues/8).

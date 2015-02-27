## App


### Attributes
| Name | Type | Description | Example |
| ------- | ------- | ------- | ------- |
| **name** | *string* | unique name of app<br/> **pattern:** <code>^[a-z][a-z0-9-]{3,30}$</code> | `"example"` |
| **repo** | *string* | the name of the repo | `"remind101/r101-api"` |
### App Create
Create a new app.

```
POST /apps
```

#### Required Parameters
| Name | Type | Description | Example |
| ------- | ------- | ------- | ------- |
| **name** | *string* | unique name of app<br/> **pattern:** <code>^[a-z][a-z0-9-]{3,30}$</code> | `"example"` |
| **repo** | *string* | the name of the repo | `"remind101/r101-api"` |



#### Curl Example
```bash
$ curl -n -X POST http://localhost:8080/apps \
  -H "Content-Type: application/json" \
 \
  -d '{
  "name": "example",
  "repo": "remind101/r101-api"
}'

```


#### Response Example
```
HTTP/1.1 201 Created
```
```json
{
  "name": "example",
  "repo": "remind101/r101-api"
}
```


## Config Vars
Configuration information for an app

### Config Vars Info
Get config-vars for app.

```
GET /apps/{app_name}/config-vars
```


#### Curl Example
```bash
$ curl -n -X GET http://localhost:8080/apps/$APP_NAME/config-vars

```


#### Response Example
```
HTTP/1.1 200 OK
```
```json
{
  "FOO": "bar",
  "BAZ": "qux"
}
```

### Config Vars Update
Update config-vars for app. You can update existing config-vars by setting them again, and remove by setting it to `NULL`.

```
PATCH /apps/{app_name}/config-vars
```


#### Curl Example
```bash
$ curl -n -X PATCH http://localhost:8080/apps/$APP_NAME/config-vars \
  -H "Content-Type: application/json" \
 \
  -d '{
  "FOO": null,
  "BAZ": "grault"
}'

```


#### Response Example
```
HTTP/1.1 200 OK
```
```json
{
  "FOO": "bar",
  "BAZ": "qux"
}
```


## Config


### Attributes
| Name | Type | Description | Example |
| ------- | ------- | ------- | ------- |
| **version** | *string* | unique identifier of config | `"0123456789abcdef0123456789abcdef"` |
| **vars** | *object* | a hash of configuration values | `{"RAILS_ENV":"production"}` |
### Config Head
Get the latest version of an app's config

```
GET /apps/{app_name}/configs/head
```


#### Curl Example
```bash
$ curl -n -X GET http://localhost:8080/apps/$APP_NAME/configs/head

```


#### Response Example
```
HTTP/1.1 200 OK
```
```json
{
  "version": "0123456789abcdef0123456789abcdef",
  "vars": {
    "RAILS_ENV": "production"
  }
}
```

### Config Info
Get a specific version of an app's config

```
GET /apps/{app_name}/configs/{config_version}
```


#### Curl Example
```bash
$ curl -n -X GET http://localhost:8080/apps/$APP_NAME/configs/$CONFIG_VERSION

```


#### Response Example
```
HTTP/1.1 200 OK
```
```json
{
  "version": "0123456789abcdef0123456789abcdef",
  "vars": {
    "RAILS_ENV": "production"
  }
}
```

### Config Update
Updates the config for an app

```
PATCH /apps/{app_name}/configs
```

#### Required Parameters
| Name | Type | Description | Example |
| ------- | ------- | ------- | ------- |
| **vars** | *object* | a hash of configuration values | `{"RAILS_ENV":"production"}` |



#### Curl Example
```bash
$ curl -n -X PATCH http://localhost:8080/apps/$APP_NAME/configs \
  -H "Content-Type: application/json" \
 \
  -d '{
  "vars": {
    "RAILS_ENV": "production"
  }
}'

```


#### Response Example
```
HTTP/1.1 200 OK
```
```json
{
  "version": "0123456789abcdef0123456789abcdef",
  "vars": {
    "RAILS_ENV": "production"
  }
}
```


## Deploy


### Attributes
| Name | Type | Description | Example |
| ------- | ------- | ------- | ------- |
| **id** | *uuid* | unique identifier of deploy | `"01234567-89ab-cdef-0123-456789abcdef"` |
| **release:id** | *uuid* | unique identifier of release | `"01234567-89ab-cdef-0123-456789abcdef"` |
| **release:version** | *number* | an incremental identifier for the version | `1` |
| **release:app:name** | *string* | unique name of app<br/> **pattern:** <code>^[a-z][a-z0-9-]{3,30}$</code> | `"example"` |
| **release:config:version** | *string* | unique identifier of config | `"0123456789abcdef0123456789abcdef"` |
| **release:slug:id** | *uuid* | unique identifier of slug | `"01234567-89ab-cdef-0123-456789abcdef"` |
### Deploy Create
Create a new deploy.

```
POST /deploys
```

#### Required Parameters
| Name | Type | Description | Example |
| ------- | ------- | ------- | ------- |
| **image:id** | *uuid* | unique identifier of image | `"0123456789abcdef0123456789abcdef"` |
| **image:repo** | *string* | the name of the repo | `"remind101/r101-api"` |



#### Curl Example
```bash
$ curl -n -X POST http://localhost:8080/deploys \
  -H "Content-Type: application/json" \
 \
  -d '{
  "image": {
    "id": "0123456789abcdef0123456789abcdef",
    "repo": "remind101/r101-api"
  }
}'

```


#### Response Example
```
HTTP/1.1 201 Created
```
```json
{
  "id": "01234567-89ab-cdef-0123-456789abcdef",
  "release": {
    "id": "01234567-89ab-cdef-0123-456789abcdef",
    "version": 1,
    "app": {
      "name": "example"
    },
    "config": {
      "version": "0123456789abcdef0123456789abcdef"
    },
    "slug": {
      "id": "01234567-89ab-cdef-0123-456789abcdef"
    }
  }
}
```


## Formation


### Formation Update
Update an apps formation

```
PATCH /apps/{app_name}/formation
```

#### Required Parameters
| Name | Type | Description | Example |
| ------- | ------- | ------- | ------- |
| **updates/process** | *string* |  |  |
| **updates/quantity** | *number* |  |  |



#### Curl Example
```bash
$ curl -n -X PATCH http://localhost:8080/apps/$APP_NAME/formation \
  -H "Content-Type: application/json" \
 \
  -d '{
  "updates": [
    {
      "process": null,
      "quantity": null
    }
  ]
}'

```


#### Response Example
```
HTTP/1.1 200 OK
```
```json
{
}
```








## App


### Attributes
| Name | Type | Description | Example |
| ------- | ------- | ------- | ------- |
| **id** | *uuid* | unique identifier of app | `"01234567-89ab-cdef-0123-456789abcdef"` |
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
  "id": "01234567-89ab-cdef-0123-456789abcdef",
  "name": "example",
  "repo": "remind101/r101-api"
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
GET /apps/{app_id_or_name}/configs/head
```


#### Curl Example
```bash
$ curl -n -X GET http://localhost:8080/apps/$APP_ID_OR_NAME/configs/head

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
GET /apps/{app_id_or_name}/configs/{config_version}
```


#### Curl Example
```bash
$ curl -n -X GET http://localhost:8080/apps/$APP_ID_OR_NAME/configs/$CONFIG_VERSION

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
PATCH /apps/{app_id_or_name}/configs
```

#### Required Parameters
| Name | Type | Description | Example |
| ------- | ------- | ------- | ------- |
| **vars** | *object* | a hash of configuration values | `{"RAILS_ENV":"production"}` |



#### Curl Example
```bash
$ curl -n -X PATCH http://localhost:8080/apps/$APP_ID_OR_NAME/configs \
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
| **release:version** | *string* | an incremental identifier for the version | `"v1"` |
| **release:app:id** | *uuid* | unique identifier of app | `"01234567-89ab-cdef-0123-456789abcdef"` |
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
    "version": "v1",
    "app": {
      "id": "01234567-89ab-cdef-0123-456789abcdef"
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








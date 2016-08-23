# Empire :: CloudFormation Resources

Empire can integrate directly with CloudFormation so that you can create Empire apps, and alter environment variables via CloudFormation custom resources. Empire supports the following custom resources:

* **Custom::EmpireApp**: This resource will create an application inside Empire. `Ref` will return the app id.
* **Custom::EmpireAppEnvironment**: This resource can be used to manage a set of environment variables on an application.

## Example

To get a better idea of how this can be used, let's take a look at a concrete example for managing the infrastructure at Acme Inc.

Acme Inc. has 2 applications; a Rails application that serves the web interface, and an internal API application, which the rails app consumes. The API application has some external dependencies:

1. A PostgreSQL database for storage.
2. An S3 bucket for storing file uploads.

The web application also wants a CloudFront distribution for serving it's static assets.

Using the Custom resources provided by Empire, we can use a single CloudFormation stack to manage the entire infrastructure, and tie it all together:

```json
{
  "Parameters": {
    "EmpireCustomResourcesSNSTopic": {
      "Type": "String",
      "Description": "The value of EMPIRE_CUSTOM_RESOURCES_TOPIC when starting Empire. CloudFormation will send requests to create custom resources here."
    }
  },
  "Resources": {
    "CDN": {
      "Type": "AWS::CloudFront::Distribution",
      "Properties": {
        ...
      }
    },
    "DB": {
      "Type": "AWS::RDS::DBInstance",
      "Properties": {
        ...
      }
    },
    "Bucket": {
      "Type": "AWS::S3::Bucket"
    },
    "WebApp": {
      "Type": "Custom::EmpireApp",
      "Properties": {
        "Name": "web"
      }
    },
    "WebEnvironment": {
      "Type": "Custom::EmpireAppEnvironment",
      "Properties": {
        "ServiceToken": { "Ref": "EmpireCustomResourcesSNSTopic" },
        "AppId": { "Ref": "WebApp" },
        "Variables": [
          {
            "Name": "ASSET_HOST",
            "Value": { "Fn::Join": ["", ["https://", { "GetAtt": ["CDN", "DomainName"] }]] }
          }
        ]
      }
    },
    "ApiApp": {
      "Type": "Custom::EmpireApp",
      "Properties": {
        "Name": "api"
      }
    },
    "ApiEnvironment": {
      "Type": "Custom::EmpireAppEnvironment",
      "Properties": {
        "ServiceToken": { "Ref": "EmpireCustomResourcesSNSTopic" },
        "AppId": { "Ref": "ApiApp" },
        "Variables": [
          {
            "Name": "DATABASE_URL",
            "Value": { "Fn::Join": ["postgres://postgres:postgres@", { "GetAtt": ["DB", "Endpoint.Address"] }, ":", { "GetAtt": ["DB", "Endpoint.Port"] }, "/postgres"] }
          },
          {
            "Name": "BUCKET",
            "Value": { "Ref": "Bucket" }
          }
        ]
      }
    }
  }
}
```

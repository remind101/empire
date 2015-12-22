# Empire :: Configuration

The following documents the various configuration parameters that you can use to tailor your Empire environment to your needs.

### GitHub Authentication

**TODO**

### GitHub Deployments

You can (optionally) trigger Deployments to your Empire environment with the [GitHub Deployments API](https://developer.github.com/v3/repos/deployments/) and something like [deploy](https://github.com/remind101/deploy).

**Step 1 - Environment Variables**

You'll need to set the following environment variables:

Environment Variable | Description
---------------------|------------
`EMPIRE_GITHUB_WEBHOOKS_SECRET` | This should be a randomly generated string that is used by GitHub to sign webhook payloads so that Empire can verify the request was from GitHub. This is the same value you will include when setting up the webhook on the repository
`EMPIRE_GITHUB_DEPLOYMENTS_ENVIRONMENT` | This should be the name of the environment that this Empire instance should respond to deployment events to. For example, if you're creating a GitHub deployment for `staging`, you'll want to set this value to `staging`
`EMPIRE_GITHUB_DEPLOYMENTS_IMAGE_TEMPLATE` | Empire makes the assumption that their is a matching Docker repository with an image tagged with the git commit sha. This is a Go text/template that will be used to determine the Docker image to deploy. It will be passed a [Deployment](https://github.com/ejholmes/hookshot/blob/master/events/deployment.go) object. The default value is `{{ .Repository.FullName }}:{{ .Deployment.Sha }}`
`EMPIRE_TUGBOAT_URL` | If you'd like to have Empire send deployment logs and status updates to a [Tugboat](https://github.com/remind101/tugboat), include the URL here.

**Step 2 - Add webhooks**

After Empire is configured to respond to GitHub webhooks, you can simply add a webhook to the repository that you want to deploy to Empire using GitHub Deployments.

1. Go into the repositories webhooks settings
2. Click **Add Webhook**
3. For the **Payload URL** field, enter the location of your Empire instance.
4. For the **Secret** field, enter the same value you used above for `EMPIRE_GITHUB_WEBHOOKS_SECRET`.
5. Select **Let me select individual events.**. Choose **Deployment** and uncheck **Push**.

After adding the webhook, you should see a successful ping event.

**Step 3 - Create GitHub Deployments**

Now you can create GitHub Deployments on the GitHub repository using a tool like the [deploy CLI](https://github.com/remind101/deploy) or [hubot-deploy](https://github.com/remind101/hubot-deploy).

### SNS Event Stream

Empire can publish internal events to an SNS topic, so that you can create consumers that publish them to, for example, a datadog event stream or a slack channel. Empire currently publishes the following events:

1. **deploy**: Triggered whenever a successful deployment completes.
2. **run**: Triggered whenever starts a one-off process.
3. **restart**: Triggered whenever an application is restarted.
4. **rollback**: Triggered when an application is rolled back to a previous version.
5. **scale**: Triggered whenever a process is scaled to a new size.

To enable publishing to an SNS topic, set the following environment variables:

Environment Variable | Description
---------------------|------------
`EMPIRE_EVENTS_BACKEND` | This should be set to `sns`
`EMPIRE_SNS_TOPIC` | The full AWS ARN for the SNS topic to publish to.

You should ensure that Empire has access to `sns:PublishEvent` in the IAM policy.

Here's an example AWS Lambda function that can be used to publish Empire events to a slack channel:

```javascript
console.log('Loading function');

const https = require('https');
const url = require('url');
// to get the slack hook url, go into slack admin and create a new "Incoming Webhook" integration
const slack_url = 'https://hooks.slack.com/services/.../...';
const slack_req_opts = url.parse(slack_url);
slack_req_opts.method = 'POST';
slack_req_opts.headers = {'Content-Type': 'application/json'};

exports.handler = function(event, context) {
  (event.Records || []).forEach(function (rec) {
    if (rec.Sns) {
      var req = https.request(slack_req_opts, function (res) {
        if (res.statusCode === 200) {
          context.succeed('posted to slack');
        } else {
          context.fail('status code: ' + res.statusCode);
        }
      });
      
      req.on('error', function(e) {
        console.log('problem with request: ' + e.message);
        context.fail(e.message);
      });
      
      var message = JSON.parse(rec.Sns.Message);
      req.write(JSON.stringify({text: message.Message})); // for testing: , channel: '@vadim'
      
      req.end();
    }
  });
};
```

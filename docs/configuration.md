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

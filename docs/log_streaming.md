# Activating log streaming

By default, log streaming is deactivated in Empire. If you try to run
`emp log -a <app>`, you will get the following response:

```console
$ emp log -a acme-inc
Logs are disabled
```

To activate log streaming on Empire, you need to set the `EMPIRE_LOG_STREAMER`
environment variable on your Empire instance(s). Right now the only value supported
is `kinesis`, but we hope to support more in the future.

When using Amazon Kinesis log streaming, Empire will try to read the logs from the
Kinesis stream named after the app id. This means that the Kinesis streams need to pre-exist
with logs in them before Empire can forward them to your terminal. We use [logspout-kinesis](https://github.com/remind101/logspout-kinesis) to do so. Our official [Empire AMI](https://github.com/remind101/empire_ami) also takes care of running logspout and activating Kinesis log streaming on Empire.

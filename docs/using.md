# Empire :: Using

1. [Overview](./README.md)
2. [Installing](./installing.md)
3. [Using](./using.md)
4. [Administering](./administering.md) **TODO**
5. [Troubleshooting](./troubleshooting.md) **TODO**
6. [Roadmap](./roadmap.md) **TODO**

Going forward from the [installing](./installing.md) guide, the first thing we'll need to do is tell the empire client where it can find the empire API. The *launch_demo* command should have printed that out for you, so make sure you've set it in your environment like so:

```
# export EMPIRE_API_URL=http://empire-60-LoadBala-1M8NAQ24SPGMP-770037928.us-east-1.elb.amazonaws.com/
```

Now lets see what apps we have deployed:

```console
# emp apps
error: Request not authenticated, API token is missing, invalid or expired Log in with `emp login`.
```

As you can see, first you're going to need to login to start using the environment. In the demo environment's case, you can login with the 'fake' user and a blank password. In a real world use case, you'd use your github credentials. So lets setup the fake login:

```console
# emp login
Enter email: fake
Enter password:
Logged in.
```

NOW lets see what apps we have deployed.

```console
# emp apps
```

Good - we haven't deployed any apps, so we shouldn't see any. Lets deploy our first app - the [acme-inc](https://github.com/remind101/acme-inc) app. It's a simple app that was written by Remind for simple testing of Empire. It has two 'processes' in the [Procfile](https://github.com/remind101/acme-inc/blob/master/Procfile) - a web and worker process.

```console
# emp deploy remind101/acme-inc:latest
Pulling repository remind101/acme-inc
345c7524bc96: Download complete
a1dd7097a8e8: Download complete
23debee88b99: Download complete
31862d352883: Download complete
c7388ff7ab91: Download complete
78fb106ed050: Download complete
133fcef559c4: Download complete
Status: Downloaded newer image for remind101/acme-inc:latest
Status: Created new release v1 for acme-inc
```

So what just happened? We just told the Empire API to go out and get the 'latest' tagged image from the remind101/acme-inc repository. The Empire daemon then pulled that image down from [hub.docker.com](http://hub.docker.com/), then extracted the *Procfile* from it to analyze what processes were available. Now lets see what apps we're running:

```console
# emp apps
acme-inc      Jun 15 20:42
```

Now we can see acme-inc is running as an app in Empire. One feature of Empire is that whenever something is deployed in it, and that image contains a 'web' process, it goes ahead and launches a single instance of that process automatically. We can see that using the *ps* sub-command (*NOTE:* You only need the image name now, not the full repository name):

```console
# emp ps -a acme-inc
v1.web.ea82cf91-5324-4a11-a7b0-8a9672a119e1  1X  RUNNING  62s  "acme-inc server"
```

Lets break down each part of the output, starting with the first field - the task name. As you can see, empire task names are broken up into 3 period (.) separated parts. The first is the 'release' version of the task, in this case **v1**. The second is the process type of the task: **web**. Finally we have a random UUID which we use to ensure that our task names are uniquely named.

The next field is the resource size of the container. Empire supports the standard Heroku container sizes (1X/2X/PX), as well as more fine grained controls (256:1GB - for 256 CPU shares and 1 gigabyte of memory, for example), but we'll go over those more later on.

Next you can see that the process is **RUNNING** - sometimes you will see a task in **PENDING** as it is being booted up or torn down. Finally you get the command that the task is running, in this case **acme-inc server** or what we define as the web process in the *Procfile*.

Next lets take a look at this release and see what we can find out about it. First, what releases does acme-inc have?

```console
# emp releases -a acme-inc
v1    Jun 15 20:42  Deploy remind101/acme-inc:latest
```

As you can see we only have a single release, the initial release that was created when we deployed the app.

Next, we want to take a look at the environment variables we've set the task up with:

```console
# emp env -a acme-inc
```

It's empty as we haven't updated the process with any variables yet. Lets do that now - lets set *FOO* to *bar* and *BAT* to *faz*:

```console
# emp set -a acme-inc FOO=bar BAT=faz
Set env vars and restarted acme-inc.
```

Now we should see those variables set when we query the environment:

```console
# emp env -a acme-inc
BAT=faz
FOO=bar
```

As well we should see a new release created for acme-inc:

```console
# emp releases -a acme-inc
v1    Jun 15 20:42  Deploy remind101/acme-inc:latest
v2    Jun 15 20:44  Set BAT,FOO config vars
```

Also, if we're quick, we'll see the old version of acme-inc running while the new version *v2* comes up:

```console
# emp ps -a acme-inc
v2.web.1774d4ed-ef0e-42c0-9e71-c752ec267cd6  1X  RUNNING  23s   "acme-inc server"
v1.web.ea82cf91-5324-4a11-a7b0-8a9672a119e1  1X  RUNNING  117s  "acme-inc server"
```

Eventually, when the new task is up and running, the old one will be terminated:

```console
# emp ps -a acme-inc
v2.web.1774d4ed-ef0e-42c0-9e71-c752ec267cd6  1X  RUNNING  59s  "acme-inc server"
```

Now what if we made a mistake, and those environment variables should not have been set? That's when rollback is useful:

```console
# emp rollback -a acme-inc v1
Rolled back acme-inc to v1 as v3.
```

Rollback creates a new release that copies all the environment variables, as well as the image number back into a new release - letting us rollback to the way things used to be:

```console
# emp releases -a acme-inc
v1    Jun 15 20:42  Deploy remind101/acme-inc:latest
v2    Jun 15 20:44  Set BAT,FOO config vars
v3    Jun 15 20:45  Rollback to v1
```

If we check the new environment for acme-inc, we should see no environment variables set, just like it was back in **v1**:

```console
# emp env -a acme-inc
```

As well we will automatically roll out new versions of the tasks - these will be labeled with the new **v3** release:

```console
# emp ps -a acme-inc
v3.web.f6337ad7-2a24-4c36-a8fb-9253581a816d  1X  RUNNING  87s  "acme-inc server"
```

Now what if things are going along, and we start to see more traffic than a single task can handle? Scaling up tasks is really simple in Empire:

```console
# emp scale -a acme-inc web=3
Scaled acme-inc to web=3:1X.
```

Here we've told Empire to bring up 2 more copies (3 total) of the web process for acme-inc, and soon we should see them running in **emp ps**:

```console
# emp ps -a acme-inc
v3.web.5526a117-2746-4965-b4ea-c6f81810198f  1X  RUNNING   2m  "acme-inc server"
v3.web.e7eff5a8-f5d0-49fd-8af9-569f5f7dbddf  1X  RUNNING   2m  "acme-inc server"
v3.web.f6337ad7-2a24-4c36-a8fb-9253581a816d  1X  RUNNING   2m  "acme-inc server"
```

So far we've only played with the web worker for acme-inc, but as we have seen in the Procfile, there is also a *worker* task that acme-inc can run. Lets create a single worker task:

```console
# emp scale -a acme-inc worker=1
Scaled acme-inc to worker=1:1X.
```

Running *emp ps* we see the worker process launching - note the **PENDING** state, which means it's not finished launching:

```console
# emp ps -a acme-inc
v3.worker.b8b06012-8354-49d6-8ae2-c6a84a73add8  1X  PENDING   3m  "acme-inc worker"
v3.web.5526a117-2746-4965-b4ea-c6f81810198f     1X  RUNNING   3m  "acme-inc server"
v3.web.e7eff5a8-f5d0-49fd-8af9-569f5f7dbddf     1X  RUNNING   3m  "acme-inc server"
v3.web.f6337ad7-2a24-4c36-a8fb-9253581a816d     1X  RUNNING   3m  "acme-inc server"
```

After a few moments the process finishes coming up:

```console
# emp ps -a acme-inc
v3.worker.b8b06012-8354-49d6-8ae2-c6a84a73add8  1X  RUNNING   3m  "acme-inc worker"
v3.web.5526a117-2746-4965-b4ea-c6f81810198f     1X  RUNNING   3m  "acme-inc server"
v3.web.e7eff5a8-f5d0-49fd-8af9-569f5f7dbddf     1X  RUNNING   3m  "acme-inc server"
v3.web.f6337ad7-2a24-4c36-a8fb-9253581a816d     1X  RUNNING   3m  "acme-inc server"
```

Finally, since we're done with acme-inc, we can destroy it - removing all tasks associated with it, as well as any load balancers and internal service discovery hostnames:

```console
# emp destroy acme-inc
warning: This will destroy acme-inc and its add-ons. Please type "acme-inc" to continue:
> acme-inc
Destroyed acme-inc.
```

And if we look to see what apps we have, we now see that there are no apps left:

```console
# emp apps
```

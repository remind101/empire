# Empire :: Troubleshooting

## Deleting an Empire CloudFormation stack

If you've created an Empire CloudFormation stack and deployed an app to it, you have created an ECS Service with an attached ELB inside the VPC of your Empire stack. Before you can delete the stack, you must no longer have any services or ELBs running inside of it. You can do this by running `emp destroy <app>` for each app in your Empire cluster.

## Launching Empire

If you're using the [empire stacker](https://github.com/remind101/stacker/tree/master/conf/empire), and you get an errors like the one below when running emp commands:
```
$ emp apps
error: Get https://empire.acme-inc.com/apps: EOF
```

This most likeley means empire had trouble launching.
To view the ansible logs, SSH into the bastion, then one of the empire controllers and check the output of the following file:
```
root@ip-10-128-10-40:~# cat /var/log/upstart/ansible.log
```
If there's an error message right after "Loading /etc/empire/seed", this probably means there's unsupported characters in one of the parameters in your .env file.

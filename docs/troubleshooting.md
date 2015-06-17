# Empire :: Troubleshooting

1. [Overview](./index.md)
2. [Installing](./installing.md)
3. [Using](./using.md)
4. [Administering](./administering.md) **TODO**
5. [Troubleshooting](./troubleshooting.md)
6. [Roadmap](./roadmap.md) **TODO**

### x509: certificate signed by unknown authority with docker-compose

If you are encountering this error with using docker-compose and boot2docker,
you need to disable TLS on boot2docker. Here is how to do so:

1. Set DOCKER_TLS in boot2docker VM
  ```console
  # ssh to boot2docker from host machine (OSX)
  boot2docker up
  boot2docker ssh

  # add DOCKER_TLS=no
  sudo vi /var/lib/boot2docker/profile # should only contain DOCKER_TLS=no

  # exit from boot2docker vm
  exit

  # restart the boot2docker vm
  boot2docker restart
  ```

2. Set the docker environment variables on the host machine
  ```console
  eval "$(boot2docker shellinit)"
  ```

# Empire syslog

This is a syslog compatible server meant to run on all container hosts and ingest logs from logspout and outputs
them to one or more https endpoints via log-shuttle.

Here is a sample configuration. Two unit files, logspout and empire.syslog

``` yaml
    - name: empire.syslog.service
      command: start
      enable: true
      content: |
        [Unit]
        Description=Syslog server that pipes logs from logspout to https endpoints via log-shuttle.

        [Service]
        TimeoutStartSec=30m
        KillMode=none

        ExecStartPre=-/usr/bin/docker pull remind101/empire-syslog:latest
        ExecStart=/usr/bin/docker run --name empire-syslog --rm -h %H -p 10514:10514/udp -e SHUTTLE_URLS=${URL1},${URL2} remind101/empire-syslog
        ExecStop=-/usr/bin/docker stop empire-syslog

        [Install]
        WantedBy=multi-user.target

        [X-Fleet]
        Global=true
    - name: logspout.service
      command: start
      enable: true
      content: |
        [Unit]
        Description=Logspout
        Requires=docker.service
        After=empire.syslog.service

        [Service]
        TimeoutStartSec=30m
        KillMode=none

        ExecStartPre=-/usr/bin/docker pull gliderlabs/logspout:latest
        ExecStart=/usr/bin/docker run --name logspout --rm -h %H -p 8000:8000 -v /var/run/docker.sock:/tmp/docker.sock progrium/logspout syslog://${PRIVATE_IP}:10514
        ExecStop=-/usr/bin/docker stop logspout

        [Install]
        WantedBy=multi-user.target

        [X-Fleet]
        Global=true

```
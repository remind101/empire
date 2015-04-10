# Relay

Relay is a small go http server that knows how to run a remote command and proxy io to this command over a tcp connection.

## Overview

### 1. Start a container by posting to the relay service.

    POST /containers
    {"image":"phusion/baseimage", "command":"/bin/bash", "env": { "TERM":"x-term"}, "attach":true}

    201 Created
    {"attachURL":"rendezvous://rendez.empire.com:5000/<token>"}

### 2. The relay service will run the equivalent of the following via the docker remote api:

    docker pull phusion/baseimage
    docker run --name <token> -e TERM=x-term phusion/baseimage /bin/bash

### 3. Client connects to relay's tcp port

The relay service is listening on rendez.empire.com:5000, after a tls handshake with a client, it waits to receive the "token"
from the client, after which it tries to attach to the docker container named the same as the token, and begins pipeing stdin,stdout,stderr over tcp to the client.

If the container has already finished running, it will send any logged output to the client, and remove the container.

## Spec
Clients connect over tls.

* When the client connects, it sends a secret followed by a `\r\n` sequence. The
  secret is the session identifier.
* `\0x03` and `\0x1C` represent SIGINT and SIGQUIT respectively. Rendezvous
  closes the tcp connection if these characters are received.
* Rendezvous will echo back what it receives from the connection, back to the
  sender.
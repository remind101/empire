#!/bin/bash

if [ ! -z "$WAIT_TIME" ]; then
  sleep "$WAIT_TIME"
fi

exec /go/bin/empire $@

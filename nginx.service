#!/bin/sh

/usr/sbin/nginx -c /etc/nginx/nginx.conf -t && \
exec /usr/sbin/nginx -c /etc/nginx/nginx.conf -g "daemon off;"

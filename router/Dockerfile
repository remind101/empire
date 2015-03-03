FROM ubuntu:14.04
MAINTAINER Benjamin Guillet <beng@remind101.com>

RUN apt-get update && apt-get install -y nginx curl \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN cd /usr/local/bin && curl -L https://github.com/kelseyhightower/confd/releases/download/v0.7.1/confd-0.7.1-linux-amd64 -o confd \
 && chmod +x /usr/local/bin/confd

RUN mkdir -p /etc/confd/conf.d /etc/confd/templates
ADD ./files/etc/confd/conf.d/nginx.toml /etc/confd/conf.d/nginx.toml
ADD ./files/etc/confd/templates/nginx.conf.tmpl /etc/confd/templates/nginx.conf.tmpl

ADD ./files/usr/local/bin/confd-watch /usr/local/bin/confd-watch
RUN chmod +x /usr/local/bin/confd-watch

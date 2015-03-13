FROM gliderlabs/alpine:3.1
MAINTAINER Benjamin Guillet <beng@remind101.com>

RUN apk-install curl nginx \
  && curl -LSs https://github.com/kelseyhightower/confd/releases/download/v0.7.1/confd-0.7.1-linux-amd64 > /usr/local/bin/confd \
  && chmod +x /usr/local/bin/confd

ADD ./files /

RUN chmod +x /usr/local/bin/confd-watch

CMD /usr/local/bin/confd-watch

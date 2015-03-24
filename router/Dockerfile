FROM remind101/base
MAINTAINER Benjamin Guillet <beng@remind101.com>

RUN apt-get update && apt-get install -y nginx curl \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN curl -LSs https://github.com/kelseyhightower/confd/releases/download/v0.8.0/confd-0.8.0-linux-amd64 > /usr/local/bin/confd \
  && chmod +x /usr/local/bin/confd

ADD ./files /

CMD /usr/local/bin/confd-watch

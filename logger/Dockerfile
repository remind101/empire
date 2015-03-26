FROM remind101/base
MAINTAINER Ben Marini <ben@remind101.com>

RUN curl -sSL https://github.com/mozilla-services/heka/releases/download/v0.9.1/heka_0.9.1_amd64.deb > /tmp/heka_0.9.1_amd64.deb \
  && dpkg -i /tmp/heka_0.9.1_amd64.deb \
  && rm /tmp/heka_0.9.1_amd64.deb

# Add template files
ADD ./files /

CMD ["/sbin/my_init"]

FROM remind101/base
MAINTAINER Ben Marini <ben@remind101.com>

RUN curl -SsL -O https://github.com/mozilla-services/heka/releases/download/v0.9.1/heka_0.9.1_amd64.deb \
  && dpkg -i heka_0.9.1_amd64.deb \
  && rm heka_0.9.1_amd64.deb

ADD hekad.toml /etc/hekad/hekad.toml
ADD lua_encoders /usr/share/heka/lua_encoders

CMD ["hekad", "-config", "/etc/hekad"]

FROM datadog/docker-dd-agent

MAINTAINER Datadog <package@datadoghq.com>

RUN echo "deb http://apt-trace.datad0g.com.s3.amazonaws.com/ stable main" > /etc/apt/sources.list.d/datadog-trace.list \
 && apt-get update \
 && apt-get install -y ca-certificates \
 && apt-get install --no-install-recommends -y dd-trace-agent \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

EXPOSE 7777/tcp

CMD ["/opt/datadog-agent/bin/trace-agent"]

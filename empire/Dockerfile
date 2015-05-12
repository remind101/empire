FROM remind101/go:1.4
MAINTAINER Eric Holmes <eric@remind101.com>

USER root
RUN addgroup -g 999 docker; addgroup app docker
USER app

ENTRYPOINT ["/go/bin/empire"]
CMD ["server"]

EXPOSE 8080

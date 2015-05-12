FROM remind101/go:1.4
MAINTAINER Eric Holmes <eric@remind101.com>

RUN addgroup -g 999 docker; addgroup app docker

ENTRYPOINT ["/go/bin/empire"]
CMD ["server"]

EXPOSE 8080

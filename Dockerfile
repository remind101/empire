FROM remind101/go:1.4-onbuild
MAINTAINER Eric Holmes <eric@remind101.com>

WORKDIR /go/src/github.com/remind101/empire
ENTRYPOINT ["/go/bin/empire"]
CMD ["server"]

EXPOSE 8080

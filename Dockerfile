FROM quay.io/remind/go1.4-onbuild
MAINTAINER Eric Holmes <eric@remind101.com>

WORKDIR /go/src/github.com/remind101/empire
ENTRYPOINT ["/go/bin/empire"]

EXPOSE 8080

FROM golang:1.7.6
MAINTAINER Eric Holmes <eric@remind101.com>

LABEL version 2.0.0-internal

RUN apt-get update -yy && \
  apt-get install -yy git make curl libxml2-dev libxmlsec1-dev liblzma-dev pkg-config xmlsec1

ADD . /go/src/github.com/remind101/empire
WORKDIR /go/src/github.com/remind101/empire
RUN go install ./cmd/empire
RUN ldd /go/bin/empire || true

ENTRYPOINT ["/go/bin/empire"]
CMD ["server"]

EXPOSE 8080

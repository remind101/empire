FROM golang:1.10.8
MAINTAINER Eric Holmes <eric@remind101.com>

RUN apt-get update -yy && \
  apt-get install -yy git make curl libxml2-dev libxmlsec1-dev liblzma-dev pkg-config xmlsec1

WORKDIR /go/src/github.com/remind101/empire

ENTRYPOINT ["make"]
CMD ["test"]

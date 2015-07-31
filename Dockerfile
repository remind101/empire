FROM golang:1.4.2
MAINTAINER Eric Holmes <eric@remind101.com>

LABEL version 0.9.1

RUN go get github.com/tools/godep
ADD . /go/src/github.com/remind101/empire
WORKDIR /go/src/github.com/remind101/empire
RUN godep go install ./cmd/empire

ENTRYPOINT ["/go/bin/empire"]
CMD ["server"]

EXPOSE 8080

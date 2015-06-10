FROM golang:1.4.2
MAINTAINER Eric Holmes <eric@remind101.com>

RUN addgroup --gid 4000 app
RUN adduser --home /home/app --gecos "" --shell /bin/sh --uid 4000 --gid 4000 --disabled-password app

ADD . /go/src/github.com/remind101/empire/empire
WORKDIR /go/src/github.com/remind101/empire/empire
RUN go get github.com/tools/godep && godep go install ./cmd/empire

USER app

ENTRYPOINT ["/go/bin/empire"]
CMD ["server"]

EXPOSE 8080

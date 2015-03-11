FROM remind101/go:1.4
MAINTAINER Eric Holmes <eric@remind101.com>

COPY . /go/src/github.com/remind101/empire
WORKDIR /go/src/github.com/remind101/empire/empire
RUN godep go install ./...
ENTRYPOINT ["/go/bin/empire"]
CMD ["server"]

EXPOSE 8080

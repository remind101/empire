FROM remind101/go:1.4
MAINTAINER Ben Marini <ben@remind101.com>

RUN mkdir -p $GOPATH/src/github.com/remind101/empire/syslog
WORKDIR $GOPATH/src/github.com/remind101/empire/syslog

COPY . $GOPATH/src/github.com/remind101/empire/syslog
RUN go get github.com/heroku/log-shuttle && go build -o /go/bin/empire-syslog ./cmd/syslog

CMD ["/go/bin/empire-syslog"]

EXPOSE 10514

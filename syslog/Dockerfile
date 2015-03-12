FROM remind101/go:1.4
MAINTAINER Ben Marini <ben@remind101.com>

RUN mkdir -p /home/app
WORKDIR /home/app

COPY . /home/app
RUN go get github.com/heroku/log-shuttle && go build -o /go/bin/empire-syslog ./...

CMD ["/go/bin/empire-syslog"]

EXPOSE 10514

FROM remind101/go:1.4-newrelic

COPY . /go/src/github.com/remind101/newrelic

WORKDIR /go/src/github.com/remind101/newrelic

RUN go-wrapper download -tags newrelic_enabled ./...
RUN go-wrapper install -tags newrelic_enabled ./...
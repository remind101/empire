.PHONY: build test docker docker_test

build:
	go build .

test: build
	go test -tags newrelic_enabled -v ./...

docker:
	docker build -t remind101/newrelic .

docker_test: docker
	docker run remind101/newrelic bash -c "cd /go/src/github.com/remind101/newrelic && make test"

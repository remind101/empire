.PHONY: cmd build test

TYPE = "patch"
VERSION = "0.9.0"

cmd:
	godep go build -o build/empire ./cmd/empire

bootstrap: cmd
	go get github.com/remind101/emp
	go build -o build/emp github.com/remind101/emp # Vendor the emp command for tests
	createdb empire
	./build/empire migrate

build: Dockerfile
	docker build --no-cache -t remind101/empire:$(VERSION) .

test:
	godep go test ./... && godep go vet ./...

release:
	pip install --upgrade bumpversion
	bumpversion ${TYPE}

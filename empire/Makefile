.PHONY: cmd build test

cmd:
	godep go build -o build/empire ./cmd/empire

bootstrap: cmd
	go build -o build/emp github.com/remind101/emp # Vendor the emp command for tests
	createdb empire
	./build/empire migrate

build: Dockerfile
	docker build --no-cache -t remind101/empire .

test:
	godep go test ./... && godep go vet ./...

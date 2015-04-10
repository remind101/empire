.PHONY: cmd build test

cmd:
	godep go build -o build/relay ./cmd/relay

build: Dockerfile
	docker build --no-cache -t remind101/relay .

test:
	godep go test ./... && godep go vet ./...

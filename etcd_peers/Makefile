.PHONY: cmd bootstrap build test

cmd:
	godep go build -o build/etcd_peers ./cmd/etcd_peers

build: Dockerfile
	docker build --no-cache -t etcd_peers .

test:
	godep go test -race ./...
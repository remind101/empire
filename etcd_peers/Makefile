.PHONY: docker

build/etcd_peers: cmd/etcd_peers/main.go etcd_peers.go
	godep go build -o build/etcd_peers ./cmd/etcd_peers

docker: Dockerfile
	docker build --no-cache -t quay.io/remind/etcd_peers .

clean:
	rm -rf build

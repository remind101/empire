.PHONY: build

build/etcd_peers: cmd/etcd_peers/main.go etcd_peers.go
	go build -o build/etcd_peers ./cmd/etcd_peers

build: Dockerfile
	docker build --no-cache -t remind101/etcd_peers .

clean:
	rm -rf build

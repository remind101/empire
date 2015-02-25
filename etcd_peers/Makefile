docker: Dockerfile
	docker build --no-cache -t quay.io/remind/empire-etcd_peers .

etcd_peers: etcd_peers.go
	go build

clean:
	rm -rf etcd_peers

.PHONY: cmd docker

cmd:
	go build -o build/syslog ./...

docker: Dockerfile
	docker build --no-cache -t quay.io/remind/syslog .

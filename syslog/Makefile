.PHONY: cmd docker

cmd:
	go build -o build/syslog ./cmd/syslog/main.go

docker: Dockerfile
	docker build --no-cache -t remind101/empire-syslog .

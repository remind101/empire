.PHONY: cmd build

cmd:
	go build -o build/syslog ./cmd/syslog/main.go

build: Dockerfile
	docker build --no-cache -t remind101/empire-syslog .

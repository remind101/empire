.PHONY: cmd build test

cmd:
	godep go build -o build/empire ./cmd/empire

bootstrap: cmd
	go get github.com/heroku/hk
	go build -o build/hk github.com/heroku/hk # The current stable release doesn't have a bugfix we need for plugins.
	createdb empire
	./build/empire migrate

build: Dockerfile
	docker build --no-cache -t remind101/empire .

test:
	godep go test ./...

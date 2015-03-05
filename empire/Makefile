.PHONY: cmd build test

cmd:
	godep go build -o build/empire ./cmd/empire

bootstrap: cmd
	createdb empire
	./build/empire migrate

build: Dockerfile
	docker build --no-cache -t empire .

test:
	godep go test ./...
.PHONY: cmd docker

cmd:
	godep go build -o build/tugboat ./cmd/tugboat
	godep go build -o build/fake ./cmd/fake

docker:
	docker build --no-cache -t remind101/tugboat .

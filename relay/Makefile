.PHONY: cmd build test cert

cmd:
	godep go build -o build/relay ./cmd/relay

build: Dockerfile
	docker build --no-cache -t remind101/relay .

test:
	godep go test ./... && godep go vet ./...

cert:
	sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ca.cert.pem
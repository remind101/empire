.PHONY: test

ARTIFACTS ?= build

cmds: build/empire build/emp

clean:
	rm -rf build/*

build/empire:
	go build -o build/empire ./cmd/empire

build/emp:
	go build -o build/emp ./cmd/emp

test: build/emp
	go test -race $(shell go list ./... | grep -v /vendor/)
	./tests/deps

vet:
	go vet $(shell go list ./... | grep -v /vendor/)

$(ARTIFACTS)/all: $(ARTIFACTS)/emp-Linux-x86_64 $(ARTIFACTS)/emp-Darwin-x86_64 $(ARTIFACTS)/empire-Linux-x86_64

$(ARTIFACTS)/emp-Linux-x86_64:
	env GOOS=linux go build -o $@ ./cmd/emp
$(ARTIFACTS)/emp-Darwin-x86_64:
	env GOOS=darwin go build -o $@ ./cmd/emp

$(ARTIFACTS)/empire-Linux-x86_64:
	env GOOS=linux go build -o $@ ./cmd/empire

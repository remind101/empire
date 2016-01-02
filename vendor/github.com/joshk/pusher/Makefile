EXAMPLES := $(wildcard examples/*.go)

examples: */**.go
	for example in $(EXAMPLES); do \
		go run $$example; \
	done

fmt: */**.go
	gofmt -w -l -tabs=false -tabwidth=4 */**.go *.go

test: */**.go
	go test ./...

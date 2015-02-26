.PHONY: cmd doc client

cmd:
	godep go build -o build/empire ./cmd/empire

client: doc
	schematic doc/schema/schema.json > client/empire/client.go

doc:
	prmd combine --meta doc/schema/meta.json doc/schema/schemata/ > doc/schema/schema.json
	prmd verify doc/schema/schema.json
	prmd doc doc/schema/schema.json > doc/schema/schema.md

bootstrap: cmd
	createdb empire
	./build/empire migrate

build: Dockerfile
	docker build --no-cache -t empire .

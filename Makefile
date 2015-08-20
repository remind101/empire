.PHONY: cmd build test bootstrap

REPO = remind101/empire
TYPE = patch

cmd:
	godep go build -o build/empire ./cmd/empire

bootstrap: cmd build/emp
	createdb empire
	./build/empire migrate

build: Dockerfile
	docker build -t ${REPO} .

test: build/emp
	godep go test ./... && godep go vet ./...

build/emp:
	go get -f -u github.com/remind101/emp
	go build -o build/emp github.com/remind101/emp # Vendor the emp command for tests

bump:
	pip install --upgrade bumpversion
	bumpversion ${TYPE}

release: test bump
	# Wait for the `master` branch to build on CircleCI before running this. We'll
	# pull that image and tag it with the new version.
	docker pull ${REPO}:latest
	docker tag ${REPO} ${REPO}:$(shell cat VERSION)
	docker push ${REPO}:$(shell cat VERSION)

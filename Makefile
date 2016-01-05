.PHONY: build test bootstrap

REPO = remind101/empire
TYPE = patch

build/empire:
	go build -o build/empire ./cmd/empire

build/emp:
	go build -o build/emp ./cmd/emp

cmds: build/empire build/emp

bootstrap: cmds
	createdb empire || true
	./build/empire migrate

build: Dockerfile
	docker build -t ${REPO} .

ci: cmds test vet

test: build/emp
	go test $(shell go list ./... | grep -v /vendor/)

vet:
	go vet $(shell go list ./... | grep -v /vendor/)

bump:
	pip install --upgrade bumpversion
	bumpversion ${TYPE}

release: test bump
	# Wait for the `master` branch to build on CircleCI before running this. We'll
	# pull that image and tag it with the new version.
	docker pull ${REPO}:latest
	docker tag ${REPO} ${REPO}:$(shell cat VERSION)
	docker push ${REPO}:$(shell cat VERSION)

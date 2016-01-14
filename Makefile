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

release: release/docker release/emp release/empire release/github

release/github::
	go get -u github.com/progrium/gh-release
	gh-release create remind101/empire $(shell cat VERSION)

release/docker::
	# Wait for the `master` branch to build on CircleCI before running this. We'll
	# pull that image and tag it with the new version.
	docker pull ${REPO}:${CIRCLE_SHA1}
	docker tag ${REPO} ${REPO}:$(shell cat VERSION)
	docker push ${REPO}:$(shell cat VERSION)

release/emp: release/emp-Linux-x86_64 release/emp-Darwin-x86_64
release/empire: release/empire-Linux-x86_64 release/empire-Darwin-x86_64

release/emp-Linux-x86_64:
	env GOOS=linux go build -o release/emp-Linux-x86_64 ./cmd/emp
release/emp-Darwin-x86_64:
	env GOOS=darwin go build -o release/emp-Linux-x86_64 ./cmd/emp

release/empire-Linux-x86_64:
	env GOOS=linux go build -o release/empire-Linux-x86_64 ./cmd/empire
release/empire-Darwin-x86_64:
	env GOOS=darwin go build -o release/empire-Linux-x86_64 ./cmd/empire

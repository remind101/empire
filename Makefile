.PHONY: build test bootstrap

REPO = remind101/empire
TYPE = patch
ARTIFACTS = ${CIRCLE_ARTIFACTS}

cmds: build/empire build/emp

clean:
	rm -rf build/*

build/empire:
	go build -o build/empire ./cmd/empire

build/emp:
	go build -o build/emp ./cmd/emp

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
	./bin/release $(ARTIFACTS)

release/docker::
	# Wait for the `master` branch to build on CircleCI before running this. We'll
	# pull that image and tag it with the new version.
	docker pull ${REPO}:${CIRCLE_SHA1}
	docker tag ${REPO}:${CIRCLE_SHA1} ${REPO}:$(shell cat VERSION)
	docker push ${REPO}:$(shell cat VERSION)

release/emp: $(ARTIFACTS)/emp-Linux-x86_64 $(ARTIFACTS)/emp-Darwin-x86_64
release/empire: $(ARTIFACTS)/empire-Linux-x86_64 $(ARTIFACTS)/empire-Darwin-x86_64

$(ARTIFACTS)/emp-Linux-x86_64:
	env GOOS=linux go build -o $@ ./cmd/emp
$(ARTIFACTS)/emp-Darwin-x86_64:
	env GOOS=darwin go build -o $@ ./cmd/emp

$(ARTIFACTS)/empire-Linux-x86_64:
	env GOOS=linux go build -o $@ ./cmd/empire
$(ARTIFACTS)/empire-Darwin-x86_64:
	env GOOS=darwin go build -o $@ ./cmd/empire

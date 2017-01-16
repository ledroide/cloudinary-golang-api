SHELL:=/bin/bash
GIT_SHA ?= $(shell git rev-parse HEAD)
SEMVER := "0.1.0"
TINYSHA ?= $(shell git rev-parse HEAD | cut -c 1-8)

.PHONY: docker.build
docker.build:
	docker build -t $(TRAVIS_REPO_SLUG):$(GIT_SHA) .

.PHONY: docker
docker: docker.build
	docker login -u $(DOCKER_USER) -p $(DOCKER_PASSWORD) $(REGISTRY_PROVIDER)
	docker tag $(TRAVIS_REPO_SLUG):$(GIT_SHA) $(REPOSITORY_NAME):latest
	docker tag $(TRAVIS_REPO_SLUG):$(GIT_SHA) $(REPOSITORY_NAME):build-$(GIT_SHA)
	docker tag $(TRAVIS_REPO_SLUG):$(GIT_SHA) $(REPOSITORY_NAME):$(SEMVER)-$(TINYSHA)
	docker push $(REPOSITORY_NAME):latest
	docker push $(REPOSITORY_NAME):build-$(GIT_SHA)
	docker push $(REPOSITORY_NAME):$(SEMVER)-$(TINYSHA)

.PHONY: default
default: docker

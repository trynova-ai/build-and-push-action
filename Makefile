IMG ?= build-and-push
IMG_VERSION ?= latest
DOCKER_REPO ?= docker.io/trynova

.PHONY: docker-build
docker-build: ## Build docker image.
	docker build -t ${IMG}:${IMG_VERSION} .

.PHONY: docker-push
docker-push: ## Tag and push docker image.
	docker tag ${IMG}:${IMG_VERSION} ${DOCKER_REPO}/${IMG}:$(tag)
	docker push ${DOCKER_REPO}/${IMG}:$(tag)
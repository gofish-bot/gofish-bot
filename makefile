# Vars
DOCKER_REPO=mogensen
APP_NAME=gofish-bots

# HELP
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help

help: ## This help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

last-version:  ## last deployed version
	git describe --tags

# Full release - build, tag and push the container, deploy to k8s
release: build publish-version deploy-k8s ## Make a release by building and publishing the `{version}` and deploying to k8s

build: ## Build docker image
	@echo 'build $(VERSION) for $(DOCKER_REPO)/$(APP_NAME)'
	docker build -t $(DOCKER_REPO)/$(APP_NAME):$(VERSION) .

publish-version: tag-version ## Publish the `{version}` taged container to Docker Hub
	@echo 'publish $(VERSION) to $(DOCKER_REPO)'
	docker push $(DOCKER_REPO)/$(APP_NAME):$(VERSION)

tag-version: ## Generate container `latest` tag
	@echo 'create git tag $(VERSION)'
	git tag $(VERSION) && git push --tags || echo 'Version already released, update your version!'

deploy-k8s: ## Deploy to Kubernetes
	@echo 'deploying version $(VERSION)'
	bash ${PWD}/deployment/deploy.sh $(VERSION)

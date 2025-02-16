.PHONY: help
.DEFAULT_GOAL := help

CURRENT_REVISION = $(shell git rev-parse --short HEAD)
BUILD_LDFLAGS = "-X main.revision=$(CURRENT_REVISION)"

DOCKER_IMAGE_NAME = "rfid-poker"

help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

generate: generate-sqlc  ## Generate code

generate-sqlc:
	sqlc generate

build: ## Build the binary
	go build -ldflags $(BUILD_LDFLAGS) -o bin/$(BINARY_NAME) cmd/cmd.go

build-docker: ## Build the docker image
	docker build -t $(DOCKER_IMAGE_NAME) .
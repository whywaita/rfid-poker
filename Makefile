.PHONY: help
.DEFAULT_GOAL := help

CURRENT_REVISION = $(shell git rev-parse --short HEAD)
GIT_VERSION = $(shell git describe --tags --match='v*' --abbrev=0 2>/dev/null || echo "dev")
BUILD_LDFLAGS ?= "-X github.com/whywaita/rfid-poker/pkg/version.gitVersion=$(GIT_VERSION) -X github.com/whywaita/rfid-poker/pkg/version.gitCommit=$(CURRENT_REVISION)"
BINARY_NAME = "cmd"

DOCKER_IMAGE_NAME = "rfid-poker"

bin:
	mkdir -p $@

.PHOHY: bin/rfid-poker
bin/rfid-poker: bin
	go build -ldflags $(BUILD_LDFLAGS) -o $@ cmd/server/main.go
.PHONY: bin/wired-client
bin/wired-client: bin
	go build -ldflags $(BUILD_LDFLAGS) -o $@ cmd/wired-client/main.go
.PHONY: bin/test-wired-client
bin/test-wired-client: bin
	go build -ldflags $(BUILD_LDFLAGS) -o $@ cmd/test-wired-client/main.go

help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

generate: generate-sqlc  ## Generate code

generate-sqlc:
	sqlc generate

build: ## Build the binary
	go build -ldflags $(BUILD_LDFLAGS) -o bin/$(BINARY_NAME) cmd/server/main.go

build-docker: ## Build the docker image
	docker build -t $(DOCKER_IMAGE_NAME) .

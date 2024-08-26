.SHELL := /usr/bin/env bash

# Environment
GO := $(shell which go)
DOCKER := $(shell which docker)
GOTEST = $(GO) test
GOLIST := $(shell $(GO) list ./... | grep -v /vendor/)
YAMLFILES := $(shell git ls-files '*.yml' '*.yaml')
PWD := $(shell pwd)
BUILD_VERSION := $(shell git describe --exact-match --tags 2> /dev/null || git rev-parse --short HEAD)

# Fancy colors
GREEN  := $(shell tput -Txterm setaf 2)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

# Configuration
GOLANGCI_LINT_IMAGE = golangci/golangci-lint:latest-alpine
YAMLLINT_IMAGE = cytopia/yamllint:latest
COVERAGE_EXPORT = false # Set to 'true' if 'COVERAGE_FILE' should be kept, otherwise it is deleted.
COVERAGE_FILE = profile.cov
BINARY_NAME = calendarsync

.PHONY: all test build vendor

all: help

## Build
build: ## Build CalendarSync
	@mkdir -p bin
	$(GO) build -race -ldflags \
		"-X 'main.Version=$(BUILD_VERSION)' \
		-X 'main.BuildTime=$(shell date)' \
		-X 'main.GoogleClientID=${CS_GOOGLE_CLIENT_ID}' \
		-X 'main.GoogleClientKey=${CS_GOOGLE_CLIENT_KEY}'" \
		-o bin/$(BINARY_NAME) cmd/calendarsync/main.go

build_goreleaser:
	goreleaser build --snapshot --clean

clean: ## Clean build assets
	rm -rf ./bin

vendor: ## Vendor dependencies
	$(GO) mod vendor

## Test

test: ## Run all tests
	$(GO) run github.com/vektra/mockery/v2@v2.42.0
	$(GOTEST) -race $(GOLIST)

coverage: ## Run tests with coverage and export it into 'profile.cov'. If 'COVERAGE_EXPORT' is true, 'COVERAGE_FILE' is written
	$(GOTEST) -cover -covermode=count -coverprofile=$(COVERAGE_FILE) ./...
	$(GO) tool cover -func $(COVERAGE_FILE)
ifeq ($(COVERAGE_EXPORT), false)
	@rm $(COVERAGE_FILE)
endif

## Lint

lint: lint-go lint-yaml lint-dockerfile ## Run all linters

lint-go: ## Lint all GO files
	$(DOCKER) run --rm -it -v $(PWD):/app -w /app $(GOLANGCI_LINT_IMAGE) golangci-lint run --go "1.23"

lint-yaml: ## Lint all YAML files
	$(DOCKER) run --rm -it -v $(PWD):/data $(YAMLLINT_IMAGE) -f parsable $(YAMLFILES)
	@echo '${CYAN}TODO${RESET}'

lint-dockerfile: ## Lint Dockerfile if there is one
	@echo '${CYAN}TODO${RESET}'

## Documentation

docs-serve: ## Run 'hugo server' to serve the documentation on localhost:1313
	@cd docs && hugo server

## Help

help: ## Show this help.
	@echo '====================='
	@echo '${GREEN}CalendarSync Makefile${RESET}'
	@echo '====================='
	@echo ''
	@echo 'Usage:'
	@echo '  make ${CYAN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${CYAN}%-20s${WHITE}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${WHITE}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)

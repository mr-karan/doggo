CLI_BIN := ./bin/doggo.bin
WEB_BIN := ./bin/doggo-web.bin

HASH := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date '+%Y-%m-%d %H:%M:%S')
VERSION := ${HASH}

.PHONY: build-cli
build-cli:
	go build -o ${CLI_BIN} -ldflags="-X 'main.buildVersion=${VERSION}' -X 'main.buildDate=${BUILD_DATE}'" ./cmd/doggo/

.PHONY: build-web
build-web:
	go build -o ${WEB_BIN} -ldflags="-X 'main.buildVersion=${VERSION}' -X 'main.buildDate=${BUILD_DATE}'" ./web/

.PHONY: run-cli
run-cli: build-cli ## Build and Execute the CLI binary after the build step.
	${CLI_BIN}

.PHONY: run-web
run-web: build-web ## Build and Execute the API binary after the build step.
	${WEB_BIN} --config config-api-sample.toml

.PHONY: clean
clean:
	go clean
	- rm -rf ./bin/

.PHONY: lint
lint:
	golangci-lint run

.PHONY: docs-dev
docs-dev: ## Start Astro docs development server
	cd docs && yarn dev

.PHONY: docs-build
docs-build: ## Build Astro docs for production
	cd docs && yarn build

.PHONY: docs-preview
docs-preview: ## Preview built docs
	cd docs && yarn preview

.PHONY: docs-install
docs-install: ## Install docs dependencies
	cd docs && yarn install

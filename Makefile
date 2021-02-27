CLI_BIN := ./bin/doggo-cli.bin
API_BIN := ./bin/doggo-api.bin

HASH := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date '+%Y-%m-%d %H:%M:%S')
VERSION := ${HASH}

.PHONY: build-cli
build-cli:
	go build -o ${CLI_BIN} -ldflags="-X 'main.buildVersion=${VERSION}' -X 'main.buildDate=${BUILD_DATE}'" ./cmd/doggo/cli/

.PHONY: build-api
build-api:
	go build -o ${API_BIN} -ldflags="-X 'main.buildVersion=${VERSION}' -X 'main.buildDate=${BUILD_DATE}'" ./cmd/doggo/api/


.PHONY: build
build: build-api build-cli

.PHONY: run-cli
run-cli: build-cli ## Build and Execute the CLI binary after the build step.
	${CLI_BIN}

.PHONY: run-api
run-api: build-api ## Build and Execute the API binary after the build step.
	${API_BIN} --config config-api-sample.toml

.PHONY: clean
clean:
	go clean
	- rm -rf ./bin/

.PHONY: lint
lint:
	golangci-lint run

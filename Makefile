BIN := ./bin/doggo

HASH := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date '+%Y-%m-%d %H:%M:%S')
VERSION := ${HASH}

.PHONY: build
build:
	go build -o ${BIN} -ldflags="-X 'main.buildVersion=${VERSION}' -X 'main.buildDate=${BUILD_DATE}'" ./cmd/doggo/

.PHONY: run
run: build ## Build and Execute the binary after the build step
	${BIN}

.PHONY: clean
clean:
	go clean
	- rm -f ${BIN}

.PHONY: lint
lint:
	golangci-lint run

.PHONY: fresh
fresh: clean build

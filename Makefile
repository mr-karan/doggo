DOGGO-BIN := doggo.bin

HASH := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date '+%Y-%m-%d %H:%M:%S')
VERSION := ${HASH}

.PHONY: build
build: ## Build the doggo binary
	mkdir -p bin/; \
	cd cmd/; \
	go build  -ldflags="-X 'main.buildVersion=${VERSION}' -X 'main.buildDate=${BUILD_DATE}'" -o ${DOGGO-BIN} ./... && \
	mv ${DOGGO-BIN} ../bin/${DOGGO-BIN}

.PHONY: run
run: build ## Build and Execute the binary after the build step
	./bin/${DOGGO-BIN}

fresh: clean build

clean:
	go clean
	- rm -f ./bin/${BIN}

# pack-releases runns stuffbin packing on a given list of
# binaries. This is used with goreleaser for packing
# release builds for cross-build targets.
pack-releases:
	$(foreach var,$(RELEASE_BUILDS),stuffbin -a stuff -in ${var} -out ${var} ${STATIC} $(var);)

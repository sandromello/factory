SHORT_NAME = factory

VERSION ?= unknown
GITCOMMIT ?= $(shell git rev-parse HEAD)
DATE ?= $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")

# Common flags passed into Go's linker.
GOTEST := go test --race -v

LDFLAGS := "-s -w \
-X github.com/sandromello/factory/pkg/version.version=${VERSION} \
-X github.com/sandromello/factory/pkg/version.gitCommit=${GITCOMMIT} \
-X github.com/sandromello/factory/pkg/version.buildDate=${DATE}"

SHELL=/bin/bash
BINARY_DEST_DIR := rootfs/usr/bin

GOOS ?= linux
GOARCH ?= amd64

generate:
	go generate ./pkg/pack/generated

build: generate
	mkdir -p ${BINARY_DEST_DIR}
	env GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags ${LDFLAGS} -o ${BINARY_DEST_DIR}/fkt cmd/main.go
	upx -9 ${BINARY_DEST_DIR}/fkt

test-unit:
	${GOTEST} ./pkg/...

.PHONY: build

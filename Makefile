MODULE   := kanbanzai
PKG      := $(MODULE)/internal/buildinfo
BINARY   := kanbanzai

# Git metadata
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GIT_SHA  := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
DIRTY    := $(shell git diff --quiet 2>/dev/null && echo false || echo true)

LDFLAGS  := -X '$(PKG).Version=$(VERSION)' \
            -X '$(PKG).GitSHA=$(GIT_SHA)' \
            -X '$(PKG).BuildTime=$(BUILD_TIME)' \
            -X '$(PKG).Dirty=$(DIRTY)'

.PHONY: build install clean test

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/kanbanzai

install: build
	go install -ldflags "$(LDFLAGS)" ./cmd/kanbanzai
	kanbanzai install-record write --by makefile

clean:
	rm -f $(BINARY)

test:
	go test -race ./...

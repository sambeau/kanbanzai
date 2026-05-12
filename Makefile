MODULE   := github.com/sambeau/kanbanzai
PKG      := $(MODULE)/internal/buildinfo
BINARY   := kbz

# Git metadata
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GIT_SHA  := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
DIRTY    := $(shell git diff --quiet 2>/dev/null && echo false || echo true)

LDFLAGS  := -X '$(PKG).Version=$(VERSION)' \
            -X '$(PKG).GitSHA=$(GIT_SHA)' \
            -X '$(PKG).BuildTime=$(BUILD_TIME)' \
            -X '$(PKG).Dirty=$(DIRTY)'

.PHONY: build install clean test test-install registry-check registry-sync claude-skills-check generate setup

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/kbz

install: build
	go install -ldflags "$(LDFLAGS)" ./cmd/kbz
	./$(BINARY) install-record write --by makefile

clean:
	rm -f $(BINARY)

test:
	go test -race ./...

test-install:
	go test ./internal/kbzinit -tags=e2e -race -run TestE2E_ -count=1

registry-check:
	go run ./cmd/kbz docs check

registry-sync:
	go run ./cmd/kbz docs sync

claude-skills-check:
	go test -v -run TestClaudeSkills ./internal/claudeskills/

setup:
	git config core.hooksPath .githooks
	@echo "Git hooks installed from .githooks/"

generate:
	go generate ./internal/binding/
	@echo "router_gen.go regenerated from routing.yaml"

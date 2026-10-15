# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

# Build directory
BUILDDIR=bin

# Install directory for `make install` (override with: make install INSTALLDIR=/usr/local/bin)
INSTALLDIR?=$(HOME)/bin

# Discover all example directories (each contains a main package)
EXAMPLES=$(shell ls -d examples/*/ 2>/dev/null | grep -v internal | sed 's|examples/||;s|/||')

.DEFAULT_GOAL := help

all: test build

build:
	@mkdir -p $(BUILDDIR)
	@for example in $(EXAMPLES); do \
		echo "building bin/$$example"; \
		$(GOBUILD) -o $(BUILDDIR)/$$example ./examples/$$example; \
	done

test:
	$(GOTEST) -v ./...

run-tests:
	$(GOTEST) ./...

fmt-check:
	@unformatted=$$(gofmt -l . | grep -v '^examples/'); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi

vet:
	$(GOCMD) vet ./...

check: fmt-check vet run-tests

clean:
	rm -rf $(BUILDDIR)
	find . -name '*~' -delete

deps:
	$(GOMOD) download
	$(GOMOD) tidy

upgrade-deps:
	$(GOCMD) get github.com/go-resty/resty/v2@v2.17.2 golang.org/x/net@v0.44.0 golang.org/x/term@v0.36.0 golang.org/x/sys@v0.38.0
	$(GOMOD) tidy

build-linux:
	@mkdir -p $(BUILDDIR)
	@for example in $(EXAMPLES); do \
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILDDIR)/$${example}_linux -v ./examples/$$example; \
	done

install: build
	@mkdir -p $(INSTALLDIR)
	@for example in $(EXAMPLES); do \
		echo "installing $(INSTALLDIR)/$$example"; \
		cp -f $(BUILDDIR)/$$example $(INSTALLDIR)/$$example; \
	done

install-hooks:
	git config core.hooksPath .githooks
	@chmod +x .githooks/* 2>/dev/null || true
	@echo "Git hooks installed (.githooks). Binaries and secrets will be blocked at commit time."

dev-deps: install-hooks
	$(GOCMD) install golang.org/x/tools/cmd/goimports@latest
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint:
	golangci-lint run

fmt:
	goimports -w .
	$(GOCMD) fmt ./...

help:
	@echo "Available targets:"
	@echo "  make all         - Run tests and build all examples"
	@echo "  make build       - Build all example programs into bin/"
	@echo "  make test        - Run all tests (verbose)"
	@echo "  make run-tests   - Run the full test suite"
	@echo "  make check       - gofmt gate + go vet + full test suite"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make deps        - Download and tidy dependencies"
	@echo "  make build-linux - Cross-compile all examples for Linux amd64"
	@echo "  make install     - Build, then copy all examples to ~/bin (override INSTALLDIR=...)"
	@echo "  make install-hooks - Install git pre-commit guard (blocks binaries/secrets)"
	@echo "  make fmt         - Format code"
	@echo "  make lint        - Run linter"
	@echo "  make help           - Show this help message"
	@echo "  make list-examples  - List all example programs"

list-examples:
	@for example in $(EXAMPLES); do echo "bin/$$example"; done

.PHONY: all build test run-tests fmt-check vet check clean deps build-linux install install-hooks dev-deps lint fmt help list-examples

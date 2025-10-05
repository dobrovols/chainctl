GO ?= go
GOCACHE ?= $(CURDIR)/.gocache
GOFLAGS ?=

GO_BIN_DIR := $(strip $(shell $(GO) env GOBIN))
ifeq ($(GO_BIN_DIR),)
GO_BIN_DIR := $(strip $(shell $(GO) env GOPATH))/bin
endif
GOFUMPT := $(GO_BIN_DIR)/gofumpt
GOLANGCI_LINT := $(GO_BIN_DIR)/golangci-lint

PACKAGES = ./...
FMT_DIRS = cmd internal pkg test

.PHONY: fmt lint test bench tidy verify

fmt:
	@echo "==> Running go fmt"
	@GOCACHE=$(GOCACHE) $(GO) fmt ./...
	@if [ -x "$(GOFUMPT)" ]; then \
		echo "==> Running gofumpt"; \
		"$(GOFUMPT)" -w $(FMT_DIRS); \
	else \
		echo "WARN: gofumpt not installed; install via '$(GO) install mvdan.cc/gofumpt@v0.6.0'"; \
	fi

lint:
	@if [ -x "$(GOLANGCI_LINT)" ]; then \
		echo "==> Running golangci-lint"; \
		GOCACHE=$(GOCACHE) "$(GOLANGCI_LINT)" run --timeout=5m ./...; \
	else \
		echo "WARN: golangci-lint not installed; install via '$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2'"; \
	fi

TEST_FLAGS ?=

test:
	@echo "==> Running unit tests"
	@GOCACHE=$(GOCACHE) $(GO) test $(GOFLAGS) $(TEST_FLAGS) $(PACKAGES)

bench:
	@echo "==> Running benchmarks"
	@GOCACHE=$(GOCACHE) $(GO) test $(GOFLAGS) -bench=. -run=^$$ $(PACKAGES)

mod:
	@GOCACHE=$(GOCACHE) $(GO) mod tidy

verify:
	@echo "==> Formatting"
	@$(MAKE) fmt
	@echo "==> Linting"
	@$(MAKE) lint
	@echo "==> Tests"
	@GOCACHE=$(GOCACHE) $(GO) test $(GOFLAGS) ./...
	@echo "==> Benchmarks"
	@GOCACHE=$(GOCACHE) $(GO) test $(GOFLAGS) -bench=. -run=^$$ ./pkg/...

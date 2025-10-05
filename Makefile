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
UNIT_PACKAGES = ./cmd/... ./internal/... ./pkg/... ./test/unit/...
INTEGRATION_PACKAGES = ./test/integration/...
E2E_PACKAGES = ./test/e2e/...
FMT_DIRS = cmd internal pkg test

CHAINCTL_SKIP_E2E ?= 0

.PHONY: fmt lint test test-unit test-integration test-e2e bench tidy verify ensure-gocache

ensure-gocache:
	@mkdir -p $(GOCACHE)

fmt: ensure-gocache
	@echo "==> Running go fmt"
	@GOCACHE=$(GOCACHE) $(GO) fmt ./...
	@if [ -x "$(GOFUMPT)" ]; then \
		echo "==> Running gofumpt"; \
		"$(GOFUMPT)" -w $(FMT_DIRS); \
	else \
		echo "WARN: gofumpt not installed; install via '$(GO) install mvdan.cc/gofumpt@v0.6.0'"; \
	fi

lint: ensure-gocache
	@if [ -x "$(GOLANGCI_LINT)" ]; then \
		echo "==> Running golangci-lint"; \
		GOCACHE=$(GOCACHE) "$(GOLANGCI_LINT)" run --timeout=5m ./...; \
	else \
		echo "WARN: golangci-lint not installed; install via '$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2'"; \
	fi

TEST_FLAGS ?=

test-unit: ensure-gocache
	@echo "==> Running unit tests"
	@GOCACHE=$(GOCACHE) $(GO) test $(GOFLAGS) $(TEST_FLAGS) $(UNIT_PACKAGES)

test-integration: ensure-gocache
	@echo "==> Running integration tests"
	@GOCACHE=$(GOCACHE) $(GO) test $(GOFLAGS) $(TEST_FLAGS) $(INTEGRATION_PACKAGES)

test-e2e: ensure-gocache
	@if [ "$(CHAINCTL_SKIP_E2E)" = "1" ]; then \
		echo "==> Skipping end-to-end tests (CHAINCTL_SKIP_E2E=1)"; \
	else \
		echo "==> Running end-to-end tests"; \
		GOCACHE=$(GOCACHE) $(GO) test $(GOFLAGS) $(TEST_FLAGS) $(E2E_PACKAGES); \
	fi

test: test-unit test-integration test-e2e

bench: ensure-gocache
	@echo "==> Running benchmarks"
	@GOCACHE=$(GOCACHE) $(GO) test $(GOFLAGS) -bench=. -run=^$$ $(PACKAGES)

mod: ensure-gocache
	@GOCACHE=$(GOCACHE) $(GO) mod tidy

verify: ensure-gocache
	@echo "==> Formatting"
	@$(MAKE) fmt
	@echo "==> Linting"
	@$(MAKE) lint
	@echo "==> Tests"
	@$(MAKE) test
	@echo "==> Benchmarks"
	@$(MAKE) bench

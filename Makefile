GO ?= go
GOCACHE ?= $(CURDIR)/.gocache
GOFLAGS ?=

PACKAGES = ./...
FMT_DIRS = cmd internal pkg test

.PHONY: fmt lint test bench tidy verify

fmt:
	@echo "==> Running gofmt"
	@$(GO)fmt ./...
	@if command -v gofumpt >/dev/null 2>&1; then \
		echo "==> Running gofumpt"; \
		gofumpt -w $(FMT_DIRS); \
	else \
		echo "WARN: gofumpt not installed; install via '$(GO) install mvdan.cc/gofumpt@latest'"; \
	fi

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "==> Running golangci-lint"; \
		GOCACHE=$(GOCACHE) golangci-lint run ./...; \
	else \
		echo "ERROR: golangci-lint not installed; install via '$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest'"; \
		exit 1; \
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

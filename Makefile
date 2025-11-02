# Basic Makefile for common tasks in this repository

SHELL := /bin/sh

GO ?= go
PKGS := $(shell $(GO) list ./...)
COVER_FILE ?= coverage.out
COVER_HTML ?= coverage.html

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@printf "Available targets:\n"
	@awk -F':|##' '/^[a-zA-Z0-9_\/\.\-%]+:.*##/ {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$NF}' $(MAKEFILE_LIST) | sort

# ------------------------------------------------------------------------------

.PHONY: all
all: test ## Run tests (alias)

.PHONY: deps
deps: ## Download dependencies
	$(GO) mod download

.PHONY: tidy
tidy: ## Tidy go.mod and go.sum
	$(GO) mod tidy

.PHONY: modverify
modverify: ## Verify dependencies
	$(GO) mod verify

.PHONY: fmt
fmt: ## Format code (gofmt) and goimports if available
	$(GO) fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		echo "running goimports"; \
		goimports -w -local github.com/atfromhome/goreus . ; \
	fi

.PHONY: fmtcheck
fmtcheck: ## Check formatting (fails if unformatted files exist)
	@unformatted=$$(gofmt -l . | grep -v '^vendor/' || true); \
	if [ -n "$$unformatted" ]; then \
		echo "Unformatted files:"; echo "$$unformatted"; exit 1; \
	fi

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: test
test: ## Run tests
	$(GO) test ./...

.PHONY: test-race
test-race: ## Run tests with race detector
	$(GO) test -race ./...

.PHONY: cover
cover: ## Run tests with coverage (writes $(COVER_FILE))
	$(GO) test -race -coverprofile=$(COVER_FILE) -covermode=atomic ./...
	@echo "Coverage written to $(COVER_FILE)"

.PHONY: coverhtml
coverhtml: cover ## Generate HTML coverage report (depends on cover)
	$(GO) tool cover -html=$(COVER_FILE) -o $(COVER_HTML)
	@echo "HTML coverage report: $(COVER_HTML)"

.PHONY: bench
bench: ## Run benchmarks
	$(GO) test -bench=. -benchmem ./...

.PHONY: ci
ci: tidy fmt vet test-race cover ## Run CI tasks locally

.PHONY: clean
clean: ## Clean generated files
	@rm -f $(COVER_FILE) $(COVER_HTML)
	@rm -rf ./bin ./dist ./out ./build ./coverage ./cover ./storage
	@echo "Cleaned."

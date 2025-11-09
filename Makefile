.PHONY: help test test-verbose test-cover integration-test-setup integration-test integration-test-teardown integration-test-all test-all build examples build-all fmt vet lint check deps tidy update-deps clean coverage goreleaser-check goreleaser-snapshot release-check release-prep release-patch release-minor release-major release ci

# Default target
help: ## Display available make targets
	@echo "Available targets:"
	@echo "  help                         Display available make targets"
	@echo "  test                         Run unit tests"
	@echo "  test-verbose                 Run unit tests with verbose output"
	@echo "  test-cover                   Run unit tests with coverage"
	@echo "  integration-test-setup       Setup integration test environment"
	@echo "  integration-test             Run integration tests (requires setup first)"
	@echo "  integration-test-teardown    Teardown integration test environment"
	@echo "  integration-test-all         Run full integration test cycle (setup -> test -> teardown)"
	@echo "  test-all                     Run both unit and integration tests"
	@echo "  build                Build all packages"
	@echo "  examples             Build all example programs"
	@echo "  build-all            Build everything (packages + examples)"
	@echo "  fmt                  Format code using gofmt"
	@echo "  vet                  Run go vet for static analysis"
	@echo "  lint                 Run formatting and static analysis"
	@echo "  check                Run all quality checks (format, vet, test)"
	@echo "  deps                 Download dependencies"
	@echo "  tidy                 Tidy go.mod and go.sum"
	@echo "  update-deps          Update dependencies to latest versions"
	@echo "  clean                Clean build artifacts and test cache"
	@echo "  coverage             Generate HTML coverage report"
	@echo "  goreleaser-check     Validate GoReleaser configuration"
	@echo "  goreleaser-snapshot  Test release locally without publishing"
	@echo "  release-check        Verify release prerequisites"
	@echo "  release-patch        Prepare patch release (auto-increment patch version)"
	@echo "  release-minor        Prepare minor release (auto-increment minor version)"
	@echo "  release-major        Prepare major release (auto-increment major version)"
	@echo "  release-prep         Prepare release (usage: make release-prep VERSION=v0.2.0)"
	@echo "  release              Full release preparation workflow"
	@echo "  ci                   Run full CI pipeline"

# Variables
GO_FILES := $(shell find . -name '*.go' -not -path './test/*' -not -path './.git/*')
EXAMPLES_DIR := ./examples
BUILD_DIR := ./build
COVERAGE_FILE := coverage.out

# Test targets
test: ## Run unit tests
	@echo "Running unit tests..."
	go test ./...

test-verbose: ## Run unit tests with verbose output
	@echo "Running unit tests with verbose output..."
	go test -v ./...

test-cover: ## Run unit tests with coverage
	@echo "Running unit tests with coverage..."
	go test -cover ./...
	go test -coverprofile=$(COVERAGE_FILE) ./...

integration-test-setup: ## Setup integration test environment
	@echo "Starting ContextForge integration test environment..."
	@./scripts/integration-test-setup.sh

integration-test: ## Run integration tests (requires setup first)
	@echo "Running integration tests..."
	INTEGRATION_TESTS=true go test -v -tags=integration -timeout=5m ./test/integration/...

integration-test-teardown: ## Teardown integration test environment
	@echo "Stopping ContextForge integration test environment..."
	@./scripts/integration-test-teardown.sh

integration-test-all: ## Run full integration test cycle (setup -> test -> teardown)
	@echo "Running full integration test cycle..."
	@./scripts/integration-test-setup.sh
	@sleep 2
	@echo "Running integration tests..."
	@INTEGRATION_TESTS=true go test -v -tags=integration -timeout=5m ./test/integration/... || (./scripts/integration-test-teardown.sh && exit 1)
	@./scripts/integration-test-teardown.sh

test-all: test integration-test-all ## Run both unit and integration tests

# Build targets
build: ## Build all packages
	@echo "Building all packages..."
	go build ./...

examples: ## Build all example programs
	@echo "Building examples..."
	@mkdir -p $(BUILD_DIR)

build-all: build examples ## Build everything (packages + examples)

# Code quality targets
fmt: ## Format code using gofmt
	@echo "Formatting code..."
	gofmt -s -w .

vet: ## Run go vet for static analysis
	@echo "Running go vet..."
	go vet ./...

lint: fmt vet ## Run formatting and static analysis

check: lint test ## Run all quality checks (format, vet, test)

# Dependency management
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download

tidy: ## Tidy go.mod and go.sum
	@echo "Tidying dependencies..."
	go mod tidy

update-deps: ## Update dependencies to latest versions
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Development targets
clean: ## Clean build artifacts and test cache
	@echo "Cleaning build artifacts..."
	go clean -testcache -cache
	rm -rf $(BUILD_DIR)
	rm -f $(COVERAGE_FILE) coverage.html
	rm -rf dist/

coverage: test-cover ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

# GoReleaser targets
goreleaser-check: ## Validate GoReleaser configuration
	@echo "Checking GoReleaser configuration..."
	@command -v goreleaser >/dev/null 2>&1 || (echo "Error: goreleaser not installed. Run: go install github.com/goreleaser/goreleaser/v2@latest" && exit 1)
	goreleaser check

goreleaser-snapshot: ## Test release locally without publishing
	@echo "Creating snapshot release..."
	@command -v goreleaser >/dev/null 2>&1 || (echo "Error: goreleaser not installed" && exit 1)
	goreleaser release --snapshot --clean

# Release targets
release-check: ## Verify release prerequisites
	@echo "Checking release prerequisites..."
	@command -v git >/dev/null 2>&1 || (echo "Error: git is required" && exit 1)
	@command -v goreleaser >/dev/null 2>&1 || (echo "Error: goreleaser not installed. Run: go install github.com/goreleaser/goreleaser/v2@latest" && exit 1)
	@git diff --quiet || (echo "Error: uncommitted changes detected" && exit 1)
	@git diff --cached --quiet || (echo "Error: staged changes detected" && exit 1)
	@[ -z "$$(git status --porcelain)" ] || (echo "Error: working directory not clean" && exit 1)
	@goreleaser check || (echo "Error: goreleaser config validation failed" && exit 1)
	@echo "Prerequisites check passed"

release-patch: release-check ## Prepare patch release (auto-increment patch version)
	@echo "Calculating patch version bump..."
	@./scripts/bump-version.sh patch
	@$(MAKE) release-prep VERSION=$$(cat .next-version)

release-minor: release-check ## Prepare minor release (auto-increment minor version)
	@echo "Calculating minor version bump..."
	@./scripts/bump-version.sh minor
	@$(MAKE) release-prep VERSION=$$(cat .next-version)

release-major: release-check ## Prepare major release (auto-increment major version)
	@echo "Calculating major version bump..."
	@./scripts/bump-version.sh major
	@$(MAKE) release-prep VERSION=$$(cat .next-version)

release-prep: ## Prepare release (usage: make release-prep VERSION=v0.2.0)
	@test -n "$(VERSION)" || (echo "Error: VERSION is required. Usage: make release-prep VERSION=v0.2.0" && exit 1)
	@echo "Preparing release $(VERSION)..."
	@./scripts/prepare-release.sh $(VERSION)

release: release-check release-prep ## Full release preparation workflow

ci: deps lint test build ## Run full CI pipeline

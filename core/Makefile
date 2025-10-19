.PHONY: all build build-gui run-gui gui-deps clean test run help install deps example

# Binary names
BINARY_NAME=devdashboard

# Build directory
BUILD_DIR=bin

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

all: deps build

## help: Display this help message
help:
	@echo "DevDashboard - Makefile Commands"
	@echo "================================="
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /'

## deps: Download and verify dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) verify
	@echo "Dependencies downloaded successfully"

## build: Build the CLI application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/devdashboard
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## build-gui: Build the optional GUI (requires 'gui' build tag)
build-gui:
	@echo "Building $(BINARY_NAME) GUI (optional)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -tags gui -o $(BUILD_DIR)/$(BINARY_NAME)-gui ./cmd/devdashboard-gui
	@echo "GUI build complete: $(BUILD_DIR)/$(BINARY_NAME)-gui"

## run-gui: Run the GUI (builds first if needed)
run-gui: build-gui
	@echo "Running $(BINARY_NAME) GUI..."
	./$(BUILD_DIR)/$(BINARY_NAME)-gui

## gui-deps: Vendor/tidy dependencies including GUI libs
gui-deps:
	@echo "Ensuring GUI dependencies (fyne) are present..."
	$(GOMOD) tidy
	$(GOMOD) vendor
	@echo "GUI dependencies ready"

## examples: Build all example programs
examples:
	@echo "Building examples..."
	@for dir in examples/*/; do \
		if [ -f "$$dir/main.go" ]; then \
			echo "Building $$(basename $$dir)..."; \
			(cd "$$dir" && $(GOBUILD) -o $$(basename $$dir)) || exit 1; \
		fi; \
	done
	@echo "All examples built successfully"

## build-examples: Alias for examples target
build-examples: examples

## clean-examples: Clean example binaries
clean-examples:
	@echo "Cleaning example binaries..."
	@for dir in examples/*/; do \
		if [ -f "$$dir/main.go" ]; then \
			rm -f "$$dir/$$(basename $$dir)"; \
			rm -f "$$dir/main"; \
		fi; \
	done
	@echo "Example binaries cleaned"

## install: Install the CLI tool to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install ./cmd/devdashboard
	@echo "Installation complete"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) --cover -count=1 ./pkg/...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## fmt: Format Go source files
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "Formatting complete"

## clean: Remove build artifacts
clean: clean-examples
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "Clean complete"

## tidy: Tidy and verify module dependencies
tidy:
	@echo "Tidying module dependencies..."
	$(GOMOD) tidy
	$(GOMOD) verify
	@echo "Tidy complete"

## run-github: Run CLI tool with GitHub example (requires env vars)
run-github: build
	@echo "Running $(BINARY_NAME) with GitHub..."
	@REPO_PROVIDER=github ./$(BUILD_DIR)/$(BINARY_NAME) $(CMD)

## run-gitlab: Run CLI tool with GitLab example (requires env vars)
run-gitlab: build
	@echo "Running $(BINARY_NAME) with GitLab..."
	@REPO_PROVIDER=gitlab ./$(BUILD_DIR)/$(BINARY_NAME) $(CMD)

## lint: Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run ./...
	@echo "Linting complete"

## check: Run various checks (fmt, vet, lint, test)
check: fmt lint
	@echo "Running go vet..."
	$(GOCMD) vet ./...
	@echo "Running tests..."
	$(GOTEST) ./...
	@echo "All checks passed"

.PHONY: all build clean test run help install deps example

# Binary names
BINARY_NAME=devdashboard
EXAMPLE_BINARY=basic_usage

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

## example: Build the example program
example:
	@echo "Building example..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(EXAMPLE_BINARY) ./examples/basic_usage.go
	@echo "Build complete: $(BUILD_DIR)/$(EXAMPLE_BINARY)"

## run-example: Build and run the example program
run-example: example
	@echo "Running example..."
	@./$(BUILD_DIR)/$(EXAMPLE_BINARY)

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
clean:
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

## check: Run various checks (fmt, vet, test)
check: fmt
	@echo "Running go vet..."
	$(GOCMD) vet ./...
	@echo "Running tests..."
	$(GOTEST) ./...
	@echo "All checks passed"

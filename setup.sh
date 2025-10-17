#!/usr/bin/env bash

set -e

# DevDashboard Setup Script
# This script helps you quickly set up and test the DevDashboard CLI tool

echo "=================================="
echo "DevDashboard Setup Script"
echo "=================================="
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    echo "Please install Go 1.21 or higher from https://golang.org/dl/"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "✓ Found Go version: $GO_VERSION"

# Download dependencies
echo ""
echo "Downloading dependencies..."
go mod download
go mod verify
echo "✓ Dependencies downloaded"

# Build the CLI tool
echo ""
echo "Building CLI tool..."
mkdir -p bin
go build -o bin/devdashboard ./cmd/devdashboard
echo "✓ CLI tool built successfully: bin/devdashboard"

# Build the example
echo ""
echo "Building example program..."
go build -o bin/basic_usage ./examples/basic_usage.go
echo "✓ Example program built successfully: bin/basic_usage"

# Run tests
echo ""
echo "Running tests..."
go test ./... -v
echo "✓ All tests passed"

# Setup complete
echo ""
echo "=================================="
echo "Setup Complete!"
echo "=================================="
echo ""
echo "Quick Start Commands:"
echo ""
echo "1. Test with a public GitHub repository:"
echo "   export REPO_PROVIDER=github"
echo "   export REPO_OWNER=golang"
echo "   export REPO_NAME=example"
echo "   ./bin/devdashboard repo-info"
echo ""
echo "2. Test with a public GitLab repository:"
echo "   export REPO_PROVIDER=gitlab"
echo "   export REPO_OWNER=gitlab-org"
echo "   export REPO_NAME=gitlab-foss"
echo "   ./bin/devdashboard repo-info"
echo ""
echo "3. Run the example program:"
echo "   ./bin/basic_usage"
echo ""
echo "4. For private repositories, set REPO_TOKEN:"
echo "   export REPO_TOKEN=your-token-here"
echo ""
echo "For more information, see README.md or QUICKSTART.md"
echo ""

# Optionally run a quick test
read -p "Would you like to run a quick test with a public repository? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo ""
    echo "Testing with golang/example repository..."
    export REPO_PROVIDER=github
    export REPO_OWNER=golang
    export REPO_NAME=example
    ./bin/devdashboard repo-info
    echo ""
    echo "✓ Test successful!"
fi

echo ""
echo "Happy coding!"

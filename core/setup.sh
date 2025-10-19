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

# Example build removed (legacy commands deprecated)
echo ""
echo "Skipping example program build (only dependency-report command is supported now)..."
echo ""
echo ""

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
echo "1. Create a config file (repos.yaml):"
echo "   cat > repos.yaml <<'EOF'"
echo "   providers:"
echo "     - name: github"
echo "       token: \"\""
echo "   repositories:"
echo "     - provider: github"
echo "       owner: golang"
echo "       repository: go"
echo "       analyzer: poetry"
echo "       packages:"
echo "         - fmt"
echo ""
echo "2. Run dependency report:"
echo "   ./bin/devdashboard dependency-report repos.yaml"
echo ""
echo "3. (Optional) JSON output: ./bin/devdashboard dependency-report repos.yaml --format json --json-indent"
echo ""
echo "For more information, see README.md or QUICKSTART.md"
echo ""

# Optionally run a quick test
read -p "Would you like to run a quick test with a public repository? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo ""
    echo "Quick test skipped: legacy repo-info command removed."
    echo "To test, create a repos.yaml and run:"
    echo "  ./bin/devdashboard dependency-report repos.yaml"
    echo ""
    echo ""
    echo ""
    echo "✓ Dependency report CLI available."
fi

echo ""
echo "Happy coding!"

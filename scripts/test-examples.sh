#!/usr/bin/env bash
# Script to test all example programs in the examples directory
# This ensures each example can build and vet successfully

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
EXAMPLES_DIR="$PROJECT_DIR/core/examples"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "======================================"
echo "Testing DevDashboard Examples"
echo "======================================"
echo ""

# Track results
TOTAL=0
PASSED=0
FAILED=0
FAILED_EXAMPLES=()

# Find all example directories with main.go
for dir in "$EXAMPLES_DIR"/*/; do
    if [ ! -f "$dir/main.go" ]; then
        continue
    fi

    TOTAL=$((TOTAL + 1))
    EXAMPLE_NAME=$(basename "$dir")

    echo "Testing: $EXAMPLE_NAME"
    echo "----------------------------------------"

    # Change to example directory
    cd "$dir"

    # Disable workspace mode - examples are independent modules with replace directives
    export GOWORK=off

    # Run go mod tidy
    echo "  → Running go mod tidy..."
    if go mod tidy 2>&1; then
        echo -e "    ${GREEN}✓${NC} go mod tidy passed"
    else
        echo -e "    ${RED}✗${NC} go mod tidy failed"
        FAILED=$((FAILED + 1))
        FAILED_EXAMPLES+=("$EXAMPLE_NAME (mod tidy)")
        echo ""
        continue
    fi

    # Run go vet
    echo "  → Running go vet..."
    if go vet 2>&1; then
        echo -e "    ${GREEN}✓${NC} go vet passed"
    else
        echo -e "    ${RED}✗${NC} go vet failed"
        FAILED=$((FAILED + 1))
        FAILED_EXAMPLES+=("$EXAMPLE_NAME (vet)")
        echo ""
        continue
    fi

    # Build the example
    echo "  → Building..."
    if go build -o /dev/null 2>&1; then
        echo -e "    ${GREEN}✓${NC} build passed"
    else
        echo -e "    ${RED}✗${NC} build failed"
        FAILED=$((FAILED + 1))
        FAILED_EXAMPLES+=("$EXAMPLE_NAME (build)")
        echo ""
        continue
    fi

    # All checks passed
    PASSED=$((PASSED + 1))
    echo -e "  ${GREEN}✓ All checks passed${NC}"
    echo ""
done

# Print summary
echo "======================================"
echo "Summary"
echo "======================================"
echo "Total examples: $TOTAL"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"

if [ $FAILED -gt 0 ]; then
    echo ""
    echo "Failed examples:"
    for example in "${FAILED_EXAMPLES[@]}"; do
        echo -e "  ${RED}✗${NC} $example"
    done
    echo ""
    exit 1
else
    echo ""
    echo -e "${GREEN}All examples passed!${NC}"
    exit 0
fi

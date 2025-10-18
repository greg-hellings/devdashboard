#!/usr/bin/env bash
# Helper script to update the vendorHash in flake.nix
# This is useful when dependencies change in go.mod

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FLAKE_DIR="$(dirname "$SCRIPT_DIR")"

cd "$FLAKE_DIR"

echo "🔍 Building to determine correct vendorHash..."
echo ""

# Attempt to build and capture the hash mismatch error
BUILD_OUTPUT=$(nix build .#devdashboard 2>&1 || true)

# Extract the actual hash from the error message
ACTUAL_HASH=$(echo "$BUILD_OUTPUT" | grep -A 1 "got:" | tail -n 1 | sed 's/^[[:space:]]*//' | sed 's/got:[[:space:]]*//')

if [ -z "$ACTUAL_HASH" ]; then
    echo "❌ Could not determine vendorHash from build output."
    echo "   The build may have succeeded, or there's a different error."
    echo ""
    echo "Build output:"
    echo "$BUILD_OUTPUT"
    exit 1
fi

echo "✅ Found correct vendorHash: $ACTUAL_HASH"
echo ""

# Get current hash from flake.nix
CURRENT_HASH=$(grep 'vendorHash = ' flake.nix | sed 's/.*vendorHash = "\(.*\)";/\1/')

if [ "$CURRENT_HASH" = "$ACTUAL_HASH" ]; then
    echo "✅ vendorHash is already correct!"
    exit 0
fi

echo "📝 Updating flake.nix..."
echo "   Old hash: $CURRENT_HASH"
echo "   New hash: $ACTUAL_HASH"
echo ""

# Update the vendorHash in flake.nix
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    sed -i '' "s|vendorHash = \".*\";|vendorHash = \"$ACTUAL_HASH\";|" flake.nix
else
    # Linux
    sed -i "s|vendorHash = \".*\";|vendorHash = \"$ACTUAL_HASH\";|" flake.nix
fi

echo "✅ Successfully updated vendorHash in flake.nix"
echo ""
echo "🔨 Verifying build..."
if nix build .#devdashboard 2>&1 | grep -q "error:"; then
    echo "❌ Build still failing. Please check manually."
    exit 1
else
    echo "✅ Build successful!"
fi

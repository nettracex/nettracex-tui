#!/bin/bash

# Setup script for GoReleaser
set -e

echo "🚀 Setting up GoReleaser for NetTraceX..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go first."
    exit 1
fi

echo "✅ Go is installed: $(go version)"

# Install GoReleaser
echo "📦 Installing GoReleaser..."
go install github.com/goreleaser/goreleaser@latest

# Verify installation
if command -v goreleaser &> /dev/null; then
    echo "✅ GoReleaser installed: $(goreleaser --version)"
else
    echo "❌ GoReleaser installation failed"
    exit 1
fi

# Check configuration
echo "🔍 Checking GoReleaser configuration..."
if goreleaser check; then
    echo "✅ GoReleaser configuration is valid"
else
    echo "❌ GoReleaser configuration has issues"
    exit 1
fi

# Test build
echo "🔨 Testing GoReleaser build..."
if goreleaser build --snapshot --clean; then
    echo "✅ GoReleaser test build successful"
    echo "📁 Build artifacts are in the 'dist' directory"
else
    echo "❌ GoReleaser test build failed"
    exit 1
fi

echo ""
echo "🎉 GoReleaser setup complete!"
echo ""
echo "Next steps:"
echo "1. Create a GitHub release tag: git tag v1.0.0 && git push origin v1.0.0"
echo "2. The GitHub Action will automatically build and release"
echo "3. Or run locally: goreleaser release --snapshot --clean"
echo ""
echo "Useful commands:"
echo "- make goreleaser-check     # Check configuration"
echo "- make goreleaser-build     # Test build"
echo "- make goreleaser-release-dry # Dry run release"
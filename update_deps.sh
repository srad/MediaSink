#!/bin/bash

# Update Dependencies Script
# Handles the retracted sqlite3 package issue automatically

echo "🔄 Updating Go dependencies..."

# Get the latest stable (non-retracted) version of sqlite3
echo "🔍 Finding latest stable SQLite driver version..."
LATEST_SQLITE_VERSION=$(go list -m -versions github.com/mattn/go-sqlite3 | awk '{print $(NF-1)}' 2>/dev/null)

if [ -z "$LATEST_SQLITE_VERSION" ]; then
    echo "⚠️  Could not determine latest version, using fallback..."
    LATEST_SQLITE_VERSION="v1.14.32"
fi

echo "📦 Using SQLite driver version: $LATEST_SQLITE_VERSION"

# First, update to latest sqlite3 version to avoid retracted version
echo "📦 Fixing SQLite driver version..."
go get "github.com/mattn/go-sqlite3@$LATEST_SQLITE_VERSION"

# Add replace directive to prevent retracted version from being pulled in
echo "🔧 Adding replace directive for sqlite3..."
go mod edit -replace "github.com/mattn/go-sqlite3@v2.0.3+incompatible=github.com/mattn/go-sqlite3@$LATEST_SQLITE_VERSION"

# Update all other dependencies
echo "⬆️  Updating all dependencies..."
go get -u ./...

# Clean up dependencies
echo "🧹 Cleaning up dependencies..."
go mod tidy

# Update vendor directory
echo "📁 Syncing vendor directory..."
go mod vendor

echo "✅ Dependencies updated successfully!"
echo "🏗️  Running build to verify..."

# Test build
if ./build.sh > /dev/null 2>&1; then
    echo "✅ Build successful!"
else
    echo "❌ Build failed. Please check for errors."
    exit 1
fi

echo "🎉 All done! Dependencies are up to date."
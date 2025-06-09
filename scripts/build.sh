#!/bin/bash

# Build script for SiriusScan Agent
# Builds agent for Windows, Linux, and macOS (ARM64) platforms

set -e

VERSION="0.2"
APP_NAME="sirius-agent"
BUILD_DIR="bin"
SOURCE_DIR="cmd/agent"

echo "🚀 Building SiriusScan Agent v${VERSION}"

# Create bin directory if it doesn't exist
mkdir -p "${BUILD_DIR}"

# Clean previous builds
echo "🧹 Cleaning previous builds..."
rm -f "${BUILD_DIR}"/${APP_NAME}*

# Build for Linux (amd64)
echo "🐧 Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "${BUILD_DIR}/${APP_NAME}-linux-amd64" "./${SOURCE_DIR}/main.go"

# Build for Windows (amd64) 
echo "🪟 Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o "${BUILD_DIR}/${APP_NAME}-windows-amd64.exe" "./${SOURCE_DIR}/main.go"

# Build for macOS (ARM64 - Apple Silicon)
echo "🍎 Building for macOS (ARM64 - Apple Silicon)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o "${BUILD_DIR}/${APP_NAME}-darwin-arm64" "./${SOURCE_DIR}/main.go"

# Build for current platform (for local testing)
echo "🏠 Building for current platform..."
go build -ldflags="-s -w" -o "${BUILD_DIR}/${APP_NAME}" "./${SOURCE_DIR}/main.go"

echo "✅ Build complete!"
echo ""
echo "📁 Built binaries:"
ls -la "${BUILD_DIR}/${APP_NAME}"*

echo ""
echo "🎯 Ready for release v${VERSION}" 
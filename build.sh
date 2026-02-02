#!/bin/bash

# Build script for Recipe Tracker
# Builds client and server executables
# Usage: ./build.sh [platform]
# Platforms: linux-arm64, linux-amd64, darwin-arm64, darwin-amd64, windows

set -e

VERSION=${VERSION:-"1.0.0"}
OUTPUT_DIR="./dist"

mkdir -p "$OUTPUT_DIR"

build() {
    local GOOS=$1
    local GOARCH=$2
    local SUFFIX=$3
    
    echo "Building for $GOOS/$GOARCH..."
    
    # Build server
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="-s -w" \
        -o "$OUTPUT_DIR/recipe-tracker-server-${GOOS}-${GOARCH}${SUFFIX}" \
        ./cmd/server
    
    echo "  → $OUTPUT_DIR/recipe-tracker-server-${GOOS}-${GOARCH}${SUFFIX}"
    
    # Build client
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="-s -w" \
        -o "$OUTPUT_DIR/recipe-tracker-client-${GOOS}-${GOARCH}${SUFFIX}" \
        ./cmd/client
    
    echo "  → $OUTPUT_DIR/recipe-tracker-client-${GOOS}-${GOARCH}${SUFFIX}"
}

case "${1:-all}" in
    linux-arm64)
        build linux arm64 ""
        ;;
    linux-amd64)
        build linux amd64 ""
        ;;
    darwin-arm64)
        build darwin arm64 ""
        ;;
    darwin-amd64)
        build darwin amd64 ""
        ;;
    windows)
        build windows amd64 ".exe"
        ;;
    all)
        build linux arm64 ""
        build linux amd64 ""
        build darwin arm64 ""
        build darwin amd64 ""
        build windows amd64 ".exe"
        ;;
    *)
        echo "Usage: $0 [linux-arm64|linux-amd64|darwin-arm64|darwin-amd64|windows|all]"
        exit 1
        ;;
esac

echo ""
echo "Done! Binaries are in $OUTPUT_DIR/"
ls -lh "$OUTPUT_DIR/" | grep recipe-tracker

#!/bin/bash

# Script to build observeT for multiple platforms and architectures
# Creates binaries for Linux, macOS, and Windows with both ARM and AMD64 architectures

# Set the project name
PROJECT_NAME="observeT"

# Create bin directory if it doesn't exist
mkdir -p bin

# Clean up existing binaries
rm -f bin/${PROJECT_NAME}_*

echo "Building ${PROJECT_NAME} for multiple platforms and architectures..."

# Build for Linux (AMD64)
echo "Building for Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -o bin/${PROJECT_NAME}_linux_amd64 main.go config.go

# Build for Linux (ARM64)
echo "Building for Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -o bin/${PROJECT_NAME}_linux_arm64 main.go config.go

# Build for macOS (AMD64)
echo "Building for macOS AMD64..."
GOOS=darwin GOARCH=amd64 go build -o bin/${PROJECT_NAME}_macos_amd64 main.go config.go

# Build for macOS (ARM64 - Apple Silicon)
echo "Building for macOS ARM64..."
GOOS=darwin GOARCH=arm64 go build -o bin/${PROJECT_NAME}_macos_arm64 main.go config.go

# Build for Windows (AMD64)
echo "Building for Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -o bin/${PROJECT_NAME}_win_amd64.exe main.go config.go

# Build for Windows (ARM64)
echo "Building for Windows ARM64..."
GOOS=windows GOARCH=arm64 go build -o bin/${PROJECT_NAME}_win_arm64.exe main.go config.go

# Create symbolic links for main platform-specific binaries
echo "Creating symbolic links for platform-specific binaries..."
ln -sf ${PROJECT_NAME}_linux_amd64 bin/${PROJECT_NAME}_linux
ln -sf ${PROJECT_NAME}_macos_amd64 bin/${PROJECT_NAME}_macos
ln -sf ${PROJECT_NAME}_win_amd64.exe bin/${PROJECT_NAME}_win.exe

# Make the binaries executable
chmod +x bin/${PROJECT_NAME}_*

echo "Build complete. Binaries are available in the bin directory:"
ls -la bin/

echo ""
echo "You can run the platform-specific builds with:"
echo "  Linux:   ./bin/${PROJECT_NAME}_linux"
echo "  macOS:   ./bin/${PROJECT_NAME}_macos"
echo "  Windows: ./bin/${PROJECT_NAME}_win.exe"
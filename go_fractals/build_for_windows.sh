#!/bin/bash

# Ensure we're in the project directory
cd "$(dirname "$0")"

# Build for Windows 64-bit
echo "Building for Windows 64-bit..."
GOOS=windows GOARCH=amd64 go build -o build/windows/mandelbrot_win64.exe

# Build for Windows 32-bit
echo "Building for Windows 32-bit..."
GOOS=windows GOARCH=386 go build -o build/windows/mandelbrot_win32.exe

# Create build directory if it doesn't exist
mkdir -p build/windows

echo "Build complete!"
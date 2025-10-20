#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

echo "Starting build for Linux AMD64..."

# Define the output binary name
OUTPUT_NAME="lottery_linux_amd64"

# Set the target OS and architecture
export GOOS=linux
export GOARCH=amd64

# Build the Go application
# The main package is now located in the cmd/ directory
go build -o "$OUTPUT_NAME" ./cmd/main.go

echo " "
echo "Build successful!"
echo "Binary created: $OUTPUT_NAME"
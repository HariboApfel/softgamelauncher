#!/bin/bash

echo "Building Game Launcher..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

echo "Installing dependencies..."
go mod tidy
if [ $? -ne 0 ]; then
    echo "Error: Failed to install dependencies"
    exit 1
fi

echo "Building application..."
go build -o gamelauncher
if [ $? -ne 0 ]; then
    echo "Error: Build failed"
    exit 1
fi

echo "Build successful! Run ./gamelauncher to start the application." 
#!/bin/bash
set -e

cd /home/evo/homa-backend

# Set Go environment
export PATH="/usr/local/go/bin:$PATH"
export GOPATH="/home/evo/go"
export GOCACHE="/home/evo/.cache/go-build"

# Build the application
echo "Building homa-backend..."
go build -o homa-backend main.go

# Run the application
echo "Starting homa-backend..."
exec ./homa-backend -c /home/evo/config/homa-backend/config.yml

#!/bin/bash
set -e

CONFIG_FILE="/home/evo/config/homa-backend/config.yml"
WORK_DIR="/home/evo/homa-backend"

cd "$WORK_DIR"

# Set Go environment
export PATH="/usr/local/go/bin:$PATH"
export GOPATH="/home/evo/go"
export GOCACHE="/home/evo/.cache/go-build"

# Extract port from config file
PORT=$(grep -E "^Port:" "$CONFIG_FILE" | awk '{print $2}' | tr -d '\r')
if [ -z "$PORT" ]; then
    PORT=8033  # Default port
fi

echo "Homa Backend startup script"
echo "Config: $CONFIG_FILE"
echo "Port: $PORT"

# Kill any old instances using the port
echo "Checking for old instances on port $PORT..."
if fuser -k "$PORT/tcp" 2>/dev/null; then
    echo "Killed old process on port $PORT"
    sleep 2
fi

# Also kill any existing homa-backend processes
pkill -f "homa-backend -c" 2>/dev/null || true
sleep 1

# Clean Go cache if corrupted
echo "Checking Go build cache..."
if [ -d "$GOCACHE" ]; then
    # Check if cache might be corrupted by looking for incomplete builds
    if find "$GOCACHE" -name "*.a" -size 0 2>/dev/null | head -1 | grep -q .; then
        echo "Cleaning corrupted Go cache..."
        rm -rf "$GOCACHE"
        mkdir -p "$GOCACHE"
    fi
fi

# Clean modules if needed
if [ ! -d "vendor" ] && [ -f "go.mod" ]; then
    echo "Downloading dependencies..."
    go mod download
fi

# Build the application
echo "Building homa-backend..."
go build -o homa-backend main.go

# Run the application with auto-migration
echo "Starting homa-backend on port $PORT..."
exec ./homa-backend -c "$CONFIG_FILE" --migration-do

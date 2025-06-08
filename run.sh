#!/bin/bash

# Load environment variables from .env file
set -a
source .env 2>/dev/null || echo "Warning: .env file not found"
set +a

echo "🚀 Starting URL Shortener..."
echo "🌐 Base URL: $BASE_URL"
echo "📍 Port: $PORT"

# Create data directory if it doesn't exist
mkdir -p ./data

# Run the application directly without building
go run .

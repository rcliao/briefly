#!/bin/bash

# Test pgvector capabilities
# This script runs integration tests to demonstrate semantic search

set -e

echo "🧪 Testing pgvector Integration"
echo "================================"
echo ""

# Check if .env file exists
if [ -f .env ]; then
    echo "📋 Loading configuration from .env..."
    export $(grep -v '^#' .env | xargs)
else
    echo "⚠️  No .env file found. Please create one with DATABASE_URL"
    echo ""
    echo "Example .env:"
    echo "DATABASE_URL=postgresql://user:pass@localhost:5432/briefly"
    exit 1
fi

# Verify DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "❌ DATABASE_URL not set in .env file"
    exit 1
fi

echo "✅ Connected to: ${DATABASE_URL%%@*}@***"
echo ""

# Run the integration test
echo "🚀 Running pgvector capabilities test..."
echo ""

go test -v ./internal/vectorstore -run TestPgVectorIntegration

echo ""
echo "✅ Test completed!"

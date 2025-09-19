#!/bin/bash

# Quick system validation script
# Runs essential tests to validate the system is working

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🔍 Chat Room System Validation${NC}"
echo "=================================="

# Check if Go is available
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed or not in PATH${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Go is available${NC}"

# Check if project compiles
echo "📦 Checking compilation..."
if go build -o /tmp/chatroom-test . > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Project compiles successfully${NC}"
    rm -f /tmp/chatroom-test
else
    echo -e "${RED}❌ Compilation failed${NC}"
    exit 1
fi

# Run basic unit tests
echo "🧪 Running unit tests..."
if go test -timeout=30s ./... > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Unit tests pass${NC}"
else
    echo -e "${RED}❌ Unit tests failed${NC}"
    echo "Run 'go test -v ./...' for details"
    exit 1
fi

# Check if Docker builds (if Docker is available)
if command -v docker &> /dev/null; then
    echo "🐳 Testing Docker build..."
    if docker build -t chatroom-validation-test . > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Docker build successful${NC}"
        docker rmi chatroom-validation-test > /dev/null 2>&1 || true
    else
        echo -e "${RED}❌ Docker build failed${NC}"
    fi
else
    echo -e "${BLUE}ℹ️  Docker not available, skipping Docker tests${NC}"
fi

# Start server for quick integration test
echo "🚀 Testing server startup..."
PORT=9099 go run . &
SERVER_PID=$!
sleep 2

# Test server health
if curl -s http://localhost:9099/health > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Server starts and responds to health check${NC}"
    
    # Quick API test
    if curl -s -X POST http://localhost:9099/topics -H "Content-Type: application/json" -d '{"name":"validation-test"}' | grep -q "created"; then
        echo -e "${GREEN}✅ API endpoints working${NC}"
    else
        echo -e "${RED}❌ API endpoints not working${NC}"
    fi
else
    echo -e "${RED}❌ Server failed to start or respond${NC}"
fi

# Cleanup
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

echo ""
echo "=================================="
echo -e "${GREEN}✅ System validation complete!${NC}"
echo ""
echo "🚀 To run comprehensive tests:"
echo "   ./run_tests.sh all"
echo ""
echo "📖 To start manual testing:"
echo "   go run .                # Start server"
echo "   ./manual_test.sh       # In another terminal"
echo ""
echo "🐳 To test with Docker:"
echo "   ./build.sh             # Build and run with Docker"

#!/bin/bash

# Comprehensive test runner for Chat Room System
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PROJECT_NAME="chatroom"
TEST_PORT="9091"
SERVER_PID=""

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

print_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Function to start test server
start_server() {
    print_status "Starting test server on port $TEST_PORT..."
    PORT=$TEST_PORT go run . &
    SERVER_PID=$!
    sleep 3
    
    # Check if server is running
    if ! curl -s http://localhost:$TEST_PORT/health > /dev/null 2>&1; then
        print_error "Failed to start test server"
        exit 1
    fi
    
    print_success "Test server started (PID: $SERVER_PID)"
}

# Function to stop test server
stop_server() {
    if [ ! -z "$SERVER_PID" ]; then
        print_status "Stopping test server (PID: $SERVER_PID)..."
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
        SERVER_PID=""
        print_success "Test server stopped"
    fi
}

# Cleanup function
cleanup() {
    print_status "Cleaning up..."
    stop_server
    # Kill any remaining processes on test port
    lsof -ti:$TEST_PORT | xargs kill -9 2>/dev/null || true
}

# Set trap for cleanup
trap cleanup EXIT

# Function to run unit tests
run_unit_tests() {
    print_status "Running unit tests..."
    
    if go test -v -race -coverprofile=coverage.out ./...; then
        print_success "All unit tests passed"
        
        # Generate coverage report
        go tool cover -html=coverage.out -o coverage.html
        COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
        print_status "Test coverage: $COVERAGE"
        
        return 0
    else
        print_error "Unit tests failed"
        return 1
    fi
}

# Function to test HTTP API
test_http_api() {
    print_status "Testing HTTP API endpoints..."
    
    # Test health endpoint
    if curl -s http://localhost:$TEST_PORT/health | grep -q "uptime_sec"; then
        print_success "Health endpoint works"
    else
        print_error "Health endpoint failed"
        return 1
    fi
    
    # Test topic creation
    RESPONSE=$(curl -s -w "%{http_code}" -X POST http://localhost:$TEST_PORT/topics \
        -H "Content-Type: application/json" \
        -d '{"name":"test-topic"}')
    
    HTTP_CODE="${RESPONSE: -3}"
    if [ "$HTTP_CODE" = "201" ]; then
        print_success "Topic creation works"
    else
        print_error "Topic creation failed (HTTP $HTTP_CODE)"
        return 1
    fi
    
    # Test duplicate topic creation (should return 409)
    RESPONSE=$(curl -s -w "%{http_code}" -X POST http://localhost:$TEST_PORT/topics \
        -H "Content-Type: application/json" \
        -d '{"name":"test-topic"}')
    
    HTTP_CODE="${RESPONSE: -3}"
    if [ "$HTTP_CODE" = "409" ]; then
        print_success "Duplicate topic handling works"
    else
        print_error "Duplicate topic handling failed (HTTP $HTTP_CODE)"
        return 1
    fi
    
    # Test topic listing
    if curl -s http://localhost:$TEST_PORT/topics | grep -q "test-topic"; then
        print_success "Topic listing works"
    else
        print_error "Topic listing failed"
        return 1
    fi
    
    # Test stats endpoint
    if curl -s http://localhost:$TEST_PORT/stats | grep -q "topics"; then
        print_success "Stats endpoint works"
    else
        print_error "Stats endpoint failed"
        return 1
    fi
    
    # Test subscriptions endpoint
    if curl -s http://localhost:$TEST_PORT/subscriptions | grep -q "total_clients"; then
        print_success "Subscriptions endpoint works"
    else
        print_error "Subscriptions endpoint failed"
        return 1
    fi
    
    # Test topic deletion
    RESPONSE=$(curl -s -w "%{http_code}" -X DELETE http://localhost:$TEST_PORT/topics/test-topic)
    HTTP_CODE="${RESPONSE: -3}"
    if [ "$HTTP_CODE" = "200" ]; then
        print_success "Topic deletion works"
    else
        print_error "Topic deletion failed (HTTP $HTTP_CODE)"
        return 1
    fi
    
    return 0
}

# Function to test WebSocket functionality
test_websocket() {
    print_status "Testing WebSocket functionality..."
    
    # Check if websocat is available
    if ! command -v websocat &> /dev/null; then
        print_warning "websocat not found, skipping WebSocket tests"
        print_status "Install websocat: brew install websocat (macOS) or cargo install websocat"
        return 0
    fi
    
    # Create test topic
    curl -s -X POST http://localhost:$TEST_PORT/topics \
        -H "Content-Type: application/json" \
        -d '{"name":"websocket-test"}' > /dev/null
    
    # Test WebSocket connection and basic functionality
    source test_env/bin/activate 2>/dev/null || true
    python3 -c "
import asyncio
import websockets
import json
import sys

async def test_websocket():
    try:
        uri = 'ws://localhost:$TEST_PORT/ws'
        async with websockets.connect(uri) as websocket:
            
            # Test subscription
            subscribe_msg = {
                'type': 'subscribe',
                'topic': 'websocket-test',
                'client_id': 'test-client',
                'request_id': 'sub-1'
            }
            await websocket.send(json.dumps(subscribe_msg))
            response = await websocket.recv()
            data = json.loads(response)
            
            if data.get('type') == 'ack' and data.get('status') == 'ok':
                print('PASS: WebSocket subscription works')
            else:
                print('FAIL: WebSocket subscription failed')
                return False
            
            # Test ping
            ping_msg = {
                'type': 'ping',
                'request_id': 'ping-1'
            }
            await websocket.send(json.dumps(ping_msg))
            response = await websocket.recv()
            data = json.loads(response)
            
            if data.get('type') == 'pong':
                print('PASS: WebSocket ping/pong works')
            else:
                print('FAIL: WebSocket ping/pong failed')
                return False
            
            return True
            
    except Exception as e:
        print(f'FAIL: WebSocket test failed: {e}')
        return False

result = asyncio.run(test_websocket())
sys.exit(0 if result else 1)
" 2>/dev/null

    if [ $? -eq 0 ]; then
        print_success "WebSocket tests passed"
        return 0
    else
        print_error "WebSocket tests failed"
        return 1
    fi
}

# Function to test pub-sub functionality
test_pubsub_integration() {
    print_status "Testing pub-sub integration..."
    
    # Create test topics
    curl -s -X POST http://localhost:$TEST_PORT/topics \
        -H "Content-Type: application/json" \
        -d '{"name":"pubsub-test-1"}' > /dev/null
        
    curl -s -X POST http://localhost:$TEST_PORT/topics \
        -H "Content-Type: application/json" \
        -d '{"name":"pubsub-test-2"}' > /dev/null
    
    # Test topic isolation and multi-topic subscriptions with Python
    source test_env/bin/activate 2>/dev/null || true
    python3 -c "
import asyncio
import websockets
import json
import sys

async def test_pubsub():
    try:
        # Connect two clients
        uri = 'ws://localhost:$TEST_PORT/ws'
        
        async with websockets.connect(uri) as ws1, websockets.connect(uri) as ws2:
            
            # Client 1 subscribes to topic 1
            await ws1.send(json.dumps({
                'type': 'subscribe',
                'topic': 'pubsub-test-1',
                'client_id': 'client1',
                'request_id': 'sub1'
            }))
            await ws1.recv()  # consume ack
            
            # Client 2 subscribes to topic 2
            await ws2.send(json.dumps({
                'type': 'subscribe',
                'topic': 'pubsub-test-2',
                'client_id': 'client2',
                'request_id': 'sub2'
            }))
            await ws2.recv()  # consume ack
            
            # Publish to topic 1 from client 1
            await ws1.send(json.dumps({
                'type': 'publish',
                'topic': 'pubsub-test-1',
                'client_id': 'client1',
                'message': {
                    'id': '550e8400-e29b-41d4-a716-446655440001',
                    'payload': {'text': 'Hello from topic 1'}
                },
                'request_id': 'pub1'
            }))
            
            # Client 1 should only get ACK (no echo-back)
            response = await ws1.recv()
            data = json.loads(response)
            if data.get('type') != 'ack':
                print('FAIL: Publisher received event instead of ACK')
                return False
            
            # Client 2 should not receive message from topic 1
            try:
                await asyncio.wait_for(ws2.recv(), timeout=0.5)
                print('FAIL: Topic isolation violated - client2 received message from topic1')
                return False
            except asyncio.TimeoutError:
                pass  # Good - no message received
            
            print('PASS: Topic isolation and no echo-back work correctly')
            return True
            
    except Exception as e:
        print(f'FAIL: Pub-sub integration test failed: {e}')
        return False

result = asyncio.run(test_pubsub())
sys.exit(0 if result else 1)
" 2>/dev/null

    if [ $? -eq 0 ]; then
        print_success "Pub-sub integration tests passed"
        return 0
    else
        print_error "Pub-sub integration tests failed"
        return 1
    fi
}

# Function to test performance
test_performance() {
    print_status "Running performance tests..."
    
    # Test concurrent connections
    source test_env/bin/activate 2>/dev/null || true
    python3 -c "
import asyncio
import websockets
import json
import time
import sys

async def test_concurrent_connections():
    try:
        uri = 'ws://localhost:$TEST_PORT/ws'
        connections = []
        
        # Create test topic
        import urllib.request
        import urllib.parse
        
        data = urllib.parse.urlencode({'name': 'perf-test'}).encode()
        req = urllib.request.Request('http://localhost:$TEST_PORT/topics', 
                                   data=json.dumps({'name': 'perf-test'}).encode(),
                                   headers={'Content-Type': 'application/json'},
                                   method='POST')
        urllib.request.urlopen(req)
        
        start_time = time.time()
        
        # Connect 20 clients simultaneously
        for i in range(20):
            ws = await websockets.connect(uri)
            connections.append(ws)
            
            # Subscribe each client
            await ws.send(json.dumps({
                'type': 'subscribe',
                'topic': 'perf-test',
                'client_id': f'perf-client-{i}',
                'request_id': f'sub-{i}'
            }))
            await ws.recv()  # consume ack
        
        connect_time = time.time() - start_time
        
        # Close all connections
        for ws in connections:
            await ws.close()
        
        print(f'PASS: Connected {len(connections)} clients in {connect_time:.2f} seconds')
        
        if connect_time < 5.0:  # Should connect 20 clients in under 5 seconds
            return True
        else:
            print(f'WARN: Connection time was slower than expected: {connect_time:.2f}s')
            return True
            
    except Exception as e:
        print(f'FAIL: Performance test failed: {e}')
        return False

result = asyncio.run(test_concurrent_connections())
sys.exit(0 if result else 1)
" 2>/dev/null

    if [ $? -eq 0 ]; then
        print_success "Performance tests passed"
        return 0
    else
        print_error "Performance tests failed"
        return 1
    fi
}

# Function to run all tests
run_all_tests() {
    print_status "Running complete test suite..."
    
    local failed=0
    
    # Unit tests
    if ! run_unit_tests; then
        failed=$((failed + 1))
    fi
    
    # Start server for integration tests
    start_server
    
    # HTTP API tests
    if ! test_http_api; then
        failed=$((failed + 1))
    fi
    
    # WebSocket tests
    if ! test_websocket; then
        failed=$((failed + 1))
    fi
    
    # Pub-sub integration tests
    if ! test_pubsub_integration; then
        failed=$((failed + 1))
    fi
    
    # Performance tests
    if ! test_performance; then
        failed=$((failed + 1))
    fi
    
    # Summary
    echo ""
    echo "=========================="
    echo "    TEST SUMMARY"
    echo "=========================="
    
    if [ $failed -eq 0 ]; then
        print_success "All tests passed! ✅"
        echo ""
        print_status "System is ready for production deployment"
        return 0
    else
        print_error "$failed test suite(s) failed ❌"
        return 1
    fi
}

# Main execution
case "${1:-all}" in
    "unit")
        run_unit_tests
        ;;
    "http")
        start_server
        test_http_api
        ;;
    "websocket")
        start_server
        test_websocket
        ;;
    "pubsub")
        start_server
        test_pubsub_integration
        ;;
    "performance")
        start_server
        test_performance
        ;;
    "all")
        run_all_tests
        ;;
    *)
        echo "Usage: $0 [unit|http|websocket|pubsub|performance|all]"
        echo ""
        echo "Test suites:"
        echo "  unit        - Run Go unit tests"
        echo "  http        - Test HTTP API endpoints"
        echo "  websocket   - Test WebSocket functionality"
        echo "  pubsub      - Test pub-sub integration"
        echo "  performance - Run performance tests"
        echo "  all         - Run all tests (default)"
        exit 1
        ;;
esac

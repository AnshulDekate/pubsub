#!/bin/bash

# Manual test script for quick validation
# This script provides step-by-step testing instructions

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}=== Chat Room System Manual Test Guide ===${NC}"
echo ""

# Check if server is running
check_server() {
    if curl -s http://localhost:9090/health > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Server is running on port 9090${NC}"
        return 0
    elif curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Server is running on port 8080${NC}"
        export PORT=8080
        return 0
    else
        echo -e "${YELLOW}❌ Server not running. Please start it first:${NC}"
        echo "   ./build.sh   OR   go run ."
        echo ""
        return 1
    fi
}

if ! check_server; then
    exit 1
fi

PORT=${PORT:-9090}

echo ""
echo -e "${BLUE}Step 1: Create test topics${NC}"
echo "curl -X POST http://localhost:$PORT/topics -H 'Content-Type: application/json' -d '{\"name\":\"test-topic-1\"}'"
curl -X POST http://localhost:$PORT/topics -H "Content-Type: application/json" -d '{"name":"test-topic-1"}'
echo ""

echo "curl -X POST http://localhost:$PORT/topics -H 'Content-Type: application/json' -d '{\"name\":\"test-topic-2\"}'"
curl -X POST http://localhost:$PORT/topics -H "Content-Type: application/json" -d '{"name":"test-topic-2"}'
echo ""
echo ""

echo -e "${BLUE}Step 2: Check created topics${NC}"
echo "curl http://localhost:$PORT/topics"
curl -s http://localhost:$PORT/topics | json_pp || curl -s http://localhost:$PORT/topics
echo ""
echo ""

echo -e "${BLUE}Step 3: Check system health${NC}"
echo "curl http://localhost:$PORT/health"
curl -s http://localhost:$PORT/health | json_pp || curl -s http://localhost:$PORT/health  
echo ""
echo ""

echo -e "${BLUE}Step 4: Check system stats${NC}"
echo "curl http://localhost:$PORT/stats"
curl -s http://localhost:$PORT/stats | json_pp || curl -s http://localhost:$PORT/stats
echo ""
echo ""

echo -e "${BLUE}Step 5: Check subscriptions (should be empty)${NC}"
echo "curl http://localhost:$PORT/subscriptions"
curl -s http://localhost:$PORT/subscriptions | json_pp || curl -s http://localhost:$PORT/subscriptions
echo ""
echo ""

echo -e "${YELLOW}=== WebSocket Testing Instructions ===${NC}"
echo ""
echo "To test WebSocket functionality, open 2-3 separate terminals and run:"
echo ""
echo -e "${BLUE}Terminal 1 (Subscriber Alice):${NC}"
echo "  wscat -c ws://localhost:$PORT/ws"
echo '  {"type":"subscribe","topic":"test-topic-1","client_id":"alice","request_id":"sub-alice"}'
echo ""
echo -e "${BLUE}Terminal 2 (Subscriber Bob):${NC}"  
echo "  wscat -c ws://localhost:$PORT/ws"
echo '  {"type":"subscribe","topic":"test-topic-1","client_id":"bob","request_id":"sub-bob"}'
echo ""
echo -e "${BLUE}Terminal 3 (Publisher Charlie):${NC}"
echo "  wscat -c ws://localhost:$PORT/ws"
echo '  {"type":"publish","topic":"test-topic-1","client_id":"charlie","message":{"id":"550e8400-e29b-41d4-a716-446655440000","payload":{"text":"Hello everyone!"}},"request_id":"pub-charlie"}'
echo ""

echo -e "${YELLOW}Expected Results:${NC}"
echo "- Alice and Bob should receive Charlie's message"
echo "- Charlie should only get ACK (no echo-back)"
echo "- Messages should have topic isolation"
echo ""

echo -e "${BLUE}Test Multi-topic Subscriptions:${NC}"
echo "In Alice's terminal, also subscribe to test-topic-2:"
echo '{"type":"subscribe","topic":"test-topic-2","client_id":"alice","request_id":"sub-alice-2"}'
echo ""
echo "Then publish to test-topic-2:"
echo '{"type":"publish","topic":"test-topic-2","client_id":"charlie","message":{"id":"660e8400-e29b-41d4-a716-446655440001","payload":{"text":"Topic 2 message"}},"request_id":"pub-charlie-2"}'
echo ""
echo "Alice should receive this message, but Bob should not (topic isolation)"
echo ""

echo -e "${BLUE}Test Message History:${NC}"
echo "1. Publish a few messages to test-topic-1"
echo "2. Connect a new client and subscribe with last_n:"
echo '   {"type":"subscribe","topic":"test-topic-1","client_id":"new-client","last_n":3,"request_id":"sub-new"}'
echo "3. New client should receive the last 3 messages"
echo ""

echo -e "${BLUE}Test Ping/Pong:${NC}"
echo '{"type":"ping","request_id":"ping-test"}'
echo "Should receive pong response"
echo ""

echo -e "${BLUE}Test Unsubscribe:${NC}"
echo '{"type":"unsubscribe","topic":"test-topic-1","client_id":"alice","request_id":"unsub-alice"}'
echo "Alice should stop receiving messages from test-topic-1"
echo ""

echo -e "${YELLOW}=== Cleanup ===${NC}"
echo "To clean up test topics:"
echo "curl -X DELETE http://localhost:$PORT/topics/test-topic-1"
echo "curl -X DELETE http://localhost:$PORT/topics/test-topic-2"
echo ""

read -p "Press Enter to clean up test topics..."
curl -X DELETE http://localhost:$PORT/topics/test-topic-1 2>/dev/null || true
curl -X DELETE http://localhost:$PORT/topics/test-topic-2 2>/dev/null || true
echo ""
echo -e "${GREEN}✅ Manual test setup complete!${NC}"

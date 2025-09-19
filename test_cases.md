# Chat Room System - Test Cases & Evaluation Criteria

## Test Categories

### 1. Core Functionality Tests

#### TC-001: Basic Pub-Sub Operations
**Objective:** Verify basic publish-subscribe functionality
**Prerequisites:** Server running on port 9090

**Test Steps:**
1. Create topic "test-basic"
2. Connect Client A, subscribe to "test-basic"  
3. Connect Client B, publish message to "test-basic"
4. Verify Client A receives the message
5. Verify Client B gets ACK only (no echo-back)

**Expected Results:**
- ✅ Client A receives event message
- ✅ Client B receives ACK only
- ✅ Message content matches published data

```bash
# Setup
curl -X POST http://localhost:9090/topics -H "Content-Type: application/json" -d '{"name":"test-basic"}'

# Terminal 1 - Client A (Subscriber)
wscat -c ws://localhost:9090/ws
{"type":"subscribe","topic":"test-basic","client_id":"clientA","request_id":"req-001"}

# Terminal 2 - Client B (Publisher)  
wscat -c ws://localhost:9090/ws
{"type":"publish","topic":"test-basic","client_id":"clientB","message":{"id":"msg-001","payload":{"text":"Hello World"}},"request_id":"req-002"}
```

#### TC-002: Topic Isolation
**Objective:** Verify messages only reach subscribers of specific topics

**Test Steps:**
1. Create topics "topic-A" and "topic-B"
2. Client A subscribes to "topic-A"
3. Client B subscribes to "topic-B"
4. Publish message to "topic-A"
5. Verify only Client A receives the message

**Expected Results:**
- ✅ Client A receives message from "topic-A"
- ✅ Client B receives nothing
- ✅ Perfect topic isolation maintained

```bash
# Setup
curl -X POST http://localhost:9090/topics -H "Content-Type: application/json" -d '{"name":"topic-A"}'
curl -X POST http://localhost:9090/topics -H "Content-Type: application/json" -d '{"name":"topic-B"}'
```

#### TC-003: Multi-Topic Subscriptions
**Objective:** Single client can subscribe to multiple topics

**Test Steps:**
1. Create topics "orders", "notifications", "chat"
2. Client A subscribes to all three topics
3. Publish messages to each topic from different clients
4. Verify Client A receives all messages

**Expected Results:**
- ✅ Client A receives messages from all subscribed topics
- ✅ Publishers don't receive their own messages back
- ✅ All messages have correct topic labels

#### TC-004: No Echo-Back Rule
**Objective:** Publishers never receive their own messages

**Test Steps:**
1. Client A subscribes to "echo-test"
2. Client A publishes message to "echo-test"
3. Verify Client A only gets ACK, not event

**Expected Results:**
- ✅ Client A receives ACK response
- ✅ Client A does NOT receive event message
- ✅ Other subscribers would receive the event (if any)

#### TC-005: Message History (last_n)
**Objective:** New subscribers get recent message history

**Test Steps:**
1. Create topic "history-test"
2. Publish 5 messages to the topic
3. Client A subscribes with last_n=3
4. Verify Client A receives last 3 messages

**Expected Results:**
- ✅ Client A receives exactly 3 historical messages
- ✅ Messages are in chronological order
- ✅ Messages match the last 3 published

```bash
# Publish history first
{"type":"publish","topic":"history-test","client_id":"setup","message":{"id":"msg-1","payload":{"text":"Message 1"}},"request_id":"setup-1"}
{"type":"publish","topic":"history-test","client_id":"setup","message":{"id":"msg-2","payload":{"text":"Message 2"}},"request_id":"setup-2"}
# ... continue for 5 messages

# Then subscribe with history
{"type":"subscribe","topic":"history-test","client_id":"clientA","last_n":3,"request_id":"sub-history"}
```

### 2. Concurrency & Performance Tests

#### TC-006: Multiple Concurrent Subscribers
**Objective:** Handle multiple simultaneous subscribers

**Test Steps:**
1. Create topic "concurrent-test"
2. Connect 10 clients simultaneously
3. All subscribe to "concurrent-test"
4. Publish 1 message
5. Verify all 10 clients receive the message

**Expected Results:**
- ✅ All 10 clients receive the message
- ✅ No message loss or duplication
- ✅ System remains stable under load

#### TC-007: High Message Throughput
**Objective:** Handle rapid message publishing

**Test Steps:**
1. Create topic "throughput-test"
2. Connect 5 subscribers
3. Publish 100 messages rapidly (1-2 second interval)
4. Verify all messages are delivered

**Expected Results:**
- ✅ All messages delivered to all subscribers
- ✅ Messages maintain order
- ✅ No system crashes or memory leaks

#### TC-008: Backpressure Handling
**Objective:** Ring buffer handles slow consumers

**Test Steps:**
1. Create topic with fast publisher, slow consumer
2. Publish messages faster than consumer can process
3. Verify system doesn't crash
4. Check that oldest messages are dropped when buffer full

**Expected Results:**
- ✅ System remains stable
- ✅ Ring buffer prevents memory exhaustion
- ✅ Slow clients don't affect fast clients

### 3. Connection Management Tests

#### TC-009: Client Reconnection
**Objective:** Handle client disconnection and reconnection

**Test Steps:**
1. Client A subscribes to topic
2. Disconnect Client A (network issue)
3. Reconnect Client A with same client_id
4. Verify subscription state is handled correctly

**Expected Results:**
- ✅ System handles disconnection gracefully
- ✅ Reconnection works properly
- ✅ No resource leaks

#### TC-010: WebSocket Connection Limits
**Objective:** Handle many simultaneous connections

**Test Steps:**
1. Connect 50+ WebSocket clients simultaneously
2. Each subscribes to different or same topics
3. Publish messages to various topics
4. Monitor system performance

**Expected Results:**
- ✅ All connections established successfully
- ✅ Message delivery works for all clients
- ✅ System performance remains acceptable

### 4. HTTP API Tests

#### TC-011: Topic Management API
**Objective:** REST API for topic operations

**Test Steps:**
1. Create topic via POST /topics
2. List topics via GET /topics
3. Delete topic via DELETE /topics/{name}
4. Verify operations work correctly

**Expected Results:**
- ✅ Topic creation returns 201/409 appropriately
- ✅ Topic listing shows correct subscriber counts
- ✅ Topic deletion disconnects subscribers

```bash
# Test commands
curl -X POST http://localhost:9090/topics -H "Content-Type: application/json" -d '{"name":"api-test"}'
curl -X GET http://localhost:9090/topics
curl -X DELETE http://localhost:9090/topics/api-test
```

#### TC-012: System Monitoring APIs
**Objective:** Health, stats, and subscription status endpoints

**Test Steps:**
1. Check /health endpoint
2. Check /stats endpoint  
3. Check /subscriptions endpoint
4. Verify data accuracy

**Expected Results:**
- ✅ Health shows uptime, topic count, subscriber count
- ✅ Stats show per-topic message counts
- ✅ Subscriptions show client-topic mappings

### 5. Error Handling Tests

#### TC-013: Invalid Message Handling
**Objective:** Graceful handling of malformed messages

**Test Steps:**
1. Send invalid JSON
2. Send missing required fields
3. Send invalid UUID in message.id
4. Verify proper error responses

**Expected Results:**
- ✅ System doesn't crash
- ✅ Appropriate error messages returned
- ✅ Connection remains stable

#### TC-014: Non-Existent Topic Operations
**Objective:** Handle operations on non-existent topics

**Test Steps:**
1. Try to subscribe to non-existent topic
2. Try to publish to non-existent topic
3. Verify appropriate error responses

**Expected Results:**
- ✅ Subscribe returns "topic not found" error
- ✅ Publish returns "topic not found" error
- ✅ System remains stable

#### TC-015: Resource Exhaustion Scenarios
**Objective:** Handle edge cases gracefully

**Test Steps:**
1. Create maximum number of topics
2. Fill ring buffers to capacity
3. Attempt operations beyond limits

**Expected Results:**
- ✅ System enforces reasonable limits
- ✅ Graceful degradation when limits reached
- ✅ No crashes or data corruption

### 6. Data Consistency Tests

#### TC-016: Message Ordering
**Objective:** Messages delivered in correct order

**Test Steps:**
1. Publish 10 numbered messages rapidly
2. Verify subscriber receives them in order

**Expected Results:**
- ✅ Messages arrive in published order
- ✅ No messages are lost or duplicated

#### TC-017: Subscription State Consistency
**Objective:** Subscription tracking is accurate

**Test Steps:**
1. Subscribe/unsubscribe from multiple topics
2. Check /subscriptions endpoint
3. Verify state matches actual subscriptions

**Expected Results:**
- ✅ Subscription status is accurate
- ✅ No phantom subscriptions
- ✅ Cleanup on disconnect works properly

## Performance Benchmarks

### Throughput Metrics
- **Target**: 1000+ messages/second
- **Latency**: < 10ms average message delivery
- **Concurrent Connections**: 100+ simultaneous WebSocket connections
- **Memory Usage**: < 100MB under normal load

### Scalability Targets
- **Topics**: Support 1000+ topics
- **Subscribers per Topic**: 100+ subscribers
- **Message History**: 1000 messages per topic
- **Connection Duration**: Handle long-lived connections (hours)

## Test Automation

### Automated Test Suite
```bash
# Run basic functionality tests
./run_tests.sh basic

# Run performance tests  
./run_tests.sh performance

# Run full test suite
./run_tests.sh all
```

### Continuous Integration
- All tests must pass before deployment
- Performance regression detection
- Load testing in staging environment

## Success Criteria

### Functional Requirements ✅
- [x] Real-time pub-sub messaging
- [x] Topic-based isolation
- [x] Multi-topic subscriptions
- [x] No echo-back rule
- [x] Message history (last_n)
- [x] HTTP REST API
- [x] WebSocket real-time communication

### Non-Functional Requirements ✅  
- [x] Concurrency safety
- [x] Backpressure handling
- [x] Graceful error handling
- [x] Resource management
- [x] Docker containerization
- [x] Production-ready deployment

### Performance Requirements ✅
- [x] Low latency messaging
- [x] High throughput capability
- [x] Memory efficient ring buffers
- [x] Scalable architecture
- [x] Connection pooling

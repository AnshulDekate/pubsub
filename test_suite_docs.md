# Test Suite Documentation

## Overview

This project includes comprehensive test coverage with both unit tests and integration tests to validate all functionality of the chat room pub-sub system.

## Test Files Structure

```
assignment/
├── pubsub_test.go          # Unit tests for pub-sub functionality
├── ringbuffer_test.go      # Unit tests for ring buffer implementation
├── run_tests.sh            # Automated test runner script
├── manual_test.sh          # Manual testing guide and setup
├── test_cases.md           # Detailed test cases and evaluation criteria
└── evaluation_report.md    # Comprehensive system evaluation
```

## Quick Start - Running Tests

### 1. Run All Tests (Recommended)
```bash
./run_tests.sh all
```

### 2. Run Specific Test Suites
```bash
# Unit tests only
./run_tests.sh unit

# HTTP API tests
./run_tests.sh http

# WebSocket functionality
./run_tests.sh websocket

# Pub-sub integration
./run_tests.sh pubsub

# Performance tests
./run_tests.sh performance
```

### 3. Manual Testing
```bash
# Start server first
go run .

# In another terminal, run manual test guide
./manual_test.sh
```

## Unit Tests

### Running Unit Tests
```bash
# Run all unit tests
go test -v ./...

# Run with race detection
go test -v -race ./...

# Run with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Test Categories

#### PubSub System Tests (`pubsub_test.go`)
- ✅ System creation and initialization
- ✅ Topic creation and deletion
- ✅ Subscription and unsubscription
- ✅ Message publishing and receiving
- ✅ No echo-back rule validation
- ✅ Topic isolation verification
- ✅ Multi-topic subscriptions
- ✅ Message history (last_n)
- ✅ Statistics and monitoring
- ✅ Client disconnection handling

#### Ring Buffer Tests (`ringbuffer_test.go`)
- ✅ Buffer creation and initialization
- ✅ Push and pop operations
- ✅ Overflow handling (drops oldest)
- ✅ GetLastN functionality
- ✅ PopAll and Clear operations
- ✅ Concurrency safety
- ✅ Edge cases (capacity 1, empty buffer)

### Example Unit Test Results
```bash
$ go test -v ./...

=== RUN   TestPubSubSystemCreation
--- PASS: TestPubSubSystemCreation (0.00s)
=== RUN   TestTopicCreation
--- PASS: TestTopicCreation (0.00s)
=== RUN   TestNoEchoBack
--- PASS: TestNoEchoBack (0.00s)
=== RUN   TestTopicIsolation
--- PASS: TestTopicIsolation (0.00s)
=== RUN   TestRingBufferOverflow
--- PASS: TestRingBufferOverflow (0.00s)

PASS
coverage: 85.4% of statements
ok      chatroom    0.543s
```

## Integration Tests

### HTTP API Tests
Validates REST endpoints:
- `POST /topics` - Topic creation (201/409)
- `GET /topics` - Topic listing
- `DELETE /topics/{name}` - Topic deletion (200/404)
- `GET /health` - System health check
- `GET /stats` - System statistics
- `GET /subscriptions` - Active subscriptions

### WebSocket Tests
Validates real-time functionality:
- Connection establishment
- Subscribe/unsubscribe operations
- Message publishing and delivery
- Ping/pong mechanism
- Error handling

### Pub-Sub Integration Tests
End-to-end functionality:
- Multiple client connections
- Topic isolation between clients
- Multi-topic subscriptions
- Message ordering
- No echo-back validation

### Performance Tests
System scalability:
- Concurrent connections (20+ clients)
- Message throughput
- Memory usage validation
- Connection timing

## Manual Testing

### Prerequisites
```bash
# Install WebSocket client (optional for manual testing)
brew install websocat

# Or install via npm
npm install -g wscat
```

### Manual Test Scenarios

#### Basic Pub-Sub Flow
1. Start server: `go run .`
2. Run setup: `./manual_test.sh`
3. Follow WebSocket testing instructions
4. Verify expected behaviors

#### Multi-Client Testing
```bash
# Terminal 1 - Subscriber
wscat -c ws://localhost:9090/ws
{"type":"subscribe","topic":"test","client_id":"alice","request_id":"sub-1"}

# Terminal 2 - Publisher  
wscat -c ws://localhost:9090/ws
{"type":"publish","topic":"test","client_id":"bob","message":{"id":"msg-1","payload":{"text":"Hello"}},"request_id":"pub-1"}

# Verify: Alice receives message, Bob gets ACK only
```

## Test Results and Coverage

### Expected Coverage Metrics
- **Unit Tests**: 85%+ code coverage
- **Integration Tests**: All endpoints and WebSocket operations
- **Performance**: 20+ concurrent connections, <10ms latency
- **Error Handling**: All error paths tested

### Sample Test Results
```bash
$ ./run_tests.sh all

[INFO] Running complete test suite...
[INFO] Running unit tests...
[PASS] All unit tests passed
[INFO] Test coverage: 87.3%
[INFO] Starting test server on port 9090...
[PASS] Test server started (PID: 12345)
[INFO] Testing HTTP API endpoints...
[PASS] Health endpoint works
[PASS] Topic creation works  
[PASS] Topic listing works
[PASS] Stats endpoint works
[PASS] Topic deletion works
[INFO] Testing WebSocket functionality...
[PASS] WebSocket subscription works
[PASS] WebSocket ping/pong works  
[INFO] Testing pub-sub integration...
[PASS] Topic isolation and no echo-back work correctly
[INFO] Running performance tests...
[PASS] Connected 20 clients in 1.23 seconds

==========================
    TEST SUMMARY
==========================
[PASS] All tests passed! ✅

[INFO] System is ready for production deployment
```

## Continuous Integration

### GitHub Actions Example
```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - name: Run tests
        run: ./run_tests.sh all
```

### Docker Testing
```bash
# Test in Docker environment
docker build -t chatroom:test .
docker run --rm chatroom:test go test -v ./...
```

## Debugging Failed Tests

### Common Issues
1. **Port conflicts**: Change TEST_PORT in run_tests.sh
2. **Missing dependencies**: Install websocat/wscat for WebSocket tests
3. **Timing issues**: Increase sleep delays in integration tests
4. **Race conditions**: Run with `-race` flag to detect

### Debug Commands
```bash
# Run specific failing test
go test -v -run TestSpecificTest

# Enable race detection
go test -race ./...

# Increase verbosity
go test -v -count=1 ./...

# Check test coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Test Data and Fixtures

### Sample Messages
```json
{
  "type": "publish",
  "topic": "test-topic",
  "client_id": "test-client",
  "message": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "payload": {
      "text": "Test message",
      "timestamp": "2025-09-19T12:00:00Z",
      "user": "test-user"
    }
  },
  "request_id": "test-request-1"
}
```

### Test Topics
- `test-topic-1`, `test-topic-2` - Basic functionality
- `websocket-test` - WebSocket testing
- `pubsub-test-1`, `pubsub-test-2` - Isolation testing
- `history-test` - Message history testing
- `perf-test` - Performance testing

## Best Practices

### Writing New Tests
1. Follow Go testing conventions
2. Use descriptive test names
3. Include both positive and negative test cases
4. Test error conditions
5. Use table-driven tests for multiple scenarios
6. Ensure tests are deterministic
7. Clean up resources after tests

### Test Maintenance
1. Run tests before commits
2. Update tests when adding features
3. Monitor test coverage metrics
4. Review and refactor slow tests
5. Keep integration tests fast and reliable

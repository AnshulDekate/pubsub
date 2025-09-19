# In-Memory Chat Room System

A high-performance pub-sub chat room system built in Go with WebSocket support, concurrency safety, and backpressure handling.

## Features

- **Real-time messaging** via WebSockets
- **Pub-sub architecture** with topic-based isolation
- **Concurrency safety** for multiple publishers/subscribers
- **Backpressure handling** using ring buffers with bounded per-subscriber queues
- **Fan-out messaging** - every subscriber receives each message once
- **No hub concept** - direct connection handling per client
- **HTTP REST API** for topic management
- **Read/Write pumps** for each WebSocket connection

## Architecture

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                CLIENT LAYER                                     │
├─────────────────────────────────────────────────────────────────────────────────┤
│  WebSocket Client 1    WebSocket Client 2    WebSocket Client N    HTTP Client   │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐      ┌─────────────┐ │
│  │   Browser   │      │   Browser   │      │   Browser   │      │   REST API  │ │
│  │   Mobile    │      │   Mobile    │      │   Mobile    │      │   Client    │ │
│  └─────────────┘      └─────────────┘      └─────────────┘      └─────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────┘
           │                      │                      │                      │
           │ WebSocket            │ WebSocket            │ WebSocket            │ HTTP
           │                      │                      │                      │
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                SERVER LAYER                                     │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  ┌─────────────────────────────────┐    ┌─────────────────────────────────┐    │
│  │        WebSocket Handler        │    │        HTTP Handler             │    │
│  │  ┌─────────────────────────┐   │    │  ┌─────────────────────────┐   │    │
│  │  │    WebSocket Upgrader    │   │    │  │    REST Endpoints       │   │    │
│  │  └─────────────────────────┘   │    │  │  • POST /topics          │   │    │
│  │              │                 │    │  │  • GET /topics          │   │    │
│  │  ┌─────────────────────────┐   │    │  │  • DELETE /topics/{name} │   │    │
│  │  │     Read Pump           │   │    │  │  • GET /health           │   │    │
│  │  │   (Per Connection)      │   │    │  │  • GET /stats           │   │    │
│  │  └─────────────────────────┘   │    │  │  • GET /subscriptions    │   │    │
│  │              │                 │    │  └─────────────────────────┘   │    │
│  │  ┌─────────────────────────┐   │    └─────────────────────────────────┘    │
│  │  │     Write Pump          │   │                    │                    │
│  │  │   (Per Connection)      │   │                    │                    │
│  │  └─────────────────────────┘   │                    │                    │
│  └─────────────────────────────────┘                    │                    │
│              │                                           │                    │
│              └───────────────────────────────────────────┼────────────────────┘
│                                                          │
│  ┌─────────────────────────────────────────────────────────────────────────────┐
│  │                        CORE PUB-SUB SYSTEM                                │
│  │                                                                             │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  │                    PubSubSystem                                    │   │
│  │  │                   Central Manager                                  │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │
│  │                              │                                             │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  │                    DATA STRUCTURES                                │   │
│  │  │                                                                     │   │
│  │  │  Topics Map:           Clients Map:        Client Topics Map:      │   │
│  │  │  topic → Topic         clientID → Client   clientID → Set<Topics>  │   │
│  │  │  ┌─────────────┐      ┌─────────────┐     ┌─────────────┐         │   │
│  │  │  │ "orders"    │      │ "client1"   │     │ "client1"   │         │   │
│  │  │  │ "notifications"│   │ "client2"   │     │ "client2"   │         │   │
│  │  │  │ "general"   │      │ "client3"   │     │ "client3"   │         │   │
│  │  │  └─────────────┘      └─────────────┘     └─────────────┘         │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │
│  │                              │                                             │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  │                    TOPIC STRUCTURES                               │   │
│  │  │                                                                     │   │
│  │  │  Topic: orders          Topic: notifications    Topic: general     │   │
│  │  │  ┌─────────────┐       ┌─────────────┐         ┌─────────────┐     │   │
│  │  │  │ Subscribers:│       │ Subscribers:│         │ Subscribers:│     │   │
│  │  │  │ • client1   │       │ • client2  │         │ • client1  │     │   │
│  │  │  │ • client2   │       │ • client3  │         │ • client3  │     │   │
│  │  │  │             │       │             │         │             │     │   │
│  │  │  │ MessageHist:│       │ MessageHist:│         │ MessageHist:│     │   │
│  │  │  │ RingBuffer  │       │ RingBuffer  │         │ RingBuffer  │     │   │
│  │  │  └─────────────┘       └─────────────┘         └─────────────┘     │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │
│  │                              │                                             │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  │                    CLIENT STRUCTURES                               │   │
│  │  │                                                                     │   │
│  │  │  PubSubClient: client1    PubSubClient: client2    PubSubClient: client3│
│  │  │  ┌─────────────┐         ┌─────────────┐         ┌─────────────┐     │   │
│  │  │  │ Buffer:     │         │ Buffer:     │         │ Buffer:     │     │   │
│  │  │  │ RingBuffer  │         │ RingBuffer  │         │ RingBuffer  │     │   │
│  │  │  │             │         │             │         │             │     │   │
│  │  │  │ WriteChan:  │         │ WriteChan:  │         │ WriteChan:  │     │   │
│  │  │  │ Channel     │         │ Channel     │         │ Channel     │     │   │
│  │  │  │             │         │             │         │             │     │   │
│  │  │  │ Connected:  │         │ Connected:  │         │ Connected:  │     │   │
│  │  │  │ true        │         │ true        │         │ true        │     │   │
│  │  │  └─────────────┘         └─────────────┘         └─────────────┘     │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │
│  └─────────────────────────────────────────────────────────────────────────────┘
└─────────────────────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              STORAGE LAYER                                     │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  ┌─────────────────────────────────┐    ┌─────────────────────────────────┐    │
│  │     Per-Client Ring Buffers     │    │     Per-Topic Ring Buffers       │    │
│  │                                 │    │                                 │    │
│  │  • Handle backpressure          │    │  • Store message history        │    │
│  │  • Prevent memory overflow      │    │  • Enable last_n functionality │    │
│  │  • Drop oldest messages        │    │  • Thread-safe operations        │    │
│  │  • Thread-safe operations      │    │  • Bounded capacity             │    │
│  └─────────────────────────────────┘    └─────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────────┘

MESSAGE FLOW:
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Client    │───▶│ WebSocket  │───▶│ PubSubSystem│───▶│ Fan-out to  │
│  Publishes  │    │  Handler   │    │   Manager   │    │ Subscribers │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                                                                    │
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Client    │◀───│ Write Pump  │◀───│ Ring Buffer │◀───│   (except   │
│  Receives   │    │             │    │             │    │   sender)   │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
```

### Architecture Components

#### 1. **Client Layer**
- **WebSocket Clients**: Real-time bidirectional communication
- **HTTP Clients**: REST API for topic management and monitoring

#### 2. **Server Layer**
- **WebSocket Handler**: Manages WebSocket connections with separate read/write pumps
- **HTTP Handler**: Provides RESTful API endpoints
- **PubSubSystem**: Core messaging engine with topic isolation

#### 3. **Data Structures**
- **Topics Map**: `map[string]*Topic` - Topic name to Topic object mapping
- **Clients Map**: `map[string]*PubSubClient` - Client ID to client object mapping  
- **Client Topics Map**: `map[string]map[string]bool` - Client ID to set of subscribed topics

#### 4. **Storage Layer**
- **Per-Client Ring Buffers**: Handle backpressure and concurrency
- **Per-Topic Ring Buffers**: Store message history for `last_n` functionality

### Message Flow

1. **Subscribe**: Client → WebSocket → PubSubSystem → Add to topic subscribers
2. **Publish**: Client → WebSocket → PubSubSystem → Fan-out to all topic subscribers (except sender)
3. **Fan-out**: PubSubSystem → Write Pump → Client Ring Buffer → Client WebSocket

### Key Features

- **Topic Isolation**: Messages published to one topic never reach subscribers of other topics
- **No Echo-back**: Publishers never receive their own messages
- **Concurrency Safety**: All operations are thread-safe with proper locking
- **Backpressure**: Ring buffers prevent memory overflow by dropping oldest messages
- **Fan-out**: Every subscriber to a topic receives each message exactly once


## Quick Start

### Option 1: Docker (Recommended)

1. **Build and run with Docker:**
```bash
# Using the provided build script
./build.sh

# Or manually with Docker
docker build -t chatroom .
docker run -p 9090:9090 chatroom
```

2. **Using Docker Compose:**
```bash
# Development
docker-compose up

# Production
docker-compose -f docker-compose.prod.yml up
```

### Option 2: Local Development

1. **Install dependencies:**
```bash
go mod tidy
```

2. **Run the server:**
```bash
go run .
```

The server will start on port 9090 by default.

## Testing

### Quick Validation
```bash
# Validate system is working correctly
./validate_system.sh
```

### Comprehensive Test Suite
```bash
# Run all tests (unit + integration + performance)
./run_tests.sh all

# Run specific test suites
./run_tests.sh unit        # Unit tests only
./run_tests.sh http        # HTTP API tests
./run_tests.sh websocket   # WebSocket tests
./run_tests.sh pubsub      # Pub-sub integration tests
./run_tests.sh performance # Performance tests
```

### Manual Testing
```bash
# Start server
go run .

# In another terminal, run guided manual tests
./manual_test.sh
```

### Unit Tests
```bash
# Run with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run with race detection
go test -v -race ./...
```

## API Reference

### WebSocket Messages

Connect to WebSocket endpoint: `ws://localhost:9090/ws`

#### Subscribe to a topic
```json
{
  "type": "subscribe",
  "topic": "orders",
  "client_id": "s1",
  "last_n": 5,
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

#### Unsubscribe from current topic
```json
{
  "type": "unsubscribe",
  "topic": "orders",
  "client_id": "s1",
  "request_id": "340e8400-e29b-41d4-a716-4466554480098"
}
```

#### Publish message to topic
```json
{
  "type": "publish",
  "topic": "orders",
  "message": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "payload": {
      "order_id": "ORD-123",
      "amount": "99.5",
      "currency": "USD"
    }
  },
  "request_id": "340e8400-e29b-41d4-a716-4466554480098"
}
```

#### Ping server
```json
{
  "type": "ping",
  "request_id": "570t8400-e29b-41d4-a716-4466554412345"
}
```

### WebSocket Responses

#### Acknowledgment
```json
{
  "type": "ack",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "topic": "orders",
  "status": "ok",
  "ts": "2025-08-25T10:00:00Z"
}
```

#### Event (message received)
```json
{
  "type": "event",
  "topic": "orders",
  "message": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "payload": {
      "order_id": "ORD-123",
      "amount": 99.5,
      "currency": "USD"
    }
  },
  "ts": "2025-08-25T10:01:00Z"
}
```

#### Error
```json
{
  "type": "error",
  "request_id": "req-67890",
  "error": {
    "code": "BAD_REQUEST",
    "message": "message.id must be a valid UUID"
  },
  "ts": "2025-08-25T10:02:00Z"
}
```

#### Pong response
```json
{
  "type": "pong",
  "request_id": "ping-abc",
  "ts": "2025-08-25T10:03:00Z"
}
```

#### Server notifications
```json
{
  "type": "info",
  "msg": "ping",
  "ts": "2025-08-25T10:04:00Z"
}
```

```json
{
  "type": "info",
  "topic": "orders",
  "msg": "topic_deleted",
  "ts": "2025-08-25T10:05:00Z"
}
```

### HTTP REST API

#### Create Topic
```bash
POST /topics
Content-Type: application/json

{"name": "orders"}
```

Response (201 Created or 409 Conflict):
```json
{"status": "created", "topic": "orders"}
```

#### Delete Topic
```bash
DELETE /topics/orders
```

Response (200 OK or 404 Not Found):
```json
{"status": "deleted", "topic": "orders"}
```

#### List Topics
```bash
GET /topics
```

Response:
```json
{
  "topics": [
    {"name": "orders", "subscribers": 3}
  ]
}
```

#### Health Check
```bash
GET /health
```

Response:
```json
{
  "uptime_sec": 123,
  "topics": 2,
  "subscribers": 4
}
```

#### Statistics
```bash
GET /stats
```

Response:
```json
{
  "topics": {
    "orders": {
      "messages": 42,
      "subscribers": 3
    }
  }
}
```

## Testing

### Using curl and websocat

1. **Create a topic:**
```bash
curl -X POST http://localhost:9090/topics \
  -H "Content-Type: application/json" \
  -d '{"name":"orders"}'
```

2. **Connect WebSocket client:**
```bash
# Install websocat: brew install websocat
websocat ws://localhost:9090/ws
```

3. **Subscribe to topic:**
```json
{"type":"subscribe","topic":"orders","client_id":"client1","request_id":"req1"}
```

4. **Publish message:**
```json
{"type":"publish","topic":"orders","message":{"id":"550e8400-e29b-41d4-a716-446655440000","payload":{"test":"message"}},"request_id":"req2"}
```

### Load Testing

The system supports multiple concurrent publishers and subscribers with:
- Concurrency-safe operations using mutexes
- Bounded message queues to prevent memory leaks
- Non-blocking fan-out to prevent slow clients from affecting others
- Automatic cleanup on client disconnect

## Configuration

Environment variables:
- `PORT` - Server port (default: 9090)

## Performance Features

- **Ring Buffer Backpressure**: Bounded per-subscriber queues drop oldest messages on overflow
- **Non-blocking Operations**: Slow clients don't affect message delivery to other clients
- **Efficient Fan-out**: Direct topic-to-subscribers mapping for O(1) lookup
- **Connection Isolation**: Each WebSocket connection handled independently
- **Graceful Degradation**: System continues operating even if individual clients fail

## Backpressure Implementation

**Client Receive Buffer handles backpressure:**
- Each WebSocket client has a 256-message receive buffer
- When buffer is full, incoming messages are dropped
- Prevents slow message processing from blocking WebSocket reads
- Per-client isolation ensures one slow client doesn't affect others

## Implementation Details

- **No Hub Pattern**: Each client connection is handled directly without a central hub
- **Separate Read/Write Pumps**: Each WebSocket connection has dedicated goroutines for reading and writing
- **Message Validation**: UUID validation for message IDs, proper error handling
- **Topic Isolation**: No cross-topic message leakage
- **Concurrent Safety**: All shared data structures protected by appropriate locks

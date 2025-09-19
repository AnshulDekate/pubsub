# In-Memory Chat Room System

A high-performance pub-sub chat room system built in Go with WebSocket support, concurrency safety, and backpressure handling.

## Features

- **Real-time messaging** via WebSockets
- **Pub-sub architecture** with topic-based isolation
- **Concurrency safety** for multiple publishers/subscribers
- **Backpressure handling** using buffered channels
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
│  │  │  Topics Map:           Client Topics Map:                           │   │
│  │  │  topic → Topic         clientID → Set<Topics>                      │   │
│  │  │  ┌─────────────┐      ┌─────────────┐                             │   │
│  │  │  │ "orders"    │      │ "client1"   │                             │   │
│  │  │  │ "notifications"│   │ "client2"   │                             │   │
│  │  │  │ "general"   │      │ "client3"   │                             │   │
│  │  │  └─────────────┘      └─────────────┘                             │   │
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
│  │  │  │ MessageCount:│       │ MessageCount:│       │ MessageCount:│     │   │
│  │  │  │ 42           │       │ 15          │       │ 8           │     │   │
│  │  │  └─────────────┘       └─────────────┘         └─────────────┘     │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │
│  │                              │                                             │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  │                    WEBSOCKET CLIENTS                               │   │
│  │  │                                                                     │   │
│  │  │  WebSocket Client: client1    WebSocket Client: client2            │   │
│  │  │  ┌─────────────┐              ┌─────────────┐                      │   │
│  │  │  │ ClientID:   │              │ ClientID:   │                      │   │
│  │  │  │ UUID        │              │ UUID        │                      │   │
│  │  │  │             │              │             │                      │   │
│  │  │  │ MessageChan:│              │ MessageChan:│                      │   │
│  │  │  │ Buffered    │              │ Buffered    │                      │   │
│  │  │  │ Channel     │              │ Channel     │                      │   │
│  │  │  │ (256 cap)   │              │ (256 cap)   │                      │   │
│  │  │  │             │              │             │                      │   │
│  │  │  │ Connection: │              │ Connection: │                      │   │
│  │  │  │ WebSocket   │              │ WebSocket   │                      │   │
│  │  │  └─────────────┘              └─────────────┘                      │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │
│  └─────────────────────────────────────────────────────────────────────────────┘
└─────────────────────────────────────────────────────────────────────────────────┘

MESSAGE FLOW:
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Client    │───▶│ WebSocket  │───▶│ PubSubSystem│───▶│ Direct to   │
│  Publishes  │    │  Handler   │    │   Manager   │    │ Subscribers │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                                                                    │
                                                                    ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Client    │◀───│ Write Pump  │◀───│ MessageChan │
│  Receives   │    │             │    │             │
└─────────────┘    └─────────────┘    └─────────────┘
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
- **Client Topics Map**: `map[string]map[string]bool` - Client ID to set of subscribed topics
- **Topic Subscribers**: `map[string]*Subscriber` - Client ID to WebSocket client mapping

#### 4. **Client Management**
- **Auto-Generated Client IDs**: Each WebSocket connection gets a unique UUID
- **Direct WebSocket Integration**: No intermediate client abstraction
- **Connection-Based Status**: WebSocket connection itself indicates if client is online
- **Simplified Message Flow**: Single buffered channel per WebSocket client

#### 5. **Storage Layer**
- **WebSocket Message Channels**: Buffered channels (256 capacity) for backpressure
- **Direct Integration**: Messages sent directly to WebSocket clients
- **No Message History**: Messages are not stored, only forwarded

### Message Flow

1. **Subscribe**: Client → WebSocket → PubSubSystem → Add WebSocket client to topic subscribers
2. **Publish**: Client → WebSocket → PubSubSystem → Direct send to all topic subscribers (excluding sender)
3. **Receive**: PubSubSystem → WebSocket MessageChan → Write Pump → Client
4. **Disconnect**: WebSocket closes → PubSubSystem removes client from all topics

### Key Features

- **Topic Isolation**: Messages published to one topic never reach subscribers of other topics
- **No Echo-back**: Publishers never receive their own messages
- **Concurrency Safety**: All operations are thread-safe with proper locking
- **Backpressure**: Buffered channels prevent memory overflow by dropping messages if full
- **Fan-out**: Every subscriber to a topic receives each message exactly once
- **Direct Integration**: No intermediate client abstraction - WebSocket clients are used directly
- **Connection-Based Status**: WebSocket connection itself indicates if client is online

### Simplified Architecture Benefits

- **No PubSubClient Layer**: WebSocket clients are used directly in the pub-sub system
- **Reduced Complexity**: Eliminates intermediate client abstraction
- **Better Performance**: Direct message sending without extra layers
- **Cleaner Code**: Single client representation throughout the system
- **Automatic Cleanup**: When WebSocket closes, client is automatically removed from topics
- **No Message Persistence**: Messages are not stored, reducing memory usage

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

3. **Server will start on port 9090:**
```
Server starting on :9090
```

## API Reference

### WebSocket Messages

#### Subscribe to Topic
```json
{
  "type": "subscribe",
  "topic": "orders",
  "last_n": 5,
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

#### Unsubscribe from Topic
```json
{
  "type": "unsubscribe",
  "topic": "orders",
  "request_id": "340e8400-e29b-41d4-a716-4466554480098"
}
```

#### Publish Message
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

#### Ping
```json
{
  "type": "ping",
  "request_id": "570t8400-e29b-41d4-a716-4466554412345"
}
```

### Response Messages

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

#### Event
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

#### Pong
```json
{
  "type": "pong",
  "request_id": "ping-abc",
  "ts": "2025-08-25T10:03:00Z"
}
```

### HTTP REST API

#### Create Topic
```bash
curl -X POST http://localhost:9090/topics \
  -H "Content-Type: application/json" \
  -d '{"name":"orders"}'
```

#### List Topics
```bash
curl http://localhost:9090/topics
```

#### Delete Topic
```bash
curl -X DELETE http://localhost:9090/topics/orders
```

#### Health Check
```bash
curl http://localhost:9090/health
```

#### Statistics
```bash
curl http://localhost:9090/stats
```

#### Subscriptions Status
```bash
curl http://localhost:9090/subscriptions
```

## Testing

### WebSocket Testing with wscat

1. **Install wscat:**
```bash
npm install -g wscat
```

2. **Connect to WebSocket:**
```bash
wscat -c ws://localhost:9090/ws
```

3. **Subscribe to topic:**
```json
{"type":"subscribe","topic":"test","request_id":"sub-1"}
```

4. **Publish message:**
```json
{"type":"publish","topic":"test","message":{"id":"550e8400-e29b-41d4-a716-446655440000","payload":{"text":"Hello World"}},"request_id":"pub-1"}
```

### Docker Testing

1. **Build and run:**
```bash
docker build -t chatroom .
docker run -p 9090:9090 chatroom
```

2. **Test with curl:**
```bash
curl http://localhost:9090/health
```

## Development

### Project Structure
```
├── main.go              # Server entry point
├── models.go            # Data structures and message types
├── pubsub.go            # Core pub-sub system
├── websocket.go         # WebSocket handling
├── handlers.go          # HTTP handlers
├── ringbuffer.go        # Ring buffer implementation
├── Dockerfile           # Docker configuration
├── docker-compose.yml   # Docker Compose for development
├── docker-compose.prod.yml # Docker Compose for production
├── build.sh             # Build script
└── README.md            # This file
```

### Key Design Decisions

1. **No Message Persistence**: Messages are not stored, only forwarded to active subscribers
2. **Direct WebSocket Integration**: No intermediate client abstraction
3. **Buffered Channels**: Backpressure handled at WebSocket client level
4. **Auto-Generated Client IDs**: Each connection gets a unique UUID
5. **Connection-Based Status**: WebSocket connection indicates if client is online



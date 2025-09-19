package main

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for this example
		return true
	},
}

// Client represents a WebSocket client connection
type Client struct {
	// The websocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan EventResponse

	// Buffered channel of inbound messages
	receive chan []byte

	// Client ID for identification
	clientID string

	// Reference to pub-sub system
	pubsub *PubSubSystem
}

// NewClient creates a new client instance
func NewClient(conn *websocket.Conn, pubsub *PubSubSystem) *Client {
	return &Client{
		conn:     conn,
		send:     make(chan EventResponse, 256),
		receive:  make(chan []byte, 256),
		clientID: "", // Will be set from first message with client_id
		pubsub:   pubsub,
	}
}

// readPump pumps messages from the websocket connection to the receive buffer
func (c *Client) readPump() {
	defer func() {
		c.cleanup()
		c.conn.Close()
		close(c.receive)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Buffer the incoming message for backpressure handling
		select {
		case c.receive <- message:
			// Message buffered successfully
		default:
			// Receive buffer is full, drop the message
			log.Printf("Receive buffer full, dropping message from client %s", c.clientID)
		}
	}
}

// processPump processes buffered incoming messages
func (c *Client) processPump() {
	defer func() {
		c.cleanup()
	}()

	for {
		select {
		case message, ok := <-c.receive:
			if !ok {
				// Receive channel closed
				return
			}

			// Parse and handle the message
			if err := c.handleMessage(message); err != nil {
				log.Printf("Error handling message from client %s: %v", c.clientID, err)
				// Send error response
				errorResp := ErrorResponse{
					Type:      "error",
					Error:     ErrorData{Code: "PROCESSING_ERROR", Message: err.Error()},
					Timestamp: time.Now(),
				}
				c.sendMessage(errorResp)
			}
		}
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("Error writing message to client %s: %v", c.clientID, err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming messages from clients
func (c *Client) handleMessage(data []byte) error {
	message, err := ParseMessage(data)
	if err != nil {
		return err
	}

	switch msg := message.(type) {
	case SubscribeRequest:
		return c.handleSubscribe(msg)
	case UnsubscribeRequest:
		return c.handleUnsubscribe(msg)
	case PublishRequest:
		return c.handlePublish(msg)
	case PingRequest:
		return c.handlePing(msg)
	default:
		return ErrorData{
			Code:    "UNKNOWN_MESSAGE_TYPE",
			Message: "Unknown message type received",
		}
	}
}

// handleSubscribe processes subscribe requests
func (c *Client) handleSubscribe(req SubscribeRequest) error {
	// Validate request ID and client ID
	if req.RequestID == "" {
		return ErrorData{Code: "BAD_REQUEST", Message: "request_id is required"}
	}
	if req.ClientID == "" {
		return ErrorData{Code: "BAD_REQUEST", Message: "client_id is required"}
	}

	// Set client ID from the message (store in connection for future use)
	if c.clientID == "" {
		c.clientID = req.ClientID
	} else if c.clientID != req.ClientID {
		return ErrorData{Code: "BAD_REQUEST", Message: "client_id mismatch with existing connection"}
	}

	lastMessages, err := c.pubsub.Subscribe(c.clientID, req.Topic, req.LastN, c.send)
	if err != nil {
		// Send error response
		errorResp := ErrorResponse{
			Type:      "error",
			RequestID: req.RequestID,
			Error:     ErrorData{Code: "SUBSCRIBE_FAILED", Message: err.Error()},
			Timestamp: time.Now(),
		}
		return c.sendMessage(errorResp)
	}

	// Send acknowledgment
	ackResp := AckResponse{
		Type:      "ack",
		RequestID: req.RequestID,
		Topic:     req.Topic,
		Status:    "ok",
		Timestamp: time.Now(),
	}

	if err := c.sendMessage(ackResp); err != nil {
		return err
	}

	// Send last N messages if any
	for _, lastMsg := range lastMessages {
		if err := c.sendMessage(lastMsg); err != nil {
			log.Printf("Error sending last message to client %s: %v", c.clientID, err)
		}
	}

	return nil
}

// handleUnsubscribe processes unsubscribe requests
func (c *Client) handleUnsubscribe(req UnsubscribeRequest) error {
	if req.RequestID == "" {
		return ErrorData{Code: "BAD_REQUEST", Message: "request_id is required"}
	}
	if req.ClientID == "" {
		return ErrorData{Code: "BAD_REQUEST", Message: "client_id is required"}
	}

	// Validate client ID matches the connection
	if c.clientID == "" {
		c.clientID = req.ClientID
	} else if c.clientID != req.ClientID {
		return ErrorData{Code: "BAD_REQUEST", Message: "client_id mismatch with existing connection"}
	}

	err := c.pubsub.Unsubscribe(c.clientID, req.Topic)
	if err != nil {
		errorResp := ErrorResponse{
			Type:      "error",
			RequestID: req.RequestID,
			Error:     ErrorData{Code: "UNSUBSCRIBE_FAILED", Message: err.Error()},
			Timestamp: time.Now(),
		}
		return c.sendMessage(errorResp)
	}

	// Send acknowledgment
	ackResp := AckResponse{
		Type:      "ack",
		RequestID: req.RequestID,
		Topic:     req.Topic,
		Status:    "ok",
		Timestamp: time.Now(),
	}

	return c.sendMessage(ackResp)
}

// handlePublish processes publish requests
func (c *Client) handlePublish(req PublishRequest) error {
	if req.RequestID == "" {
		return ErrorData{Code: "BAD_REQUEST", Message: "request_id is required"}
	}

	// Handle client_id: use from connection if already set, or set it from the request
	if c.clientID == "" {
		if req.ClientID == "" {
			return ErrorData{Code: "BAD_REQUEST", Message: "client_id is required for first message on this connection"}
		}
		c.clientID = req.ClientID
	} else if req.ClientID != "" && c.clientID != req.ClientID {
		return ErrorData{Code: "BAD_REQUEST", Message: "client_id mismatch with existing connection"}
	}

	// Validate message ID is a valid UUID
	if req.Message.ID == "" {
		errorResp := ErrorResponse{
			Type:      "error",
			RequestID: req.RequestID,
			Error:     ErrorData{Code: "BAD_REQUEST", Message: "message.id must be a valid UUID"},
			Timestamp: time.Now(),
		}
		return c.sendMessage(errorResp)
	}

	// Validate UUID format
	if _, err := uuid.Parse(req.Message.ID); err != nil {
		errorResp := ErrorResponse{
			Type:      "error",
			RequestID: req.RequestID,
			Error:     ErrorData{Code: "BAD_REQUEST", Message: "message.id must be a valid UUID"},
			Timestamp: time.Now(),
		}
		return c.sendMessage(errorResp)
	}

	// Use the stored client_id from the connection
	err := c.pubsub.Publish(req.Topic, req.Message, c.clientID)
	if err != nil {
		errorResp := ErrorResponse{
			Type:      "error",
			RequestID: req.RequestID,
			Error:     ErrorData{Code: "PUBLISH_FAILED", Message: err.Error()},
			Timestamp: time.Now(),
		}
		return c.sendMessage(errorResp)
	}

	// Send acknowledgment
	ackResp := AckResponse{
		Type:      "ack",
		RequestID: req.RequestID,
		Topic:     req.Topic,
		Status:    "ok",
		Timestamp: time.Now(),
	}

	return c.sendMessage(ackResp)
}

// handlePing processes ping requests
func (c *Client) handlePing(req PingRequest) error {
	if req.RequestID == "" {
		return ErrorData{Code: "BAD_REQUEST", Message: "request_id is required"}
	}

	pongResp := PongResponse{
		Type:      "pong",
		RequestID: req.RequestID,
		Timestamp: time.Now(),
	}

	return c.sendMessage(pongResp)
}

// sendMessage sends a message to the client
func (c *Client) sendMessage(message interface{}) error {
	// Convert message to EventResponse format for the send channel
	var eventMsg EventResponse

	switch msg := message.(type) {
	case EventResponse:
		eventMsg = msg
	case AckResponse:
		// Convert AckResponse to EventResponse format
		eventMsg = EventResponse{
			Type:      msg.Type,
			Topic:     msg.Topic,
			Message:   MessageData{ID: msg.RequestID, Payload: map[string]interface{}{"status": msg.Status}},
			Timestamp: msg.Timestamp,
		}
	case ErrorResponse:
		// Convert ErrorResponse to EventResponse format
		eventMsg = EventResponse{
			Type:      msg.Type,
			Topic:     "",
			Message:   MessageData{ID: msg.RequestID, Payload: msg.Error},
			Timestamp: msg.Timestamp,
		}
	case PongResponse:
		// Convert PongResponse to EventResponse format
		eventMsg = EventResponse{
			Type:      msg.Type,
			Topic:     "",
			Message:   MessageData{ID: msg.RequestID, Payload: "pong"},
			Timestamp: msg.Timestamp,
		}
	default:
		return ErrorData{Code: "INTERNAL_ERROR", Message: "Unknown message type to send"}
	}

	// Try to send message without blocking
	select {
	case c.send <- eventMsg:
		return nil
	default:
		// Channel is full, client is slow
		log.Printf("Client %s send channel is full, dropping message", c.clientID)
		return ErrorData{Code: "CLIENT_OVERLOADED", Message: "Client send buffer is full"}
	}
}

// cleanup handles client disconnection
func (c *Client) cleanup() {
	// Disconnect client from pub-sub system
	c.pubsub.DisconnectClient(c.clientID)

	// Close send channel
	close(c.send)

	log.Printf("Client %s disconnected", c.clientID)
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(pubsub *PubSubSystem) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		client := NewClient(conn, pubsub)
		log.Printf("New client connected: %s", client.clientID)

		// Start read, process, and write pumps in separate goroutines
		go client.readPump()
		go client.processPump()
		go client.writePump()
	}
}

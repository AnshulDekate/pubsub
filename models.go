package main

import (
	"encoding/json"
	"time"
)

// Request message types
type SubscribeRequest struct {
	Type      string `json:"type"`
	Topic     string `json:"topic"`
	ClientID  string `json:"client_id"`
	LastN     int    `json:"last_n,omitempty"`
	RequestID string `json:"request_id"`
}

type UnsubscribeRequest struct {
	Type      string `json:"type"`
	Topic     string `json:"topic"`
	ClientID  string `json:"client_id"`
	RequestID string `json:"request_id"`
}

type PublishRequest struct {
	Type      string      `json:"type"`
	Topic     string      `json:"topic"`
	Message   MessageData `json:"message"`
	ClientID  string      `json:"client_id,omitempty"` // Optional - used to set client ID if not already set
	RequestID string      `json:"request_id"`
}

type PingRequest struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id"`
}

type MessageData struct {
	ID      string      `json:"id"`
	Payload interface{} `json:"payload"`
}

// Response message types
type AckResponse struct {
	Type      string    `json:"type"`
	RequestID string    `json:"request_id"`
	Topic     string    `json:"topic,omitempty"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"ts"`
}

type EventResponse struct {
	Type      string      `json:"type"`
	Topic     string      `json:"topic"`
	Message   MessageData `json:"message"`
	Timestamp time.Time   `json:"ts"`
}

type ErrorResponse struct {
	Type      string    `json:"type"`
	RequestID string    `json:"request_id,omitempty"`
	Error     ErrorData `json:"error"`
	Timestamp time.Time `json:"ts"`
}

type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e ErrorData) Error() string {
	return e.Message
}

type PongResponse struct {
	Type      string    `json:"type"`
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"ts"`
}

type InfoResponse struct {
	Type      string    `json:"type"`
	Topic     string    `json:"topic,omitempty"`
	Message   string    `json:"msg"`
	Timestamp time.Time `json:"ts"`
}

// HTTP API models
type CreateTopicRequest struct {
	Name string `json:"name"`
}

type CreateTopicResponse struct {
	Status string `json:"status"`
	Topic  string `json:"topic"`
}

type DeleteTopicResponse struct {
	Status string `json:"status"`
	Topic  string `json:"topic"`
}

type TopicInfo struct {
	Name        string `json:"name"`
	Subscribers int    `json:"subscribers"`
}

type TopicsResponse struct {
	Topics []TopicInfo `json:"topics"`
}

type HealthResponse struct {
	UptimeSeconds int `json:"uptime_sec"`
	Topics        int `json:"topics"`
	Subscribers   int `json:"subscribers"`
}

type TopicStats struct {
	Messages    int64 `json:"messages"`
	Subscribers int   `json:"subscribers"`
}

type StatsResponse struct {
	Topics map[string]TopicStats `json:"topics"`
}

type ClientSubscription struct {
	ClientID string   `json:"client_id"`
	Topics   []string `json:"topics"`
}

type SubscriptionsStatusResponse struct {
	TotalClients   int                  `json:"total_clients"`
	TotalTopics    int                  `json:"total_topics"`
	Subscriptions  []ClientSubscription `json:"subscriptions"`
	TopicBreakdown map[string][]string  `json:"topic_breakdown"` // topic -> list of client_ids
}

// Generic message wrapper for parsing incoming JSON
type IncomingMessage struct {
	Type string `json:"type"`
}

// ParseMessage parses incoming JSON and returns the appropriate struct
func ParseMessage(data []byte) (interface{}, error) {
	var incoming IncomingMessage
	if err := json.Unmarshal(data, &incoming); err != nil {
		return nil, err
	}

	switch incoming.Type {
	case "subscribe":
		var msg SubscribeRequest
		err := json.Unmarshal(data, &msg)
		return msg, err
	case "unsubscribe":
		var msg UnsubscribeRequest
		err := json.Unmarshal(data, &msg)
		return msg, err
	case "publish":
		var msg PublishRequest
		err := json.Unmarshal(data, &msg)
		return msg, err
	case "ping":
		var msg PingRequest
		err := json.Unmarshal(data, &msg)
		return msg, err
	default:
		return nil, ErrorData{
			Code:    "INVALID_MESSAGE_TYPE",
			Message: "Unknown message type: " + incoming.Type,
		}
	}
}

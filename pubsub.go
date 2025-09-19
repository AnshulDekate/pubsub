package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultBufferSize      = 100  // Default ring buffer size per subscriber
	TopicHistoryBufferSize = 1000 // Default ring buffer size per topic for message history
)

// PubSubClient represents a client in the pub-sub system
type PubSubClient struct {
	ClientID   string
	Buffer     *RingBuffer        // Per-client ring buffer for concurrency/backpressure
	WriteChan  chan EventResponse // Channel to send messages to client's write pump
	LastActive time.Time
	Connected  bool
}

// Subscriber represents a client subscribed to a topic (just a link)
type Subscriber struct {
	ClientID string
	Topic    string
	Client   *PubSubClient // Reference to the actual client
}

// Topic represents a chat room topic
type Topic struct {
	Name           string
	Subscribers    map[string]*Subscriber // clientID -> Subscriber
	MessageCount   int64
	CreatedAt      time.Time
	MessageHistory *RingBuffer // Topic-level message history for last_n
	mutex          sync.RWMutex
}

// PubSubSystem manages the entire pub-sub system
type PubSubSystem struct {
	// Topic -> client_ids mapping for fan-out
	topics map[string]*Topic

	// client_id -> PubSubClient mapping (separate client registry)
	clients map[string]*PubSubClient

	// client_id -> set of topics mapping (client can subscribe to multiple topics)
	clientTopics map[string]map[string]bool

	// System-wide mutex for topic operations
	topicsMutex sync.RWMutex

	// client mapping mutex
	clientMutex sync.RWMutex

	// System stats
	startTime time.Time
}

// NewPubSubSystem creates a new pub-sub system
func NewPubSubSystem() *PubSubSystem {
	return &PubSubSystem{
		topics:       make(map[string]*Topic),
		clients:      make(map[string]*PubSubClient),
		clientTopics: make(map[string]map[string]bool),
		startTime:    time.Now(),
	}
}

// CreateTopic creates a new topic
func (ps *PubSubSystem) CreateTopic(name string) error {
	ps.topicsMutex.Lock()
	defer ps.topicsMutex.Unlock()

	if _, exists := ps.topics[name]; exists {
		return fmt.Errorf("topic %s already exists", name)
	}

	ps.topics[name] = &Topic{
		Name:           name,
		Subscribers:    make(map[string]*Subscriber),
		CreatedAt:      time.Now(),
		MessageHistory: NewRingBuffer(TopicHistoryBufferSize),
	}

	return nil
}

// DeleteTopic deletes a topic and disconnects all subscribers
func (ps *PubSubSystem) DeleteTopic(name string) error {
	ps.topicsMutex.Lock()
	defer ps.topicsMutex.Unlock()

	topic, exists := ps.topics[name]
	if !exists {
		return fmt.Errorf("topic %s not found", name)
	}

	// Notify all subscribers about topic deletion
	topic.mutex.Lock()
	for _, subscriber := range topic.Subscribers {
		// Send topic deletion notice
		notice := InfoResponse{
			Type:      "info",
			Topic:     name,
			Message:   "topic_deleted",
			Timestamp: time.Now(),
		}

		// Try to send notice, don't block if client is slow
		select {
		case subscriber.Client.WriteChan <- EventResponse{
			Type:      notice.Type,
			Topic:     notice.Topic,
			Message:   MessageData{ID: uuid.New().String(), Payload: notice.Message},
			Timestamp: notice.Timestamp,
		}:
		default:
			// Client write channel is full, skip
		}

		// Remove from client mapping
		ps.clientMutex.Lock()
		if clientTopics, exists := ps.clientTopics[subscriber.ClientID]; exists {
			delete(clientTopics, name)
			if len(clientTopics) == 0 {
				delete(ps.clientTopics, subscriber.ClientID)
			}
		}
		ps.clientMutex.Unlock()
	}
	topic.mutex.Unlock()

	// Delete the topic
	delete(ps.topics, name)
	return nil
}

// Subscribe adds a client to a topic
func (ps *PubSubSystem) Subscribe(clientID, topicName string, lastN int, writeChan chan EventResponse) ([]EventResponse, error) {
	// Check if topic exists
	ps.topicsMutex.RLock()
	topic, exists := ps.topics[topicName]
	ps.topicsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("topic %s not found", topicName)
	}

	// Add client to the topic mapping (allow multiple topic subscriptions)
	ps.clientMutex.Lock()
	if ps.clientTopics[clientID] == nil {
		ps.clientTopics[clientID] = make(map[string]bool)
	}
	ps.clientTopics[clientID][topicName] = true
	ps.clientMutex.Unlock()

	// Register or get existing client
	ps.clientMutex.Lock()
	client, exists := ps.clients[clientID]
	if !exists {
		client = &PubSubClient{
			ClientID:   clientID,
			Buffer:     NewRingBuffer(DefaultBufferSize),
			WriteChan:  writeChan,
			LastActive: time.Now(),
			Connected:  true,
		}
		ps.clients[clientID] = client
	} else {
		// Update existing client's write channel (for reconnections)
		client.WriteChan = writeChan
		client.Connected = true
		client.LastActive = time.Now()
	}
	ps.clientMutex.Unlock()

	// Add subscriber to topic
	topic.mutex.Lock()
	defer topic.mutex.Unlock()

	subscriber := &Subscriber{
		ClientID: clientID,
		Topic:    topicName,
		Client:   client,
	}

	topic.Subscribers[clientID] = subscriber

	// Return last N messages if requested from topic's message history
	var lastMessages []EventResponse
	if lastN > 0 {
		lastMessages = topic.MessageHistory.GetLastN(lastN)
	}

	return lastMessages, nil
}

// Unsubscribe removes a client from a specific topic
func (ps *PubSubSystem) Unsubscribe(clientID, topicName string) error {
	ps.clientMutex.Lock()
	clientTopics, exists := ps.clientTopics[clientID]
	if !exists || !clientTopics[topicName] {
		ps.clientMutex.Unlock()
		return fmt.Errorf("client %s is not subscribed to topic %s", clientID, topicName)
	}
	delete(clientTopics, topicName)
	if len(clientTopics) == 0 {
		delete(ps.clientTopics, clientID)
	}
	ps.clientMutex.Unlock()

	// Remove from topic
	ps.topicsMutex.RLock()
	topic, exists := ps.topics[topicName]
	ps.topicsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("topic %s not found", topicName)
	}

	topic.mutex.Lock()
	defer topic.mutex.Unlock()

	delete(topic.Subscribers, clientID)
	return nil
}

// Publish sends a message to all subscribers of a topic except the sender
func (ps *PubSubSystem) Publish(topicName string, message MessageData, senderClientID string) error {
	ps.topicsMutex.RLock()
	topic, exists := ps.topics[topicName]
	ps.topicsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("topic %s not found", topicName)
	}

	// Create event message
	event := EventResponse{
		Type:      "event",
		Topic:     topicName,
		Message:   message,
		Timestamp: time.Now(),
	}

	topic.mutex.Lock()
	topic.MessageCount++

	// Add message to topic's history for last_n functionality
	topic.MessageHistory.Push(event)

	for _, subscriber := range topic.Subscribers {
		if !subscriber.Client.Connected {
			continue
		}

		// Send message to all subscribers (including sender)
		// Add to client's ring buffer for backpressure handling
		subscriber.Client.Buffer.Push(event)

		// Try to send immediately via write channel, don't block
		select {
		case subscriber.Client.WriteChan <- event:
			subscriber.Client.LastActive = time.Now()
		default:
			// Write channel is full, message is in buffer for later retrieval
		}
	}
	topic.mutex.Unlock()

	return nil
}

// GetTopics returns all topics with subscriber counts
func (ps *PubSubSystem) GetTopics() []TopicInfo {
	ps.topicsMutex.RLock()
	defer ps.topicsMutex.RUnlock()

	topics := make([]TopicInfo, 0, len(ps.topics))
	for _, topic := range ps.topics {
		topic.mutex.RLock()
		topics = append(topics, TopicInfo{
			Name:        topic.Name,
			Subscribers: len(topic.Subscribers),
		})
		topic.mutex.RUnlock()
	}

	return topics
}

// GetStats returns detailed statistics
func (ps *PubSubSystem) GetStats() StatsResponse {
	ps.topicsMutex.RLock()
	defer ps.topicsMutex.RUnlock()

	stats := StatsResponse{
		Topics: make(map[string]TopicStats),
	}

	for name, topic := range ps.topics {
		topic.mutex.RLock()
		stats.Topics[name] = TopicStats{
			Messages:    topic.MessageCount,
			Subscribers: len(topic.Subscribers),
		}
		topic.mutex.RUnlock()
	}

	return stats
}

// GetHealth returns system health information
func (ps *PubSubSystem) GetHealth() HealthResponse {
	ps.topicsMutex.RLock()
	defer ps.topicsMutex.RUnlock()

	totalSubscribers := 0
	for _, topic := range ps.topics {
		topic.mutex.RLock()
		totalSubscribers += len(topic.Subscribers)
		topic.mutex.RUnlock()
	}

	return HealthResponse{
		UptimeSeconds: int(time.Since(ps.startTime).Seconds()),
		Topics:        len(ps.topics),
		Subscribers:   totalSubscribers,
	}
}

// GetClientTopics returns all topics a client is subscribed to
func (ps *PubSubSystem) GetClientTopics(clientID string) []string {
	ps.clientMutex.RLock()
	defer ps.clientMutex.RUnlock()

	topicsMap, exists := ps.clientTopics[clientID]
	if !exists {
		return []string{}
	}

	topics := make([]string, 0, len(topicsMap))
	for topic := range topicsMap {
		topics = append(topics, topic)
	}
	return topics
}

// GetSubscriptionsStatus returns detailed subscription information for all clients
func (ps *PubSubSystem) GetSubscriptionsStatus() SubscriptionsStatusResponse {
	ps.clientMutex.RLock()
	ps.topicsMutex.RLock()
	defer ps.clientMutex.RUnlock()
	defer ps.topicsMutex.RUnlock()

	// Build client subscriptions list
	subscriptions := make([]ClientSubscription, 0, len(ps.clientTopics))
	for clientID, topicsMap := range ps.clientTopics {
		topics := make([]string, 0, len(topicsMap))
		for topic := range topicsMap {
			topics = append(topics, topic)
		}
		subscriptions = append(subscriptions, ClientSubscription{
			ClientID: clientID,
			Topics:   topics,
		})
	}

	// Build topic breakdown (topic -> list of client_ids)
	topicBreakdown := make(map[string][]string)
	for topicName, topic := range ps.topics {
		topic.mutex.RLock()
		clients := make([]string, 0, len(topic.Subscribers))
		for clientID := range topic.Subscribers {
			clients = append(clients, clientID)
		}
		topicBreakdown[topicName] = clients
		topic.mutex.RUnlock()
	}

	return SubscriptionsStatusResponse{
		TotalClients:   len(ps.clientTopics),
		TotalTopics:    len(ps.topics),
		Subscriptions:  subscriptions,
		TopicBreakdown: topicBreakdown,
	}
}

// DisconnectClient cleans up when a client disconnects from all topics
func (ps *PubSubSystem) DisconnectClient(clientID string) {
	ps.clientMutex.Lock()

	// Mark client as disconnected
	if client, exists := ps.clients[clientID]; exists {
		client.Connected = false
	}

	topicsMap, exists := ps.clientTopics[clientID]
	if exists {
		delete(ps.clientTopics, clientID)
	}
	ps.clientMutex.Unlock()

	if !exists {
		return
	}

	// Remove from all subscribed topics
	ps.topicsMutex.RLock()
	for topicName := range topicsMap {
		if topic, exists := ps.topics[topicName]; exists {
			topic.mutex.Lock()
			delete(topic.Subscribers, clientID)
			topic.mutex.Unlock()
		}
	}
	ps.topicsMutex.RUnlock()
}

package main

import (
	"testing"
	"time"
)

func TestPubSubSystemCreation(t *testing.T) {
	ps := NewPubSubSystem()
	if ps == nil {
		t.Fatal("PubSubSystem creation failed")
	}

	if ps.topics == nil {
		t.Error("topics map not initialized")
	}

	if ps.clients == nil {
		t.Error("clients map not initialized")
	}

	if ps.clientTopics == nil {
		t.Error("clientTopics map not initialized")
	}
}

func TestTopicCreation(t *testing.T) {
	ps := NewPubSubSystem()

	// Test topic creation
	err := ps.CreateTopic("test-topic")
	if err != nil {
		t.Fatalf("Failed to create topic: %v", err)
	}

	// Test duplicate topic creation
	err = ps.CreateTopic("test-topic")
	if err == nil {
		t.Error("Expected error for duplicate topic creation")
	}
}

func TestTopicDeletion(t *testing.T) {
	ps := NewPubSubSystem()

	// Create and delete topic
	ps.CreateTopic("delete-topic")
	err := ps.DeleteTopic("delete-topic")
	if err != nil {
		t.Fatalf("Failed to delete topic: %v", err)
	}

	// Test deleting non-existent topic
	err = ps.DeleteTopic("non-existent")
	if err == nil {
		t.Error("Expected error for deleting non-existent topic")
	}
}

func TestSubscription(t *testing.T) {
	ps := NewPubSubSystem()
	ps.CreateTopic("sub-topic")

	// Create a write channel for testing
	writeChan := make(chan EventResponse, 10)

	// Test subscription
	_, err := ps.Subscribe("client1", "sub-topic", 0, writeChan)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Test subscription to non-existent topic
	_, err = ps.Subscribe("client2", "non-existent", 0, writeChan)
	if err == nil {
		t.Error("Expected error for subscribing to non-existent topic")
	}
}

func TestUnsubscription(t *testing.T) {
	ps := NewPubSubSystem()
	ps.CreateTopic("unsub-topic")

	writeChan := make(chan EventResponse, 10)
	ps.Subscribe("client1", "unsub-topic", 0, writeChan)

	// Test unsubscription
	err := ps.Unsubscribe("client1", "unsub-topic")
	if err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}

	// Test unsubscribing non-subscribed client
	err = ps.Unsubscribe("client2", "unsub-topic")
	if err == nil {
		t.Error("Expected error for unsubscribing non-subscribed client")
	}
}

func TestPublishSubscribe(t *testing.T) {
	ps := NewPubSubSystem()
	ps.CreateTopic("pub-topic")

	writeChan1 := make(chan EventResponse, 10)
	writeChan2 := make(chan EventResponse, 10)

	// Subscribe two clients
	ps.Subscribe("client1", "pub-topic", 0, writeChan1)
	ps.Subscribe("client2", "pub-topic", 0, writeChan2)

	// Publish message
	message := MessageData{
		ID:      "test-message-1",
		Payload: map[string]interface{}{"text": "Hello World"},
	}

	err := ps.Publish("pub-topic", message, "publisher")
	if err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	// Check both subscribers received message
	select {
	case event := <-writeChan1:
		if event.Type != "event" {
			t.Error("Expected event message")
		}
		if event.Topic != "pub-topic" {
			t.Error("Wrong topic in event")
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for message on writeChan1")
	}

	select {
	case event := <-writeChan2:
		if event.Type != "event" {
			t.Error("Expected event message")
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for message on writeChan2")
	}
}

func TestNoEchoBack(t *testing.T) {
	ps := NewPubSubSystem()
	ps.CreateTopic("echo-topic")

	writeChan := make(chan EventResponse, 10)

	// Subscribe client as both subscriber and publisher
	ps.Subscribe("client1", "echo-topic", 0, writeChan)

	// Publish message from same client
	message := MessageData{
		ID:      "echo-test-1",
		Payload: map[string]interface{}{"text": "Echo test"},
	}

	err := ps.Publish("echo-topic", message, "client1")
	if err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	// Should not receive own message
	select {
	case <-writeChan:
		t.Error("Client received its own message (echo-back violation)")
	case <-time.After(100 * time.Millisecond):
		// Correct behavior - no echo-back
	}
}

func TestTopicIsolation(t *testing.T) {
	ps := NewPubSubSystem()
	ps.CreateTopic("topic-a")
	ps.CreateTopic("topic-b")

	writeChanA := make(chan EventResponse, 10)
	writeChanB := make(chan EventResponse, 10)

	// Subscribe to different topics
	ps.Subscribe("clientA", "topic-a", 0, writeChanA)
	ps.Subscribe("clientB", "topic-b", 0, writeChanB)

	// Publish to topic-a
	message := MessageData{
		ID:      "isolation-test",
		Payload: map[string]interface{}{"text": "Topic A message"},
	}

	err := ps.Publish("topic-a", message, "publisher")
	if err != nil {
		t.Fatalf("Failed to publish to topic-a: %v", err)
	}

	// Client A should receive message
	select {
	case event := <-writeChanA:
		if event.Topic != "topic-a" {
			t.Error("Wrong topic in received message")
		}
	case <-time.After(time.Second):
		t.Error("Client A didn't receive message from topic-a")
	}

	// Client B should NOT receive message
	select {
	case <-writeChanB:
		t.Error("Client B received message from topic-a (isolation violation)")
	case <-time.After(100 * time.Millisecond):
		// Correct behavior - topic isolation maintained
	}
}

func TestMultiTopicSubscription(t *testing.T) {
	ps := NewPubSubSystem()
	ps.CreateTopic("topic-1")
	ps.CreateTopic("topic-2")

	writeChan := make(chan EventResponse, 10)

	// Subscribe to multiple topics
	ps.Subscribe("multi-client", "topic-1", 0, writeChan)
	ps.Subscribe("multi-client", "topic-2", 0, writeChan)

	// Publish to both topics
	message1 := MessageData{ID: "msg-1", Payload: map[string]interface{}{"topic": "1"}}
	message2 := MessageData{ID: "msg-2", Payload: map[string]interface{}{"topic": "2"}}

	ps.Publish("topic-1", message1, "publisher")
	ps.Publish("topic-2", message2, "publisher")

	// Should receive both messages
	receivedTopics := make(map[string]bool)

	for i := 0; i < 2; i++ {
		select {
		case event := <-writeChan:
			receivedTopics[event.Topic] = true
		case <-time.After(time.Second):
			t.Error("Timeout waiting for messages")
		}
	}

	if !receivedTopics["topic-1"] || !receivedTopics["topic-2"] {
		t.Error("Did not receive messages from both topics")
	}
}

func TestMessageHistory(t *testing.T) {
	ps := NewPubSubSystem()
	ps.CreateTopic("history-topic")

	// Publish some messages first
	for i := 1; i <= 5; i++ {
		message := MessageData{
			ID:      "hist-msg-" + string(rune('0'+i)),
			Payload: map[string]interface{}{"sequence": i},
		}
		ps.Publish("history-topic", message, "publisher")
	}

	// Subscribe with last_n=3
	writeChan := make(chan EventResponse, 10)
	lastMessages, err := ps.Subscribe("late-joiner", "history-topic", 3, writeChan)

	if err != nil {
		t.Fatalf("Failed to subscribe with history: %v", err)
	}

	if len(lastMessages) != 3 {
		t.Errorf("Expected 3 historical messages, got %d", len(lastMessages))
	}

	// Verify messages are in chronological order
	for i, msg := range lastMessages {
		expectedSeq := i + 3 // Should get messages 3, 4, 5
		if payload, ok := msg.Message.Payload.(map[string]interface{}); ok {
			if seq, ok := payload["sequence"].(int); ok {
				if seq != expectedSeq {
					t.Errorf("Expected sequence %d, got %d", expectedSeq, seq)
				}
			}
		}
	}
}

func TestGetStats(t *testing.T) {
	ps := NewPubSubSystem()
	ps.CreateTopic("stats-topic")

	writeChan := make(chan EventResponse, 10)
	ps.Subscribe("stats-client", "stats-topic", 0, writeChan)

	// Publish some messages
	for i := 0; i < 3; i++ {
		message := MessageData{
			ID:      "stats-msg-" + string(rune('0'+i)),
			Payload: map[string]interface{}{"count": i},
		}
		ps.Publish("stats-topic", message, "publisher")
	}

	stats := ps.GetStats()

	if len(stats.Topics) != 1 {
		t.Errorf("Expected 1 topic in stats, got %d", len(stats.Topics))
	}

	topicStats, exists := stats.Topics["stats-topic"]
	if !exists {
		t.Error("stats-topic not found in stats")
	}

	if topicStats.Messages != 3 {
		t.Errorf("Expected 3 messages in stats, got %d", topicStats.Messages)
	}

	if topicStats.Subscribers != 1 {
		t.Errorf("Expected 1 subscriber in stats, got %d", topicStats.Subscribers)
	}
}

func TestDisconnectClient(t *testing.T) {
	ps := NewPubSubSystem()
	ps.CreateTopic("disconnect-topic")

	writeChan := make(chan EventResponse, 10)
	ps.Subscribe("disconnect-client", "disconnect-topic", 0, writeChan)

	// Verify client is subscribed
	topics := ps.GetClientTopics("disconnect-client")
	if len(topics) != 1 || topics[0] != "disconnect-topic" {
		t.Error("Client not properly subscribed")
	}

	// Disconnect client
	ps.DisconnectClient("disconnect-client")

	// Verify client is no longer subscribed
	topics = ps.GetClientTopics("disconnect-client")
	if len(topics) != 0 {
		t.Error("Client still has subscriptions after disconnect")
	}

	// Verify client is removed from topic subscribers
	stats := ps.GetStats()
	if stats.Topics["disconnect-topic"].Subscribers != 0 {
		t.Error("Topic still has subscribers after client disconnect")
	}
}

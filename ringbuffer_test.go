package main

import (
	"testing"
	"time"
)

func TestRingBufferCreation(t *testing.T) {
	rb := NewRingBuffer(5)
	if rb == nil {
		t.Fatal("RingBuffer creation failed")
	}

	if rb.capacity != 5 {
		t.Errorf("Expected capacity 5, got %d", rb.capacity)
	}

	if rb.Size() != 0 {
		t.Errorf("Expected size 0, got %d", rb.Size())
	}

	if rb.IsFull() {
		t.Error("New buffer should not be full")
	}
}

func TestRingBufferPushPop(t *testing.T) {
	rb := NewRingBuffer(3)

	// Test pushing messages
	msg1 := EventResponse{Type: "event", Message: MessageData{ID: "1", Payload: "test1"}}
	msg2 := EventResponse{Type: "event", Message: MessageData{ID: "2", Payload: "test2"}}

	rb.Push(msg1)
	rb.Push(msg2)

	if rb.Size() != 2 {
		t.Errorf("Expected size 2, got %d", rb.Size())
	}

	// Test popping messages
	popped := rb.Pop()
	if popped == nil {
		t.Fatal("Pop returned nil")
	}

	if popped.Message.ID != "1" {
		t.Errorf("Expected ID '1', got '%s'", popped.Message.ID)
	}

	if rb.Size() != 1 {
		t.Errorf("Expected size 1 after pop, got %d", rb.Size())
	}
}

func TestRingBufferOverflow(t *testing.T) {
	rb := NewRingBuffer(2)

	// Fill buffer
	msg1 := EventResponse{Message: MessageData{ID: "1", Payload: "test1"}}
	msg2 := EventResponse{Message: MessageData{ID: "2", Payload: "test2"}}
	msg3 := EventResponse{Message: MessageData{ID: "3", Payload: "test3"}}

	rb.Push(msg1)
	rb.Push(msg2)

	if !rb.IsFull() {
		t.Error("Buffer should be full")
	}

	// Overflow - should drop oldest message
	rb.Push(msg3)

	if rb.Size() != 2 {
		t.Errorf("Expected size 2 after overflow, got %d", rb.Size())
	}

	// First pop should be message 2 (oldest message 1 was dropped)
	popped := rb.Pop()
	if popped.Message.ID != "2" {
		t.Errorf("Expected ID '2' after overflow, got '%s'", popped.Message.ID)
	}
}

func TestRingBufferGetLastN(t *testing.T) {
	rb := NewRingBuffer(5)

	// Push messages
	for i := 1; i <= 4; i++ {
		msg := EventResponse{Message: MessageData{ID: string(rune('0' + i)), Payload: i}}
		rb.Push(msg)
	}

	// Get last 2 messages
	last2 := rb.GetLastN(2)
	if len(last2) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(last2))
	}

	// Should get messages in chronological order (3, 4)
	if last2[0].Message.ID != "3" || last2[1].Message.ID != "4" {
		t.Error("GetLastN returned wrong messages or wrong order")
	}

	// Get more messages than available
	last10 := rb.GetLastN(10)
	if len(last10) != 4 {
		t.Errorf("Expected 4 messages (all available), got %d", len(last10))
	}
}

func TestRingBufferPopAll(t *testing.T) {
	rb := NewRingBuffer(3)

	// Push messages
	for i := 1; i <= 3; i++ {
		msg := EventResponse{Message: MessageData{ID: string(rune('0' + i)), Payload: i}}
		rb.Push(msg)
	}

	// PopAll should return all messages and clear buffer
	all := rb.PopAll()
	if len(all) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(all))
	}

	if rb.Size() != 0 {
		t.Errorf("Expected empty buffer after PopAll, got size %d", rb.Size())
	}

	// Verify messages are in chronological order
	for i, msg := range all {
		expectedID := string(rune('1' + i))
		if msg.Message.ID != expectedID {
			t.Errorf("Expected ID '%s', got '%s'", expectedID, msg.Message.ID)
		}
	}
}

func TestRingBufferClear(t *testing.T) {
	rb := NewRingBuffer(3)

	// Push messages
	for i := 1; i <= 2; i++ {
		msg := EventResponse{Message: MessageData{ID: string(rune('0' + i)), Payload: i}}
		rb.Push(msg)
	}

	if rb.Size() != 2 {
		t.Errorf("Expected size 2 before clear, got %d", rb.Size())
	}

	// Clear buffer
	rb.Clear()

	if rb.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", rb.Size())
	}

	if rb.IsFull() {
		t.Error("Buffer should not be full after clear")
	}

	// Pop should return nil from cleared buffer
	popped := rb.Pop()
	if popped != nil {
		t.Error("Expected nil from cleared buffer")
	}
}

func TestRingBufferConcurrency(t *testing.T) {
	rb := NewRingBuffer(100)
	done := make(chan bool)

	// Producer goroutine
	go func() {
		for i := 0; i < 50; i++ {
			msg := EventResponse{Message: MessageData{ID: string(rune('A' + i)), Payload: i}}
			rb.Push(msg)
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Consumer goroutine
	go func() {
		for i := 0; i < 30; i++ {
			rb.Pop()
			time.Sleep(time.Microsecond * 2)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Buffer should have remaining messages
	if rb.Size() == 0 {
		t.Error("Expected some messages remaining after concurrent operations")
	}

	if rb.Size() > 50 {
		t.Error("Buffer size inconsistent after concurrent operations")
	}
}

func TestRingBufferEdgeCases(t *testing.T) {
	// Test with capacity 1
	rb := NewRingBuffer(1)

	msg1 := EventResponse{Message: MessageData{ID: "1", Payload: "test1"}}
	msg2 := EventResponse{Message: MessageData{ID: "2", Payload: "test2"}}

	rb.Push(msg1)
	if !rb.IsFull() {
		t.Error("Buffer with capacity 1 should be full after one push")
	}

	rb.Push(msg2) // Should replace msg1
	if rb.Size() != 1 {
		t.Errorf("Expected size 1, got %d", rb.Size())
	}

	popped := rb.Pop()
	if popped.Message.ID != "2" {
		t.Error("Should get the second message (first was overwritten)")
	}

	// Test GetLastN with empty buffer
	empty := NewRingBuffer(5)
	lastN := empty.GetLastN(3)
	if lastN != nil {
		t.Error("GetLastN on empty buffer should return nil")
	}

	// Test GetLastN with 0 or negative n
	rb.Push(msg1)
	lastZero := rb.GetLastN(0)
	if lastZero != nil {
		t.Error("GetLastN(0) should return nil")
	}

	lastNegative := rb.GetLastN(-1)
	if lastNegative != nil {
		t.Error("GetLastN(-1) should return nil")
	}
}

package main

import (
	"sync"
)

// RingBuffer implements a bounded circular buffer for message queuing
// Drops oldest messages when capacity is exceeded (overflow handling)
type RingBuffer struct {
	buffer   []EventResponse
	head     int  // Points to the next write position
	tail     int  // Points to the oldest message
	size     int  // Current number of messages
	capacity int  // Maximum capacity
	full     bool // Whether buffer is at capacity
	mutex    sync.RWMutex
}

// NewRingBuffer creates a new ring buffer with specified capacity
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		buffer:   make([]EventResponse, capacity),
		capacity: capacity,
	}
}

// Push adds a new message to the buffer
// If at capacity, overwrites the oldest message
func (rb *RingBuffer) Push(message EventResponse) {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	rb.buffer[rb.head] = message
	rb.head = (rb.head + 1) % rb.capacity

	if rb.full {
		// Buffer is full, advance tail to drop oldest message
		rb.tail = (rb.tail + 1) % rb.capacity
	} else {
		// Buffer not full yet
		rb.size++
		if rb.size == rb.capacity {
			rb.full = true
		}
	}
}

// Pop removes and returns the oldest message
// Returns nil if buffer is empty
func (rb *RingBuffer) Pop() *EventResponse {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	if rb.size == 0 {
		return nil
	}

	message := rb.buffer[rb.tail]
	rb.tail = (rb.tail + 1) % rb.capacity
	rb.size--
	rb.full = false

	return &message
}

// PopAll returns all messages in chronological order and clears the buffer
func (rb *RingBuffer) PopAll() []EventResponse {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	if rb.size == 0 {
		return nil
	}

	messages := make([]EventResponse, rb.size)
	for i := 0; i < rb.size; i++ {
		messages[i] = rb.buffer[(rb.tail+i)%rb.capacity]
	}

	// Reset buffer
	rb.head = 0
	rb.tail = 0
	rb.size = 0
	rb.full = false

	return messages
}

// GetLastN returns the last N messages in chronological order without removing them
func (rb *RingBuffer) GetLastN(n int) []EventResponse {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()

	if rb.size == 0 || n <= 0 {
		return nil
	}

	// Determine how many messages to return
	count := n
	if count > rb.size {
		count = rb.size
	}

	messages := make([]EventResponse, count)

	// Calculate starting position (count messages back from head)
	start := rb.head - count
	if start < 0 {
		start += rb.capacity
	}

	for i := 0; i < count; i++ {
		messages[i] = rb.buffer[(start+i)%rb.capacity]
	}

	return messages
}

// Size returns the current number of messages in the buffer
func (rb *RingBuffer) Size() int {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()
	return rb.size
}

// IsFull returns true if the buffer is at capacity
func (rb *RingBuffer) IsFull() bool {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()
	return rb.full
}

// Clear empties the buffer
func (rb *RingBuffer) Clear() {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	rb.head = 0
	rb.tail = 0
	rb.size = 0
	rb.full = false
}

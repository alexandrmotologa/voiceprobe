package dsp

import (
	"sync"
)

// RingBuffer provides a concurrency-safe circular buffer for streaming audio samples.
type RingBuffer struct {
	mu       sync.RWMutex
	data     []float32
	capacity int
	writeIdx int
	totalIn  int64
}

// NewRingBuffer allocates a circular buffer with the specified capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 8192
	}
	return &RingBuffer{
		data:     make([]float32, capacity),
		capacity: capacity,
	}
}

// Write appends incoming audio samples to the ring buffer.
func (rb *RingBuffer) Write(samples []float32) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for _, s := range samples {
		rb.data[rb.writeIdx] = s
		rb.writeIdx = (rb.writeIdx + 1) % rb.capacity
		rb.totalIn++
	}
}

// Available returns how many samples are currently buffered, capped by capacity.
func (rb *RingBuffer) Available() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.totalIn < int64(rb.capacity) {
		return int(rb.totalIn)
	}
	return rb.capacity
}

// ReadLatest copies the most recent `count` samples into dst without consuming them.
// If count exceeds buffered samples, only available samples are read, aligned to the tail.
func (rb *RingBuffer) ReadLatest(dst []float32) int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	avail := int(rb.totalIn)
	if avail > rb.capacity {
		avail = rb.capacity
	}

	count := len(dst)
	if count > avail {
		count = avail
	}
	if count == 0 {
		return 0
	}

	startIdx := (rb.writeIdx - count + rb.capacity) % rb.capacity
	for i := 0; i < count; i++ {
		dst[i] = rb.data[(startIdx+i)%rb.capacity]
	}
	return count
}

// Reset clears the buffer pointer.
func (rb *RingBuffer) Reset() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.writeIdx = 0
	rb.totalIn = 0
	for i := range rb.data {
		rb.data[i] = 0
	}
}

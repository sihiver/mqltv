package streaming

import (
	"sync"
)

// RingBuffer is a circular buffer for stream data
type RingBuffer struct {
	buffer []byte
	size   int
	head   int
	tail   int
	mux    sync.RWMutex
}

// NewRingBuffer creates a new ring buffer
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		buffer: make([]byte, size),
		size:   size,
	}
}

// Write writes data to the ring buffer
func (rb *RingBuffer) Write(data []byte) int {
	rb.mux.Lock()
	defer rb.mux.Unlock()

	n := len(data)
	if n > rb.size {
		// If data is larger than buffer, only keep the last part
		data = data[n-rb.size:]
		n = rb.size
	}

	for i := 0; i < n; i++ {
		rb.buffer[rb.head] = data[i]
		rb.head = (rb.head + 1) % rb.size
		if rb.head == rb.tail {
			rb.tail = (rb.tail + 1) % rb.size
		}
	}

	return n
}

// Read reads data from the ring buffer
func (rb *RingBuffer) Read(data []byte) int {
	rb.mux.RLock()
	defer rb.mux.RUnlock()

	if rb.head == rb.tail {
		return 0
	}

	n := 0
	for rb.tail != rb.head && n < len(data) {
		data[n] = rb.buffer[rb.tail]
		rb.tail = (rb.tail + 1) % rb.size
		n++
	}

	return n
}

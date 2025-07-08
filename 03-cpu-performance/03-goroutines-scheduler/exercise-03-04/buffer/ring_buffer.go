package buffer

import (
	"sync/atomic"
	"unsafe"
)

// RingBuffer is a lock-free ring buffer implementation
type RingBuffer struct {
	buffer   []unsafe.Pointer
	capacity int64
	mask     int64
	head     int64
	tail     int64
}

// NewRingBuffer creates a new ring buffer with power-of-2 capacity
func NewRingBuffer(capacity int) *RingBuffer {
	// Ensure capacity is power of 2
	cap := int64(1)
	for cap < int64(capacity) {
		cap <<= 1
	}
	
	return &RingBuffer{
		buffer:   make([]unsafe.Pointer, cap),
		capacity: cap,
		mask:     cap - 1,
	}
}

// Send adds an item to the ring buffer (non-blocking)
func (rb *RingBuffer) Send(item interface{}) bool {
	for {
		tail := atomic.LoadInt64(&rb.tail)
		head := atomic.LoadInt64(&rb.head)
		
		// Check if buffer is full
		if tail-head >= rb.capacity {
			return false
		}
		
		// Try to claim the slot
		if atomic.CompareAndSwapInt64(&rb.tail, tail, tail+1) {
			// Store the item
			atomic.StorePointer(&rb.buffer[tail&rb.mask], unsafe.Pointer(&item))
			return true
		}
	}
}

// Receive gets an item from the ring buffer (non-blocking)
func (rb *RingBuffer) Receive() (interface{}, bool) {
	for {
		head := atomic.LoadInt64(&rb.head)
		tail := atomic.LoadInt64(&rb.tail)
		
		// Check if buffer is empty
		if head >= tail {
			return nil, false
		}
		
		// Try to claim the slot
		if atomic.CompareAndSwapInt64(&rb.head, head, head+1) {
			// Load the item
			ptr := atomic.LoadPointer(&rb.buffer[head&rb.mask])
			if ptr == nil {
				continue
			}
			
			item := *(*interface{})(ptr)
			// Clear the slot
			atomic.StorePointer(&rb.buffer[head&rb.mask], nil)
			return item, true
		}
	}
}

// Size returns the current number of items in the buffer
func (rb *RingBuffer) Size() int {
	tail := atomic.LoadInt64(&rb.tail)
	head := atomic.LoadInt64(&rb.head)
	return int(tail - head)
}

// Capacity returns the buffer capacity
func (rb *RingBuffer) Capacity() int {
	return int(rb.capacity)
}

// IsEmpty checks if the buffer is empty
func (rb *RingBuffer) IsEmpty() bool {
	return atomic.LoadInt64(&rb.head) >= atomic.LoadInt64(&rb.tail)
}

// IsFull checks if the buffer is full
func (rb *RingBuffer) IsFull() bool {
	tail := atomic.LoadInt64(&rb.tail)
	head := atomic.LoadInt64(&rb.head)
	return tail-head >= rb.capacity
}

// BatchRingBuffer provides batch operations on ring buffer
type BatchRingBuffer struct {
	*RingBuffer
}

// NewBatchRingBuffer creates a new batch ring buffer
func NewBatchRingBuffer(capacity int) *BatchRingBuffer {
	return &BatchRingBuffer{
		RingBuffer: NewRingBuffer(capacity),
	}
}

// SendBatch sends multiple items at once
func (brb *BatchRingBuffer) SendBatch(items []interface{}) int {
	sent := 0
	for _, item := range items {
		if brb.Send(item) {
			sent++
		} else {
			break
		}
	}
	return sent
}

// ReceiveBatch receives multiple items at once
func (brb *BatchRingBuffer) ReceiveBatch(maxItems int) []interface{} {
	items := make([]interface{}, 0, maxItems)
	
	for i := 0; i < maxItems; i++ {
		if item, ok := brb.Receive(); ok {
			items = append(items, item)
		} else {
			break
		}
	}
	
	return items
}

// MultiProducerRingBuffer supports multiple producers
type MultiProducerRingBuffer struct {
	buffer     []unsafe.Pointer
	capacity   int64
	mask       int64
	head       int64
	tail       int64
	sequences  []int64
}

// NewMultiProducerRingBuffer creates a new multi-producer ring buffer
func NewMultiProducerRingBuffer(capacity int) *MultiProducerRingBuffer {
	// Ensure capacity is power of 2
	cap := int64(1)
	for cap < int64(capacity) {
		cap <<= 1
	}
	
	return &MultiProducerRingBuffer{
		buffer:    make([]unsafe.Pointer, cap),
		capacity:  cap,
		mask:      cap - 1,
		sequences: make([]int64, cap),
	}
}

// Send adds an item with multi-producer support
func (mprb *MultiProducerRingBuffer) Send(item interface{}) bool {
	for {
		tail := atomic.LoadInt64(&mprb.tail)
		head := atomic.LoadInt64(&mprb.head)
		
		// Check if buffer is full
		if tail-head >= mprb.capacity {
			return false
		}
		
		// Try to claim the slot
		if atomic.CompareAndSwapInt64(&mprb.tail, tail, tail+1) {
			index := tail & mprb.mask
			
			// Wait for the slot to be available
			for atomic.LoadInt64(&mprb.sequences[index]) != tail {
				// Spin wait
			}
			
			// Store the item
			atomic.StorePointer(&mprb.buffer[index], unsafe.Pointer(&item))
			
			// Update sequence
			atomic.StoreInt64(&mprb.sequences[index], tail+1)
			return true
		}
	}
}

// Receive gets an item with multi-producer support
func (mprb *MultiProducerRingBuffer) Receive() (interface{}, bool) {
	for {
		head := atomic.LoadInt64(&mprb.head)
		tail := atomic.LoadInt64(&mprb.tail)
		
		// Check if buffer is empty
		if head >= tail {
			return nil, false
		}
		
		index := head & mprb.mask
		
		// Wait for the item to be available
		if atomic.LoadInt64(&mprb.sequences[index]) <= head {
			return nil, false
		}
		
		// Try to claim the slot
		if atomic.CompareAndSwapInt64(&mprb.head, head, head+1) {
			// Load the item
			ptr := atomic.LoadPointer(&mprb.buffer[index])
			if ptr == nil {
				continue
			}
			
			item := *(*interface{})(ptr)
			
			// Clear the slot and update sequence
			atomic.StorePointer(&mprb.buffer[index], nil)
			atomic.StoreInt64(&mprb.sequences[index], head+mprb.capacity)
			
			return item, true
		}
	}
}

// Size returns the current number of items
func (mprb *MultiProducerRingBuffer) Size() int {
	tail := atomic.LoadInt64(&mprb.tail)
	head := atomic.LoadInt64(&mprb.head)
	return int(tail - head)
}

// Capacity returns the buffer capacity
func (mprb *MultiProducerRingBuffer) Capacity() int {
	return int(mprb.capacity)
}
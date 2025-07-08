package queue

import (
	"sync/atomic"
)

// BoundedQueueSlot represents a slot in the bounded queue
type BoundedQueueSlot struct {
	data     interface{}
	sequence uint64
	_        [14]uintptr // Cache line padding
}

// BoundedQueue implements a bounded lock-free queue using a ring buffer
type BoundedQueue struct {
	buffer   []BoundedQueueSlot
	mask     uint64
	enqPos   uint64 // Enqueue position
	_        [15]uintptr // Cache line padding
	deqPos   uint64 // Dequeue position
	_        [15]uintptr // Cache line padding
}

// NewBoundedQueue creates a new bounded queue with the specified capacity
// Capacity must be a power of 2
func NewBoundedQueue(capacity uint64) *BoundedQueue {
	if capacity == 0 || (capacity&(capacity-1)) != 0 {
		panic("Capacity must be a power of 2")
	}
	
	q := &BoundedQueue{
		buffer: make([]BoundedQueueSlot, capacity),
		mask:   capacity - 1,
	}
	
	// Initialize sequence numbers
	for i := uint64(0); i < capacity; i++ {
		atomic.StoreUint64(&q.buffer[i].sequence, i)
	}
	
	return q
}

// Enqueue adds an item to the queue
// Returns true if successful, false if queue is full
func (q *BoundedQueue) Enqueue(data interface{}) bool {
	for {
		pos := atomic.LoadUint64(&q.enqPos)
		slot := &q.buffer[pos&q.mask]
		seq := atomic.LoadUint64(&slot.sequence)
		diff := int64(seq) - int64(pos)
		
		if diff == 0 {
			// Slot is available for enqueue
			if atomic.CompareAndSwapUint64(&q.enqPos, pos, pos+1) {
				// Successfully claimed the slot
				slot.data = data
				atomic.StoreUint64(&slot.sequence, pos+1)
				return true
			}
		} else if diff < 0 {
			// Queue is full
			return false
		}
		// else: slot is not ready yet, retry
	}
}

// Dequeue removes and returns an item from the queue
// Returns (data, true) if successful, (nil, false) if queue is empty
func (q *BoundedQueue) Dequeue() (interface{}, bool) {
	for {
		pos := atomic.LoadUint64(&q.deqPos)
		slot := &q.buffer[pos&q.mask]
		seq := atomic.LoadUint64(&slot.sequence)
		diff := int64(seq) - int64(pos+1)
		
		if diff == 0 {
			// Slot has data ready for dequeue
			if atomic.CompareAndSwapUint64(&q.deqPos, pos, pos+1) {
				// Successfully claimed the slot
				data := slot.data
				slot.data = nil // Clear reference to help GC
				atomic.StoreUint64(&slot.sequence, pos+q.mask+1)
				return data, true
			}
		} else if diff < 0 {
			// Queue is empty
			return nil, false
		}
		// else: slot is not ready yet, retry
	}
}

// TryEnqueue attempts to enqueue without blocking
// Returns true if successful, false if queue is full
func (q *BoundedQueue) TryEnqueue(data interface{}) bool {
	pos := atomic.LoadUint64(&q.enqPos)
	slot := &q.buffer[pos&q.mask]
	seq := atomic.LoadUint64(&slot.sequence)
	diff := int64(seq) - int64(pos)
	
	if diff == 0 {
		if atomic.CompareAndSwapUint64(&q.enqPos, pos, pos+1) {
			slot.data = data
			atomic.StoreUint64(&slot.sequence, pos+1)
			return true
		}
	}
	
	return false
}

// TryDequeue attempts to dequeue without blocking
// Returns (data, true) if successful, (nil, false) if queue is empty
func (q *BoundedQueue) TryDequeue() (interface{}, bool) {
	pos := atomic.LoadUint64(&q.deqPos)
	slot := &q.buffer[pos&q.mask]
	seq := atomic.LoadUint64(&slot.sequence)
	diff := int64(seq) - int64(pos+1)
	
	if diff == 0 {
		if atomic.CompareAndSwapUint64(&q.deqPos, pos, pos+1) {
			data := slot.data
			slot.data = nil
			atomic.StoreUint64(&slot.sequence, pos+q.mask+1)
			return data, true
		}
	}
	
	return nil, false
}

// IsEmpty checks if the queue is empty
func (q *BoundedQueue) IsEmpty() bool {
	enqPos := atomic.LoadUint64(&q.enqPos)
	deqPos := atomic.LoadUint64(&q.deqPos)
	return enqPos == deqPos
}

// IsFull checks if the queue is full
func (q *BoundedQueue) IsFull() bool {
	enqPos := atomic.LoadUint64(&q.enqPos)
	deqPos := atomic.LoadUint64(&q.deqPos)
	return enqPos-deqPos == uint64(len(q.buffer))
}

// Size returns the current size of the queue
func (q *BoundedQueue) Size() int {
	enqPos := atomic.LoadUint64(&q.enqPos)
	deqPos := atomic.LoadUint64(&q.deqPos)
	return int(enqPos - deqPos)
}

// Capacity returns the capacity of the queue
func (q *BoundedQueue) Capacity() uint64 {
	return uint64(len(q.buffer))
}

// MultiEnqueue attempts to enqueue multiple items atomically
// Returns the number of items successfully enqueued
func (q *BoundedQueue) MultiEnqueue(items []interface{}) int {
	count := 0
	for _, item := range items {
		if !q.Enqueue(item) {
			break
		}
		count++
	}
	return count
}

// MultiDequeue attempts to dequeue multiple items
// Returns a slice of dequeued items
func (q *BoundedQueue) MultiDequeue(maxItems int) []interface{} {
	result := make([]interface{}, 0, maxItems)
	for i := 0; i < maxItems; i++ {
		if item, ok := q.Dequeue(); ok {
			result = append(result, item)
		} else {
			break
		}
	}
	return result
}

// Stats returns queue statistics
func (q *BoundedQueue) Stats() (size, capacity uint64, utilization float64) {
	size = uint64(q.Size())
	capacity = q.Capacity()
	utilization = float64(size) / float64(capacity)
	return size, capacity, utilization
}

// Reset clears the queue (not thread-safe)
func (q *BoundedQueue) Reset() {
	atomic.StoreUint64(&q.enqPos, 0)
	atomic.StoreUint64(&q.deqPos, 0)
	
	for i := uint64(0); i < uint64(len(q.buffer)); i++ {
		q.buffer[i].data = nil
		atomic.StoreUint64(&q.buffer[i].sequence, i)
	}
}

// BoundedQueueWithBackpressure extends BoundedQueue with backpressure handling
type BoundedQueueWithBackpressure struct {
	*BoundedQueue
	backpressureThreshold float64
	backpressureCallback  func(utilization float64)
}

// NewBoundedQueueWithBackpressure creates a bounded queue with backpressure handling
func NewBoundedQueueWithBackpressure(capacity uint64, threshold float64, callback func(float64)) *BoundedQueueWithBackpressure {
	return &BoundedQueueWithBackpressure{
		BoundedQueue:          NewBoundedQueue(capacity),
		backpressureThreshold: threshold,
		backpressureCallback:  callback,
	}
}

// EnqueueWithBackpressure enqueues with backpressure monitoring
func (q *BoundedQueueWithBackpressure) EnqueueWithBackpressure(data interface{}) bool {
	success := q.BoundedQueue.Enqueue(data)
	
	if success && q.backpressureCallback != nil {
		_, _, utilization := q.Stats()
		if utilization >= q.backpressureThreshold {
			q.backpressureCallback(utilization)
		}
	}
	
	return success
}
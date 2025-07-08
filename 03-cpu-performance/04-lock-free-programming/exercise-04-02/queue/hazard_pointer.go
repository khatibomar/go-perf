package queue

import (
	"sync/atomic"
	"unsafe"

	"exercise-04-02/memory"
)

// HPNode represents a queue node for hazard pointer queue
type HPNode struct {
	data interface{}
	next unsafe.Pointer // *HPNode
	_    [14]uintptr   // Cache line padding
}

// HazardPointerQueue implements a lock-free queue using hazard pointers
type HazardPointerQueue struct {
	head unsafe.Pointer // *HPNode
	tail unsafe.Pointer // *HPNode
	hpm  *memory.HazardPointerManager
	_    [12]uintptr // Cache line padding
}

// NewHazardPointerQueue creates a new hazard pointer queue
func NewHazardPointerQueue() *HazardPointerQueue {
	// Create dummy node
	dummy := &HPNode{}
	q := &HazardPointerQueue{}
	atomic.StorePointer(&q.head, unsafe.Pointer(dummy))
	atomic.StorePointer(&q.tail, unsafe.Pointer(dummy))
	
	// Initialize hazard pointer manager with max pointers
	q.hpm = memory.NewHazardPointerManager(8)
	
	return q
}

// Enqueue adds an item to the tail of the queue
func (q *HazardPointerQueue) Enqueue(data interface{}) bool {
	newNode := &HPNode{data: data}
	newNodePtr := unsafe.Pointer(newNode)

	for {
		// Acquire hazard pointer for tail
		tailHP := q.hpm.AcquireHazardPointer(nil)
		tail := atomic.LoadPointer(&q.tail)
		tailHP.SetPointer(tail)
		
		// Verify tail hasn't changed
		if tail != atomic.LoadPointer(&q.tail) {
			q.hpm.ReleaseHazardPointer(tailHP)
			continue
		}
		
		tailNode := (*HPNode)(tail)
		
		// Acquire hazard pointer for next
		nextHP := q.hpm.AcquireHazardPointer(nil)
		next := atomic.LoadPointer(&tailNode.next)
		nextHP.SetPointer(next)
		
		// Verify next hasn't changed
		if next != atomic.LoadPointer(&tailNode.next) {
			q.hpm.ReleaseHazardPointer(tailHP)
			q.hpm.ReleaseHazardPointer(nextHP)
			continue
		}

		// Check consistency
		if tail == atomic.LoadPointer(&q.tail) {
			if next == nil {
				// Tail is pointing to last node, try to link new node
				if atomic.CompareAndSwapPointer(&tailNode.next, nil, newNodePtr) {
					q.hpm.ReleaseHazardPointer(tailHP)
					q.hpm.ReleaseHazardPointer(nextHP)
					break // Successfully linked
				}
			} else {
				// Tail is lagging, try to advance it
				atomic.CompareAndSwapPointer(&q.tail, tail, next)
			}
		}
		
		q.hpm.ReleaseHazardPointer(tailHP)
		q.hpm.ReleaseHazardPointer(nextHP)
	}

	// Try to advance tail pointer
	tail := atomic.LoadPointer(&q.tail)
	atomic.CompareAndSwapPointer(&q.tail, tail, newNodePtr)
	return true
}

// Dequeue removes and returns an item from the head of the queue
func (q *HazardPointerQueue) Dequeue() (interface{}, bool) {
	for {
		// Acquire hazard pointer for head
		headHP := q.hpm.AcquireHazardPointer(nil)
		head := atomic.LoadPointer(&q.head)
		headHP.SetPointer(head)
		
		// Verify head hasn't changed
		if head != atomic.LoadPointer(&q.head) {
			q.hpm.ReleaseHazardPointer(headHP)
			continue
		}
		
		// Acquire hazard pointer for tail
		tailHP := q.hpm.AcquireHazardPointer(nil)
		tail := atomic.LoadPointer(&q.tail)
		tailHP.SetPointer(tail)
		
		headNode := (*HPNode)(head)
		
		// Acquire hazard pointer for next
		nextHP := q.hpm.AcquireHazardPointer(nil)
		next := atomic.LoadPointer(&headNode.next)
		nextHP.SetPointer(next)
		
		// Verify next hasn't changed
		if next != atomic.LoadPointer(&headNode.next) {
			q.hpm.ReleaseHazardPointer(headHP)
			q.hpm.ReleaseHazardPointer(tailHP)
			q.hpm.ReleaseHazardPointer(nextHP)
			continue
		}

		// Check consistency
		if head == atomic.LoadPointer(&q.head) {
			if head == tail {
				if next == nil {
					// Queue is empty
					q.hpm.ReleaseHazardPointer(headHP)
					q.hpm.ReleaseHazardPointer(tailHP)
					q.hpm.ReleaseHazardPointer(nextHP)
					return nil, false
				}
				// Tail is lagging, try to advance it
				atomic.CompareAndSwapPointer(&q.tail, tail, next)
			} else {
				if next == nil {
					// Inconsistent state, retry
					q.hpm.ReleaseHazardPointer(headHP)
					q.hpm.ReleaseHazardPointer(tailHP)
					q.hpm.ReleaseHazardPointer(nextHP)
					continue
				}

				nextNode := (*HPNode)(next)
				data := nextNode.data

				// Try to move head to next node
				if atomic.CompareAndSwapPointer(&q.head, head, next) {
					// Successfully dequeued, retire the old head
					q.hpm.RetirePointer(head)
					q.hpm.ReleaseHazardPointer(headHP)
					q.hpm.ReleaseHazardPointer(tailHP)
					q.hpm.ReleaseHazardPointer(nextHP)
					return data, true
				}
			}
		}
		
		q.hpm.ReleaseHazardPointer(headHP)
		q.hpm.ReleaseHazardPointer(tailHP)
		q.hpm.ReleaseHazardPointer(nextHP)
	}
}

// IsEmpty checks if the queue is empty
func (q *HazardPointerQueue) IsEmpty() bool {
	headHP := q.hpm.AcquireHazardPointer(nil)
	head := atomic.LoadPointer(&q.head)
	headHP.SetPointer(head)
	defer q.hpm.ReleaseHazardPointer(headHP)
	
	tailHP := q.hpm.AcquireHazardPointer(nil)
	tail := atomic.LoadPointer(&q.tail)
	tailHP.SetPointer(tail)
	defer q.hpm.ReleaseHazardPointer(tailHP)
	
	headNode := (*HPNode)(head)
	nextHP := q.hpm.AcquireHazardPointer(nil)
	next := atomic.LoadPointer(&headNode.next)
	nextHP.SetPointer(next)
	defer q.hpm.ReleaseHazardPointer(nextHP)
	
	return head == tail && next == nil
}

// Size returns approximate size of the queue (not linearizable)
func (q *HazardPointerQueue) Size() int {
	count := 0
	currentHP := q.hpm.AcquireHazardPointer(nil)
	current := atomic.LoadPointer(&q.head)
	currentHP.SetPointer(current)
	defer q.hpm.ReleaseHazardPointer(currentHP)
	
	currentNode := (*HPNode)(current)

	for {
		nextHP := q.hpm.AcquireHazardPointer(nil)
		next := atomic.LoadPointer(&currentNode.next)
		nextHP.SetPointer(next)
		
		if next == nil {
			q.hpm.ReleaseHazardPointer(nextHP)
			break
		}
		
		count++
		current = next
		currentNode = (*HPNode)(current)
		
		// Update current hazard pointer
		currentHP.SetPointer(current)
		q.hpm.ReleaseHazardPointer(nextHP)
	}

	return count
}

// Cleanup forces cleanup of retired nodes
func (q *HazardPointerQueue) Cleanup() {
	q.hpm.ForceCleanup()
}

// Stats returns hazard pointer statistics
func (q *HazardPointerQueue) Stats() (activePointers, retiredPointers int) {
	return q.hpm.Stats()
}
package queue

import (
	"sync/atomic"
	"unsafe"
)

// VersionedPointer combines a pointer with a version counter
type VersionedPointer struct {
	ptr     unsafe.Pointer
	version uint64
}

// VersionedNode represents a queue node with version information
type VersionedNode struct {
	data interface{}
	next VersionedPointer
	_    [6]uintptr // Cache line padding
}

// VersionedQueue implements a lock-free queue using versioned pointers
type VersionedQueue struct {
	head VersionedPointer
	tail VersionedPointer
	_    [12]uintptr // Cache line padding
}

// NewVersionedQueue creates a new versioned queue
func NewVersionedQueue() *VersionedQueue {
	// Create dummy node
	dummy := &VersionedNode{}
	q := &VersionedQueue{}
	q.head = VersionedPointer{ptr: unsafe.Pointer(dummy), version: 0}
	q.tail = VersionedPointer{ptr: unsafe.Pointer(dummy), version: 0}
	return q
}

// loadVersionedPointer atomically loads a versioned pointer
func (q *VersionedQueue) loadVersionedPointer(vp *VersionedPointer) VersionedPointer {
	for {
		ptr := atomic.LoadPointer(&vp.ptr)
		version := atomic.LoadUint64(&vp.version)
		// Double-check for consistency
		if ptr == atomic.LoadPointer(&vp.ptr) {
			return VersionedPointer{ptr: ptr, version: version}
		}
	}
}

// compareAndSwapVersionedPointer atomically compares and swaps a versioned pointer
func (q *VersionedQueue) compareAndSwapVersionedPointer(vp *VersionedPointer, old, new VersionedPointer) bool {
	// First try to CAS the version (this acts as a lock)
	if !atomic.CompareAndSwapUint64(&vp.version, old.version, new.version) {
		return false
	}
	
	// Then CAS the pointer
	if atomic.CompareAndSwapPointer(&vp.ptr, old.ptr, new.ptr) {
		return true
	}
	
	// If pointer CAS failed, restore the old version
	atomic.StoreUint64(&vp.version, old.version)
	return false
}

// Enqueue adds an item to the tail of the queue
func (q *VersionedQueue) Enqueue(data interface{}) bool {
	newNode := &VersionedNode{data: data}
	newNodePtr := unsafe.Pointer(newNode)

	for {
		tail := q.loadVersionedPointer(&q.tail)
		tailNode := (*VersionedNode)(tail.ptr)
		next := q.loadVersionedPointer(&tailNode.next)

		// Check consistency
		if tail.ptr == q.loadVersionedPointer(&q.tail).ptr && tail.version == q.loadVersionedPointer(&q.tail).version {
			if next.ptr == nil {
				// Tail is pointing to last node, try to link new node
				newNext := VersionedPointer{ptr: newNodePtr, version: next.version + 1}
				if q.compareAndSwapVersionedPointer(&tailNode.next, next, newNext) {
					break // Successfully linked
				}
			} else {
				// Tail is lagging, try to advance it
				newTail := VersionedPointer{ptr: next.ptr, version: tail.version + 1}
				q.compareAndSwapVersionedPointer(&q.tail, tail, newTail)
			}
		}
	}

	// Try to advance tail pointer
	tail := q.loadVersionedPointer(&q.tail)
	newTail := VersionedPointer{ptr: newNodePtr, version: tail.version + 1}
	q.compareAndSwapVersionedPointer(&q.tail, tail, newTail)
	return true
}

// Dequeue removes and returns an item from the head of the queue
func (q *VersionedQueue) Dequeue() (interface{}, bool) {
	for {
		head := q.loadVersionedPointer(&q.head)
		tail := q.loadVersionedPointer(&q.tail)
		headNode := (*VersionedNode)(head.ptr)
		next := q.loadVersionedPointer(&headNode.next)

		// Check consistency
		if head.ptr == q.loadVersionedPointer(&q.head).ptr && head.version == q.loadVersionedPointer(&q.head).version {
			if head.ptr == tail.ptr {
				if next.ptr == nil {
					// Queue is empty
					return nil, false
				}
				// Tail is lagging, try to advance it
				newTail := VersionedPointer{ptr: next.ptr, version: tail.version + 1}
				q.compareAndSwapVersionedPointer(&q.tail, tail, newTail)
			} else {
				if next.ptr == nil {
					// Inconsistent state, retry
					continue
				}

				nextNode := (*VersionedNode)(next.ptr)
				data := nextNode.data

				// Try to move head to next node
				newHead := VersionedPointer{ptr: next.ptr, version: head.version + 1}
				if q.compareAndSwapVersionedPointer(&q.head, head, newHead) {
					// Successfully dequeued
					return data, true
				}
			}
		}
	}
}

// IsEmpty checks if the queue is empty
func (q *VersionedQueue) IsEmpty() bool {
	head := q.loadVersionedPointer(&q.head)
	tail := q.loadVersionedPointer(&q.tail)
	headNode := (*VersionedNode)(head.ptr)
	next := q.loadVersionedPointer(&headNode.next)
	return head.ptr == tail.ptr && next.ptr == nil
}

// Size returns approximate size of the queue (not linearizable)
func (q *VersionedQueue) Size() int {
	count := 0
	current := q.loadVersionedPointer(&q.head)
	currentNode := (*VersionedNode)(current.ptr)

	for {
		next := q.loadVersionedPointer(&currentNode.next)
		if next.ptr == nil {
			break
		}
		count++
		current = next
		currentNode = (*VersionedNode)(current.ptr)
	}

	return count
}
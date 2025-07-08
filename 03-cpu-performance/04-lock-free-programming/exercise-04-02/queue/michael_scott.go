package queue

import (
	"sync/atomic"
	"unsafe"
)

// Node represents a queue node
type Node struct {
	data interface{}
	next unsafe.Pointer // *Node
}

// MichaelScottQueue implements the classic Michael & Scott lock-free queue
type MichaelScottQueue struct {
	head unsafe.Pointer // *Node
	tail unsafe.Pointer // *Node
	_    [14]uintptr    // Cache line padding
}

// NewMichaelScottQueue creates a new Michael & Scott queue
func NewMichaelScottQueue() *MichaelScottQueue {
	// Create dummy node
	dummy := &Node{}
	q := &MichaelScottQueue{}
	atomic.StorePointer(&q.head, unsafe.Pointer(dummy))
	atomic.StorePointer(&q.tail, unsafe.Pointer(dummy))
	return q
}

// Enqueue adds an item to the tail of the queue
func (q *MichaelScottQueue) Enqueue(data interface{}) bool {
	newNode := &Node{data: data}
	newNodePtr := unsafe.Pointer(newNode)

	var tail unsafe.Pointer
	for {
		tail = atomic.LoadPointer(&q.tail)
		tailNode := (*Node)(tail)
		next := atomic.LoadPointer(&tailNode.next)

		// Check consistency
		if tail == atomic.LoadPointer(&q.tail) {
			if next == nil {
				// Tail is pointing to last node, try to link new node
				if atomic.CompareAndSwapPointer(&tailNode.next, nil, newNodePtr) {
					break // Successfully linked
				}
			} else {
				// Tail is lagging, try to advance it
				atomic.CompareAndSwapPointer(&q.tail, tail, next)
			}
		}
	}

	// Try to advance tail pointer
	atomic.CompareAndSwapPointer(&q.tail, tail, newNodePtr)
	return true
}

// Dequeue removes and returns an item from the head of the queue
func (q *MichaelScottQueue) Dequeue() (interface{}, bool) {
	for {
		head := atomic.LoadPointer(&q.head)
		tail := atomic.LoadPointer(&q.tail)
		headNode := (*Node)(head)
		next := atomic.LoadPointer(&headNode.next)

		// Check consistency
		if head == atomic.LoadPointer(&q.head) {
			if head == tail {
				if next == nil {
					// Queue is empty
					return nil, false
				}
				// Tail is lagging, try to advance it
				atomic.CompareAndSwapPointer(&q.tail, tail, next)
			} else {
				if next == nil {
					// Inconsistent state, retry
					continue
				}

				nextNode := (*Node)(next)
				data := nextNode.data

				// Try to move head to next node
				if atomic.CompareAndSwapPointer(&q.head, head, next) {
					// Successfully dequeued
					return data, true
				}
			}
		}
	}
}

// IsEmpty checks if the queue is empty
func (q *MichaelScottQueue) IsEmpty() bool {
	head := atomic.LoadPointer(&q.head)
	tail := atomic.LoadPointer(&q.tail)
	headNode := (*Node)(head)
	next := atomic.LoadPointer(&headNode.next)
	return head == tail && next == nil
}

// Size returns approximate size of the queue (not linearizable)
func (q *MichaelScottQueue) Size() int {
	count := 0
	current := atomic.LoadPointer(&q.head)
	currentNode := (*Node)(current)

	for {
		next := atomic.LoadPointer(&currentNode.next)
		if next == nil {
			break
		}
		count++
		current = next
		currentNode = (*Node)(current)
	}

	return count
}
package list

import (
	"sync/atomic"
	"unsafe"
)

// OptimisticListNode represents a node in the optimistic list
type OptimisticListNode struct {
	key    int64
	value  interface{}
	next   unsafe.Pointer // *OptimisticListNode
	marked int32          // Deletion mark
}

// OptimisticList uses optimistic synchronization
type OptimisticList struct {
	head *OptimisticListNode
	tail *OptimisticListNode
}

func NewOptimisticList() *OptimisticList {
	head := &OptimisticListNode{key: -1}
	tail := &OptimisticListNode{key: 1<<63 - 1}
	head.next = unsafe.Pointer(tail)
	
	return &OptimisticList{
		head: head,
		tail: tail,
	}
}

func (l *OptimisticList) validate(pred, curr *OptimisticListNode) bool {
	return atomic.LoadInt32(&pred.marked) == 0 &&
		atomic.LoadInt32(&curr.marked) == 0 &&
		atomic.LoadPointer(&pred.next) == unsafe.Pointer(curr)
}

func (l *OptimisticList) optimisticSearch(key int64) (*OptimisticListNode, *OptimisticListNode) {
	pred := l.head
	curr := (*OptimisticListNode)(atomic.LoadPointer(&pred.next))
	
	for curr.key < key {
		pred = curr
		curr = (*OptimisticListNode)(atomic.LoadPointer(&curr.next))
	}
	
	return pred, curr
}

func (l *OptimisticList) Insert(key int64, value interface{}) bool {
	for {
		pred, curr := l.optimisticSearch(key)
		
		if curr.key == key {
			return false
		}
		
		newNode := &OptimisticListNode{
			key:   key,
			value: value,
			next:  unsafe.Pointer(curr),
		}
		
		// Validate that pred and curr are still valid
		if l.validate(pred, curr) {
			if atomic.CompareAndSwapPointer(&pred.next, unsafe.Pointer(curr), unsafe.Pointer(newNode)) {
				return true
			}
		}
	}
}

func (l *OptimisticList) Delete(key int64) bool {
	for {
		pred, curr := l.optimisticSearch(key)
		
		if curr.key != key {
			return false
		}
		
		// Validate that pred and curr are still valid
		if l.validate(pred, curr) {
			// Mark node for deletion
			if atomic.CompareAndSwapInt32(&curr.marked, 0, 1) {
				next := (*OptimisticListNode)(atomic.LoadPointer(&curr.next))
				// Try to physically remove
				atomic.CompareAndSwapPointer(&pred.next, unsafe.Pointer(curr), unsafe.Pointer(next))
				return true
			}
		}
	}
}

func (l *OptimisticList) Contains(key int64) bool {
	pred, curr := l.optimisticSearch(key)
	
	// Validate the search result
	if l.validate(pred, curr) {
		return curr.key == key && atomic.LoadInt32(&curr.marked) == 0
	}
	
	// If validation failed, retry
	return l.Contains(key)
}

func (l *OptimisticList) Get(key int64) (interface{}, bool) {
	pred, curr := l.optimisticSearch(key)
	
	// Validate the search result
	if l.validate(pred, curr) {
		if curr.key == key && atomic.LoadInt32(&curr.marked) == 0 {
			return curr.value, true
		}
		return nil, false
	}
	
	// If validation failed, retry
	return l.Get(key)
}

func (l *OptimisticList) Update(key int64, value interface{}) bool {
	for {
		pred, curr := l.optimisticSearch(key)
		
		if curr.key != key {
			return false
		}
		
		// Validate that pred and curr are still valid
		if l.validate(pred, curr) {
			if atomic.LoadInt32(&curr.marked) == 0 {
				curr.value = value
				return true
			}
			return false
		}
	}
}

func (l *OptimisticList) Size() int {
	count := 0
	curr := (*OptimisticListNode)(atomic.LoadPointer(&l.head.next))
	
	for curr != l.tail {
		if atomic.LoadInt32(&curr.marked) == 0 {
			count++
		}
		curr = (*OptimisticListNode)(atomic.LoadPointer(&curr.next))
	}
	
	return count
}

func (l *OptimisticList) IsEmpty() bool {
	curr := (*OptimisticListNode)(atomic.LoadPointer(&l.head.next))
	return curr == l.tail
}

// Cleanup removes marked nodes from the list
func (l *OptimisticList) Cleanup() int {
	cleanedCount := 0
	pred := l.head
	curr := (*OptimisticListNode)(atomic.LoadPointer(&pred.next))
	
	for curr != l.tail {
		next := (*OptimisticListNode)(atomic.LoadPointer(&curr.next))
		
		if atomic.LoadInt32(&curr.marked) == 1 {
			if atomic.CompareAndSwapPointer(&pred.next, unsafe.Pointer(curr), unsafe.Pointer(next)) {
				cleanedCount++
			}
		} else {
			pred = curr
		}
		
		curr = next
	}
	
	return cleanedCount
}
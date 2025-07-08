package stack

import (
	"sync/atomic"
	"unsafe"
)

// VersionedStackNode represents a node with explicit versioning
type VersionedStackNode struct {
	data    interface{}
	next    unsafe.Pointer // *VersionedStackNode
	version int64
	deleted int32 // 0 = active, 1 = deleted
}

// VersionedStack implements version-based stack with explicit version tracking
type VersionedStack struct {
	head           unsafe.Pointer // *VersionedStackNode
	globalVersion  int64
	operationCount int64
}

func NewVersionedStack() *VersionedStack {
	return &VersionedStack{}
}

func (s *VersionedStack) Push(data interface{}) {
	newNode := &VersionedStackNode{
		data:    data,
		version: atomic.AddInt64(&s.globalVersion, 1),
	}
	
	atomic.AddInt64(&s.operationCount, 1)
	
	for {
		head := atomic.LoadPointer(&s.head)
		newNode.next = head
		
		if atomic.CompareAndSwapPointer(&s.head, head, unsafe.Pointer(newNode)) {
			return
		}
	}
}

func (s *VersionedStack) Pop() (interface{}, bool) {
	atomic.AddInt64(&s.operationCount, 1)
	
	for {
		head := atomic.LoadPointer(&s.head)
		if head == nil {
			return nil, false
		}
		
		headNode := (*VersionedStackNode)(head)
		
		// Check if node is already deleted
		if atomic.LoadInt32(&headNode.deleted) == 1 {
			continue
		}
		
		next := atomic.LoadPointer(&headNode.next)
		
		// Mark as deleted first
		if !atomic.CompareAndSwapInt32(&headNode.deleted, 0, 1) {
			continue
		}
		
		// Then try to update head
		if atomic.CompareAndSwapPointer(&s.head, head, next) {
			return headNode.data, true
		} else {
			// Rollback deletion mark if head update failed
			atomic.StoreInt32(&headNode.deleted, 0)
		}
	}
}

func (s *VersionedStack) IsEmpty() bool {
	head := atomic.LoadPointer(&s.head)
	if head == nil {
		return true
	}
	
	// Check if head is deleted
	headNode := (*VersionedStackNode)(head)
	return atomic.LoadInt32(&headNode.deleted) == 1
}

func (s *VersionedStack) Size() int {
	count := 0
	current := atomic.LoadPointer(&s.head)
	
	for current != nil {
		currentNode := (*VersionedStackNode)(current)
		if atomic.LoadInt32(&currentNode.deleted) == 0 {
			count++
		}
		current = atomic.LoadPointer(&currentNode.next)
	}
	
	return count
}

func (s *VersionedStack) Peek() (interface{}, bool) {
	head := atomic.LoadPointer(&s.head)
	if head == nil {
		return nil, false
	}
	
	headNode := (*VersionedStackNode)(head)
	if atomic.LoadInt32(&headNode.deleted) == 1 {
		return nil, false
	}
	
	return headNode.data, true
}

// GetGlobalVersion returns the current global version
func (s *VersionedStack) GetGlobalVersion() int64 {
	return atomic.LoadInt64(&s.globalVersion)
}

// GetOperationCount returns the total number of operations
func (s *VersionedStack) GetOperationCount() int64 {
	return atomic.LoadInt64(&s.operationCount)
}

// Cleanup removes deleted nodes from the stack
func (s *VersionedStack) Cleanup() int {
	cleanedCount := 0
	current := atomic.LoadPointer(&s.head)
	var prev *VersionedStackNode
	
	for current != nil {
		currentNode := (*VersionedStackNode)(current)
		next := atomic.LoadPointer(&currentNode.next)
		
		if atomic.LoadInt32(&currentNode.deleted) == 1 {
			if prev != nil {
				atomic.StorePointer(&prev.next, next)
			} else {
				atomic.StorePointer(&s.head, next)
			}
			cleanedCount++
		} else {
			prev = currentNode
		}
		
		current = next
	}
	
	return cleanedCount
}
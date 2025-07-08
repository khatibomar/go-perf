package stack

import (
	"sync/atomic"
	"unsafe"
)

// VersionedNode represents a stack node with version for ABA safety
type VersionedNode struct {
	data    interface{}
	next    unsafe.Pointer // *VersionedNode
	version int64
}

// ABASafeStack implements ABA-resistant stack
type ABASafeStack struct {
	head    unsafe.Pointer // *VersionedNode
	version int64
}

func NewABASafeStack() *ABASafeStack {
	return &ABASafeStack{}
}

func (s *ABASafeStack) Push(data interface{}) {
	newNode := &VersionedNode{
		data:    data,
		version: atomic.AddInt64(&s.version, 1),
	}
	
	for {
		head := atomic.LoadPointer(&s.head)
		newNode.next = head
		
		if atomic.CompareAndSwapPointer(&s.head, head, unsafe.Pointer(newNode)) {
			return
		}
	}
}

func (s *ABASafeStack) Pop() (interface{}, bool) {
	for {
		head := atomic.LoadPointer(&s.head)
		if head == nil {
			return nil, false
		}
		
		headNode := (*VersionedNode)(head)
		next := atomic.LoadPointer(&headNode.next)
		
		// Verify version hasn't changed (ABA protection)
		if atomic.LoadPointer(&s.head) != head {
			continue
		}
		
		if atomic.CompareAndSwapPointer(&s.head, head, next) {
			return headNode.data, true
		}
	}
}

func (s *ABASafeStack) IsEmpty() bool {
	return atomic.LoadPointer(&s.head) == nil
}

func (s *ABASafeStack) Size() int {
	count := 0
	current := atomic.LoadPointer(&s.head)
	
	for current != nil {
		count++
		currentNode := (*VersionedNode)(current)
		current = atomic.LoadPointer(&currentNode.next)
	}
	
	return count
}

func (s *ABASafeStack) Peek() (interface{}, bool) {
	head := atomic.LoadPointer(&s.head)
	if head == nil {
		return nil, false
	}
	
	headNode := (*VersionedNode)(head)
	return headNode.data, true
}

// GetVersion returns the current version of the stack
func (s *ABASafeStack) GetVersion() int64 {
	return atomic.LoadInt64(&s.version)
}

// ValidateNode checks if a node is still valid (not affected by ABA)
func (s *ABASafeStack) ValidateNode(node *VersionedNode, expectedVersion int64) bool {
	return atomic.LoadInt64(&node.version) == expectedVersion
}
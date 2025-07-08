package stack

import (
	"sync/atomic"
	"unsafe"
)

// Node represents a stack node
type Node struct {
	data interface{}
	next unsafe.Pointer // *Node
}

// LockFreeStack implements a lock-free stack using CAS
type LockFreeStack struct {
	head unsafe.Pointer // *Node
}

func NewLockFreeStack() *LockFreeStack {
	return &LockFreeStack{}
}

func (s *LockFreeStack) Push(data interface{}) {
	newNode := &Node{data: data}
	
	for {
		head := atomic.LoadPointer(&s.head)
		newNode.next = head
		
		if atomic.CompareAndSwapPointer(&s.head, head, unsafe.Pointer(newNode)) {
			return
		}
	}
}

func (s *LockFreeStack) Pop() (interface{}, bool) {
	for {
		head := atomic.LoadPointer(&s.head)
		if head == nil {
			return nil, false
		}
		
		headNode := (*Node)(head)
		next := atomic.LoadPointer(&headNode.next)
		
		if atomic.CompareAndSwapPointer(&s.head, head, next) {
			return headNode.data, true
		}
	}
}

func (s *LockFreeStack) IsEmpty() bool {
	return atomic.LoadPointer(&s.head) == nil
}

func (s *LockFreeStack) Size() int {
	count := 0
	current := atomic.LoadPointer(&s.head)
	
	for current != nil {
		count++
		currentNode := (*Node)(current)
		current = atomic.LoadPointer(&currentNode.next)
	}
	
	return count
}

// Peek returns the top element without removing it
func (s *LockFreeStack) Peek() (interface{}, bool) {
	head := atomic.LoadPointer(&s.head)
	if head == nil {
		return nil, false
	}
	
	headNode := (*Node)(head)
	return headNode.data, true
}
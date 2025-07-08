package list

import (
	"sync/atomic"
	"unsafe"
)

// ListNode represents a node in the linked list
type ListNode struct {
	key    int64
	value  interface{}
	next   unsafe.Pointer // *ListNode
	marked int32          // Deletion mark
}

// LockFreeList implements Harris's lock-free linked list
type LockFreeList struct {
	head *ListNode
	tail *ListNode
}

func NewLockFreeList() *LockFreeList {
	head := &ListNode{key: -1} // Sentinel
	tail := &ListNode{key: 1<<63 - 1} // Max sentinel
	head.next = unsafe.Pointer(tail)
	
	return &LockFreeList{
		head: head,
		tail: tail,
	}
}

// Search finds the position for a key
func (l *LockFreeList) Search(key int64) (*ListNode, *ListNode) {
	var pred, curr *ListNode
	
retry:
	pred = l.head
	curr = (*ListNode)(atomic.LoadPointer(&pred.next))
	
	for {
		// Remove marked nodes
		for atomic.LoadInt32(&curr.marked) == 1 {
			next := (*ListNode)(atomic.LoadPointer(&curr.next))
			if !atomic.CompareAndSwapPointer(&pred.next, unsafe.Pointer(curr), unsafe.Pointer(next)) {
				goto retry
			}
			curr = next
		}
		
		if curr.key >= key {
			return pred, curr
		}
		
		pred = curr
		curr = (*ListNode)(atomic.LoadPointer(&curr.next))
	}
}

func (l *LockFreeList) Insert(key int64, value interface{}) bool {
	newNode := &ListNode{
		key:   key,
		value: value,
	}
	
	for {
		pred, curr := l.Search(key)
		
		if curr.key == key {
			return false // Key already exists
		}
		
		newNode.next = unsafe.Pointer(curr)
		
		if atomic.CompareAndSwapPointer(&pred.next, unsafe.Pointer(curr), unsafe.Pointer(newNode)) {
			return true
		}
	}
}

func (l *LockFreeList) Delete(key int64) bool {
	for {
		pred, curr := l.Search(key)
		
		if curr.key != key {
			return false // Key not found
		}
		
		next := (*ListNode)(atomic.LoadPointer(&curr.next))
		
		// Mark node for deletion
		if !atomic.CompareAndSwapInt32(&curr.marked, 0, 1) {
			continue // Already marked
		}
		
		// Try to physically remove
		atomic.CompareAndSwapPointer(&pred.next, unsafe.Pointer(curr), unsafe.Pointer(next))
		return true
	}
}

func (l *LockFreeList) Contains(key int64) bool {
	curr := (*ListNode)(atomic.LoadPointer(&l.head.next))
	
	for curr.key < key {
		curr = (*ListNode)(atomic.LoadPointer(&curr.next))
	}
	
	return curr.key == key && atomic.LoadInt32(&curr.marked) == 0
}

func (l *LockFreeList) Size() int {
	count := 0
	curr := (*ListNode)(atomic.LoadPointer(&l.head.next))
	
	for curr != l.tail {
		if atomic.LoadInt32(&curr.marked) == 0 {
			count++
		}
		curr = (*ListNode)(atomic.LoadPointer(&curr.next))
	}
	
	return count
}

func (l *LockFreeList) IsEmpty() bool {
	curr := (*ListNode)(atomic.LoadPointer(&l.head.next))
	return curr == l.tail
}

// Get retrieves the value for a given key
func (l *LockFreeList) Get(key int64) (interface{}, bool) {
	curr := (*ListNode)(atomic.LoadPointer(&l.head.next))
	
	for curr.key < key {
		curr = (*ListNode)(atomic.LoadPointer(&curr.next))
	}
	
	if curr.key == key && atomic.LoadInt32(&curr.marked) == 0 {
		return curr.value, true
	}
	return nil, false
}

// Update modifies the value for an existing key
func (l *LockFreeList) Update(key int64, value interface{}) bool {
	curr := (*ListNode)(atomic.LoadPointer(&l.head.next))
	
	for curr.key < key {
		curr = (*ListNode)(atomic.LoadPointer(&curr.next))
	}
	
	if curr.key == key && atomic.LoadInt32(&curr.marked) == 0 {
		curr.value = value
		return true
	}
	return false
}
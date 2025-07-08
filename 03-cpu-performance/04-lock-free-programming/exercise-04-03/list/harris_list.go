package list

import (
	"sync/atomic"
	"unsafe"
)

// HarrisListNode represents a node in Harris's lock-free list
type HarrisListNode struct {
	key    int64
	value  interface{}
	next   unsafe.Pointer // *HarrisListNode
	marked int32          // Deletion mark
}

// HarrisLockFreeList implements Harris's lock-free linked list algorithm
type HarrisLockFreeList struct {
	head *HarrisListNode
	tail *HarrisListNode
}

func NewHarrisLockFreeList() *HarrisLockFreeList {
	head := &HarrisListNode{key: -1} // Sentinel
	tail := &HarrisListNode{key: 1<<63 - 1} // Max sentinel
	head.next = unsafe.Pointer(tail)
	
	return &HarrisLockFreeList{
		head: head,
		tail: tail,
	}
}

// searchFrom searches from a given starting node
func (l *HarrisLockFreeList) searchFrom(startNode *HarrisListNode, key int64) (*HarrisListNode, *HarrisListNode) {
	var pred, curr, next *HarrisListNode
	
retry:
	pred = startNode
	curr = (*HarrisListNode)(atomic.LoadPointer(&pred.next))
	
	for {
		if curr == nil {
			return pred, l.tail
		}
		
		next = (*HarrisListNode)(atomic.LoadPointer(&curr.next))
		
		// Check if current node is marked for deletion
		if atomic.LoadInt32(&curr.marked) == 1 {
			// Try to physically remove the marked node
			if !atomic.CompareAndSwapPointer(&pred.next, unsafe.Pointer(curr), unsafe.Pointer(next)) {
				goto retry // CAS failed, restart
			}
			curr = next
		} else {
			if curr.key >= key {
				return pred, curr
			}
			pred = curr
			curr = next
		}
	}
}

// Search finds the position for a key
func (l *HarrisLockFreeList) Search(key int64) (*HarrisListNode, *HarrisListNode) {
	return l.searchFrom(l.head, key)
}

func (l *HarrisLockFreeList) Insert(key int64, value interface{}) bool {
	newNode := &HarrisListNode{
		key:   key,
		value: value,
	}
	
	for {
		pred, curr := l.Search(key)
		
		if curr != l.tail && curr.key == key {
			return false // Key already exists
		}
		
		newNode.next = unsafe.Pointer(curr)
		
		if atomic.CompareAndSwapPointer(&pred.next, unsafe.Pointer(curr), unsafe.Pointer(newNode)) {
			return true
		}
	}
}

func (l *HarrisLockFreeList) Delete(key int64) bool {
	for {
		pred, curr := l.Search(key)
		
		if curr == l.tail || curr.key != key {
			return false // Key not found
		}
		
		next := (*HarrisListNode)(atomic.LoadPointer(&curr.next))
		
		// Logically delete by marking
		if !atomic.CompareAndSwapInt32(&curr.marked, 0, 1) {
			continue // Already marked, retry
		}
		
		// Try to physically remove
		atomic.CompareAndSwapPointer(&pred.next, unsafe.Pointer(curr), unsafe.Pointer(next))
		return true
	}
}

func (l *HarrisLockFreeList) Contains(key int64) bool {
	curr := (*HarrisListNode)(atomic.LoadPointer(&l.head.next))
	
	for curr != l.tail && curr.key < key {
		curr = (*HarrisListNode)(atomic.LoadPointer(&curr.next))
	}
	
	return curr != l.tail && curr.key == key && atomic.LoadInt32(&curr.marked) == 0
}

func (l *HarrisLockFreeList) Get(key int64) (interface{}, bool) {
	curr := (*HarrisListNode)(atomic.LoadPointer(&l.head.next))
	
	for curr != l.tail && curr.key < key {
		curr = (*HarrisListNode)(atomic.LoadPointer(&curr.next))
	}
	
	if curr != l.tail && curr.key == key && atomic.LoadInt32(&curr.marked) == 0 {
		return curr.value, true
	}
	return nil, false
}

func (l *HarrisLockFreeList) Update(key int64, value interface{}) bool {
	curr := (*HarrisListNode)(atomic.LoadPointer(&l.head.next))
	
	for curr != l.tail && curr.key < key {
		curr = (*HarrisListNode)(atomic.LoadPointer(&curr.next))
	}
	
	if curr != l.tail && curr.key == key && atomic.LoadInt32(&curr.marked) == 0 {
		curr.value = value
		return true
	}
	return false
}

func (l *HarrisLockFreeList) Size() int {
	count := 0
	curr := (*HarrisListNode)(atomic.LoadPointer(&l.head.next))
	
	for curr != l.tail {
		if atomic.LoadInt32(&curr.marked) == 0 {
			count++
		}
		curr = (*HarrisListNode)(atomic.LoadPointer(&curr.next))
	}
	
	return count
}

func (l *HarrisLockFreeList) IsEmpty() bool {
	curr := (*HarrisListNode)(atomic.LoadPointer(&l.head.next))
	return curr == l.tail
}

// Cleanup performs garbage collection of marked nodes
func (l *HarrisLockFreeList) Cleanup() int {
	cleanedCount := 0
	_, _ = l.searchFrom(l.head, 1<<63-1) // Traverse entire list to clean up
	return cleanedCount
}

// Range iterates over all elements in the list
func (l *HarrisLockFreeList) Range(fn func(key int64, value interface{}) bool) {
	curr := (*HarrisListNode)(atomic.LoadPointer(&l.head.next))
	
	for curr != l.tail {
		if atomic.LoadInt32(&curr.marked) == 0 {
			if !fn(curr.key, curr.value) {
				break
			}
		}
		curr = (*HarrisListNode)(atomic.LoadPointer(&curr.next))
	}
}
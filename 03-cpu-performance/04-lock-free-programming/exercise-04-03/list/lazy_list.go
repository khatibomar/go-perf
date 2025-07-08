package list

import (
	"sync/atomic"
	"unsafe"
)

// LazyListNode represents a node in the lazy list
type LazyListNode struct {
	key    int64
	value  interface{}
	next   unsafe.Pointer // *LazyListNode
	marked int32          // Deletion mark
	locked int32          // Lock for this node
}

// LazyList implements lazy synchronization list
type LazyList struct {
	head *LazyListNode
	tail *LazyListNode
}

func NewLazyList() *LazyList {
	head := &LazyListNode{key: -1}
	tail := &LazyListNode{key: 1<<63 - 1}
	head.next = unsafe.Pointer(tail)
	
	return &LazyList{
		head: head,
		tail: tail,
	}
}

// lock attempts to lock a node
func (l *LazyList) lock(node *LazyListNode) bool {
	return atomic.CompareAndSwapInt32(&node.locked, 0, 1)
}

// unlock releases the lock on a node
func (l *LazyList) unlock(node *LazyListNode) {
	atomic.StoreInt32(&node.locked, 0)
}

// validate checks if the node is still reachable and not marked
func (l *LazyList) validate(pred, curr *LazyListNode) bool {
	return atomic.LoadInt32(&pred.marked) == 0 &&
		atomic.LoadInt32(&curr.marked) == 0 &&
		atomic.LoadPointer(&pred.next) == unsafe.Pointer(curr)
}

// locate finds the position for a key with lazy validation
func (l *LazyList) locate(key int64) (*LazyListNode, *LazyListNode) {
	for {
		pred := l.head
		curr := (*LazyListNode)(atomic.LoadPointer(&pred.next))
		
		// Traverse to find position
		for curr.key < key {
			pred = curr
			curr = (*LazyListNode)(atomic.LoadPointer(&curr.next))
		}
		
		// Lock both nodes
		if l.lock(pred) {
			if l.lock(curr) {
				// Validate after locking
				if l.validate(pred, curr) {
					return pred, curr
				}
				l.unlock(curr)
			}
			l.unlock(pred)
		}
	}
}

func (l *LazyList) Insert(key int64, value interface{}) bool {
	for {
		pred, curr := l.locate(key)
		
		if curr.key == key {
			l.unlock(pred)
			l.unlock(curr)
			return false // Key already exists
		}
		
		newNode := &LazyListNode{
			key:   key,
			value: value,
			next:  unsafe.Pointer(curr),
		}
		
		atomic.StorePointer(&pred.next, unsafe.Pointer(newNode))
		l.unlock(pred)
		l.unlock(curr)
		return true
	}
}

func (l *LazyList) Delete(key int64) bool {
	for {
		pred, curr := l.locate(key)
		
		if curr.key != key {
			l.unlock(pred)
			l.unlock(curr)
			return false // Key not found
		}
		
		// Mark for deletion
		atomic.StoreInt32(&curr.marked, 1)
		
		// Physical removal
		next := (*LazyListNode)(atomic.LoadPointer(&curr.next))
		atomic.StorePointer(&pred.next, unsafe.Pointer(next))
		
		l.unlock(pred)
		l.unlock(curr)
		return true
	}
}

func (l *LazyList) Contains(key int64) bool {
	curr := (*LazyListNode)(atomic.LoadPointer(&l.head.next))
	
	for curr.key < key {
		curr = (*LazyListNode)(atomic.LoadPointer(&curr.next))
	}
	
	return curr.key == key && atomic.LoadInt32(&curr.marked) == 0
}

func (l *LazyList) Get(key int64) (interface{}, bool) {
	curr := (*LazyListNode)(atomic.LoadPointer(&l.head.next))
	
	for curr.key < key {
		curr = (*LazyListNode)(atomic.LoadPointer(&curr.next))
	}
	
	if curr.key == key && atomic.LoadInt32(&curr.marked) == 0 {
		return curr.value, true
	}
	return nil, false
}

func (l *LazyList) Update(key int64, value interface{}) bool {
	for {
		pred, curr := l.locate(key)
		
		if curr.key != key {
			l.unlock(pred)
			l.unlock(curr)
			return false
		}
		
		if atomic.LoadInt32(&curr.marked) == 0 {
			curr.value = value
			l.unlock(pred)
			l.unlock(curr)
			return true
		}
		
		l.unlock(pred)
		l.unlock(curr)
		return false
	}
}

func (l *LazyList) Size() int {
	count := 0
	curr := (*LazyListNode)(atomic.LoadPointer(&l.head.next))
	
	for curr != l.tail {
		if atomic.LoadInt32(&curr.marked) == 0 {
			count++
		}
		curr = (*LazyListNode)(atomic.LoadPointer(&curr.next))
	}
	
	return count
}

func (l *LazyList) IsEmpty() bool {
	curr := (*LazyListNode)(atomic.LoadPointer(&l.head.next))
	return curr == l.tail
}

// Cleanup removes marked nodes (should be called periodically)
func (l *LazyList) Cleanup() int {
	cleanedCount := 0
	pred := l.head
	
	for {
		curr := (*LazyListNode)(atomic.LoadPointer(&pred.next))
		if curr == l.tail {
			break
		}
		
		if atomic.LoadInt32(&curr.marked) == 1 {
			next := (*LazyListNode)(atomic.LoadPointer(&curr.next))
			if atomic.CompareAndSwapPointer(&pred.next, unsafe.Pointer(curr), unsafe.Pointer(next)) {
				cleanedCount++
				continue
			}
		}
		
		pred = curr
	}
	
	return cleanedCount
}
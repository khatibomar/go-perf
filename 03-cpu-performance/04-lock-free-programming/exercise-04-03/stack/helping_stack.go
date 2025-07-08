package stack

import (
	"sync/atomic"
	"unsafe"
)

// HelpingStackOperation represents an operation that can be helped
type HelpingStackOperation struct {
	operationType int32 // 0 = push, 1 = pop
	data          interface{}
	result        interface{}
	success       int32 // 0 = pending, 1 = success, -1 = failure
	completed     int32 // 0 = not completed, 1 = completed
}

const (
	HelpingOpPush = iota
	HelpingOpPop
)

// HelpingStackNode represents a node in the helping stack
type HelpingStackNode struct {
	data interface{}
	next unsafe.Pointer // *HelpingStackNode
}

// HelpingStack implements wait-free stack with helping mechanism
type HelpingStack struct {
	head      unsafe.Pointer // *HelpingStackNode
	operation unsafe.Pointer // *HelpingStackOperation
}

func NewHelpingStack() *HelpingStack {
	return &HelpingStack{}
}

func (s *HelpingStack) Push(data interface{}) {
	op := &HelpingStackOperation{
		operationType: HelpingOpPush,
		data:          data,
	}
	
	// Try to install our operation
	for {
		currentOp := (*HelpingStackOperation)(atomic.LoadPointer(&s.operation))
		
		// Help complete any pending operation
		if currentOp != nil {
			s.helpComplete(currentOp)
		}
		
		// Try to install our operation
		if atomic.CompareAndSwapPointer(&s.operation, unsafe.Pointer(currentOp), unsafe.Pointer(op)) {
			break
		}
	}
	
	// Complete our operation
	s.helpComplete(op)
}

func (s *HelpingStack) Pop() (interface{}, bool) {
	op := &HelpingStackOperation{
		operationType: HelpingOpPop,
	}
	
	// Try to install our operation
	for {
		currentOp := (*HelpingStackOperation)(atomic.LoadPointer(&s.operation))
		
		// Help complete any pending operation
		if currentOp != nil {
			s.helpComplete(currentOp)
		}
		
		// Try to install our operation
		if atomic.CompareAndSwapPointer(&s.operation, unsafe.Pointer(currentOp), unsafe.Pointer(op)) {
			break
		}
	}
	
	// Complete our operation
	s.helpComplete(op)
	
	success := atomic.LoadInt32(&op.success)
	if success == 1 {
		return op.result, true
	}
	return nil, false
}

func (s *HelpingStack) helpComplete(op *HelpingStackOperation) {
	if atomic.LoadInt32(&op.completed) == 1 {
		return
	}
	
	switch op.operationType {
	case HelpingOpPush:
		s.helpPush(op)
	case HelpingOpPop:
		s.helpPop(op)
	}
	
	atomic.StoreInt32(&op.completed, 1)
	
	// Clear the operation if it's still ours
	atomic.CompareAndSwapPointer(&s.operation, unsafe.Pointer(op), nil)
}

func (s *HelpingStack) helpPush(op *HelpingStackOperation) {
	newNode := &HelpingStackNode{data: op.data}
	
	for {
		head := atomic.LoadPointer(&s.head)
		newNode.next = head
		
		if atomic.CompareAndSwapPointer(&s.head, head, unsafe.Pointer(newNode)) {
			atomic.StoreInt32(&op.success, 1)
			return
		}
	}
}

func (s *HelpingStack) helpPop(op *HelpingStackOperation) {
	for {
		head := atomic.LoadPointer(&s.head)
		if head == nil {
			atomic.StoreInt32(&op.success, -1)
			return
		}
		
		headNode := (*HelpingStackNode)(head)
		next := atomic.LoadPointer(&headNode.next)
		
		if atomic.CompareAndSwapPointer(&s.head, head, next) {
			op.result = headNode.data
			atomic.StoreInt32(&op.success, 1)
			return
		}
	}
}

func (s *HelpingStack) IsEmpty() bool {
	return atomic.LoadPointer(&s.head) == nil
}

func (s *HelpingStack) Size() int {
	count := 0
	current := atomic.LoadPointer(&s.head)
	
	for current != nil {
		count++
		currentNode := (*HelpingStackNode)(current)
		current = atomic.LoadPointer(&currentNode.next)
	}
	
	return count
}

func (s *HelpingStack) Peek() (interface{}, bool) {
	head := atomic.LoadPointer(&s.head)
	if head == nil {
		return nil, false
	}
	
	headNode := (*HelpingStackNode)(head)
	return headNode.data, true
}

// HelpOthers helps complete any pending operations
func (s *HelpingStack) HelpOthers() {
	currentOp := (*HelpingStackOperation)(atomic.LoadPointer(&s.operation))
	if currentOp != nil {
		s.helpComplete(currentOp)
	}
}
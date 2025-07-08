package cas

import (
	"sync/atomic"
	"unsafe"
)

// HelpingOperation represents an operation that can be helped by other threads
type HelpingOperation struct {
	operationType int32
	arg1          int64
	arg2          int64
	result        int64
	completed     int32
}

const (
	OpIncrement = iota
	OpAdd
	OpMultiply
)

// HelpingCounter implements a wait-free counter with helping
type HelpingCounter struct {
	value     int64
	operation unsafe.Pointer // *HelpingOperation
}

func NewHelpingCounter() *HelpingCounter {
	return &HelpingCounter{}
}

func (c *HelpingCounter) Increment() int64 {
	return c.Add(1)
}

func (c *HelpingCounter) Add(delta int64) int64 {
	op := &HelpingOperation{
		operationType: OpAdd,
		arg1:          delta,
	}
	
	// Try to install our operation
	for {
		currentOp := (*HelpingOperation)(atomic.LoadPointer(&c.operation))
		
		// Help complete any pending operation
		if currentOp != nil {
			c.helpComplete(currentOp)
		}
		
		// Try to install our operation
		if atomic.CompareAndSwapPointer(&c.operation, unsafe.Pointer(currentOp), unsafe.Pointer(op)) {
			break
		}
	}
	
	// Complete our operation
	c.helpComplete(op)
	return atomic.LoadInt64(&op.result)
}

func (c *HelpingCounter) helpComplete(op *HelpingOperation) {
	if atomic.LoadInt32(&op.completed) == 1 {
		return
	}
	
	var result int64
	switch op.operationType {
	case OpIncrement:
		result = atomic.AddInt64(&c.value, 1)
	case OpAdd:
		result = atomic.AddInt64(&c.value, op.arg1)
	case OpMultiply:
		// This is more complex and would need special handling
		for {
			old := atomic.LoadInt64(&c.value)
			new := old * op.arg1
			if atomic.CompareAndSwapInt64(&c.value, old, new) {
				result = new
				break
			}
		}
	}
	
	atomic.StoreInt64(&op.result, result)
	atomic.StoreInt32(&op.completed, 1)
	
	// Clear the operation if it's still ours
	atomic.CompareAndSwapPointer(&c.operation, unsafe.Pointer(op), nil)
}

func (c *HelpingCounter) Get() int64 {
	return atomic.LoadInt64(&c.value)
}

// HelpOthers helps complete any pending operations
func (c *HelpingCounter) HelpOthers() {
	currentOp := (*HelpingOperation)(atomic.LoadPointer(&c.operation))
	if currentOp != nil {
		c.helpComplete(currentOp)
	}
}
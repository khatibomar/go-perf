package atomic

import (
	"testing"
	"unsafe"
)

// TestAtomicCounter tests basic atomic counter operations
func TestAtomicCounter(t *testing.T) {
	counter := NewAtomicCounter()
	
	// Test initial value
	if counter.Load() != 0 {
		t.Errorf("Expected initial value 0, got %d", counter.Load())
	}
	
	// Test Store
	counter.Store(42)
	if counter.Load() != 42 {
		t.Errorf("Expected 42, got %d", counter.Load())
	}
	
	// Test Add
	counter.Add(8)
	if counter.Load() != 50 {
		t.Errorf("Expected 50, got %d", counter.Load())
	}
	
	// Test CompareAndSwap
	if !counter.CompareAndSwap(50, 100) {
		t.Error("CompareAndSwap should have succeeded")
	}
	if counter.Load() != 100 {
		t.Errorf("Expected 100, got %d", counter.Load())
	}
	
	// Test failed CompareAndSwap
	if counter.CompareAndSwap(99, 200) {
		t.Error("CompareAndSwap should have failed")
	}
	if counter.Load() != 100 {
		t.Errorf("Expected 100, got %d", counter.Load())
	}
	
	// Test Swap
	old := counter.Swap(300)
	if old != 100 {
		t.Errorf("Expected old value 100, got %d", old)
	}
	if counter.Load() != 300 {
		t.Errorf("Expected 300, got %d", counter.Load())
	}
}

// TestAtomicPointer tests atomic pointer operations
func TestAtomicPointer(t *testing.T) {
	ptr := NewAtomicPointer()
	value1 := 42
	value2 := 84
	
	// Test initial value
	if ptr.Load() != nil {
		t.Error("Expected initial value nil")
	}
	
	// Test Store
	ptr.Store(unsafe.Pointer(&value1))
	loadedPtr := ptr.Load()
	if loadedPtr == nil {
		t.Error("Expected non-nil pointer")
	} else if *(*int)(loadedPtr) != 42 {
		t.Errorf("Expected 42, got %d", *(*int)(loadedPtr))
	}
	
	// Test CompareAndSwap
	if !ptr.CompareAndSwap(unsafe.Pointer(&value1), unsafe.Pointer(&value2)) {
		t.Error("CompareAndSwap should have succeeded")
	}
	loadedPtr = ptr.Load()
	if loadedPtr == nil {
		t.Error("Expected non-nil pointer")
	} else if *(*int)(loadedPtr) != 84 {
		t.Errorf("Expected 84, got %d", *(*int)(loadedPtr))
	}
	
	// Test Swap
	oldPtr := ptr.Swap(unsafe.Pointer(&value1))
	if oldPtr == nil {
		t.Error("Expected non-nil old pointer")
	} else if *(*int)(oldPtr) != 84 {
		t.Errorf("Expected old value 84, got %d", *(*int)(oldPtr))
	}
	loadedPtr = ptr.Load()
	if loadedPtr == nil {
		t.Error("Expected non-nil pointer")
	} else if *(*int)(loadedPtr) != 42 {
		t.Errorf("Expected 42, got %d", *(*int)(loadedPtr))
	}
}

// TestAtomicBool tests atomic boolean operations
func TestAtomicBool(t *testing.T) {
	boolVal := NewAtomicBool()
	
	// Test initial value
	if boolVal.Load() != false {
		t.Error("Expected initial value false")
	}
	
	// Test Store
	boolVal.Store(true)
	if boolVal.Load() != true {
		t.Error("Expected true")
	}
	
	// Test CompareAndSwap
	if !boolVal.CompareAndSwap(true, false) {
		t.Error("CompareAndSwap should have succeeded")
	}
	if boolVal.Load() != false {
		t.Error("Expected false")
	}
	
	// Test failed CompareAndSwap
	if boolVal.CompareAndSwap(true, true) {
		t.Error("CompareAndSwap should have failed")
	}
	if boolVal.Load() != false {
		t.Error("Expected false")
	}
	
	// Test Swap
	old := boolVal.Swap(true)
	if old != false {
		t.Error("Expected old value false")
	}
	if boolVal.Load() != true {
		t.Error("Expected true")
	}
}

// TestCASCounter tests CAS counter operations
func TestCASCounter(t *testing.T) {
	casCounter := NewCASCounter()
	
	// Test SimpleCAS
	if !casCounter.SimpleCAS(0, 10) {
		t.Error("SimpleCAS should have succeeded")
	}
	if casCounter.Load() != 10 {
		t.Errorf("Expected 10, got %d", casCounter.Load())
	}
	
	// Test CASLoop
	casCounter.CASLoop(5)
	if casCounter.Load() != 15 {
		t.Errorf("Expected 15, got %d", casCounter.Load())
	}
	
	// Test CASWithRetry
	casCounter.CASWithRetry(3, 5)
	if casCounter.Load() != 18 {
		t.Errorf("Expected 18, got %d", casCounter.Load())
	}
	
	// Test CASWithBackoff
	casCounter.CASWithBackoff(2)
	if casCounter.Load() != 20 {
		t.Errorf("Expected 20, got %d", casCounter.Load())
	}
}

// TestArithmeticCounter tests arithmetic operations
func TestArithmeticCounter(t *testing.T) {
	counter := NewArithmeticCounter()
	
	// Test Increment
	counter.Increment()
	if counter.Load() != 1 {
		t.Errorf("Expected 1, got %d", counter.Load())
	}
	
	// Test Add
	counter.Add(5)
	if counter.Load() != 6 {
		t.Errorf("Expected 6, got %d", counter.Load())
	}
	
	// Test Decrement
	counter.Decrement()
	if counter.Load() != 5 {
		t.Errorf("Expected 5, got %d", counter.Load())
	}
	
	// Test Store
	counter.Store(100)
	if counter.Load() != 100 {
		t.Errorf("Expected 100, got %d", counter.Load())
	}
	
	// Test Reset
	counter.Reset()
	if counter.Load() != 0 {
		t.Errorf("Expected 0, got %d", counter.Load())
	}
}

// TestBitwiseOperations tests bitwise operations
func TestBitwiseOperations(t *testing.T) {
	bitwise := NewBitwiseOperations()
	
	// Test SetBit
	bitwise.SetBit(0)
	if !bitwise.TestBit(0) {
		t.Error("Bit 0 should be set")
	}
	
	bitwise.SetBit(5)
	if !bitwise.TestBit(5) {
		t.Error("Bit 5 should be set")
	}
	
	// Test ClearBit
	bitwise.ClearBit(0)
	if bitwise.TestBit(0) {
		t.Error("Bit 0 should be clear")
	}
	
	// Test ToggleBit
	bitwise.ToggleBit(5)
	if bitwise.TestBit(5) {
		t.Error("Bit 5 should be clear after toggle")
	}
	
	bitwise.ToggleBit(5)
	if !bitwise.TestBit(5) {
		t.Error("Bit 5 should be set after second toggle")
	}
	
	// Test PopCount
	bitwise.Store(0b1010101) // 4 bits set
	if bitwise.PopCount() != 4 {
		t.Errorf("Expected popcount 4, got %d", bitwise.PopCount())
	}
}

// TestAtomicFloat64 tests atomic float operations
func TestAtomicFloat64(t *testing.T) {
	floatOps := NewAtomicFloat64()
	
	// Test Store and Load
	floatOps.Store(3.14)
	if floatOps.Load() != 3.14 {
		t.Errorf("Expected 3.14, got %f", floatOps.Load())
	}
	
	// Test CompareAndSwap
	if !floatOps.CompareAndSwap(3.14, 2.71) {
		t.Error("CompareAndSwap should have succeeded")
	}
	if floatOps.Load() != 2.71 {
		t.Errorf("Expected 2.71, got %f", floatOps.Load())
	}
	
	// Test failed CompareAndSwap
	if floatOps.CompareAndSwap(3.14, 1.41) {
		t.Error("CompareAndSwap should have failed")
	}
	if floatOps.Load() != 2.71 {
		t.Errorf("Expected 2.71, got %f", floatOps.Load())
	}
}

// TestMemoryOrdering tests memory ordering operations
func TestMemoryOrdering(t *testing.T) {
	demo := NewMemoryOrderingDemo()
	
	// Test relaxed operations
	demo.RelaxedWrite(42)
	value, ok := demo.RelaxedRead()
	if !ok {
		t.Error("RelaxedRead should have succeeded")
	}
	if value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}
	
	// Test acquire-release operations
	demo.AcquireReleaseWrite(84)
	value, ok = demo.AcquireReleaseRead()
	if !ok {
		t.Error("AcquireReleaseRead should have succeeded")
	}
	if value != 84 {
		t.Errorf("Expected 84, got %d", value)
	}
	
	// Test sequential consistency
	demo.SequentialWrite(126)
	value, ok = demo.SequentialRead()
	if !ok {
		t.Error("SequentialRead should have succeeded")
	}
	if value != 126 {
		t.Errorf("Expected 126, got %d", value)
	}
	
	// Test Reset
	demo.Reset()
	value, ok = demo.RelaxedRead()
	if ok {
		t.Error("RelaxedRead should have failed after reset")
	}
}
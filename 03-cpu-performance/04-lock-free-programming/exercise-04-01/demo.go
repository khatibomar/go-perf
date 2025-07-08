package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"exercise-04-01/atomic"
	"exercise-04-01/cache"
)

func main() {
	fmt.Println("=== Atomic Operations Demo ===")

	// Basic atomic counter
	counter := atomic.NewAtomicCounter()
	counter.Store(10)
	fmt.Printf("Counter value: %d\n", counter.Load())
	counter.Add(5)
	fmt.Printf("After adding 5: %d\n", counter.Load())

	// Atomic pointer operations
	ptr := atomic.NewAtomicPointer()
	value1 := 42
	value2 := 100
	ptr.Store(unsafe.Pointer(&value1))
	loadedPtr := ptr.Load()
	if loadedPtr != nil {
		fmt.Printf("Pointer value: %d\n", *(*int)(loadedPtr))
	}
	ptr.Swap(unsafe.Pointer(&value2))
	loadedPtr = ptr.Load()
	if loadedPtr != nil {
		fmt.Printf("After swap: %d\n", *(*int)(loadedPtr))
	}

	// Atomic boolean
	boolVal := atomic.NewAtomicBool()
	boolVal.Store(true)
	fmt.Printf("Boolean value: %t\n", boolVal.Load())
	oldBool := boolVal.Swap(false)
	fmt.Printf("Swapped from %t to %t\n", oldBool, boolVal.Load())

	// CAS operations
	casCounter := atomic.NewCASCounter()
	fmt.Printf("\n=== CAS Operations ===\n")
	fmt.Printf("Initial CAS counter: %d\n", casCounter.Load())
	success := casCounter.SimpleCAS(0, 10)
	fmt.Printf("CAS 0->10 success: %t, value: %d\n", success, casCounter.Load())

	// Demonstrate contention with goroutines
	fmt.Println("\n=== Concurrent Operations Demo ===")
	demonstrateContention()

	// Cache effects demo
	fmt.Println("\n=== Cache Effects Demo ===")
	demonstrateCache()
}

func demonstrateContention() {
	counter := atomic.NewAtomicCounter()
	var wg sync.WaitGroup
	numGoroutines := runtime.NumCPU()
	operationsPerGoroutine := 10000

	start := time.Now()
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				counter.Add(1)
			}
		}()
	}
	wg.Wait()
	duration := time.Since(start)

	expected := int64(numGoroutines * operationsPerGoroutine)
	actual := counter.Load()
	fmt.Printf("Goroutines: %d, Operations each: %d\n", numGoroutines, operationsPerGoroutine)
	fmt.Printf("Expected: %d, Actual: %d, Correct: %t\n", expected, actual, expected == actual)
	fmt.Printf("Time taken: %v\n", duration)
}

func demonstrateCache() {
	// Create cache-aligned and false-sharing examples
	aligned := cache.NewAlignedData()
	falseSharingExample := cache.NewFalseSharingExample()

	// Set some values
	aligned.Store(42)
	falseSharingExample.IncrementCounter1()
	falseSharingExample.IncrementCounter2()

	fmt.Printf("Aligned data address: %p\n", aligned)
	fmt.Printf("False sharing example address: %p\n", falseSharingExample)
	fmt.Printf("Cache line size: %d bytes\n", cache.CacheLineSize)
	fmt.Printf("Aligned data value: %d\n", aligned.Load())
	c1, c2 := falseSharingExample.GetCounters()
	fmt.Printf("False sharing counters: %d, %d\n", c1, c2)

	// Check alignment
	alignedAddr := uintptr(unsafe.Pointer(aligned))
	falseSharingAddr := uintptr(unsafe.Pointer(falseSharingExample))
	fmt.Printf("Aligned data is cache-line aligned: %t\n", alignedAddr%cache.CacheLineSize == 0)
	fmt.Printf("False sharing example is cache-line aligned: %t\n", falseSharingAddr%cache.CacheLineSize == 0)
}

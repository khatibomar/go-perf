package main

import (
	"fmt"
	"testing"
	"time"
	"unsafe"
)

// TestMemoryArena tests memory arena allocation
func TestMemoryArena(t *testing.T) {
	arena := NewMemoryArena(1024) // 1KB blocks
	
	// Test basic allocation
	ptr1 := arena.Allocate(64)
	if ptr1 == nil {
		t.Fatal("Failed to allocate 64 bytes")
	}
	
	ptr2 := arena.Allocate(128)
	if ptr2 == nil {
		t.Fatal("Failed to allocate 128 bytes")
	}
	
	// Test that pointers are different
	if ptr1 == ptr2 {
		t.Error("Allocated pointers should be different")
	}
	
	// Test allocation larger than block size (should allocate new block)
	ptr3 := arena.Allocate(2048)
	if ptr3 == nil {
		t.Fatal("Failed to allocate 2048 bytes")
	}
	
	// Test reset
	arena.Reset()
	ptr4 := arena.Allocate(64)
	if ptr4 == nil {
		t.Fatal("Failed to allocate after reset")
	}
	
	// After reset, we should be able to allocate from the beginning again
	// (though we can't guarantee the same pointer due to implementation details)
}

// TestStringInterner tests string interning
func TestStringInterner(t *testing.T) {
	interner := NewStringInterner()
	
	// Test basic interning
	str1 := interner.Intern("hello")
	str2 := interner.Intern("hello")
	
	// Should return the same string instance
	if str1 != str2 {
		t.Error("Interned strings should be identical")
	}
	
	// Test different strings
	str3 := interner.Intern("world")
	if str1 == str3 {
		t.Error("Different strings should not be identical")
	}
	
	// Test stats
	unique, total := interner.Stats()
	if unique != 2 {
		t.Errorf("Expected 2 unique strings, got %d", unique)
	}
	if total != 3 {
		t.Errorf("Expected 3 total interns, got %d", total)
	}
	
	// Test memory savings
	savings := interner.MemorySavings()
	if savings <= 0 {
		t.Errorf("Expected positive memory savings, got %d", savings)
	}
}

// TestBufferPool tests buffer pooling
func TestBufferPool(t *testing.T) {
	pool := NewBufferPool(1024, 10) // 1KB buffers, max 10 in pool
	
	// Test get and put
	buf1 := pool.Get()
	if len(buf1) != 0 || cap(buf1) < 1024 {
		t.Errorf("Expected empty buffer with capacity >= 1024, got len=%d cap=%d", len(buf1), cap(buf1))
	}
	
	buf2 := pool.Get()
	if len(buf2) != 0 || cap(buf2) < 1024 {
		t.Errorf("Expected empty buffer with capacity >= 1024, got len=%d cap=%d", len(buf2), cap(buf2))
	}
	
	// Use buffers
	buf1 = append(buf1, []byte("hello")...)
	buf2 = append(buf2, []byte("world")...)
	
	// Return to pool
	pool.Put(buf1)
	pool.Put(buf2)
	
	// Get again - should be reused and reset
	buf3 := pool.Get()
	if len(buf3) != 0 {
		t.Errorf("Expected reset buffer to be empty, got length %d", len(buf3))
	}
	
	// Test stats
	pooled, created := pool.Stats()
	if created < 2 {
		t.Errorf("Expected at least 2 buffers created, got %d", created)
	}
	if pooled == 0 {
		t.Errorf("Expected some buffers in pool, got %d", pooled)
	}
}

// TestSlabAllocatorEdgeCases tests edge cases for slab allocator
func TestSlabAllocatorEdgeCases(t *testing.T) {
	sa := NewSlabAllocator(32, 1024) // 32-byte objects, 1KB slabs
	
	// Allocate all objects in first slab
	objectsPerSlab := 1024 / 32
	ptrs := make([]unsafe.Pointer, objectsPerSlab+5) // More than one slab
	
	for i := range ptrs {
		ptrs[i] = sa.Allocate()
		if ptrs[i] == nil {
			t.Fatalf("Failed to allocate object %d", i)
		}
	}
	
	// Should have allocated multiple slabs
	slabs, capacity, allocated := sa.Stats()
	if slabs < 2 {
		t.Errorf("Expected at least 2 slabs, got %d", slabs)
	}
	if allocated != len(ptrs) {
		t.Errorf("Expected %d allocated objects, got %d", len(ptrs), allocated)
	}
	if capacity < allocated {
		t.Errorf("Capacity (%d) should be >= allocated (%d)", capacity, allocated)
	}
	
	// Free some objects
	for i := 0; i < 5; i++ {
		sa.Free(ptrs[i])
	}
	
	// Check stats after freeing
	_, _, allocatedAfter := sa.Stats()
	if allocatedAfter != allocated-5 {
		t.Errorf("Expected %d allocated after freeing, got %d", allocated-5, allocatedAfter)
	}
}

// TestRingBufferEdgeCases tests edge cases for ring buffer
func TestRingBufferEdgeCases(t *testing.T) {
	rb := NewRingBuffer(10)
	
	// Test writing more than capacity
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	written := rb.Write(data)
	if written != 10 {
		t.Errorf("Expected to write 10 bytes (capacity), wrote %d", written)
	}
	
	// Buffer should be full
	if rb.Available() != 10 {
		t.Errorf("Expected 10 bytes available, got %d", rb.Available())
	}
	
	// Read some data
	readBuf := make([]byte, 5)
	read := rb.Read(readBuf)
	if read != 5 {
		t.Errorf("Expected to read 5 bytes, read %d", read)
	}
	
	// Now we should have space for more writes
	moreData := []byte{13, 14, 15, 16, 17}
	written2 := rb.Write(moreData)
	if written2 != 5 {
		t.Errorf("Expected to write 5 bytes, wrote %d", written2)
	}
	
	// Test wrap-around reading
	allData := make([]byte, 15)
	totalRead := rb.Read(allData)
	if totalRead != 10 {
		t.Errorf("Expected to read 10 bytes total, read %d", totalRead)
	}
	
	// Verify the data sequence
	expected := []byte{6, 7, 8, 9, 10, 13, 14, 15, 16, 17}
	for i, b := range expected {
		if allData[i] != b {
			t.Errorf("Expected byte %d at position %d, got %d", b, i, allData[i])
		}
	}
}

// Benchmark memory arena vs individual allocations with different sizes
func BenchmarkMemoryArenaVsSizes(b *testing.B) {
	sizes := []int{16, 64, 256, 1024}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Arena_%d", size), func(b *testing.B) {
			arena := NewMemoryArena(64 * 1024) // 64KB blocks
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ptr := arena.Allocate(size)
				// Simulate using the memory
				*(*byte)(ptr) = byte(i)
				if i%10000 == 0 {
					arena.Reset()
				}
			}
		})
		
		b.Run(fmt.Sprintf("Individual_%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				data := make([]byte, size)
				data[0] = byte(i)
			}
		})
	}
}

// Benchmark slab allocator vs individual allocations
func BenchmarkSlabAllocator(b *testing.B) {
	sa := NewSlabAllocator(64, 64*1024) // 64-byte objects, 64KB slabs
	
	b.Run("SlabAllocator", func(b *testing.B) {
		ptrs := make([]unsafe.Pointer, 1000)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			idx := i % len(ptrs)
			if ptrs[idx] != nil {
				sa.Free(ptrs[idx])
			}
			ptrs[idx] = sa.Allocate()
			// Simulate using the memory
			*(*byte)(ptrs[idx]) = byte(i)
		}
		// Cleanup
		for _, ptr := range ptrs {
			if ptr != nil {
				sa.Free(ptr)
			}
		}
	})
	
	b.Run("IndividualAlloc", func(b *testing.B) {
		ptrs := make([]*[64]byte, 1000)
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			idx := i % len(ptrs)
			ptrs[idx] = &[64]byte{}
			// Simulate using the memory
			ptrs[idx][0] = byte(i)
		}
	})
}

// Benchmark string interning vs regular strings
func BenchmarkStringInterning(b *testing.B) {
	interner := NewStringInterner()
	strings := []string{
		"hello", "world", "golang", "performance", "optimization",
		"memory", "allocation", "benchmark", "testing", "string",
	}
	
	b.Run("WithInterning", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			str := strings[i%len(strings)]
			interned := interner.Intern(str)
		SinkString = interned
		}
	})
	
	b.Run("WithoutInterning", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			str := strings[i%len(strings)]
			// Simulate string duplication
		duplicated := string([]byte(str))
		SinkString = duplicated
		}
	})
}

// Benchmark buffer pool vs direct allocation
func BenchmarkBufferPoolVsDirect(b *testing.B) {
	pool := NewBufferPool(1024, 100)
	
	b.Run("BufferPool", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := pool.Get()
			// Simulate work
			buf = append(buf, []byte("some data")...)
			SinkBytes = buf
			pool.Put(buf)
		}
	})
	
	b.Run("DirectAllocation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := make([]byte, 0, 1024)
			// Simulate work
			buf = append(buf, []byte("some data")...)
			SinkBytes = buf
		}
	})
}

// Benchmark ring buffer vs channel
func BenchmarkRingBufferVsChannel(b *testing.B) {
	const bufferSize = 1024
	
	b.Run("RingBuffer", func(b *testing.B) {
		rb := NewRingBuffer(bufferSize)
		data := make([]byte, 64)
		readBuf := make([]byte, 64)
		
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Write data
			data[0] = byte(i)
			written := rb.Write(data)
			
			// Read data if available
			if rb.Available() >= 64 {
				read := rb.Read(readBuf)
				SinkInt = written + read
			}
		}
	})
	
	b.Run("Channel", func(b *testing.B) {
		ch := make(chan []byte, bufferSize/64) // Approximate same capacity
		data := make([]byte, 64)
		
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Write data
			data[0] = byte(i)
			select {
			case ch <- data:
				// Successfully sent
			default:
				// Channel full, try to read
				select {
				case received := <-ch:
					SinkBytes = received
					ch <- data // Try to send again
				default:
					// Still full, skip
				}
			}
		}
	})
}

// Benchmark object pool with different pool sizes
func BenchmarkObjectPoolSizes(b *testing.B) {
	poolSizes := []int{10, 50, 100, 500}
	
	for _, poolSize := range poolSizes {
		b.Run(fmt.Sprintf("PoolSize_%d", poolSize), func(b *testing.B) {
			pool := NewObjectPool(poolSize, func() interface{} {
				return make([]byte, 256)
			})
			
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				obj := pool.Get().([]byte)
				// Simulate work
				obj[0] = byte(i)
				pool.Put(obj)
			}
		})
	}
}

// Test concurrent access to object pool
func TestObjectPoolConcurrent(t *testing.T) {
	pool := NewObjectPool(100, func() interface{} {
		return make([]byte, 64)
	})
	
	const numGoroutines = 10
	const operationsPerGoroutine = 1000
	
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < operationsPerGoroutine; j++ {
				obj := pool.Get().([]byte)
				// Simulate some work
				obj[0] = byte(id)
				obj[1] = byte(j)
				time.Sleep(time.Microsecond) // Small delay to increase contention
				pool.Put(obj)
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	
	// Pool should still be functional
	obj := pool.Get().([]byte)
	if len(obj) != 64 {
		t.Errorf("Expected object length 64, got %d", len(obj))
	}
	pool.Put(obj)
}

// Test memory arena with concurrent access
func TestMemoryArenaConcurrent(t *testing.T) {
	// Note: This test assumes the arena is NOT thread-safe
	// and is just testing that it doesn't crash under concurrent access
	arena := NewMemoryArena(4096)
	
	const numGoroutines = 5
	const allocationsPerGoroutine = 100
	
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < allocationsPerGoroutine; j++ {
				ptr := arena.Allocate(32)
				if ptr != nil {
					// Write to the allocated memory
					*(*byte)(ptr) = byte(id)
				}
				if j%50 == 0 {
					arena.Reset()
				}
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	
	// Arena should still be functional
	ptr := arena.Allocate(64)
	if ptr == nil {
		t.Error("Arena should still be functional after concurrent access")
	}
}

// Additional sink variables for this test file
var (
	SinkPtr unsafe.Pointer
	SinkBytes []byte
	SinkString string
	SinkInt int
)
package main

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"
)

// BenchmarkBufferPool benchmarks buffer pool performance
func BenchmarkBufferPool(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		pool := NewBufferPool()
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					buf := pool.GetBuffer(1024)
					
					// Simulate usage
					for i := 0; i < 100; i++ {
						buf = append(buf, byte(i))
					}
					
					pool.Put(buf)
				}
			})
	})
	
	b.Run("WithoutPool", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := make([]byte, 0, 1024)
				
				// Simulate usage
				for i := 0; i < 100; i++ {
					buf = append(buf, byte(i))
				}
				
				_ = buf
			}
		})
	})
	
	b.Run("SyncPool", func(b *testing.B) {
		pool := &sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 1024)
			},
		}
		
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := pool.Get().([]byte)
				buf = buf[:0]
				
				// Simulate usage
				for i := 0; i < 100; i++ {
					buf = append(buf, byte(i))
				}
				
				pool.Put(buf)
			}
		})
	})
}

// BenchmarkStringBuilderPool benchmarks string builder pool performance
func BenchmarkStringBuilderPool(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		pool := NewStringBuilderPool(1024)
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				sb := pool.Get().(*strings.Builder)
				
				// Simulate usage
				for i := 0; i < 10; i++ {
					sb.WriteString("hello world ")
				}
				
				result := sb.String()
				_ = result
				
				pool.Put(sb)
			}
		})
	})
	
	b.Run("WithoutPool", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var sb strings.Builder
				sb.Grow(1024)
				
				// Simulate usage
				for i := 0; i < 10; i++ {
					sb.WriteString("hello world ")
				}
				
				result := sb.String()
				_ = result
			}
		})
	})
	
	b.Run("StringConcatenation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var result string
				
				// Simulate usage
				for i := 0; i < 10; i++ {
					result += "hello world "
				}
				
				_ = result
			}
		})
	})
}

// BenchmarkCustomAllocators benchmarks custom allocator performance
func BenchmarkCustomAllocators(b *testing.B) {
	b.Run("FixedSizePool", func(b *testing.B) {
		pool := NewFixedSizePool(64, 1000)
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			ptr := pool.Allocate(64)
			if ptr != nil {
				pool.Deallocate(ptr)
			}
		}
	})
	
	b.Run("SizeClassAllocator", func(b *testing.B) {
		allocator := NewSizeClassAllocator()
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			size := uintptr(32 + (i%8)*16) // Varying sizes
			ptr := allocator.Allocate(size)
			if ptr != nil {
				allocator.Deallocate(ptr)
			}
		}
	})
	
	b.Run("StackAllocator", func(b *testing.B) {
		allocator := NewStackAllocator(1024 * 1024) // 1MB
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			allocator.PushMarker()
			
			// Allocate several objects
			for j := 0; j < 10; j++ {
				allocator.Allocate(64)
			}
			
			allocator.PopMarker()
		}
	})
	
	b.Run("StandardAllocation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			_ = make([]byte, 64)
		}
	})
}

// BenchmarkWorkerPool benchmarks worker pool performance
func BenchmarkWorkerPool(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		pool := NewWorkerPool(10, 1024)
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					worker := pool.Get().(*Worker)
					
					// Simulate work
					worker.buffer = append(worker.buffer[:0], []byte("work data")...)
					
					pool.Put(worker)
				}
			})
	})
	
	b.Run("WithoutPool", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				worker := &Worker{
					buffer: make([]byte, 0, 1024),
					tasks:  make(chan func(), 1),
					done:   make(chan bool, 1),
				}
				
				// Simulate work
				worker.buffer = append(worker.buffer, []byte("work data")...)
				
				_ = worker
			}
		})
	})
}

// BenchmarkConnectionPool benchmarks connection pool performance
func BenchmarkConnectionPool(b *testing.B) {
	createConn := func() *Connection {
		return &Connection{
			data: make(map[string]interface{}),
		}
	}
	
	b.Run("WithPool", func(b *testing.B) {
		pool := NewConnectionPool(5, createConn)
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					conn := pool.Get().(*Connection)
					
					// Simulate usage
					conn.data["key"] = "value"
					
					pool.Put(conn)
				}
			})
	})
	
	b.Run("WithoutPool", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				conn := createConn()
				
				// Simulate usage
				conn.data["key"] = "value"
				
				_ = conn
			}
		})
	})
}

// BenchmarkPoolContention benchmarks pool performance under contention
func BenchmarkPoolContention(b *testing.B) {
	goroutineCounts := []int{1, 2, 4, 8, 16, 32}
	
	for _, numGoroutines := range goroutineCounts {
		b.Run(fmt.Sprintf("Goroutines_%d", numGoroutines), func(b *testing.B) {
			pool := NewBufferPool()
			b.ResetTimer()
			b.ReportAllocs()
			
			b.SetParallelism(numGoroutines)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					buf := pool.GetBuffer(1024)
					
					// Simulate work
					for i := 0; i < 50; i++ {
						buf = append(buf, byte(i))
					}
					
					pool.Put(buf)
				}
			})
		})
	}
}

// BenchmarkPoolSizes benchmarks different pool sizes
func BenchmarkPoolSizes(b *testing.B) {
	sizes := []int{64, 128, 256, 512, 1024, 2048, 4096}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			pool := NewBufferPool()
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				buf := pool.GetBuffer(size)
				
				// Simulate usage
				for j := 0; j < size/10; j++ {
					buf = append(buf, byte(j))
				}
				
				pool.Put(buf)
			}
		})
	}
}

// BenchmarkMemoryUsage benchmarks memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("HighAllocationRate", func(b *testing.B) {
		pool := NewBufferPool()
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			// Allocate many buffers quickly
				var buffers [][]byte
				for j := 0; j < 100; j++ {
					buf := pool.GetBuffer(256)
					buffers = append(buffers, buf)
				}
			
			// Return them all
			for _, buf := range buffers {
				pool.Put(buf)
			}
		}
	})
	
	b.Run("LowAllocationRate", func(b *testing.B) {
		pool := NewBufferPool()
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
				buf := pool.GetBuffer(256)
				
				// Simulate longer usage
				time.Sleep(time.Microsecond)
				
				pool.Put(buf)
			}
	})
}

// TestPoolCorrectness tests pool correctness
func TestPoolCorrectness(t *testing.T) {
	t.Run("BufferPool", func(t *testing.T) {
		pool := NewBufferPool()
		
		// Test basic functionality
		buf1 := pool.GetBuffer(100)
		if cap(buf1) < 100 {
			t.Errorf("Buffer capacity %d < requested %d", cap(buf1), 100)
		}
		
		// Test reuse
		pool.Put(buf1)
		buf2 := pool.GetBuffer(100)
		
		// Should reuse the same buffer
		if cap(buf2) < 100 {
			t.Errorf("Reused buffer capacity %d < requested %d", cap(buf2), 100)
		}
		
		// Test statistics
		stats := pool.Stats()
		if stats.Gets < 2 {
			t.Errorf("Expected at least 2 gets, got %d", stats.Gets)
		}
	})
	
	t.Run("StringBuilderPool", func(t *testing.T) {
		pool := NewStringBuilderPool(100)
		
		sb1 := pool.Get().(*strings.Builder)
		sb1.WriteString("test")
		result1 := sb1.String()
		
		pool.Put(sb1)
		
		sb2 := pool.Get().(*strings.Builder)
		if sb2.Len() != 0 {
			t.Errorf("Expected reset string builder, got length %d", sb2.Len())
		}
		
		sb2.WriteString("test2")
		result2 := sb2.String()
		
		if result1 == result2 {
			t.Error("String builders should produce different results")
		}
		
		pool.Put(sb2)
	})
	
	t.Run("FixedSizePool", func(t *testing.T) {
		pool := NewFixedSizePool(64, 10)
		
		// Allocate all blocks
		var ptrs []unsafe.Pointer
		for i := 0; i < 10; i++ {
			ptr := pool.Allocate(64)
			if ptr == nil {
				t.Errorf("Failed to allocate block %d", i)
			}
			ptrs = append(ptrs, ptr)
		}
		
		// Should fail to allocate more
		ptr := pool.Allocate(64)
		if ptr != nil {
			t.Error("Should have failed to allocate when pool is full")
		}
		
		// Deallocate one and try again
		pool.Deallocate(ptrs[0])
		ptr = pool.Allocate(64)
		if ptr == nil {
			t.Error("Should have succeeded after deallocation")
		}
	})
	
	t.Run("StackAllocator", func(t *testing.T) {
		allocator := NewStackAllocator(1024)
		
		// Test basic allocation
		ptr1 := allocator.Allocate(100)
		if ptr1 == nil {
			t.Error("Failed to allocate from stack")
		}
		
		// Test marker functionality
		allocator.PushMarker()
		ptr2 := allocator.Allocate(200)
		ptr3 := allocator.Allocate(300)
		
		if ptr2 == nil || ptr3 == nil {
			t.Error("Failed to allocate after marker")
		}
		
		// Pop marker should free ptr2 and ptr3
		allocator.PopMarker()
		
		// Should be able to allocate in the freed space
		ptr4 := allocator.Allocate(400)
		if ptr4 == nil {
			t.Error("Failed to allocate after pop marker")
		}
	})
}

// TestPoolStats tests pool statistics accuracy
func TestPoolStats(t *testing.T) {
	pool := NewBufferPool()
	
	// Perform operations
		for i := 0; i < 10; i++ {
			buf := pool.GetBuffer(256)
			pool.Put(buf)
		}
	
	stats := pool.Stats()
	
	if stats.Gets != 10 {
		t.Errorf("Expected 10 gets, got %d", stats.Gets)
	}
	
	if stats.Puts != 10 {
		t.Errorf("Expected 10 puts, got %d", stats.Puts)
	}
	
	if stats.HitRate <= 0 {
		t.Errorf("Expected positive hit rate, got %f", stats.HitRate)
	}
}

// TestConcurrentAccess tests concurrent pool access
func TestConcurrentAccess(t *testing.T) {
	pool := NewBufferPool()
	const numGoroutines = 100
	const operationsPerGoroutine = 100
	
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			
			for j := 0; j < operationsPerGoroutine; j++ {
				buf := pool.GetBuffer(256)
				
				// Simulate work
				for k := 0; k < 10; k++ {
					buf = append(buf, byte(k))
				}
				
				pool.Put(buf)
			}
		}()
	}
	
	wg.Wait()
	
	stats := pool.Stats()
	expectedOps := int64(numGoroutines * operationsPerGoroutine)
	
	if stats.Gets != expectedOps {
		t.Errorf("Expected %d gets, got %d", expectedOps, stats.Gets)
	}
	
	if stats.Puts != expectedOps {
		t.Errorf("Expected %d puts, got %d", expectedOps, stats.Puts)
	}
}

// TestMemoryLeaks tests for memory leaks in pools
func TestMemoryLeaks(t *testing.T) {
	var m1, m2 runtime.MemStats
	
	// Measure initial memory
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	// Create and use pools
	pool := NewBufferPool()
	for i := 0; i < 10000; i++ {
		buf := pool.GetBuffer(1024)
		pool.Put(buf)
	}
	
	// Force GC and measure again
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	// Check for excessive memory growth
	memoryGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	if memoryGrowth > 1024*1024 { // 1MB threshold
		t.Errorf("Excessive memory growth: %d bytes", memoryGrowth)
	}
	
	t.Logf("Memory growth: %d bytes", memoryGrowth)
}

// Example of running benchmarks with memory profiling:
// go test -bench=. -benchmem -memprofile=mem.prof
// go tool pprof mem.prof
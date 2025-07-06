package main

import (
	"runtime"
	"sync"
	"testing"
)

const (
	testBufferSize = 1024
	testPoolSize   = 100
	testIterations = 10000
)

// Benchmark no pooling (baseline)
func BenchmarkNoPooling(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := make([]byte, testBufferSize)
			// Simulate some work
			for i := 0; i < len(buf); i += 64 {
				buf[i] = byte(i)
			}
			_ = buf
		}
	})
}

// Benchmark sync.Pool-based pooling
func BenchmarkSyncPooling(b *testing.B) {
	pool := NewSyncBufferPool()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get(testBufferSize)
			// Simulate some work
			for i := 0; i < len(buf); i += 64 {
				buf[i] = byte(i)
			}
			pool.Put(buf)
		}
	})
}

// Benchmark custom buffer pooling
func BenchmarkCustomPooling(b *testing.B) {
	pool := NewCustomBufferPool(testBufferSize, testPoolSize)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get(testBufferSize)
			// Simulate some work
			for i := 0; i < len(buf); i += 64 {
				buf[i] = byte(i)
			}
			pool.Put(buf)
		}
	})
}

// Benchmark dynamic buffer pooling
func BenchmarkDynamicPooling(b *testing.B) {
	pool := NewDynamicBufferPool(testPoolSize)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get(testBufferSize)
			// Simulate some work
			for i := 0; i < len(buf); i += 64 {
				buf[i] = byte(i)
			}
			pool.Put(buf)
		}
	})
}

// Benchmark variable size allocations
func BenchmarkVariableSizeNoPooling(b *testing.B) {
	sizes := []int{64, 128, 256, 512, 1024, 2048, 4096}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			size := sizes[i%len(sizes)]
			buf := make([]byte, size)
			// Simulate some work
			for j := 0; j < len(buf); j += 64 {
				buf[j] = byte(j)
			}
			_ = buf
			i++
		}
	})
}

func BenchmarkVariableSizeSyncPooling(b *testing.B) {
	pool := NewSyncBufferPool()
	sizes := []int{64, 128, 256, 512, 1024, 2048, 4096}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			size := sizes[i%len(sizes)]
			buf := pool.Get(size)
			// Simulate some work
			for j := 0; j < len(buf); j += 64 {
				buf[j] = byte(j)
			}
			pool.Put(buf)
			i++
		}
	})
}

func BenchmarkVariableSizeDynamicPooling(b *testing.B) {
	pool := NewDynamicBufferPool(50)
	sizes := []int{64, 128, 256, 512, 1024, 2048, 4096}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			size := sizes[i%len(sizes)]
			buf := pool.Get(size)
			// Simulate some work
			for j := 0; j < len(buf); j += 64 {
				buf[j] = byte(j)
			}
			pool.Put(buf)
			i++
		}
	})
}

// Test pool correctness
func TestSyncBufferPool(t *testing.T) {
	pool := NewSyncBufferPool()
	
	// Test basic get/put
	buf1 := pool.Get(100)
	if len(buf1) != 100 {
		t.Errorf("Expected buffer length 100, got %d", len(buf1))
	}
	
	// Modify buffer
	buf1[0] = 42
	pool.Put(buf1)
	
	// Get another buffer and verify it's clean
	buf2 := pool.Get(100)
	if buf2[0] != 0 {
		t.Errorf("Expected clean buffer, got %d at index 0", buf2[0])
	}
	pool.Put(buf2)
	
	// Test stats
	stats := pool.Stats()
	if stats.Hits == 0 && stats.Misses == 0 {
		t.Error("Expected some hits or misses in stats")
	}
}

func TestCustomBufferPool(t *testing.T) {
	pool := NewCustomBufferPool(1024, 10)
	
	// Test basic functionality
	buf := pool.Get(512)
	if len(buf) != 512 {
		t.Errorf("Expected buffer length 512, got %d", len(buf))
	}
	
	buf[0] = 42
	pool.Put(buf)
	
	// Get another buffer and verify it's clean
	buf2 := pool.Get(512)
	if buf2[0] != 0 {
		t.Errorf("Expected clean buffer, got %d at index 0", buf2[0])
	}
	pool.Put(buf2)
	
	// Test oversized request
	buf3 := pool.Get(2048)
	if len(buf3) != 2048 {
		t.Errorf("Expected buffer length 2048, got %d", len(buf3))
	}
	// This should not be accepted by the pool
	pool.Put(buf3)
}

func TestDynamicBufferPool(t *testing.T) {
	pool := NewDynamicBufferPool(5)
	
	// Test different sizes
	sizes := []int{64, 128, 256, 512, 1024}
	buffers := make([][]byte, len(sizes))
	
	for i, size := range sizes {
		buffers[i] = pool.Get(size)
		if len(buffers[i]) != size {
			t.Errorf("Expected buffer length %d, got %d", size, len(buffers[i]))
		}
		buffers[i][0] = byte(i)
	}
	
	// Return all buffers
	for _, buf := range buffers {
		pool.Put(buf)
	}
	
	// Get buffers again and verify they're clean
	for _, size := range sizes {
		buf := pool.Get(size)
		if buf[0] != 0 {
			t.Errorf("Expected clean buffer for size %d, got %d at index 0", size, buf[0])
		}
		pool.Put(buf)
	}
}

// Benchmark memory allocation patterns
func BenchmarkMemoryPressure(b *testing.B) {
	pool := NewSyncBufferPool()
	
	b.Run("NoPooling", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := make([]byte, 1024)
			_ = buf
			_ = i // Use the variable to avoid unused variable error
		}
		b.StopTimer()
		
		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc), "bytes/op")
	})
	
	b.Run("WithPooling", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := pool.Get(1024)
			pool.Put(buf)
			_ = i // Use the variable to avoid unused variable error
		}
		b.StopTimer()
		
		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc), "bytes/op")
	})
}

// Test concurrent access
func TestConcurrentAccess(t *testing.T) {
	pool := NewSyncBufferPool()
	const numGoroutines = 100
	const numOperations = 1000
	
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			_ = goroutineID // Use the parameter to avoid unused variable error
			for j := 0; j < numOperations; j++ {
				buf := pool.Get(512)
				// Simulate work
				for k := 0; k < len(buf); k++ {
					buf[k] = byte(k)
				}
				pool.Put(buf)
			}
		}(i)
	}
	
	wg.Wait()
	
	stats := pool.Stats()
	expectedOperations := int64(numGoroutines * numOperations)
	totalOperations := stats.Hits + stats.Misses
	
	if totalOperations != expectedOperations {
		t.Errorf("Expected %d total operations, got %d", expectedOperations, totalOperations)
	}
}
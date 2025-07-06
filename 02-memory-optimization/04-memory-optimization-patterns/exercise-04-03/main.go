package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

func main() {
	fmt.Println("=== Buffer Pooling Strategies Demo ===")
	fmt.Println()
	
	// Demonstrate different pooling strategies
	demonstratePoolingStrategies()
	
	// Show pool statistics
	demonstratePoolStatistics()
	
	// Memory usage comparison
	demonstrateMemoryUsage()
	
	// Concurrent access demonstration
	demonstrateConcurrentAccess()
}

func demonstratePoolingStrategies() {
	fmt.Println("1. Pooling Strategies Comparison")
	fmt.Println("================================")
	
	const iterations = 10000
	const bufferSize = 1024
	
	// No pooling baseline
	start := time.Now()
	for i := 0; i < iterations; i++ {
		buf := make([]byte, bufferSize)
		// Simulate work
		for j := 0; j < len(buf); j += 64 {
			buf[j] = byte(j)
		}
		_ = buf
	}
	noPoolTime := time.Since(start)
	fmt.Printf("No pooling: %v\n", noPoolTime)
	
	// sync.Pool strategy
	syncPool := NewSyncBufferPool()
	start = time.Now()
	for i := 0; i < iterations; i++ {
		buf := syncPool.Get(bufferSize)
		// Simulate work
		for j := 0; j < len(buf); j += 64 {
			buf[j] = byte(j)
		}
		syncPool.Put(buf)
	}
	syncPoolTime := time.Since(start)
	fmt.Printf("sync.Pool: %v (%.2fx faster)\n", syncPoolTime, float64(noPoolTime)/float64(syncPoolTime))
	
	// Custom pool strategy
	customPool := NewCustomBufferPool(bufferSize, 100)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		buf := customPool.Get(bufferSize)
		// Simulate work
		for j := 0; j < len(buf); j += 64 {
			buf[j] = byte(j)
		}
		customPool.Put(buf)
	}
	customPoolTime := time.Since(start)
	fmt.Printf("Custom pool: %v (%.2fx faster)\n", customPoolTime, float64(noPoolTime)/float64(customPoolTime))
	
	// Dynamic pool strategy
	dynamicPool := NewDynamicBufferPool(100)
	start = time.Now()
	for i := 0; i < iterations; i++ {
		buf := dynamicPool.Get(bufferSize)
		// Simulate work
		for j := 0; j < len(buf); j += 64 {
			buf[j] = byte(j)
		}
		dynamicPool.Put(buf)
	}
	dynamicPoolTime := time.Since(start)
	fmt.Printf("Dynamic pool: %v (%.2fx faster)\n", dynamicPoolTime, float64(noPoolTime)/float64(dynamicPoolTime))
	
	fmt.Println()
}

func demonstratePoolStatistics() {
	fmt.Println("2. Pool Statistics Analysis")
	fmt.Println("===========================")
	
	pools := map[string]BufferPool{
		"sync.Pool":    NewSyncBufferPool(),
		"Custom Pool":  NewCustomBufferPool(1024, 50),
		"Dynamic Pool": NewDynamicBufferPool(50),
	}
	
	const operations = 5000
	sizes := []int{64, 128, 256, 512, 1024, 2048}
	
	for name, pool := range pools {
		fmt.Printf("\n%s:\n", name)
		
		// Perform operations with varying sizes
		for i := 0; i < operations; i++ {
			size := sizes[i%len(sizes)]
			buf := pool.Get(size)
			// Simulate work
			for j := 0; j < len(buf); j += 32 {
				buf[j] = byte(j)
			}
			pool.Put(buf)
		}
		
		stats := pool.Stats()
		hitRate := float64(stats.Hits) / float64(stats.Hits+stats.Misses) * 100
		
		fmt.Printf("  Hits: %d\n", stats.Hits)
		fmt.Printf("  Misses: %d\n", stats.Misses)
		fmt.Printf("  Hit Rate: %.2f%%\n", hitRate)
		fmt.Printf("  Allocations: %d\n", stats.Allocations)
		fmt.Printf("  Pool Size: %d\n", stats.PoolSize)
	}
	
	fmt.Println()
}

func demonstrateMemoryUsage() {
	fmt.Println("3. Memory Usage Comparison")
	fmt.Println("==========================")
	
	const iterations = 10000
	const bufferSize = 1024
	
	// Measure memory without pooling
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	for i := 0; i < iterations; i++ {
		buf := make([]byte, bufferSize)
		_ = buf
	}
	
	runtime.GC()
	runtime.ReadMemStats(&m2)
	noPoolAllocs := m2.TotalAlloc - m1.TotalAlloc
	
	fmt.Printf("No pooling: %d bytes allocated\n", noPoolAllocs)
	
	// Measure memory with pooling
	pool := NewSyncBufferPool()
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	for i := 0; i < iterations; i++ {
		buf := pool.Get(bufferSize)
		pool.Put(buf)
	}
	
	runtime.GC()
	runtime.ReadMemStats(&m2)
	poolAllocs := m2.TotalAlloc - m1.TotalAlloc
	
	fmt.Printf("With pooling: %d bytes allocated\n", poolAllocs)
	fmt.Printf("Reduction: %.2f%% (%.2fx less)\n", 
		float64(noPoolAllocs-poolAllocs)/float64(noPoolAllocs)*100,
		float64(noPoolAllocs)/float64(poolAllocs))
	
	fmt.Println()
}

func demonstrateConcurrentAccess() {
	fmt.Println("4. Concurrent Access Performance")
	fmt.Println("===============================")
	
	const numGoroutines = 10
	const operationsPerGoroutine = 1000
	const bufferSize = 1024
	
	// Test without pooling
	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				buf := make([]byte, bufferSize)
				// Simulate work
				for k := 0; k < len(buf); k += 64 {
					buf[k] = byte(k)
				}
				_ = buf
			}
		}()
	}
	
	wg.Wait()
	noPoolConcurrent := time.Since(start)
	fmt.Printf("No pooling (concurrent): %v\n", noPoolConcurrent)
	
	// Test with pooling
	pool := NewSyncBufferPool()
	start = time.Now()
	wg.Add(numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				buf := pool.Get(bufferSize)
				// Simulate work
				for k := 0; k < len(buf); k += 64 {
					buf[k] = byte(k)
				}
				pool.Put(buf)
			}
		}()
	}
	
	wg.Wait()
	poolConcurrent := time.Since(start)
	fmt.Printf("With pooling (concurrent): %v\n", poolConcurrent)
	fmt.Printf("Improvement: %.2fx faster\n", float64(noPoolConcurrent)/float64(poolConcurrent))
	
	// Show final pool statistics
	stats := pool.Stats()
	hitRate := float64(stats.Hits) / float64(stats.Hits+stats.Misses) * 100
	fmt.Printf("\nFinal pool statistics:\n")
	fmt.Printf("  Hit rate: %.2f%%\n", hitRate)
	fmt.Printf("  Total allocations: %d\n", stats.Allocations)
	fmt.Printf("  Pool size classes: %d\n", stats.PoolSize)
	
	fmt.Println()
	fmt.Println("=== Demo Complete ===")
}
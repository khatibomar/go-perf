package main

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// BenchmarkGCPressureOptimizations benchmarks different GC pressure optimization techniques
func BenchmarkGCPressureOptimizations(b *testing.B) {
	tests := []struct {
		name        string
		optimized   func(int)
		unoptimized func(int)
	}{
		{
			name: "StringConcatenation",
			optimized: func(n int) {
				var builder strings.Builder
				builder.Grow(n * 10) // Pre-allocate
				for i := 0; i < n; i++ {
					builder.WriteString(fmt.Sprintf("item_%d_", i))
				}
				_ = builder.String()
			},
			unoptimized: func(n int) {
				result := ""
				for i := 0; i < n; i++ {
					result += fmt.Sprintf("item_%d_", i)
				}
				_ = result
			},
		},
		{
			name: "SliceGrowth",
			optimized: func(n int) {
				slice := make([]int, 0, n) // Pre-allocate capacity
				for i := 0; i < n; i++ {
					slice = append(slice, i)
				}
				_ = slice
			},
			unoptimized: func(n int) {
				var slice []int // No pre-allocation
				for i := 0; i < n; i++ {
					slice = append(slice, i)
				}
				_ = slice
			},
		},
		{
			name: "MapAllocations",
			optimized: func(n int) {
				m := make(map[int]string, n) // Pre-size map
				for i := 0; i < n; i++ {
					m[i] = fmt.Sprintf("value_%d", i)
				}
				_ = m
			},
			unoptimized: func(n int) {
				m := make(map[int]string) // No pre-sizing
				for i := 0; i < n; i++ {
					m[i] = fmt.Sprintf("value_%d", i)
				}
				_ = m
			},
		},
		{
			name: "BufferReuse",
			optimized: func(n int) {
				buffer := make([]byte, 1024)
				for i := 0; i < n; i++ {
					// Reuse buffer
					for j := range buffer {
						buffer[j] = byte(i % 256)
					}
					_ = buffer
				}
			},
			unoptimized: func(n int) {
				for i := 0; i < n; i++ {
					// Allocate new buffer each time
					buffer := make([]byte, 1024)
					for j := range buffer {
						buffer[j] = byte(i % 256)
					}
					_ = buffer
				}
			},
		},
	}
	
	for _, tt := range tests {
		b.Run(tt.name+"_Optimized", func(b *testing.B) {
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tt.optimized(100)
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportGCMetrics(b, startStats, endStats)
		})
		
		b.Run(tt.name+"_Unoptimized", func(b *testing.B) {
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tt.unoptimized(100)
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportGCMetrics(b, startStats, endStats)
		})
	}
}

// BenchmarkObjectPooling benchmarks object pooling vs direct allocation
func BenchmarkObjectPooling(b *testing.B) {
	// Buffer pool benchmark
	b.Run("BufferPool", func(b *testing.B) {
		pool := NewBufferPool([]int{64, 256, 1024, 4096}, 8192)
		
		b.Run("WithPool", func(b *testing.B) {
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buffer := pool.GetBuffer(1024)
				// Use buffer
				for j := range buffer {
					buffer[j] = byte(j % 256)
				}
				pool.PutBuffer(buffer)
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportGCMetrics(b, startStats, endStats)
			b.ReportMetric(pool.Stats().HitRate, "hit-rate-%")
		})
		
		b.Run("WithoutPool", func(b *testing.B) {
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buffer := make([]byte, 1024)
				// Use buffer
				for j := range buffer {
					buffer[j] = byte(j % 256)
				}
				_ = buffer
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportGCMetrics(b, startStats, endStats)
		})
	})
	
	// String builder pool benchmark
	b.Run("StringBuilderPool", func(b *testing.B) {
		pool := NewStringBuilderPool(4096)
		
		b.Run("WithPool", func(b *testing.B) {
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				builder := pool.GetBuilder()
				for j := 0; j < 10; j++ {
					builder.WriteString(fmt.Sprintf("item_%d_", j))
				}
				result := builder.String()
				_ = result
				pool.PutBuilder(builder)
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportGCMetrics(b, startStats, endStats)
			b.ReportMetric(pool.Stats().HitRate, "hit-rate-%")
		})
		
		b.Run("WithoutPool", func(b *testing.B) {
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var builder strings.Builder
				for j := 0; j < 10; j++ {
					builder.WriteString(fmt.Sprintf("item_%d_", j))
				}
				result := builder.String()
				_ = result
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportGCMetrics(b, startStats, endStats)
		})
	})
	
	// Object pool benchmark
	b.Run("ObjectPool", func(b *testing.B) {
		type TestObject struct {
			Data []int
			Name string
		}
		
		pool := NewObjectPool(func() interface{} {
			return &TestObject{
				Data: make([]int, 100),
				Name: "",
			}
		})
		
		pool.SetReset(func(obj interface{}) {
			test := obj.(*TestObject)
			for i := range test.Data {
				test.Data[i] = 0
			}
			test.Name = ""
		})
		
		b.Run("WithPool", func(b *testing.B) {
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				obj := pool.Get().(*TestObject)
				obj.Name = fmt.Sprintf("object_%d", i)
				for j := range obj.Data {
					obj.Data[j] = j
				}
				pool.Put(obj)
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportGCMetrics(b, startStats, endStats)
			b.ReportMetric(pool.Stats().HitRate, "hit-rate-%")
		})
		
		b.Run("WithoutPool", func(b *testing.B) {
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				obj := &TestObject{
					Data: make([]int, 100),
					Name: fmt.Sprintf("object_%d", i),
				}
				for j := range obj.Data {
					obj.Data[j] = j
				}
				_ = obj
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportGCMetrics(b, startStats, endStats)
		})
	})
}

// BenchmarkBatchProcessing benchmarks batch processing vs individual processing
func BenchmarkBatchProcessing(b *testing.B) {
	batchSizes := []int{1, 10, 50, 100, 500}
	
	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Process data in batches
				data := make([]int, 1000)
				for j := range data {
					data[j] = j
				}
				
				for start := 0; start < len(data); start += batchSize {
					end := start + batchSize
					if end > len(data) {
						end = len(data)
					}
					
					batch := data[start:end]
					// Process batch
					sum := 0
					for _, val := range batch {
						sum += val
					}
					_ = sum
				}
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportGCMetrics(b, startStats, endStats)
		})
	}
}

// BenchmarkLazyInitialization benchmarks lazy vs eager initialization
func BenchmarkLazyInitialization(b *testing.B) {
	b.Run("LazyInitialization", func(b *testing.B) {
		type LazyResource struct {
			data []byte
			once sync.Once
		}
		
		var startStats, endStats runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&startStats)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resource := &LazyResource{}
			
			// Only initialize when needed (10% of the time)
			if i%10 == 0 {
				resource.once.Do(func() {
					resource.data = make([]byte, 1024)
				})
				_ = resource.data
			}
		}
		b.StopTimer()
		
		runtime.ReadMemStats(&endStats)
		reportGCMetrics(b, startStats, endStats)
	})
	
	b.Run("EagerInitialization", func(b *testing.B) {
		type EagerResource struct {
			data []byte
		}
		
		var startStats, endStats runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&startStats)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resource := &EagerResource{
				data: make([]byte, 1024), // Always initialize
			}
			
			// Only use 10% of the time
			if i%10 == 0 {
				_ = resource.data
			}
		}
		b.StopTimer()
		
		runtime.ReadMemStats(&endStats)
		reportGCMetrics(b, startStats, endStats)
	})
}

// BenchmarkConcurrentGCPressure benchmarks GC pressure under concurrent load
func BenchmarkConcurrentGCPressure(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8, 16}
	
	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d_Optimized", workers), func(b *testing.B) {
			// Use object pooling for concurrent allocations
			pool := NewBufferPool([]int{1024}, 8192)
			
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				for w := 0; w < workers; w++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for j := 0; j < 100/workers; j++ {
							buffer := pool.GetBuffer(1024)
							// Use buffer
							for k := range buffer {
								buffer[k] = byte(k % 256)
							}
							pool.PutBuffer(buffer)
						}
					}()
				}
				wg.Wait()
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportGCMetrics(b, startStats, endStats)
			b.ReportMetric(pool.Stats().HitRate, "hit-rate-%")
		})
		
		b.Run(fmt.Sprintf("Workers_%d_Unoptimized", workers), func(b *testing.B) {
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				for w := 0; w < workers; w++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for j := 0; j < 100/workers; j++ {
							buffer := make([]byte, 1024)
							// Use buffer
							for k := range buffer {
								buffer[k] = byte(k % 256)
							}
							_ = buffer
						}
					}()
				}
				wg.Wait()
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportGCMetrics(b, startStats, endStats)
		})
	}
}

// BenchmarkRealWorldScenarios benchmarks realistic GC pressure scenarios
func BenchmarkRealWorldScenarios(b *testing.B) {
	b.Run("WebServerOptimized", func(b *testing.B) {
		// Optimized web server simulation with pooling
		bufferPool := NewBufferPool([]int{2048, 4096}, 8192)
		stringPool := NewStringBuilderPool(4096)
		
		var startStats, endStats runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&startStats)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate HTTP request processing
			request := bufferPool.GetBuffer(2048)
			
			// Process headers
			headers := make(map[string]string, 10) // Pre-size
			for j := 0; j < 10; j++ {
				headers[fmt.Sprintf("header-%d", j)] = fmt.Sprintf("value-%d", j)
			}
			
			// Build response
			builder := stringPool.GetBuilder()
			builder.WriteString("HTTP/1.1 200 OK\r\n")
			for k, v := range headers {
				builder.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
			}
			builder.WriteString("\r\n")
			response := builder.String()
			
			_ = request
			_ = response
			
			bufferPool.PutBuffer(request)
			stringPool.PutBuilder(builder)
		}
		b.StopTimer()
		
		runtime.ReadMemStats(&endStats)
		reportGCMetrics(b, startStats, endStats)
	})
	
	b.Run("WebServerUnoptimized", func(b *testing.B) {
		var startStats, endStats runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&startStats)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate HTTP request processing without pooling
			request := make([]byte, 2048)
			
			// Process headers
			headers := make(map[string]string)
			for j := 0; j < 10; j++ {
				headers[fmt.Sprintf("header-%d", j)] = fmt.Sprintf("value-%d", j)
			}
			
			// Build response
			response := "HTTP/1.1 200 OK\r\n"
			for k, v := range headers {
				response += fmt.Sprintf("%s: %s\r\n", k, v)
			}
			response += "\r\n"
			
			_ = request
			_ = response
		}
		b.StopTimer()
		
		runtime.ReadMemStats(&endStats)
		reportGCMetrics(b, startStats, endStats)
	})
	
	b.Run("JSONProcessingOptimized", func(b *testing.B) {
		// Optimized JSON processing with pre-allocated structures
		type JSONData struct {
			Fields map[string]interface{}
		}
		
		pool := NewObjectPool(func() interface{} {
			return &JSONData{
				Fields: make(map[string]interface{}, 20),
			}
		})
		
		pool.SetReset(func(obj interface{}) {
			data := obj.(*JSONData)
			for k := range data.Fields {
				delete(data.Fields, k)
			}
		})
		
		var startStats, endStats runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&startStats)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data := pool.Get().(*JSONData)
			
			for j := 0; j < 20; j++ {
				key := fmt.Sprintf("field_%d", j)
				if j%3 == 0 {
					data.Fields[key] = fmt.Sprintf("string_value_%d", j)
				} else if j%3 == 1 {
					data.Fields[key] = j
				} else {
					data.Fields[key] = []int{j, j + 1, j + 2}
				}
			}
			
			pool.Put(data)
		}
		b.StopTimer()
		
		runtime.ReadMemStats(&endStats)
		reportGCMetrics(b, startStats, endStats)
	})
	
	b.Run("JSONProcessingUnoptimized", func(b *testing.B) {
		var startStats, endStats runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&startStats)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data := make(map[string]interface{})
			
			for j := 0; j < 20; j++ {
				key := fmt.Sprintf("field_%d", j)
				if j%3 == 0 {
					data[key] = fmt.Sprintf("string_value_%d", j)
				} else if j%3 == 1 {
					data[key] = j
				} else {
					data[key] = []int{j, j + 1, j + 2}
				}
			}
			
			_ = data
		}
		b.StopTimer()
		
		runtime.ReadMemStats(&endStats)
		reportGCMetrics(b, startStats, endStats)
	})
}

// reportGCMetrics reports GC-related metrics for benchmarks
func reportGCMetrics(b *testing.B, start, end runtime.MemStats) {
	gcCycles := end.NumGC - start.NumGC
	totalPause := end.PauseTotalNs - start.PauseTotalNs
	allocated := end.TotalAlloc - start.TotalAlloc
	heapSize := end.HeapSys
	heapInUse := end.HeapInuse
	
	b.ReportMetric(float64(gcCycles), "gc-cycles")
	b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
	b.ReportMetric(float64(allocated)/(1024*1024), "allocated-mb")
	b.ReportMetric(float64(heapSize)/(1024*1024), "heap-size-mb")
	b.ReportMetric(float64(heapInUse)/(1024*1024), "heap-inuse-mb")
	
	if gcCycles > 0 {
		b.ReportMetric(float64(totalPause)/float64(gcCycles)/1e6, "avg-pause-ms")
	}
}

// TestGCPressureOptimizer tests the GC pressure optimizer functionality
func TestGCPressureOptimizer(t *testing.T) {
	optimizer := NewGCPressureOptimizer()
	
	// Generate some GC pressure
	for i := 0; i < 1000; i++ {
		_ = make([]byte, 1024)
	}
	
	// Analyze pressure
	analysis := optimizer.AnalyzePressure()
	
	if analysis.PressureScore < 0 || analysis.PressureScore > 100 {
		t.Errorf("Invalid pressure score: %f", analysis.PressureScore)
	}
	
	if len(analysis.Hotspots) == 0 {
		t.Error("Expected hotspots to be identified")
	}
	
	if len(analysis.Optimizations) == 0 {
		t.Error("Expected optimizations to be generated")
	}
	
	if len(analysis.Recommendations) == 0 {
		t.Error("Expected recommendations to be generated")
	}
}

// TestObjectPooling tests object pooling functionality
func TestObjectPooling(t *testing.T) {
	// Test buffer pool
	bufferPool := NewBufferPool([]int{64, 256, 1024}, 2048)
	
	buffer := bufferPool.GetBuffer(512)
	if len(buffer) != 512 {
		t.Errorf("Expected buffer length 512, got %d", len(buffer))
	}
	
	bufferPool.PutBuffer(buffer)
	stats := bufferPool.Stats()
	if stats.Gets != 1 || stats.Puts != 1 {
		t.Errorf("Expected 1 get and 1 put, got %d gets and %d puts", stats.Gets, stats.Puts)
	}
	
	// Test string builder pool
	stringPool := NewStringBuilderPool(1024)
	
	builder := stringPool.GetBuilder()
	builder.WriteString("test")
	result := builder.String()
	if result != "test" {
		t.Errorf("Expected 'test', got '%s'", result)
	}
	
	stringPool.PutBuilder(builder)
	stringStats := stringPool.Stats()
	if stringStats.Gets != 1 || stringStats.Puts != 1 {
		t.Errorf("Expected 1 get and 1 put, got %d gets and %d puts", stringStats.Gets, stringStats.Puts)
	}
	
	// Test object pool
	type TestStruct struct {
		Value int
	}
	
	objectPool := NewObjectPool(func() interface{} {
		return &TestStruct{}
	})
	
	obj := objectPool.Get().(*TestStruct)
	obj.Value = 42
	objectPool.Put(obj)
	
	objectStats := objectPool.Stats()
	if objectStats.Gets != 1 || objectStats.Puts != 1 {
		t.Errorf("Expected 1 get and 1 put, got %d gets and %d puts", objectStats.Gets, objectStats.Puts)
	}
}

// TestConcurrentPoolAccess tests concurrent access to pools
func TestConcurrentPoolAccess(t *testing.T) {
	bufferPool := NewBufferPool([]int{1024}, 2048)
	
	var wg sync.WaitGroup
	workers := 10
	operations := 100
	
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				buffer := bufferPool.GetBuffer(1024)
				// Use buffer
				for k := range buffer {
					buffer[k] = byte(k % 256)
				}
				bufferPool.PutBuffer(buffer)
			}
		}()
	}
	
	wg.Wait()
	
	stats := bufferPool.Stats()
	expectedOps := workers * operations
	if int(stats.Gets) != expectedOps || int(stats.Puts) != expectedOps {
		t.Errorf("Expected %d gets and puts, got %d gets and %d puts", 
			expectedOps, stats.Gets, stats.Puts)
	}
}

// TestWorkerPool tests worker pool functionality
func TestWorkerPool(t *testing.T) {
	workerPool := NewWorkerPool(4, 100)
	defer workerPool.Stop()
	
	var wg sync.WaitGroup
	var counter int
	var mu sync.Mutex
	
	for i := 0; i < 100; i++ {
		wg.Add(1)
		workerPool.Submit(func() {
			defer wg.Done()
			mu.Lock()
			counter++
			mu.Unlock()
		})
	}
	
	wg.Wait()
	
	if counter != 100 {
		t.Errorf("Expected counter to be 100, got %d", counter)
	}
	
	stats := workerPool.Stats()
	if stats.Gets != 100 {
		t.Errorf("Expected 100 gets, got %d", stats.Gets)
	}
}

// TestMemoryLeakDetection tests for memory leaks in pools
func TestMemoryLeakDetection(t *testing.T) {
	var startStats, endStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startStats)
	
	// Create and use pools extensively
	bufferPool := NewBufferPool([]int{1024}, 2048)
	
	for i := 0; i < 10000; i++ {
		buffer := bufferPool.GetBuffer(1024)
		bufferPool.PutBuffer(buffer)
	}
	
	// Force GC and check for leaks
	runtime.GC()
	runtime.GC() // Double GC to ensure cleanup
	runtime.ReadMemStats(&endStats)
	
	// Use signed arithmetic to avoid wraparound
	heapGrowth := int64(endStats.HeapInuse) - int64(startStats.HeapInuse)
	
	// Allow some growth but not excessive (5MB threshold)
	if heapGrowth > 5*1024*1024 {
		t.Errorf("Potential memory leak detected: heap grew by %d bytes", heapGrowth)
	}
	
	// Log the growth for debugging
	t.Logf("Heap growth: %d bytes", heapGrowth)
}
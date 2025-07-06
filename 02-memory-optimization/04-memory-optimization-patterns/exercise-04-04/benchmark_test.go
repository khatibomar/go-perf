package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"exercise-04-04/implementations"
	"exercise-04-04/memory"
	"exercise-04-04/pipeline"
)

// BenchmarkPipelineWithPool benchmarks the pipeline with buffer pooling
func BenchmarkPipelineWithPool(b *testing.B) {
	benchmarkPipeline(b, true)
}

// BenchmarkPipelineWithoutPool benchmarks the pipeline without buffer pooling
func BenchmarkPipelineWithoutPool(b *testing.B) {
	benchmarkPipeline(b, false)
}

// benchmarkPipeline runs the pipeline benchmark
func benchmarkPipeline(b *testing.B, usePool bool) {
	// Create test data
	testData := strings.Repeat("This is a test line for benchmarking memory usage.\n", 1000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var pool *memory.BufferPool
		if usePool {
			pool = memory.NewBufferPool()
		} else {
			pool = &memory.BufferPool{} // Empty pool that doesn't reuse buffers
		}

		// Create pipeline components
		reader := implementations.NewStringReader(testData, pool)
		processor := implementations.NewTextProcessor(pool, implementations.UppercaseTransform)
		writer := implementations.NewBufferedWriter()

		// Create pipeline with small memory limit to test pressure handling
		config := &pipeline.Config{
			ChunkSize:       1024,
			BufferSize:      100,
			WorkerCount:     2,
			MemoryLimit:     10 * 1024 * 1024, // 10MB
			ProcessTimeout:  time.Second,
			BackpressureMax: 50,
		}

		pipe := pipeline.NewPipeline(reader, processor, writer, config)

		// Run pipeline
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pipe.Start(ctx)

		// Wait for completion
		for pipe.IsRunning() {
			time.Sleep(10 * time.Millisecond)
		}

		pipe.Stop()
		reader.Close()
		processor.Close()
		writer.Close()
		cancel()
	}
}

// BenchmarkBufferPool benchmarks buffer pool operations
func BenchmarkBufferPool(b *testing.B) {
	pool := memory.NewBufferPool()
	sizes := []int{64, 256, 1024, 4096, 8192}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, size := range sizes {
			buf := pool.Get(size)
			// Simulate some work
			for j := 0; j < len(buf) && j < 100; j++ {
				buf[j] = byte(j % 256)
			}
			pool.Put(buf)
		}
	}
}

// BenchmarkBufferPoolVsAlloc compares buffer pool vs direct allocation
func BenchmarkBufferPoolVsAlloc(b *testing.B) {
	b.Run("BufferPool", func(b *testing.B) {
		pool := memory.NewBufferPool()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			buf := pool.Get(1024)
			// Simulate work
			buf[0] = byte(i)
			pool.Put(buf)
		}
	})

	b.Run("DirectAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			buf := make([]byte, 1024)
			// Simulate work
			buf[0] = byte(i)
			// Buffer goes out of scope and becomes eligible for GC
			_ = buf
		}
	})
}

// BenchmarkTextProcessors benchmarks different text processors
func BenchmarkTextProcessors(b *testing.B) {
	pool := memory.NewBufferPool()
	testText := "This is a sample text for processing benchmarks with multiple words."

	processors := map[string]implementations.TransformFunc{
		"Uppercase": implementations.UppercaseTransform,
		"Lowercase": implementations.LowercaseTransform,
		"WordCount": implementations.WordCountTransform,
		"Reverse":   implementations.ReverseTransform,
		"TrimSpace": implementations.TrimSpaceTransform,
	}

	for name, transform := range processors {
		b.Run(name, func(b *testing.B) {
			processor := implementations.NewTextProcessor(pool, transform)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				chunk := &pipeline.Chunk{
					ID:       int64(i),
					Data:     []byte(testText),
					Metadata: make(map[string]interface{}),
				}

				ctx := context.Background()
				result, err := processor.Process(ctx, chunk)
				if err != nil {
					b.Fatal(err)
				}
				if result != nil {
					result.Release()
				}
			}
		})
	}
}

// BenchmarkMemoryManager benchmarks memory manager operations
func BenchmarkMemoryManager(b *testing.B) {
	manager := memory.NewManager(100 * 1024 * 1024) // 100MB limit

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate memory monitoring operations
		_ = manager.IsUnderPressure()
		_ = manager.GetUsage()
		_ = manager.GetLimit()

		// Occasionally trigger cleanup
		if i%1000 == 0 {
			manager.TriggerCleanup()
		}
	}
}

// BenchmarkConcurrentPipeline benchmarks pipeline with different worker counts
func BenchmarkConcurrentPipeline(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8}
	testData := strings.Repeat("Concurrent processing test line.\n", 500)

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("Workers%d", workers), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				pool := memory.NewBufferPool()
				reader := implementations.NewStringReader(testData, pool)
				processor := implementations.NewTextProcessor(pool, implementations.UppercaseTransform)
				writer := implementations.NewBufferedWriter()

				config := &pipeline.Config{
					ChunkSize:       512,
					BufferSize:      50,
					WorkerCount:     workers,
					MemoryLimit:     50 * 1024 * 1024,
					ProcessTimeout:  time.Second,
					BackpressureMax: 25,
				}

				pipe := pipeline.NewPipeline(reader, processor, writer, config)

				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				pipe.Start(ctx)

				for pipe.IsRunning() {
					time.Sleep(5 * time.Millisecond)
				}

				pipe.Stop()
				reader.Close()
				processor.Close()
				writer.Close()
				cancel()
			}
		})
	}
}
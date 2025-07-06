package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"exercise-04-04/implementations"
	"exercise-04-04/memory"
	"exercise-04-04/pipeline"
)

func main() {
	// Command line flags
	inputFile := flag.String("input", "sample.txt", "Input file path")
	outputFile := flag.String("output", "output.txt", "Output file path")
	memoryLimit := flag.Int64("memory", 50*1024*1024, "Memory limit in bytes (default: 50MB)")
	workers := flag.Int("workers", 4, "Number of worker goroutines")
	chunkSize := flag.Int("chunk-size", 8192, "Chunk size in bytes")
	processorType := flag.String("processor", "uppercase", "Processor type (uppercase, lowercase, wordcount, reverse)")
	monitor := flag.Bool("monitor", false, "Enable real-time monitoring")
	createSample := flag.Bool("create-sample", false, "Create sample input file and exit")
	flag.Parse()

	// Handle create sample flag
	if *createSample {
		if err := createSampleFile(*inputFile); err != nil {
			log.Fatalf("Failed to create sample file: %v", err)
		}
		fmt.Printf("Sample file created: %s\n", *inputFile)
		return
	}

	// Create sample input file if it doesn't exist
	if err := createSampleFile(*inputFile); err != nil {
		log.Printf("Warning: Could not create sample file: %v", err)
	}

	// Create buffer pool for memory efficiency
	bufferPool := memory.NewBufferPool()

	// Create pipeline configuration
	config := &pipeline.Config{
		ChunkSize:       *chunkSize,
		BufferSize:      1024,
		WorkerCount:     *workers,
		MemoryLimit:     *memoryLimit,
		ProcessTimeout:  5 * time.Second,
		BackpressureMax: 100,
	}

	// Create pipeline components
	reader := implementations.NewFileReader(*inputFile, bufferPool)
	processor := createProcessor(*processorType, bufferPool)
	writer := implementations.NewFileWriter(*outputFile)

	// Create pipeline
	pipe := pipeline.NewPipeline(reader, processor, writer, config)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutdown signal received, stopping pipeline...")
		cancel()
	}()

	// Start monitoring if requested
	if *monitor {
		go monitorPipeline(ctx, pipe)
	}

	fmt.Printf("Starting memory-conscious pipeline...\n")
	fmt.Printf("Input: %s\n", *inputFile)
	fmt.Printf("Output: %s\n", *outputFile)
	fmt.Printf("Memory limit: %d MB\n", *memoryLimit/(1024*1024))
	fmt.Printf("Workers: %d\n", *workers)
	fmt.Printf("Chunk size: %d bytes\n", *chunkSize)
	fmt.Printf("Processor: %s\n", *processorType)

	// Start pipeline
	start := time.Now()
	if err := pipe.Start(ctx); err != nil {
		log.Fatalf("Failed to start pipeline: %v", err)
	}

	// Wait for completion or cancellation
	for pipe.IsRunning() {
		select {
		case <-ctx.Done():
			goto shutdown
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

shutdown:
	// Stop pipeline
	if err := pipe.Stop(); err != nil {
		log.Printf("Error stopping pipeline: %v", err)
	}

	// Close components
	reader.Close()
	processor.Close()
	writer.Close()

	duration := time.Since(start)
	stats := pipe.GetMetrics()

	fmt.Printf("\nPipeline completed in %v\n", duration)
	fmt.Printf("Processed: %d chunks\n", stats.Processed)
	fmt.Printf("Written: %d chunks\n", stats.Written)
	fmt.Printf("Errors: %d\n", stats.Errors)
	fmt.Printf("Memory pressure events: %d\n", stats.MemoryPressure)
	fmt.Printf("Average latency: %v\n", stats.AvgLatency)
	fmt.Printf("Throughput: %.2f chunks/sec\n", stats.Throughput)
}

// createProcessor creates a processor based on the specified type
func createProcessor(processorType string, pool *memory.BufferPool) pipeline.Processor {
	switch processorType {
	case "uppercase":
		return implementations.NewTextProcessor(pool, implementations.UppercaseTransform)
	case "lowercase":
		return implementations.NewTextProcessor(pool, implementations.LowercaseTransform)
	case "wordcount":
		return implementations.NewTextProcessor(pool, implementations.WordCountTransform)
	case "reverse":
		return implementations.NewTextProcessor(pool, implementations.ReverseTransform)
	case "trim":
		return implementations.NewTextProcessor(pool, implementations.TrimSpaceTransform)
	case "chain":
		// Example of chaining transformations
		chained := implementations.ChainTransforms(
			implementations.TrimSpaceTransform,
			implementations.UppercaseTransform,
			implementations.WordCountTransform,
		)
		return implementations.NewTextProcessor(pool, chained)
	case "json":
		return implementations.NewJSONProcessor(pool)
	default:
		fmt.Printf("Unknown processor type '%s', using uppercase\n", processorType)
		return implementations.NewTextProcessor(pool, implementations.UppercaseTransform)
	}
}

// monitorPipeline monitors pipeline metrics in real-time
func monitorPipeline(ctx context.Context, pipe *pipeline.Pipeline) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := pipe.GetMetrics()
			fmt.Printf("[MONITOR] Processed: %d, Written: %d, Errors: %d, Memory Pressure: %d, Throughput: %.2f/sec\n",
				stats.Processed, stats.Written, stats.Errors, stats.MemoryPressure, stats.Throughput)
		}
	}
}

// createSampleFile creates a sample input file for demonstration
func createSampleFile(filename string) error {
	// Check if file already exists
	if _, err := os.Stat(filename); err == nil {
		return nil // File exists, don't overwrite
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write sample data
	sampleData := `This is a sample text file for memory-conscious processing.
Each line will be processed by the pipeline.
The system demonstrates efficient memory usage through:
- Buffer pooling for memory reuse
- Chunked processing to limit memory footprint
- Memory pressure monitoring and cleanup
- Configurable worker pools for concurrency

This file contains multiple lines of text.
Some lines are longer than others to test variable chunk sizes.
The pipeline will transform this text according to the specified processor.

Memory optimization techniques used:
1. Object pooling to reduce garbage collection
2. Streaming processing to handle large files
3. Backpressure control to prevent memory overflow
4. Efficient buffer management
5. Real-time memory monitoring

End of sample file.
`

	_, err = file.WriteString(sampleData)
	return err
}
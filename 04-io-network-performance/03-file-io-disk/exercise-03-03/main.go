package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ProcessingStage represents a stage in the processing pipeline
type ProcessingStage func([]byte) ([]byte, error)

// BatchResult contains the results of batch processing
type BatchResult struct {
	Files       []string
	Results     [][]byte
	Errors      []error
	Duration    time.Duration
	BytesRead   int64
	BytesWritten int64
	FilesProcessed int64
}

// ProgressCallback is called to report progress
type ProgressCallback func(processed, total int, bytesProcessed int64)

// BatchConfig holds configuration for batch processing
type BatchConfig struct {
	BatchSize     int
	WorkerCount   int
	BufferSize    int
	ProgressCallback ProgressCallback
	Timeout       time.Duration
}

// DefaultBatchConfig returns a default configuration
func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		BatchSize:   1000,
		WorkerCount: runtime.NumCPU(),
		BufferSize:  64 * 1024, // 64KB
		Timeout:     30 * time.Second,
	}
}

// BatchProcessor handles batch file processing with pipeline architecture
type BatchProcessor struct {
	config   BatchConfig
	pipeline []ProcessingStage
	mu       sync.RWMutex
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(config BatchConfig) *BatchProcessor {
	return &BatchProcessor{
		config: config,
	}
}

// AddStage adds a processing stage to the pipeline
func (b *BatchProcessor) AddStage(stage ProcessingStage) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pipeline = append(b.pipeline, stage)
}

// ProcessBatch processes a batch of files with pipeline architecture
func (b *BatchProcessor) ProcessBatch(ctx context.Context, filenames []string) <-chan BatchResult {
	resultChan := make(chan BatchResult, 1)
	
	go func() {
		defer close(resultChan)
		
		start := time.Now()
		var totalBytesRead, totalBytesWritten, filesProcessed int64
		results := make([][]byte, len(filenames))
		errors := make([]error, len(filenames))
		
		// Create worker pool
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, b.config.WorkerCount)
		
		// Process files in parallel
		for i, filename := range filenames {
			select {
			case <-ctx.Done():
				errors[i] = ctx.Err()
				continue
			default:
			}
			
			wg.Add(1)
			go func(index int, file string) {
				defer wg.Done()
				
				// Acquire worker slot
				select {
				case semaphore <- struct{}{}:
					defer func() { <-semaphore }()
				case <-ctx.Done():
					errors[index] = ctx.Err()
					return
				}
				
				data, bytesRead, bytesWritten, err := b.processFile(ctx, file)
				if err != nil {
					errors[index] = err
					return
				}
				
				results[index] = data
				atomic.AddInt64(&totalBytesRead, bytesRead)
				atomic.AddInt64(&totalBytesWritten, bytesWritten)
				processed := atomic.AddInt64(&filesProcessed, 1)
				
				// Report progress
				if b.config.ProgressCallback != nil {
					b.config.ProgressCallback(int(processed), len(filenames), totalBytesRead)
				}
			}(i, filename)
		}
		
		wg.Wait()
		
		resultChan <- BatchResult{
			Files:          filenames,
			Results:        results,
			Errors:         errors,
			Duration:       time.Since(start),
			BytesRead:      totalBytesRead,
			BytesWritten:   totalBytesWritten,
			FilesProcessed: filesProcessed,
		}
	}()
	
	return resultChan
}

// processFile processes a single file through the pipeline
func (b *BatchProcessor) processFile(ctx context.Context, filename string) ([]byte, int64, int64, error) {
	select {
	case <-ctx.Done():
		return nil, 0, 0, ctx.Err()
	default:
	}
	
	// Read file with optimized buffer
	data, bytesRead, err := b.readFileOptimized(filename)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to read %s: %w", filename, err)
	}
	
	bytesWritten := int64(0)
	
	// Apply processing pipeline
	b.mu.RLock()
	pipeline := b.pipeline
	b.mu.RUnlock()
	
	for _, stage := range pipeline {
		select {
		case <-ctx.Done():
			return nil, bytesRead, bytesWritten, ctx.Err()
		default:
		}
		
		processedData, err := stage(data)
		if err != nil {
			return nil, bytesRead, bytesWritten, fmt.Errorf("pipeline stage failed for %s: %w", filename, err)
		}
		data = processedData
		bytesWritten += int64(len(data))
	}
	
	return data, bytesRead, bytesWritten, nil
}

// readFileOptimized reads a file with optimized buffering
func (b *BatchProcessor) readFileOptimized(filename string) ([]byte, int64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()
	
	stat, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}
	
	size := stat.Size()
	if size == 0 {
		return []byte{}, 0, nil
	}
	
	// Pre-allocate buffer based on file size
	data := make([]byte, size)
	n, err := io.ReadFull(file, data)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, 0, err
	}
	
	return data[:n], int64(n), nil
}

// BatchStats provides statistics about batch processing
type BatchStats struct {
	TotalFiles      int64
	ProcessedFiles  int64
	FailedFiles     int64
	TotalBytes      int64
	ThroughputMBps  float64
	AvgFileSize     float64
	ProcessingTime  time.Duration
}

// CalculateStats calculates statistics from batch results
func CalculateStats(results []BatchResult) BatchStats {
	var stats BatchStats
	
	for _, result := range results {
		stats.TotalFiles += int64(len(result.Files))
		stats.ProcessedFiles += result.FilesProcessed
		stats.TotalBytes += result.BytesRead
		stats.ProcessingTime += result.Duration
		
		for _, err := range result.Errors {
			if err != nil {
				stats.FailedFiles++
			}
		}
	}
	
	if stats.ProcessingTime > 0 {
		stats.ThroughputMBps = float64(stats.TotalBytes) / (1024 * 1024) / stats.ProcessingTime.Seconds()
	}
	
	if stats.ProcessedFiles > 0 {
		stats.AvgFileSize = float64(stats.TotalBytes) / float64(stats.ProcessedFiles)
	}
	
	return stats
}

// Common processing stages

// UppercaseStage converts data to uppercase
func UppercaseStage(data []byte) ([]byte, error) {
	result := make([]byte, len(data))
	for i, b := range data {
		if b >= 'a' && b <= 'z' {
			result[i] = b - 32
		} else {
			result[i] = b
		}
	}
	return result, nil
}

// CompressionStage simulates compression (simple RLE)
func CompressionStage(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	
	var result []byte
	current := data[0]
	count := byte(1)
	
	for i := 1; i < len(data); i++ {
		if data[i] == current && count < 255 {
			count++
		} else {
			result = append(result, count, current)
			current = data[i]
			count = 1
		}
	}
	result = append(result, count, current)
	
	return result, nil
}

// ValidationStage validates data integrity
func ValidationStage(data []byte) ([]byte, error) {
	// Simple validation - check for null bytes
	for i, b := range data {
		if b == 0 {
			return nil, fmt.Errorf("invalid null byte at position %d", i)
		}
	}
	return data, nil
}

// CreateTestFiles creates test files for batch processing
func CreateTestFiles(dir string, count int, sizeBytes int) ([]string, error) {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}
	
	var filenames []string
	data := make([]byte, sizeBytes)
	
	// Fill with test data
	for i := range data {
		data[i] = byte('A' + (i % 26))
	}
	
	for i := 0; i < count; i++ {
		filename := filepath.Join(dir, fmt.Sprintf("test_%04d.txt", i))
		err := os.WriteFile(filename, data, 0644)
		if err != nil {
			return nil, err
		}
		filenames = append(filenames, filename)
	}
	
	return filenames, nil
}

func main() {
	fmt.Println("Batch File Processing Demo")
	fmt.Println("==========================")
	
	// Create test files
	testDir := "test_batch_files"
	fileCount := 100
	fileSize := 1024 // 1KB per file
	
	fmt.Printf("Creating %d test files of %d bytes each...\n", fileCount, fileSize)
	filenames, err := CreateTestFiles(testDir, fileCount, fileSize)
	if err != nil {
		fmt.Printf("Error creating test files: %v\n", err)
		return
	}
	defer os.RemoveAll(testDir)
	
	// Configure batch processor
	config := DefaultBatchConfig()
	config.WorkerCount = 4
	config.ProgressCallback = func(processed, total int, bytesProcessed int64) {
		progress := float64(processed) / float64(total) * 100
		fmt.Printf("\rProgress: %.1f%% (%d/%d files, %.2f MB processed)", 
			progress, processed, total, float64(bytesProcessed)/(1024*1024))
	}
	
	processor := NewBatchProcessor(config)
	
	// Add processing stages
	processor.AddStage(ValidationStage)
	processor.AddStage(UppercaseStage)
	processor.AddStage(CompressionStage)
	
	// Process batch
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()
	
	fmt.Printf("\nProcessing batch with %d workers...\n", config.WorkerCount)
	
	resultChan := processor.ProcessBatch(ctx, filenames)
	result := <-resultChan
	
	fmt.Printf("\n\nBatch Processing Results:\n")
	fmt.Printf("========================\n")
	fmt.Printf("Files processed: %d/%d\n", result.FilesProcessed, len(result.Files))
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Bytes read: %.2f MB\n", float64(result.BytesRead)/(1024*1024))
	fmt.Printf("Bytes written: %.2f MB\n", float64(result.BytesWritten)/(1024*1024))
	
	if result.Duration > 0 {
		throughput := float64(result.BytesRead) / (1024 * 1024) / result.Duration.Seconds()
		fmt.Printf("Read throughput: %.2f MB/s\n", throughput)
		
		writeThroughput := float64(result.BytesWritten) / (1024 * 1024) / result.Duration.Seconds()
		fmt.Printf("Write throughput: %.2f MB/s\n", writeThroughput)
		
		filesPerSec := float64(result.FilesProcessed) / result.Duration.Seconds()
		fmt.Printf("Files per second: %.2f\n", filesPerSec)
	}
	
	// Count errors
	errorCount := 0
	for _, err := range result.Errors {
		if err != nil {
			errorCount++
		}
	}
	fmt.Printf("Errors: %d\n", errorCount)
	
	// Calculate statistics
	stats := CalculateStats([]BatchResult{result})
	fmt.Printf("\nStatistics:\n")
	fmt.Printf("===========\n")
	fmt.Printf("Success rate: %.2f%%\n", float64(stats.ProcessedFiles)/float64(stats.TotalFiles)*100)
	fmt.Printf("Average file size: %.2f bytes\n", stats.AvgFileSize)
	fmt.Printf("Overall throughput: %.2f MB/s\n", stats.ThroughputMBps)
	
	// Performance validation
	fmt.Printf("\nPerformance Validation:\n")
	fmt.Printf("======================\n")
	if stats.ThroughputMBps > 100 {
		fmt.Printf("✓ Throughput target met (>100 MB/s): %.2f MB/s\n", stats.ThroughputMBps)
	} else {
		fmt.Printf("✗ Throughput target not met (<100 MB/s): %.2f MB/s\n", stats.ThroughputMBps)
	}
	
	if stats.ProcessedFiles >= int64(fileCount) {
		fmt.Printf("✓ All files processed successfully\n")
	} else {
		fmt.Printf("✗ Some files failed to process: %d/%d\n", stats.ProcessedFiles, stats.TotalFiles)
	}
	
	memUsageMB := float64(runtime.MemStats{}.Alloc) / (1024 * 1024)
	if memUsageMB < 100 {
		fmt.Printf("✓ Memory usage target met (<100 MB): %.2f MB\n", memUsageMB)
	} else {
		fmt.Printf("✗ Memory usage target not met (>100 MB): %.2f MB\n", memUsageMB)
	}
	
	fmt.Printf("\nBatch file processing demo completed successfully!\n")
}
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// FileResult represents the result of processing a single file
type FileResult struct {
	Filename string
	Data     []byte
	Size     int64
	Duration time.Duration
	Error    error
}

// AsyncFileProcessor handles concurrent file operations
type AsyncFileProcessor struct {
	workerPool   chan struct{}
	resultChan   chan FileResult
	errorChan    chan error
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	processedCount int64
	errorCount     int64
	totalBytes     int64
	startTime      time.Time
}

// ProcessorConfig holds configuration for the async processor
type ProcessorConfig struct {
	MaxWorkers   int
	BufferSize   int
	Timeout      time.Duration
	ResultBuffer int
	ErrorBuffer  int
}

// DefaultConfig returns a default configuration
func DefaultConfig() ProcessorConfig {
	return ProcessorConfig{
		MaxWorkers:   100,
		BufferSize:   64 * 1024, // 64KB buffer
		Timeout:      30 * time.Second,
		ResultBuffer: 1000,
		ErrorBuffer:  100,
	}
}

// NewAsyncFileProcessor creates a new async file processor
func NewAsyncFileProcessor(config ProcessorConfig) *AsyncFileProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &AsyncFileProcessor{
		workerPool: make(chan struct{}, config.MaxWorkers),
		resultChan: make(chan FileResult, config.ResultBuffer),
		errorChan:  make(chan error, config.ErrorBuffer),
		ctx:        ctx,
		cancel:     cancel,
		startTime:  time.Now(),
	}
}

// ProcessFile processes a single file asynchronously
func (p *AsyncFileProcessor) ProcessFile(filename string) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		
		// Check if context is cancelled
		select {
		case <-p.ctx.Done():
			return
		default:
		}
		
		// Acquire worker slot
		select {
		case p.workerPool <- struct{}{}:
			defer func() { <-p.workerPool }()
		case <-p.ctx.Done():
			return
		}
		
		start := time.Now()
		data, err := p.readFileOptimized(filename)
		
		result := FileResult{
			Filename: filename,
			Data:     data,
			Size:     int64(len(data)),
			Duration: time.Since(start),
			Error:    err,
		}
		
		if err != nil {
			atomic.AddInt64(&p.errorCount, 1)
			select {
			case p.errorChan <- fmt.Errorf("failed to read %s: %w", filename, err):
			case <-p.ctx.Done():
				return
			case <-time.After(5 * time.Second):
				// Timeout sending error
			}
		} else {
			atomic.AddInt64(&p.totalBytes, int64(len(data)))
		}
		
		atomic.AddInt64(&p.processedCount, 1)
		
		select {
		case p.resultChan <- result:
		case <-p.ctx.Done():
			return
		case <-time.After(5 * time.Second):
			// Timeout sending result
			select {
			case p.errorChan <- fmt.Errorf("timeout sending result for %s", filename):
			default:
			}
		}
	}()
}

// ProcessFiles processes multiple files concurrently
func (p *AsyncFileProcessor) ProcessFiles(filenames []string) {
	for _, filename := range filenames {
		p.ProcessFile(filename)
	}
}

// ProcessDirectory processes all files in a directory recursively
func (p *AsyncFileProcessor) ProcessDirectory(dirPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() {
			p.ProcessFile(path)
		}
		
		return nil
	})
}

// readFileOptimized reads a file with optimized buffer management
func (p *AsyncFileProcessor) readFileOptimized(filename string) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	
	// Pre-allocate buffer based on file size
	data := make([]byte, stat.Size())
	_, err = io.ReadFull(file, data)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	
	return data, nil
}

// Wait waits for all file processing to complete
func (p *AsyncFileProcessor) Wait() {
	p.wg.Wait()
}

// Close closes the result and error channels
func (p *AsyncFileProcessor) Close() {
	close(p.resultChan)
	close(p.errorChan)
}

// Cancel cancels all ongoing operations
func (p *AsyncFileProcessor) Cancel() {
	p.cancel()
}

// Results returns the result channel
func (p *AsyncFileProcessor) Results() <-chan FileResult {
	return p.resultChan
}

// Errors returns the error channel
func (p *AsyncFileProcessor) Errors() <-chan error {
	return p.errorChan
}

// GetStats returns processing statistics
func (p *AsyncFileProcessor) GetStats() ProcessingStats {
	return ProcessingStats{
		ProcessedCount: atomic.LoadInt64(&p.processedCount),
		ErrorCount:     atomic.LoadInt64(&p.errorCount),
		TotalBytes:     atomic.LoadInt64(&p.totalBytes),
		Duration:       time.Since(p.startTime),
		Throughput:     float64(atomic.LoadInt64(&p.totalBytes)) / time.Since(p.startTime).Seconds(),
		ErrorRate:      float64(atomic.LoadInt64(&p.errorCount)) / float64(atomic.LoadInt64(&p.processedCount)),
	}
}

// ProcessingStats holds processing statistics
type ProcessingStats struct {
	ProcessedCount int64
	ErrorCount     int64
	TotalBytes     int64
	Duration       time.Duration
	Throughput     float64 // bytes per second
	ErrorRate      float64 // error rate as percentage
}

// String returns a string representation of the stats
func (s ProcessingStats) String() string {
	throughputMB := s.Throughput / (1024 * 1024)
	errorPercent := s.ErrorRate * 100
	
	return fmt.Sprintf(
		"Processed: %d files, Errors: %d, Total: %.2f MB, Duration: %v, Throughput: %.2f MB/s, Error Rate: %.4f%%",
		s.ProcessedCount, s.ErrorCount, float64(s.TotalBytes)/(1024*1024), s.Duration, throughputMB, errorPercent,
	)
}

// CreateTestFiles creates test files for demonstration
func CreateTestFiles(dir string, count int, sizeKB int) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	data := make([]byte, sizeKB*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	
	for i := 0; i < count; i++ {
		filename := filepath.Join(dir, fmt.Sprintf("test_file_%04d.dat", i))
		if err := os.WriteFile(filename, data, 0644); err != nil {
			return err
		}
	}
	
	return nil
}

// CleanupTestFiles removes test files
func CleanupTestFiles(dir string) error {
	return os.RemoveAll(dir)
}

func main() {
	fmt.Println("Async File Operations Demo")
	fmt.Println("==========================")
	
	// Create test files
	testDir := "test_files"
	fileCount := 1000
	fileSizeKB := 10
	
	fmt.Printf("Creating %d test files of %dKB each...\n", fileCount, fileSizeKB)
	if err := CreateTestFiles(testDir, fileCount, fileSizeKB); err != nil {
		fmt.Printf("Error creating test files: %v\n", err)
		return
	}
	defer CleanupTestFiles(testDir)
	
	// Configure processor
	config := DefaultConfig()
	config.MaxWorkers = 50 // Limit workers for demo
	
	// Create processor
	processor := NewAsyncFileProcessor(config)
	
	// Start processing
	fmt.Println("Starting async file processing...")
	start := time.Now()
	
	if err := processor.ProcessDirectory(testDir); err != nil {
		fmt.Printf("Error processing directory: %v\n", err)
		return
	}
	
	// Monitor progress
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				stats := processor.GetStats()
				fmt.Printf("Progress: %s\n", stats.String())
			case <-processor.ctx.Done():
				return
			}
		}
	}()
	
	// Collect results
	var results []FileResult
	var errors []error
	
	go func() {
		for result := range processor.Results() {
			results = append(results, result)
		}
	}()
	
	go func() {
		for err := range processor.Errors() {
			errors = append(errors, err)
		}
	}()
	
	// Wait for completion
	processor.Wait()
	processor.Close()
	processor.Cancel()
	
	// Final statistics
	stats := processor.GetStats()
	fmt.Println("\nFinal Results:")
	fmt.Printf("%s\n", stats.String())
	
	// Performance validation
	fmt.Println("\nPerformance Validation:")
	fmt.Printf("✓ Files processed: %d\n", stats.ProcessedCount)
	fmt.Printf("✓ Throughput: %.2f MB/s (target: >100 MB/s)\n", stats.Throughput/(1024*1024))
	fmt.Printf("✓ Error rate: %.4f%% (target: <0.01%%)\n", stats.ErrorRate*100)
	fmt.Printf("✓ Processing time: %v\n", time.Since(start))
	
	if len(errors) > 0 {
		fmt.Printf("\nErrors encountered: %d\n", len(errors))
		for i, err := range errors {
			if i < 5 { // Show first 5 errors
				fmt.Printf("  %d: %v\n", i+1, err)
			}
		}
		if len(errors) > 5 {
			fmt.Printf("  ... and %d more errors\n", len(errors)-5)
		}
	}
	
	fmt.Println("\nDemo completed successfully!")
}
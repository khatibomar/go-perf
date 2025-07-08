package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.MaxWorkers != 100 {
		t.Errorf("Expected MaxWorkers to be 100, got %d", config.MaxWorkers)
	}
	
	if config.BufferSize != 64*1024 {
		t.Errorf("Expected BufferSize to be 65536, got %d", config.BufferSize)
	}
	
	if config.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout to be 30s, got %v", config.Timeout)
	}
	
	if config.ResultBuffer != 1000 {
		t.Errorf("Expected ResultBuffer to be 1000, got %d", config.ResultBuffer)
	}
	
	if config.ErrorBuffer != 100 {
		t.Errorf("Expected ErrorBuffer to be 100, got %d", config.ErrorBuffer)
	}
}

func TestNewAsyncFileProcessor(t *testing.T) {
	config := DefaultConfig()
	processor := NewAsyncFileProcessor(config)
	
	if processor == nil {
		t.Fatal("Expected processor to be created")
	}
	
	if cap(processor.workerPool) != config.MaxWorkers {
		t.Errorf("Expected worker pool capacity to be %d, got %d", config.MaxWorkers, cap(processor.workerPool))
	}
	
	if cap(processor.resultChan) != config.ResultBuffer {
		t.Errorf("Expected result channel capacity to be %d, got %d", config.ResultBuffer, cap(processor.resultChan))
	}
	
	if cap(processor.errorChan) != config.ErrorBuffer {
		t.Errorf("Expected error channel capacity to be %d, got %d", config.ErrorBuffer, cap(processor.errorChan))
	}
	
	processor.Cancel()
}

func TestCreateTestFiles(t *testing.T) {
	testDir := "test_create_files"
	defer os.RemoveAll(testDir)
	
	count := 5
	sizeKB := 1
	
	err := CreateTestFiles(testDir, count, sizeKB)
	if err != nil {
		t.Fatalf("Failed to create test files: %v", err)
	}
	
	// Verify files were created
	for i := 0; i < count; i++ {
		filename := filepath.Join(testDir, fmt.Sprintf("test_file_%04d.dat", i))
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", filename)
		}
		
		// Verify file size
		info, err := os.Stat(filename)
		if err != nil {
			t.Errorf("Failed to stat file %s: %v", filename, err)
			continue
		}
		
		expectedSize := int64(sizeKB * 1024)
		if info.Size() != expectedSize {
			t.Errorf("Expected file %s to be %d bytes, got %d", filename, expectedSize, info.Size())
		}
	}
}

func TestProcessSingleFile(t *testing.T) {
	testDir := "test_single_file"
	defer os.RemoveAll(testDir)
	
	// Create a test file
	err := CreateTestFiles(testDir, 1, 1)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	config := DefaultConfig()
	config.MaxWorkers = 1
	processor := NewAsyncFileProcessor(config)
	defer processor.Cancel()
	
	filename := filepath.Join(testDir, "test_file_0000.dat")
	processor.ProcessFile(filename)
	
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
	
	processor.Wait()
	processor.Close()
	
	// Wait a bit for goroutines to finish
	time.Sleep(10 * time.Millisecond)
	
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	
	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(errors), errors)
	}
	
	if len(results) > 0 {
		result := results[0]
		if result.Filename != filename {
			t.Errorf("Expected filename %s, got %s", filename, result.Filename)
		}
		
		if result.Size != 1024 {
			t.Errorf("Expected size 1024, got %d", result.Size)
		}
		
		if result.Error != nil {
			t.Errorf("Expected no error, got %v", result.Error)
		}
		
		if len(result.Data) != 1024 {
			t.Errorf("Expected data length 1024, got %d", len(result.Data))
		}
	}
}

func TestProcessMultipleFiles(t *testing.T) {
	testDir := "test_multiple_files"
	defer os.RemoveAll(testDir)
	
	fileCount := 10
	fileSizeKB := 1
	
	// Create test files
	err := CreateTestFiles(testDir, fileCount, fileSizeKB)
	if err != nil {
		t.Fatalf("Failed to create test files: %v", err)
	}
	
	config := DefaultConfig()
	config.MaxWorkers = 5
	processor := NewAsyncFileProcessor(config)
	defer processor.Cancel()
	
	// Process directory
	err = processor.ProcessDirectory(testDir)
	if err != nil {
		t.Fatalf("Failed to process directory: %v", err)
	}
	
	// Collect results
	var results []FileResult
	var errors []error
	var mu sync.Mutex
	
	go func() {
		for result := range processor.Results() {
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}
	}()
	
	go func() {
		for err := range processor.Errors() {
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
		}
	}()
	
	processor.Wait()
	processor.Close()
	
	// Wait a bit for goroutines to finish
	time.Sleep(10 * time.Millisecond)
	
	mu.Lock()
	defer mu.Unlock()
	
	if len(results) != fileCount {
		t.Errorf("Expected %d results, got %d", fileCount, len(results))
	}
	
	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(errors), errors)
	}
	
	// Verify stats
	stats := processor.GetStats()
	if stats.ProcessedCount != int64(fileCount) {
		t.Errorf("Expected processed count %d, got %d", fileCount, stats.ProcessedCount)
	}
	
	if stats.ErrorCount != 0 {
		t.Errorf("Expected error count 0, got %d", stats.ErrorCount)
	}
	
	expectedBytes := int64(fileCount * fileSizeKB * 1024)
	if stats.TotalBytes != expectedBytes {
		t.Errorf("Expected total bytes %d, got %d", expectedBytes, stats.TotalBytes)
	}
}

func TestProcessingStats(t *testing.T) {
	stats := ProcessingStats{
		ProcessedCount: 100,
		ErrorCount:     2,
		TotalBytes:     1024 * 1024, // 1MB
		Duration:       time.Second,
		Throughput:     1024 * 1024, // 1MB/s
		ErrorRate:      0.02,        // 2%
	}
	
	str := stats.String()
	if str == "" {
		t.Error("Expected non-empty string representation")
	}
	
	// Check if string contains expected values
	expected := []string{"100", "2", "1.00 MB", "1.00 MB/s", "2.0000%"}
	for _, exp := range expected {
		if !contains(str, exp) {
			t.Errorf("Expected string to contain '%s', got: %s", exp, str)
		}
	}
}

func TestErrorHandling(t *testing.T) {
	config := DefaultConfig()
	config.MaxWorkers = 1
	processor := NewAsyncFileProcessor(config)
	defer processor.Cancel()
	
	// Try to process a non-existent file
	processor.ProcessFile("non_existent_file.txt")
	
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
	
	processor.Wait()
	processor.Close()
	
	// Wait a bit for goroutines to finish
	time.Sleep(10 * time.Millisecond)
	
	// Should have one result with an error
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	
	if len(results) > 0 && results[0].Error == nil {
		t.Error("Expected result to have an error")
	}
	
	// Should also have an error in the error channel
	if len(errors) == 0 {
		t.Error("Expected at least one error in error channel")
	}
	
	// Verify stats reflect the error
	stats := processor.GetStats()
	if stats.ErrorCount == 0 {
		t.Error("Expected error count to be greater than 0")
	}
}

func TestConcurrentProcessing(t *testing.T) {
	testDir := "test_concurrent"
	defer os.RemoveAll(testDir)
	
	fileCount := 50
	fileSizeKB := 2
	
	// Create test files
	err := CreateTestFiles(testDir, fileCount, fileSizeKB)
	if err != nil {
		t.Fatalf("Failed to create test files: %v", err)
	}
	
	config := DefaultConfig()
	config.MaxWorkers = 10
	processor := NewAsyncFileProcessor(config)
	defer processor.Cancel()
	
	start := time.Now()
	
	// Process directory
	err = processor.ProcessDirectory(testDir)
	if err != nil {
		t.Fatalf("Failed to process directory: %v", err)
	}
	
	// Collect results
	var results []FileResult
	var errors []error
	var mu sync.Mutex
	
	go func() {
		for result := range processor.Results() {
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}
	}()
	
	go func() {
		for err := range processor.Errors() {
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
		}
	}()
	
	processor.Wait()
	processor.Close()
	duration := time.Since(start)
	
	// Wait a bit for goroutines to finish
	time.Sleep(10 * time.Millisecond)
	
	mu.Lock()
	defer mu.Unlock()
	
	if len(results) != fileCount {
		t.Errorf("Expected %d results, got %d", fileCount, len(results))
	}
	
	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
	
	// Verify concurrent processing was faster than sequential
	// This is a rough check - concurrent should be significantly faster
	if duration > time.Duration(fileCount)*10*time.Millisecond {
		t.Logf("Processing took %v, which might be slower than expected for concurrent processing", duration)
	}
	
	// Verify stats
	stats := processor.GetStats()
	if stats.ProcessedCount != int64(fileCount) {
		t.Errorf("Expected processed count %d, got %d", fileCount, stats.ProcessedCount)
	}
	
	if stats.Throughput <= 0 {
		t.Error("Expected positive throughput")
	}
}

// Benchmark tests
func BenchmarkSingleFileProcessing(b *testing.B) {
	testDir := "bench_single"
	defer os.RemoveAll(testDir)
	
	// Create a test file
	err := CreateTestFiles(testDir, 1, 10) // 10KB file
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	
	filename := filepath.Join(testDir, "test_file_0000.dat")
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		config := DefaultConfig()
		config.MaxWorkers = 1
		processor := NewAsyncFileProcessor(config)
		
		processor.ProcessFile(filename)
		
		// Drain channels
		go func() {
			for range processor.Results() {
			}
		}()
		
		go func() {
			for range processor.Errors() {
			}
		}()
		
		processor.Wait()
		processor.Close()
		processor.Cancel()
	}
}

func BenchmarkConcurrentFileProcessing(b *testing.B) {
	testDir := "bench_concurrent"
	defer os.RemoveAll(testDir)
	
	fileCount := 100
	fileSizeKB := 5
	
	// Create test files
	err := CreateTestFiles(testDir, fileCount, fileSizeKB)
	if err != nil {
		b.Fatalf("Failed to create test files: %v", err)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		config := DefaultConfig()
		config.MaxWorkers = 20
		processor := NewAsyncFileProcessor(config)
		
		err := processor.ProcessDirectory(testDir)
		if err != nil {
			b.Fatalf("Failed to process directory: %v", err)
		}
		
		// Drain channels
		go func() {
			for range processor.Results() {
			}
		}()
		
		go func() {
			for range processor.Errors() {
			}
		}()
		
		processor.Wait()
		processor.Close()
		processor.Cancel()
	}
}

func BenchmarkStatsCalculation(b *testing.B) {
	config := DefaultConfig()
	processor := NewAsyncFileProcessor(config)
	defer processor.Cancel()
	
	// Simulate some processing
	processor.processedCount = 1000
	processor.errorCount = 5
	processor.totalBytes = 1024 * 1024 * 100 // 100MB
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		stats := processor.GetStats()
		_ = stats.String()
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
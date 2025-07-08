package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDefaultBatchConfig(t *testing.T) {
	config := DefaultBatchConfig()
	
	if config.BatchSize != 1000 {
		t.Errorf("Expected BatchSize 1000, got %d", config.BatchSize)
	}
	
	if config.WorkerCount != runtime.NumCPU() {
		t.Errorf("Expected WorkerCount %d, got %d", runtime.NumCPU(), config.WorkerCount)
	}
	
	if config.BufferSize != 64*1024 {
		t.Errorf("Expected BufferSize 65536, got %d", config.BufferSize)
	}
	
	if config.Timeout != 30*time.Second {
		t.Errorf("Expected Timeout 30s, got %v", config.Timeout)
	}
}

func TestCreateTestFiles(t *testing.T) {
	testDir := t.TempDir()
	fileCount := 10
	fileSize := 1024
	
	filenames, err := CreateTestFiles(testDir, fileCount, fileSize)
	if err != nil {
		t.Fatalf("CreateTestFiles failed: %v", err)
	}
	
	if len(filenames) != fileCount {
		t.Errorf("Expected %d files, got %d", fileCount, len(filenames))
	}
	
	// Verify files exist and have correct size
	for i, filename := range filenames {
		stat, err := os.Stat(filename)
		if err != nil {
			t.Errorf("File %d does not exist: %v", i, err)
			continue
		}
		
		if stat.Size() != int64(fileSize) {
			t.Errorf("File %d has wrong size: expected %d, got %d", i, fileSize, stat.Size())
		}
	}
}

func TestBatchProcessor_AddStage(t *testing.T) {
	config := DefaultBatchConfig()
	processor := NewBatchProcessor(config)
	
	// Initially no stages
	if len(processor.pipeline) != 0 {
		t.Errorf("Expected 0 stages initially, got %d", len(processor.pipeline))
	}
	
	// Add stages
	processor.AddStage(UppercaseStage)
	processor.AddStage(ValidationStage)
	
	if len(processor.pipeline) != 2 {
		t.Errorf("Expected 2 stages, got %d", len(processor.pipeline))
	}
}

func TestProcessingStages(t *testing.T) {
	tests := []struct {
		name     string
		stage    ProcessingStage
		input    []byte
		expected []byte
		shouldErr bool
	}{
		{
			name:     "UppercaseStage",
			stage:    UppercaseStage,
			input:    []byte("hello world"),
			expected: []byte("HELLO WORLD"),
			shouldErr: false,
		},
		{
			name:     "ValidationStage_Valid",
			stage:    ValidationStage,
			input:    []byte("valid data"),
			expected: []byte("valid data"),
			shouldErr: false,
		},
		{
			name:     "ValidationStage_Invalid",
			stage:    ValidationStage,
			input:    []byte{65, 0, 66}, // Contains null byte
			expected: nil,
			shouldErr: true,
		},
		{
			name:     "CompressionStage",
			stage:    CompressionStage,
			input:    []byte("AAABBC"),
			expected: []byte{3, 'A', 2, 'B', 1, 'C'},
			shouldErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.stage(tt.input)
			
			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if len(result) != len(tt.expected) {
				t.Errorf("Length mismatch: expected %d, got %d", len(tt.expected), len(result))
				return
			}
			
			for i, b := range tt.expected {
				if result[i] != b {
					t.Errorf("Byte mismatch at position %d: expected %d, got %d", i, b, result[i])
					break
				}
			}
		})
	}
}

func TestBatchProcessor_ProcessBatch(t *testing.T) {
	testDir := t.TempDir()
	fileCount := 50
	fileSize := 512
	
	// Create test files
	filenames, err := CreateTestFiles(testDir, fileCount, fileSize)
	if err != nil {
		t.Fatalf("CreateTestFiles failed: %v", err)
	}
	
	// Configure processor
	config := DefaultBatchConfig()
	config.WorkerCount = 4
	config.Timeout = 10 * time.Second
	
	var progressCalls int64
	config.ProgressCallback = func(processed, total int, bytesProcessed int64) {
		atomic.AddInt64(&progressCalls, 1)
	}
	
	processor := NewBatchProcessor(config)
	processor.AddStage(ValidationStage)
	processor.AddStage(UppercaseStage)
	
	// Process batch
	ctx := context.Background()
	resultChan := processor.ProcessBatch(ctx, filenames)
	result := <-resultChan
	
	// Verify results
	if result.FilesProcessed != int64(fileCount) {
		t.Errorf("Expected %d files processed, got %d", fileCount, result.FilesProcessed)
	}
	
	if result.BytesRead != int64(fileCount*fileSize) {
		t.Errorf("Expected %d bytes read, got %d", fileCount*fileSize, result.BytesRead)
	}
	
	if result.Duration <= 0 {
		t.Errorf("Expected positive duration, got %v", result.Duration)
	}
	
	// Check for errors
	errorCount := 0
	for _, err := range result.Errors {
		if err != nil {
			errorCount++
			t.Logf("Processing error: %v", err)
		}
	}
	
	if errorCount > 0 {
		t.Errorf("Expected no errors, got %d", errorCount)
	}
	
	// Verify progress callbacks were called
	if atomic.LoadInt64(&progressCalls) == 0 {
		t.Errorf("Expected progress callbacks to be called")
	}
	
	// Verify results are uppercase (due to UppercaseStage)
	for i, data := range result.Results {
		if data == nil {
			continue
		}
		
		// Check that data is uppercase
		str := string(data)
		if strings.ToUpper(str) != str {
			t.Errorf("Result %d is not uppercase", i)
			break
		}
	}
}

func TestBatchProcessor_ContextCancellation(t *testing.T) {
	testDir := t.TempDir()
	fileCount := 100
	fileSize := 1024
	
	// Create test files
	filenames, err := CreateTestFiles(testDir, fileCount, fileSize)
	if err != nil {
		t.Fatalf("CreateTestFiles failed: %v", err)
	}
	
	// Configure processor with slow stage
	config := DefaultBatchConfig()
	config.WorkerCount = 2
	
	processor := NewBatchProcessor(config)
	
	// Add a slow processing stage
	slowStage := func(data []byte) ([]byte, error) {
		time.Sleep(100 * time.Millisecond) // Simulate slow processing
		return data, nil
	}
	processor.AddStage(slowStage)
	
	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	
	// Process batch
	resultChan := processor.ProcessBatch(ctx, filenames)
	result := <-resultChan
	
	// Should have some context cancellation errors
	errorCount := 0
	for _, err := range result.Errors {
		if err != nil {
			errorCount++
		}
	}
	
	// We expect some errors due to context cancellation
	if errorCount == 0 {
		t.Logf("Warning: Expected some context cancellation errors, got none")
	}
	
	// Files processed should be less than total due to cancellation
	if result.FilesProcessed >= int64(fileCount) {
		t.Logf("Warning: Expected fewer files processed due to cancellation, got %d/%d", result.FilesProcessed, fileCount)
	}
}

func TestCalculateStats(t *testing.T) {
	// Create mock batch results
	results := []BatchResult{
		{
			Files:          []string{"file1.txt", "file2.txt"},
			FilesProcessed: 2,
			BytesRead:      2048,
			Duration:       1 * time.Second,
			Errors:         []error{nil, nil},
		},
		{
			Files:          []string{"file3.txt", "file4.txt"},
			FilesProcessed: 1,
			BytesRead:      1024,
			Duration:       500 * time.Millisecond,
			Errors:         []error{nil, fmt.Errorf("test error")},
		},
	}
	
	stats := CalculateStats(results)
	
	if stats.TotalFiles != 4 {
		t.Errorf("Expected TotalFiles 4, got %d", stats.TotalFiles)
	}
	
	if stats.ProcessedFiles != 3 {
		t.Errorf("Expected ProcessedFiles 3, got %d", stats.ProcessedFiles)
	}
	
	if stats.FailedFiles != 1 {
		t.Errorf("Expected FailedFiles 1, got %d", stats.FailedFiles)
	}
	
	if stats.TotalBytes != 3072 {
		t.Errorf("Expected TotalBytes 3072, got %d", stats.TotalBytes)
	}
	
	if stats.ProcessingTime != 1500*time.Millisecond {
		t.Errorf("Expected ProcessingTime 1.5s, got %v", stats.ProcessingTime)
	}
	
	// Check throughput calculation (approximately)
	expectedThroughput := float64(3072) / (1024 * 1024) / 1.5 // MB/s
	if stats.ThroughputMBps < expectedThroughput*0.9 || stats.ThroughputMBps > expectedThroughput*1.1 {
		t.Errorf("Expected throughput around %.2f MB/s, got %.2f MB/s", expectedThroughput, stats.ThroughputMBps)
	}
	
	// Check average file size
	expectedAvgSize := float64(3072) / 3
	if stats.AvgFileSize != expectedAvgSize {
		t.Errorf("Expected AvgFileSize %.2f, got %.2f", expectedAvgSize, stats.AvgFileSize)
	}
}

func TestBatchProcessor_EmptyFiles(t *testing.T) {
	testDir := t.TempDir()
	
	// Create empty files
	var filenames []string
	for i := 0; i < 5; i++ {
		filename := filepath.Join(testDir, fmt.Sprintf("empty_%d.txt", i))
		err := os.WriteFile(filename, []byte{}, 0644)
		if err != nil {
			t.Fatalf("Failed to create empty file: %v", err)
		}
		filenames = append(filenames, filename)
	}
	
	// Process empty files
	config := DefaultBatchConfig()
	processor := NewBatchProcessor(config)
	processor.AddStage(ValidationStage)
	
	ctx := context.Background()
	resultChan := processor.ProcessBatch(ctx, filenames)
	result := <-resultChan
	
	// Verify results
	if result.FilesProcessed != int64(len(filenames)) {
		t.Errorf("Expected %d files processed, got %d", len(filenames), result.FilesProcessed)
	}
	
	if result.BytesRead != 0 {
		t.Errorf("Expected 0 bytes read, got %d", result.BytesRead)
	}
	
	// Check for errors
	for i, err := range result.Errors {
		if err != nil {
			t.Errorf("Unexpected error for empty file %d: %v", i, err)
		}
	}
}

func TestBatchProcessor_LargeFiles(t *testing.T) {
	testDir := t.TempDir()
	fileCount := 10
	fileSize := 1024 * 1024 // 1MB per file
	
	// Create large test files
	filenames, err := CreateTestFiles(testDir, fileCount, fileSize)
	if err != nil {
		t.Fatalf("CreateTestFiles failed: %v", err)
	}
	
	// Configure processor
	config := DefaultBatchConfig()
	config.WorkerCount = 2 // Limit workers for large files
	config.Timeout = 30 * time.Second
	
	processor := NewBatchProcessor(config)
	processor.AddStage(ValidationStage)
	
	// Process batch
	ctx := context.Background()
	start := time.Now()
	resultChan := processor.ProcessBatch(ctx, filenames)
	result := <-resultChan
	duration := time.Since(start)
	
	// Verify results
	if result.FilesProcessed != int64(fileCount) {
		t.Errorf("Expected %d files processed, got %d", fileCount, result.FilesProcessed)
	}
	
	expectedBytes := int64(fileCount * fileSize)
	if result.BytesRead != expectedBytes {
		t.Errorf("Expected %d bytes read, got %d", expectedBytes, result.BytesRead)
	}
	
	// Calculate throughput
	if duration > 0 {
		throughputMBps := float64(result.BytesRead) / (1024 * 1024) / duration.Seconds()
		t.Logf("Large file processing throughput: %.2f MB/s", throughputMBps)
		
		// Should achieve reasonable throughput for large files
		if throughputMBps < 10 {
			t.Logf("Warning: Low throughput for large files: %.2f MB/s", throughputMBps)
		}
	}
	
	// Check for errors
	errorCount := 0
	for _, err := range result.Errors {
		if err != nil {
			errorCount++
		}
	}
	
	if errorCount > 0 {
		t.Errorf("Expected no errors for large files, got %d", errorCount)
	}
}

func TestBatchProcessor_NonExistentFiles(t *testing.T) {
	// Test with non-existent files
	filenames := []string{
		"non_existent_1.txt",
		"non_existent_2.txt",
		"non_existent_3.txt",
	}
	
	config := DefaultBatchConfig()
	processor := NewBatchProcessor(config)
	
	ctx := context.Background()
	resultChan := processor.ProcessBatch(ctx, filenames)
	result := <-resultChan
	
	// Should have errors for all files
	errorCount := 0
	for _, err := range result.Errors {
		if err != nil {
			errorCount++
		}
	}
	
	if errorCount != len(filenames) {
		t.Errorf("Expected %d errors for non-existent files, got %d", len(filenames), errorCount)
	}
	
	if result.FilesProcessed != 0 {
		t.Errorf("Expected 0 files processed, got %d", result.FilesProcessed)
	}
	
	if result.BytesRead != 0 {
		t.Errorf("Expected 0 bytes read, got %d", result.BytesRead)
	}
}

// Benchmark tests

func BenchmarkBatchProcessor_SmallFiles(b *testing.B) {
	testDir := b.TempDir()
	fileCount := 100
	fileSize := 1024 // 1KB
	
	filenames, err := CreateTestFiles(testDir, fileCount, fileSize)
	if err != nil {
		b.Fatalf("CreateTestFiles failed: %v", err)
	}
	
	config := DefaultBatchConfig()
	config.WorkerCount = runtime.NumCPU()
	processor := NewBatchProcessor(config)
	processor.AddStage(ValidationStage)
	processor.AddStage(UppercaseStage)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		resultChan := processor.ProcessBatch(ctx, filenames)
		result := <-resultChan
		
		if result.FilesProcessed != int64(fileCount) {
			b.Errorf("Expected %d files processed, got %d", fileCount, result.FilesProcessed)
		}
	}
}

func BenchmarkBatchProcessor_LargeFiles(b *testing.B) {
	testDir := b.TempDir()
	fileCount := 10
	fileSize := 1024 * 1024 // 1MB
	
	filenames, err := CreateTestFiles(testDir, fileCount, fileSize)
	if err != nil {
		b.Fatalf("CreateTestFiles failed: %v", err)
	}
	
	config := DefaultBatchConfig()
	config.WorkerCount = runtime.NumCPU()
	processor := NewBatchProcessor(config)
	processor.AddStage(ValidationStage)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		resultChan := processor.ProcessBatch(ctx, filenames)
		result := <-resultChan
		
		if result.FilesProcessed != int64(fileCount) {
			b.Errorf("Expected %d files processed, got %d", fileCount, result.FilesProcessed)
		}
	}
}

func BenchmarkProcessingStages(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte('a' + (i % 26))
	}
	
	b.Run("UppercaseStage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := UppercaseStage(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("ValidationStage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := ValidationStage(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("CompressionStage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := CompressionStage(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
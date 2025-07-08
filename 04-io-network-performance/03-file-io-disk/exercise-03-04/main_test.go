package main

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestDefaultBufferConfig(t *testing.T) {
	config := DefaultBufferConfig()
	
	if config.ReadBufferSize != 64*1024 {
		t.Errorf("Expected ReadBufferSize to be 65536, got %d", config.ReadBufferSize)
	}
	
	if config.WriteBufferSize != 64*1024 {
		t.Errorf("Expected WriteBufferSize to be 65536, got %d", config.WriteBufferSize)
	}
	
	if !config.UseBuffering {
		t.Error("Expected UseBuffering to be true")
	}
	
	if !config.AlignToPage {
		t.Error("Expected AlignToPage to be true")
	}
	
	if !config.PoolBuffers {
		t.Error("Expected PoolBuffers to be true")
	}
}

func TestCreateAndCleanupTestFiles(t *testing.T) {
	filenames, err := CreateTestFiles(5, 1) // 5 files, 1KB each
	if err != nil {
		t.Fatalf("Failed to create test files: %v", err)
	}
	
	if len(filenames) != 5 {
		t.Errorf("Expected 5 filenames, got %d", len(filenames))
	}
	
	// Verify files exist
	for _, filename := range filenames {
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			t.Errorf("File %s does not exist", filename)
		}
	}
	
	// Cleanup
	CleanupTestFiles(filenames)
	
	// Verify files are removed
	for _, filename := range filenames {
		if _, err := os.Stat(filename); !os.IsNotExist(err) {
			t.Errorf("File %s still exists after cleanup", filename)
		}
	}
}

func TestIOProfiler(t *testing.T) {
	profiler := NewIOProfiler()
	
	// Test initial state
	stats := profiler.GetStats()
	if stats.TotalReads != 0 {
		t.Errorf("Expected TotalReads to be 0, got %d", stats.TotalReads)
	}
	
	if stats.TotalWrites != 0 {
		t.Errorf("Expected TotalWrites to be 0, got %d", stats.TotalWrites)
	}
	
	// Record some operations
	profiler.recordRead(100 * time.Microsecond)
	profiler.recordWrite(200 * time.Microsecond)
	profiler.recordBytesRead(1024)
	profiler.recordBytesWritten(2048)
	profiler.recordBufferHit()
	profiler.recordBufferMiss()
	
	stats = profiler.GetStats()
	if stats.TotalReads != 1 {
		t.Errorf("Expected TotalReads to be 1, got %d", stats.TotalReads)
	}
	
	if stats.TotalWrites != 1 {
		t.Errorf("Expected TotalWrites to be 1, got %d", stats.TotalWrites)
	}
	
	if stats.BytesRead != 1024 {
		t.Errorf("Expected BytesRead to be 1024, got %d", stats.BytesRead)
	}
	
	if stats.BytesWritten != 2048 {
		t.Errorf("Expected BytesWritten to be 2048, got %d", stats.BytesWritten)
	}
	
	if stats.BufferHitRatio != 0.5 {
		t.Errorf("Expected BufferHitRatio to be 0.5, got %f", stats.BufferHitRatio)
	}
}

func TestOptimizedFileReader(t *testing.T) {
	config := DefaultBufferConfig()
	profiler := NewIOProfiler()
	reader := NewOptimizedFileReader(profiler, config)
	
	// Create test file
	testData := []byte("Hello, World! This is a test file for reading.")
	testFile := "test_reader.txt"
	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	_, err = file.Write(testData)
	file.Close()
	if err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	defer os.Remove(testFile)
	
	// Test reading
	data, err := reader.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	
	if string(data) != string(testData) {
		t.Errorf("Read data doesn't match written data")
	}
	
	// Check profiler stats
	stats := profiler.GetStats()
	if stats.TotalReads != 1 {
		t.Errorf("Expected TotalReads to be 1, got %d", stats.TotalReads)
	}
	
	if stats.BytesRead != int64(len(testData)) {
		t.Errorf("Expected BytesRead to be %d, got %d", len(testData), stats.BytesRead)
	}
}

func TestOptimizedFileWriter(t *testing.T) {
	config := DefaultBufferConfig()
	profiler := NewIOProfiler()
	writer := NewOptimizedFileWriter(profiler, config)
	
	// Test writing
	testData := []byte("Hello, World! This is a test file for writing.")
	testFile := "test_writer.txt"
	defer os.Remove(testFile)
	
	err := writer.WriteFile(testFile, testData)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	
	// Verify file contents
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	
	if string(data) != string(testData) {
		t.Errorf("Written data doesn't match expected data")
	}
	
	// Check profiler stats
	stats := profiler.GetStats()
	if stats.TotalWrites != 1 {
		t.Errorf("Expected TotalWrites to be 1, got %d", stats.TotalWrites)
	}
	
	if stats.BytesWritten != int64(len(testData)) {
		t.Errorf("Expected BytesWritten to be %d, got %d", len(testData), stats.BytesWritten)
	}
}

func TestDiskIOOptimizer_ProcessFiles(t *testing.T) {
	config := DefaultBufferConfig()
	optimizer := NewDiskIOOptimizer(config)
	
	// Create test files
	filenames, err := CreateTestFiles(10, 1) // 10 files, 1KB each
	if err != nil {
		t.Fatalf("Failed to create test files: %v", err)
	}
	defer CleanupTestFiles(filenames)
	
	// Process files
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	err = optimizer.ProcessFiles(ctx, filenames, 4)
	if err != nil {
		t.Fatalf("Failed to process files: %v", err)
	}
	
	// Verify processed files exist
	for _, filename := range filenames {
		processedFile := filename + ".processed"
		if _, err := os.Stat(processedFile); os.IsNotExist(err) {
			t.Errorf("Processed file %s does not exist", processedFile)
		}
	}
	
	// Check profiler stats
	stats := optimizer.profiler.GetStats()
	if stats.TotalReads != int64(len(filenames)) {
		t.Errorf("Expected TotalReads to be %d, got %d", len(filenames), stats.TotalReads)
	}
	
	if stats.TotalWrites != int64(len(filenames)) {
		t.Errorf("Expected TotalWrites to be %d, got %d", len(filenames), stats.TotalWrites)
	}
	
	if stats.BytesRead == 0 {
		t.Error("Expected BytesRead to be greater than 0")
	}
	
	if stats.BytesWritten == 0 {
		t.Error("Expected BytesWritten to be greater than 0")
	}
}

func TestDiskIOOptimizer_ContextCancellation(t *testing.T) {
	config := DefaultBufferConfig()
	optimizer := NewDiskIOOptimizer(config)
	
	// Create test files
	filenames, err := CreateTestFiles(100, 10) // 100 files, 10KB each
	if err != nil {
		t.Fatalf("Failed to create test files: %v", err)
	}
	defer CleanupTestFiles(filenames)
	
	// Process files with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	
	err = optimizer.ProcessFiles(ctx, filenames, 2)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
	
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func TestOptimizeBufferSize(t *testing.T) {
	config := DefaultBufferConfig()
	optimizer := NewDiskIOOptimizer(config)
	
	testSizes := []int{4096, 8192, 16384}
	testFile := "buffer_optimization_test.dat"
	defer os.Remove(testFile)
	
	optimalSize, err := optimizer.OptimizeBufferSize(testFile, testSizes)
	if err != nil {
		t.Fatalf("Failed to optimize buffer size: %v", err)
	}
	
	if optimalSize == 0 {
		t.Error("Expected optimal size to be greater than 0")
	}
	
	// Verify optimal size is one of the test sizes
	found := false
	for _, size := range testSizes {
		if optimalSize == size {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("Optimal size %d is not in test sizes %v", optimalSize, testSizes)
	}
}

func TestBufferedVsDirectIO(t *testing.T) {
	// Test buffered I/O
	bufferedConfig := DefaultBufferConfig()
	bufferedConfig.UseBuffering = true
	bufferedOptimizer := NewDiskIOOptimizer(bufferedConfig)
	
	// Test direct I/O
	directConfig := DefaultBufferConfig()
	directConfig.UseBuffering = false
	directOptimizer := NewDiskIOOptimizer(directConfig)
	
	// Create test files
	filenames, err := CreateTestFiles(20, 5) // 20 files, 5KB each
	if err != nil {
		t.Fatalf("Failed to create test files: %v", err)
	}
	defer CleanupTestFiles(filenames)
	
	ctx := context.Background()
	
	// Test buffered I/O
	err = bufferedOptimizer.ProcessFiles(ctx, filenames, 4)
	if err != nil {
		t.Fatalf("Failed to process files with buffered I/O: %v", err)
	}
	bufferedStats := bufferedOptimizer.profiler.GetStats()
	
	// Clean up processed files
	for _, filename := range filenames {
		os.Remove(filename + ".processed")
	}
	
	// Test direct I/O
	err = directOptimizer.ProcessFiles(ctx, filenames, 4)
	if err != nil {
		t.Fatalf("Failed to process files with direct I/O: %v", err)
	}
	directStats := directOptimizer.profiler.GetStats()
	
	// Both should process the same amount of data
	if bufferedStats.BytesRead != directStats.BytesRead {
		t.Errorf("Buffered and direct I/O should read same amount of data")
	}
	
	if bufferedStats.BytesWritten != directStats.BytesWritten {
		t.Errorf("Buffered and direct I/O should write same amount of data")
	}
	
	// Buffered I/O should have higher buffer hit ratio
	if bufferedStats.BufferHitRatio <= directStats.BufferHitRatio {
		t.Errorf("Buffered I/O should have higher buffer hit ratio")
	}
}

func TestLargeFileProcessing(t *testing.T) {
	config := DefaultBufferConfig()
	optimizer := NewDiskIOOptimizer(config)
	
	// Create large test files
	filenames, err := CreateTestFiles(5, 1024) // 5 files, 1MB each
	if err != nil {
		t.Fatalf("Failed to create large test files: %v", err)
	}
	defer CleanupTestFiles(filenames)
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	start := time.Now()
	err = optimizer.ProcessFiles(ctx, filenames, 4)
	processingTime := time.Since(start)
	
	if err != nil {
		t.Fatalf("Failed to process large files: %v", err)
	}
	
	stats := optimizer.profiler.GetStats()
	
	// Calculate throughput
	totalMB := float64(stats.BytesRead) / (1024 * 1024)
	throughput := totalMB / processingTime.Seconds()
	
	t.Logf("Processed %.2f MB in %v (%.2f MB/s)", totalMB, processingTime, throughput)
	
	// Performance assertions
	if stats.ReadThroughput < 1.0 {
		t.Errorf("Read throughput too low: %.2f MB/s", stats.ReadThroughput)
	}
	
	if stats.WriteThroughput < 1.0 {
		t.Errorf("Write throughput too low: %.2f MB/s", stats.WriteThroughput)
	}
	
	if stats.IOPS < 1.0 {
		t.Errorf("IOPS too low: %.2f", stats.IOPS)
	}
}

func TestMemoryEfficiency(t *testing.T) {
	config := DefaultBufferConfig()
	config.PoolBuffers = true
	optimizer := NewDiskIOOptimizer(config)
	
	// Create test files
	filenames, err := CreateTestFiles(50, 20) // 50 files, 20KB each
	if err != nil {
		t.Fatalf("Failed to create test files: %v", err)
	}
	defer CleanupTestFiles(filenames)
	
	ctx := context.Background()
	
	err = optimizer.ProcessFiles(ctx, filenames, 8)
	if err != nil {
		t.Fatalf("Failed to process files: %v", err)
	}
	
	stats := optimizer.profiler.GetStats()
	
	// With buffer pooling, we should have good buffer hit ratio
	if stats.BufferHitRatio < 0.1 {
		t.Errorf("Buffer hit ratio too low with pooling: %.2f", stats.BufferHitRatio)
	}
	
	// Should not have any file descriptor leaks
	if stats.OpenFileCount != 0 {
		t.Errorf("File descriptor leak detected: %d open files", stats.OpenFileCount)
	}
}

// Benchmark tests
func BenchmarkOptimizedFileReader_SmallFiles(b *testing.B) {
	config := DefaultBufferConfig()
	profiler := NewIOProfiler()
	reader := NewOptimizedFileReader(profiler, config)
	
	// Create test file
	testData := make([]byte, 1024) // 1KB
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	
	testFile := "bench_small.dat"
	file, _ := os.Create(testFile)
	file.Write(testData)
	file.Close()
	defer os.Remove(testFile)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reader.ReadFile(testFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOptimizedFileReader_LargeFiles(b *testing.B) {
	config := DefaultBufferConfig()
	profiler := NewIOProfiler()
	reader := NewOptimizedFileReader(profiler, config)
	
	// Create test file
	testData := make([]byte, 1024*1024) // 1MB
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	
	testFile := "bench_large.dat"
	file, _ := os.Create(testFile)
	file.Write(testData)
	file.Close()
	defer os.Remove(testFile)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reader.ReadFile(testFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDiskIOOptimizer_ProcessFiles(b *testing.B) {
	config := DefaultBufferConfig()
	optimizer := NewDiskIOOptimizer(config)
	
	// Create test files
	filenames, _ := CreateTestFiles(10, 10) // 10 files, 10KB each
	defer CleanupTestFiles(filenames)
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset profiler for each iteration
		optimizer.profiler = NewIOProfiler()
		optimizer.reader = NewOptimizedFileReader(optimizer.profiler, config)
		optimizer.writer = NewOptimizedFileWriter(optimizer.profiler, config)
		
		err := optimizer.ProcessFiles(ctx, filenames, 4)
		if err != nil {
			b.Fatal(err)
		}
		
		// Clean up processed files
		for _, filename := range filenames {
			os.Remove(filename + ".processed")
		}
	}
}

func BenchmarkBufferSizeOptimization(b *testing.B) {
	config := DefaultBufferConfig()
	optimizer := NewDiskIOOptimizer(config)
	
	testSizes := []int{4096, 8192, 16384, 32768}
	testFile := "bench_buffer_opt.dat"
	defer os.Remove(testFile)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := optimizer.OptimizeBufferSize(testFile, testSizes)
		if err != nil {
			b.Fatal(err)
		}
	}
}
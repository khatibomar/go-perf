package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultMMapConfig(t *testing.T) {
	config := DefaultMMapConfig()
	
	if !config.ReadOnly {
		t.Error("Expected ReadOnly to be true")
	}
	if config.Preload {
		t.Error("Expected Preload to be false")
	}
	if config.Advise != 0 {
		t.Error("Expected Advise to be 0")
	}
	if config.BufferSize != 64*1024 {
		t.Errorf("Expected BufferSize to be 65536, got %d", config.BufferSize)
	}
}

func TestCreateTestFile(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "test.dat")
	size := int64(1024 * 1024) // 1MB
	
	err := CreateTestFile(testFile, size)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Verify file exists and has correct size
	stat, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}
	
	if stat.Size() != size {
		t.Errorf("Expected file size %d, got %d", size, stat.Size())
	}
}

func TestOpenMMapFile(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "test.dat")
	size := int64(64 * 1024) // 64KB
	
	// Create test file
	err := CreateTestFile(testFile, size)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Test opening with default config
	config := DefaultMMapConfig()
	mmap, err := OpenMMapFile(testFile, config)
	if err != nil {
		t.Fatalf("Failed to open mmap file: %v", err)
	}
	defer mmap.Close()
	
	if mmap.Size() != size {
		t.Errorf("Expected size %d, got %d", size, mmap.Size())
	}
	
	if !mmap.readOnly {
		t.Error("Expected file to be read-only")
	}
}

func TestCreateMMapFile(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "create_test.dat")
	size := int64(32 * 1024) // 32KB
	
	mmap, err := CreateMMapFile(testFile, size)
	if err != nil {
		t.Fatalf("Failed to create mmap file: %v", err)
	}
	defer mmap.Close()
	
	if mmap.Size() != size {
		t.Errorf("Expected size %d, got %d", size, mmap.Size())
	}
	
	if mmap.readOnly {
		t.Error("Expected file to be writable")
	}
	
	// Verify file exists
	stat, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat created file: %v", err)
	}
	
	if stat.Size() != size {
		t.Errorf("Expected file size %d, got %d", size, stat.Size())
	}
}

func TestMMapFileReadAt(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "read_test.dat")
	size := int64(1024)
	
	// Create test file with known pattern
	err := CreateTestFile(testFile, size)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	config := DefaultMMapConfig()
	mmap, err := OpenMMapFile(testFile, config)
	if err != nil {
		t.Fatalf("Failed to open mmap file: %v", err)
	}
	defer mmap.Close()
	
	// Test reading at different offsets
	tests := []struct {
		offset int64
		length int64
	}{
		{0, 100},
		{100, 200},
		{500, 100},
		{1000, 24}, // Near end
	}
	
	for _, test := range tests {
		data, err := mmap.ReadAt(test.offset, test.length)
		if err != nil {
			t.Errorf("Failed to read at offset %d: %v", test.offset, err)
			continue
		}
		
		expectedLength := test.length
		if test.offset+test.length > size {
			expectedLength = size - test.offset
		}
		
		if int64(len(data)) != expectedLength {
			t.Errorf("Expected length %d, got %d", expectedLength, len(data))
		}
		
		// Verify pattern
		for i, b := range data {
			expected := byte((test.offset + int64(i)) % 256)
			if b != expected {
				t.Errorf("Data mismatch at offset %d: expected %d, got %d", test.offset+int64(i), expected, b)
				break
			}
		}
	}
}

func TestMMapFileReadAtNoCopy(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "nocopy_test.dat")
	size := int64(1024)
	
	err := CreateTestFile(testFile, size)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	config := DefaultMMapConfig()
	mmap, err := OpenMMapFile(testFile, config)
	if err != nil {
		t.Fatalf("Failed to open mmap file: %v", err)
	}
	defer mmap.Close()
	
	// Test zero-copy read
	data, err := mmap.ReadAtNoCopy(0, 100)
	if err != nil {
		t.Fatalf("Failed to read no-copy: %v", err)
	}
	
	if len(data) != 100 {
		t.Errorf("Expected length 100, got %d", len(data))
	}
	
	// Verify pattern
	for i, b := range data {
		expected := byte(i % 256)
		if b != expected {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, expected, b)
			break
		}
	}
}

func TestMMapFileWriteAt(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "write_test.dat")
	size := int64(1024)
	
	mmap, err := CreateMMapFile(testFile, size)
	if err != nil {
		t.Fatalf("Failed to create mmap file: %v", err)
	}
	defer mmap.Close()
	
	// Test writing data
	testData := []byte("Hello, Memory-Mapped World!")
	err = mmap.WriteAt(testData, 100)
	if err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}
	
	// Read back and verify
	readData, err := mmap.ReadAt(100, int64(len(testData)))
	if err != nil {
		t.Fatalf("Failed to read back data: %v", err)
	}
	
	if !bytes.Equal(testData, readData) {
		t.Errorf("Data mismatch: expected %s, got %s", string(testData), string(readData))
	}
}

func TestMMapFileWriteAtReadOnly(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "readonly_test.dat")
	size := int64(1024)
	
	err := CreateTestFile(testFile, size)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	config := DefaultMMapConfig() // Read-only by default
	mmap, err := OpenMMapFile(testFile, config)
	if err != nil {
		t.Fatalf("Failed to open mmap file: %v", err)
	}
	defer mmap.Close()
	
	// Try to write to read-only file
	testData := []byte("Should fail")
	err = mmap.WriteAt(testData, 0)
	if err == nil {
		t.Error("Expected error when writing to read-only file")
	}
}

func TestMMapFileSync(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "sync_test.dat")
	size := int64(1024)
	
	mmap, err := CreateMMapFile(testFile, size)
	if err != nil {
		t.Fatalf("Failed to create mmap file: %v", err)
	}
	defer mmap.Close()
	
	// Write some data
	testData := []byte("Sync test data")
	err = mmap.WriteAt(testData, 0)
	if err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}
	
	// Sync to disk
	err = mmap.Sync()
	if err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}
}

func TestMMapReader(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "reader_test.dat")
	size := int64(1024)
	
	err := CreateTestFile(testFile, size)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	config := DefaultMMapConfig()
	mmap, err := OpenMMapFile(testFile, config)
	if err != nil {
		t.Fatalf("Failed to open mmap file: %v", err)
	}
	defer mmap.Close()
	
	reader := NewMMapReader(mmap)
	
	// Test reading
	buffer := make([]byte, 100)
	n, err := reader.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	
	if n != 100 {
		t.Errorf("Expected to read 100 bytes, got %d", n)
	}
	
	// Verify pattern
	for i, b := range buffer {
		expected := byte(i % 256)
		if b != expected {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, expected, b)
			break
		}
	}
	
	// Test seeking
	pos, err := reader.Seek(500, io.SeekStart)
	if err != nil {
		t.Fatalf("Failed to seek: %v", err)
	}
	
	if pos != 500 {
		t.Errorf("Expected position 500, got %d", pos)
	}
	
	// Read after seek
	n, err = reader.Read(buffer[:50])
	if err != nil {
		t.Fatalf("Failed to read after seek: %v", err)
	}
	
	if n != 50 {
		t.Errorf("Expected to read 50 bytes, got %d", n)
	}
	
	// Verify pattern after seek
	for i, b := range buffer[:50] {
		expected := byte((500 + i) % 256)
		if b != expected {
			t.Errorf("Data mismatch at index %d after seek: expected %d, got %d", i, expected, b)
			break
		}
	}
}

func TestMMapStats(t *testing.T) {
	stats := MMapStats{
		ReadOperations:  10,
		WriteOperations: 5,
		BytesRead:       1024,
		BytesWritten:    512,
		TotalReadTime:   time.Millisecond * 100,
		TotalWriteTime:  time.Millisecond * 50,
	}
	
	avgReadTime := stats.GetAverageReadTime()
	expectedAvgRead := time.Millisecond * 10
	if avgReadTime != expectedAvgRead {
		t.Errorf("Expected average read time %v, got %v", expectedAvgRead, avgReadTime)
	}
	
	avgWriteTime := stats.GetAverageWriteTime()
	expectedAvgWrite := time.Millisecond * 10
	if avgWriteTime != expectedAvgWrite {
		t.Errorf("Expected average write time %v, got %v", expectedAvgWrite, avgWriteTime)
	}
	
	readThroughput := stats.GetReadThroughput()
	expectedReadThroughput := float64(1024) / 0.1 // 1024 bytes / 0.1 seconds
	if readThroughput != expectedReadThroughput {
		t.Errorf("Expected read throughput %f, got %f", expectedReadThroughput, readThroughput)
	}
	
	writeThroughput := stats.GetWriteThroughput()
	expectedWriteThroughput := float64(512) / 0.05 // 512 bytes / 0.05 seconds
	if writeThroughput != expectedWriteThroughput {
		t.Errorf("Expected write throughput %f, got %f", expectedWriteThroughput, writeThroughput)
	}
}

func TestBenchmarkRandomAccess(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "benchmark_test.dat")
	size := int64(64 * 1024) // 64KB
	
	err := CreateTestFile(testFile, size)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	config := DefaultMMapConfig()
	mmap, err := OpenMMapFile(testFile, config)
	if err != nil {
		t.Fatalf("Failed to open mmap file: %v", err)
	}
	defer mmap.Close()
	
	stats, err := BenchmarkRandomAccess(mmap, 100)
	if err != nil {
		t.Fatalf("Benchmark failed: %v", err)
	}
	
	if stats.ReadOperations != 100 {
		t.Errorf("Expected 100 read operations, got %d", stats.ReadOperations)
	}
	
	if stats.BytesRead == 0 {
		t.Error("Expected some bytes to be read")
	}
	
	// On fast systems, read time might be very small or zero
	// Just check that we have valid stats
	if stats.ReadOperations == 0 {
		t.Error("Expected some read operations")
	}
}

func TestErrorHandling(t *testing.T) {
	// Test opening non-existent file
	config := DefaultMMapConfig()
	_, err := OpenMMapFile("non_existent_file.dat", config)
	if err == nil {
		t.Error("Expected error when opening non-existent file")
	}
	
	// Test reading out of bounds
	testFile := filepath.Join(t.TempDir(), "bounds_test.dat")
	size := int64(100)
	
	err = CreateTestFile(testFile, size)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	mmap, err := OpenMMapFile(testFile, config)
	if err != nil {
		t.Fatalf("Failed to open mmap file: %v", err)
	}
	defer mmap.Close()
	
	// Test reading beyond file size
	_, err = mmap.ReadAt(200, 10)
	if err == nil {
		t.Error("Expected error when reading beyond file size")
	}
	
	// Test negative offset
	_, err = mmap.ReadAt(-1, 10)
	if err == nil {
		t.Error("Expected error when reading with negative offset")
	}
}

// Benchmarks
func BenchmarkMMapReadAt(b *testing.B) {
	testFile := filepath.Join(b.TempDir(), "bench_read.dat")
	size := int64(1024 * 1024) // 1MB
	
	err := CreateTestFile(testFile, size)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	
	config := DefaultMMapConfig()
	mmap, err := OpenMMapFile(testFile, config)
	if err != nil {
		b.Fatalf("Failed to open mmap file: %v", err)
	}
	defer mmap.Close()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		offset := int64(i % 1000) * 1024
		_, err := mmap.ReadAt(offset, 1024)
		if err != nil {
			b.Fatalf("Read failed: %v", err)
		}
	}
}

func BenchmarkMMapReadAtNoCopy(b *testing.B) {
	testFile := filepath.Join(b.TempDir(), "bench_nocopy.dat")
	size := int64(1024 * 1024) // 1MB
	
	err := CreateTestFile(testFile, size)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	
	config := DefaultMMapConfig()
	mmap, err := OpenMMapFile(testFile, config)
	if err != nil {
		b.Fatalf("Failed to open mmap file: %v", err)
	}
	defer mmap.Close()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		offset := int64(i % 1000) * 1024
		_, err := mmap.ReadAtNoCopy(offset, 1024)
		if err != nil {
			b.Fatalf("Read failed: %v", err)
		}
	}
}

func BenchmarkMMapWriteAt(b *testing.B) {
	testFile := filepath.Join(b.TempDir(), "bench_write.dat")
	size := int64(1024 * 1024) // 1MB
	
	mmap, err := CreateMMapFile(testFile, size)
	if err != nil {
		b.Fatalf("Failed to create mmap file: %v", err)
	}
	defer mmap.Close()
	
	testData := make([]byte, 1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		offset := int64(i % 1000) * 1024
		err := mmap.WriteAt(testData, offset)
		if err != nil {
			b.Fatalf("Write failed: %v", err)
		}
	}
}

func BenchmarkRandomAccessPerformance(b *testing.B) {
	testFile := filepath.Join(b.TempDir(), "bench_random.dat")
	size := int64(10 * 1024 * 1024) // 10MB
	
	err := CreateTestFile(testFile, size)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	
	config := DefaultMMapConfig()
	mmap, err := OpenMMapFile(testFile, config)
	if err != nil {
		b.Fatalf("Failed to open mmap file: %v", err)
	}
	defer mmap.Close()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, err := BenchmarkRandomAccess(mmap, 100)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}
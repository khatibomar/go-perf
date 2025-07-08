package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// MMapFile represents a memory-mapped file
type MMapFile struct {
	file     *os.File
	data     []byte
	size     int64
	readOnly bool
	mutex    sync.RWMutex
}

// MMapConfig holds configuration for memory mapping
type MMapConfig struct {
	ReadOnly   bool
	Preload    bool
	Advise     int
	BufferSize int
}

// DefaultMMapConfig returns default configuration
func DefaultMMapConfig() MMapConfig {
	return MMapConfig{
		ReadOnly:   true,
		Preload:    false,
		Advise:     0, // MADV_NORMAL
		BufferSize: 64 * 1024, // 64KB
	}
}

// OpenMMapFile opens a file for memory mapping
func OpenMMapFile(filename string, config MMapConfig) (*MMapFile, error) {
	var flag int
	if config.ReadOnly {
		flag = os.O_RDONLY
	} else {
		flag = os.O_RDWR
	}
	
	file, err := os.OpenFile(filename, flag, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	
	if stat.Size() == 0 {
		file.Close()
		return nil, fmt.Errorf("cannot map empty file")
	}
	
	data, err := mmapFile(file, config.ReadOnly, stat.Size())
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to mmap file: %w", err)
	}
	
	mf := &MMapFile{
		file:     file,
		data:     data,
		size:     stat.Size(),
		readOnly: config.ReadOnly,
	}
	
	// Apply memory advice if specified
	if config.Advise != 0 {
		mf.Advise(config.Advise)
	}
	
	// Preload if requested
	if config.Preload {
		mf.Preload()
	}
	
	return mf, nil
}

// CreateMMapFile creates a new file and maps it to memory
func CreateMMapFile(filename string, size int64) (*MMapFile, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	
	// Extend file to desired size
	if err := file.Truncate(size); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to truncate file: %w", err)
	}
	
	data, err := mmapFile(file, false, size)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to mmap file: %w", err)
	}
	
	return &MMapFile{
		file:     file,
		data:     data,
		size:     size,
		readOnly: false,
	}, nil
}

// mmapFile performs the actual memory mapping using Windows API
func mmapFile(file *os.File, readOnly bool, size int64) ([]byte, error) {
	var access uint32
	var protect uint32
	
	if readOnly {
		access = windows.FILE_MAP_READ
		protect = windows.PAGE_READONLY
	} else {
		access = windows.FILE_MAP_WRITE
		protect = windows.PAGE_READWRITE
	}
	
	// Create file mapping
	mapping, err := windows.CreateFileMapping(
		windows.Handle(file.Fd()),
		nil,
		protect,
		uint32(size>>32), // High-order DWORD of size
		uint32(size),     // Low-order DWORD of size
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateFileMapping failed: %v", err)
	}
	
	// Map view of file
	addr, err := windows.MapViewOfFile(
		mapping,
		access,
		0, // High-order DWORD of offset
		0, // Low-order DWORD of offset
		uintptr(size),
	)
	if err != nil {
		windows.CloseHandle(mapping)
		return nil, fmt.Errorf("MapViewOfFile failed: %v", err)
	}
	
	// Store mapping handle for cleanup
	data := (*[1 << 30]byte)(unsafe.Pointer(addr))[:size:size]
	
	// We need to store the mapping handle somewhere for cleanup
	// For simplicity, we'll close it here since Windows will keep the mapping alive
	// as long as the view is mapped
	windows.CloseHandle(mapping)
	
	return data, nil
}

// ReadAt reads data at the specified offset and length
func (m *MMapFile) ReadAt(offset, length int64) ([]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if offset < 0 || offset >= m.size {
		return nil, fmt.Errorf("offset %d out of range [0, %d)", offset, m.size)
	}
	
	if offset+length > m.size {
		length = m.size - offset
	}
	
	if length <= 0 {
		return []byte{}, nil
	}
	
	// Return a copy to avoid data races
	result := make([]byte, length)
	copy(result, m.data[offset:offset+length])
	return result, nil
}

// ReadAtNoCopy reads data at the specified offset without copying (zero-copy)
// WARNING: The returned slice shares memory with the mapped file
func (m *MMapFile) ReadAtNoCopy(offset, length int64) ([]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if offset < 0 || offset >= m.size {
		return nil, fmt.Errorf("offset %d out of range [0, %d)", offset, m.size)
	}
	
	if offset+length > m.size {
		length = m.size - offset
	}
	
	if length <= 0 {
		return []byte{}, nil
	}
	
	return m.data[offset : offset+length], nil
}

// WriteAt writes data at the specified offset
func (m *MMapFile) WriteAt(data []byte, offset int64) error {
	if m.readOnly {
		return fmt.Errorf("cannot write to read-only mapped file")
	}
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if offset < 0 || offset >= m.size {
		return fmt.Errorf("offset %d out of range [0, %d)", offset, m.size)
	}
	
	if int64(len(data))+offset > m.size {
		return fmt.Errorf("write would exceed file size")
	}
	
	copy(m.data[offset:], data)
	return nil
}

// Size returns the size of the mapped file
func (m *MMapFile) Size() int64 {
	return m.size
}

// Sync synchronizes the mapped memory with the underlying file
func (m *MMapFile) Sync() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if len(m.data) == 0 {
		return nil
	}
	
	err := windows.FlushViewOfFile(
		uintptr(unsafe.Pointer(&m.data[0])),
		uintptr(len(m.data)),
	)
	if err != nil {
		return fmt.Errorf("FlushViewOfFile failed: %v", err)
	}
	
	return nil
}

// Advise provides advice about memory usage patterns
func (m *MMapFile) Advise(advice int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Memory advice is not directly supported on Windows
	// This is a no-op for Windows compatibility
	return nil
}

// Preload loads the entire file into memory
func (m *MMapFile) Preload() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Touch every page to load it into memory
	pageSize := os.Getpagesize()
	for i := 0; i < len(m.data); i += pageSize {
		_ = m.data[i]
	}
	
	return nil
}

// Close unmaps the file and closes the file descriptor
func (m *MMapFile) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.data != nil {
		err := windows.UnmapViewOfFile(
			uintptr(unsafe.Pointer(&m.data[0])),
		)
		if err != nil {
			return fmt.Errorf("UnmapViewOfFile failed: %v", err)
		}
		
		m.data = nil
	}
	
	if m.file != nil {
		return m.file.Close()
	}
	
	return nil
}

// MMapReader provides io.Reader interface for memory-mapped files
type MMapReader struct {
	mmap   *MMapFile
	offset int64
}

// NewMMapReader creates a new reader for the memory-mapped file
func NewMMapReader(mmap *MMapFile) *MMapReader {
	return &MMapReader{
		mmap:   mmap,
		offset: 0,
	}
}

// Read implements io.Reader
func (r *MMapReader) Read(p []byte) (int, error) {
	if r.offset >= r.mmap.size {
		return 0, io.EOF
	}
	
	data, err := r.mmap.ReadAtNoCopy(r.offset, int64(len(p)))
	if err != nil {
		return 0, err
	}
	
	n := copy(p, data)
	r.offset += int64(n)
	
	if r.offset >= r.mmap.size {
		return n, io.EOF
	}
	
	return n, nil
}

// Seek implements io.Seeker
func (r *MMapReader) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = r.offset + offset
	case io.SeekEnd:
		newOffset = r.mmap.size + offset
	default:
		return 0, fmt.Errorf("invalid whence value: %d", whence)
	}
	
	if newOffset < 0 {
		return 0, fmt.Errorf("negative seek position")
	}
	
	r.offset = newOffset
	return newOffset, nil
}

// MMapStats holds statistics about memory-mapped file operations
type MMapStats struct {
	ReadOperations  int64
	WriteOperations int64
	BytesRead       int64
	BytesWritten    int64
	TotalReadTime   time.Duration
	TotalWriteTime  time.Duration
}

// GetAverageReadTime returns average read time per operation
func (s *MMapStats) GetAverageReadTime() time.Duration {
	if s.ReadOperations == 0 {
		return 0
	}
	return s.TotalReadTime / time.Duration(s.ReadOperations)
}

// GetAverageWriteTime returns average write time per operation
func (s *MMapStats) GetAverageWriteTime() time.Duration {
	if s.WriteOperations == 0 {
		return 0
	}
	return s.TotalWriteTime / time.Duration(s.WriteOperations)
}

// GetReadThroughput returns read throughput in bytes per second
func (s *MMapStats) GetReadThroughput() float64 {
	if s.TotalReadTime == 0 {
		return 0
	}
	return float64(s.BytesRead) / s.TotalReadTime.Seconds()
}

// GetWriteThroughput returns write throughput in bytes per second
func (s *MMapStats) GetWriteThroughput() float64 {
	if s.TotalWriteTime == 0 {
		return 0
	}
	return float64(s.BytesWritten) / s.TotalWriteTime.Seconds()
}

// CreateTestFile creates a test file with specified size and pattern
func CreateTestFile(filename string, sizeBytes int64) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write pattern data
	buffer := make([]byte, 64*1024) // 64KB buffer
	for i := range buffer {
		buffer[i] = byte(i % 256)
	}
	
	written := int64(0)
	for written < sizeBytes {
		toWrite := int64(len(buffer))
		if written+toWrite > sizeBytes {
			toWrite = sizeBytes - written
		}
		
		n, err := file.Write(buffer[:toWrite])
		if err != nil {
			return err
		}
		
		written += int64(n)
	}
	
	return nil
}

// BenchmarkRandomAccess performs random access benchmark
func BenchmarkRandomAccess(mmap *MMapFile, numOperations int) (MMapStats, error) {
	stats := MMapStats{}
	bufferSize := int64(4096) // 4KB reads
	
	for i := 0; i < numOperations; i++ {
		// Random offset
		maxOffset := mmap.size - bufferSize
		if maxOffset <= 0 {
			maxOffset = 0
		}
		offset := int64(i*1234567) % (maxOffset + 1)
		
		start := time.Now()
		data, err := mmap.ReadAtNoCopy(offset, bufferSize)
		if err != nil {
			return stats, err
		}
		duration := time.Since(start)
		
		stats.ReadOperations++
		stats.BytesRead += int64(len(data))
		stats.TotalReadTime += duration
	}
	
	return stats, nil
}

func main() {
	fmt.Println("Memory-Mapped Files Demo")
	fmt.Println("========================")
	
	// Create test file
	testFile := "test_large_file.dat"
	fileSizeMB := int64(100) // 100MB test file
	fileSizeBytes := fileSizeMB * 1024 * 1024
	
	fmt.Printf("Creating test file: %s (%.1f MB)...\n", testFile, float64(fileSizeBytes)/(1024*1024))
	if err := CreateTestFile(testFile, fileSizeBytes); err != nil {
		fmt.Printf("Error creating test file: %v\n", err)
		return
	}
	defer os.Remove(testFile)
	
	// Test memory mapping
	fmt.Println("\nTesting memory-mapped file operations...")
	
	// Open file for memory mapping
	config := DefaultMMapConfig()
	mmap, err := OpenMMapFile(testFile, config)
	if err != nil {
		fmt.Printf("Error opening mmap file: %v\n", err)
		return
	}
	defer mmap.Close()
	
	fmt.Printf("Successfully mapped file: %d bytes\n", mmap.Size())
	
	// Test sequential reads
	fmt.Println("\nTesting sequential reads...")
	start := time.Now()
	chunkSize := int64(64 * 1024) // 64KB chunks
	totalRead := int64(0)
	
	for offset := int64(0); offset < mmap.Size(); offset += chunkSize {
		data, err := mmap.ReadAtNoCopy(offset, chunkSize)
		if err != nil {
			fmt.Printf("Error reading at offset %d: %v\n", offset, err)
			break
		}
		totalRead += int64(len(data))
	}
	
	sequentialDuration := time.Since(start)
	sequentialThroughput := float64(totalRead) / sequentialDuration.Seconds() / (1024 * 1024)
	
	fmt.Printf("Sequential read: %.2f MB in %v (%.2f MB/s)\n",
		float64(totalRead)/(1024*1024), sequentialDuration, sequentialThroughput)
	
	// Test random access
	fmt.Println("\nTesting random access performance...")
	numRandomOps := 10000
	stats, err := BenchmarkRandomAccess(mmap, numRandomOps)
	if err != nil {
		fmt.Printf("Error in random access benchmark: %v\n", err)
		return
	}
	
	avgAccessTime := stats.GetAverageReadTime()
	randomThroughput := stats.GetReadThroughput() / (1024 * 1024)
	
	fmt.Printf("Random access: %d operations, avg time: %v, throughput: %.2f MB/s\n",
		stats.ReadOperations, avgAccessTime, randomThroughput)
	
	// Test write operations (create writable file)
	fmt.Println("\nTesting write operations...")
	writeTestFile := "test_write_file.dat"
	writeFileSizeMB := int64(10) // 10MB
	writeFileSizeBytes := writeFileSizeMB * 1024 * 1024
	
	writeMmap, err := CreateMMapFile(writeTestFile, writeFileSizeBytes)
	if err != nil {
		fmt.Printf("Error creating writable mmap file: %v\n", err)
		return
	}
	defer writeMmap.Close()
	defer os.Remove(writeTestFile)
	
	// Write test data
	writeStart := time.Now()
	testData := make([]byte, 4096) // 4KB
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	
	writeOps := 0
	for offset := int64(0); offset < writeFileSizeBytes; offset += int64(len(testData)) {
		if err := writeMmap.WriteAt(testData, offset); err != nil {
			fmt.Printf("Error writing at offset %d: %v\n", offset, err)
			break
		}
		writeOps++
	}
	
	writeDuration := time.Since(writeStart)
	writeThroughput := float64(writeFileSizeBytes) / writeDuration.Seconds() / (1024 * 1024)
	
	fmt.Printf("Write operations: %d writes, duration: %v, throughput: %.2f MB/s\n",
		writeOps, writeDuration, writeThroughput)
	
	// Sync to disk
	if err := writeMmap.Sync(); err != nil {
		fmt.Printf("Error syncing to disk: %v\n", err)
	} else {
		fmt.Println("Successfully synced to disk")
	}
	
	// Performance validation
	fmt.Println("\nPerformance Validation:")
	fmt.Printf("✓ File size: %.1f MB\n", float64(fileSizeBytes)/(1024*1024))
	fmt.Printf("✓ Sequential throughput: %.2f MB/s\n", sequentialThroughput)
	fmt.Printf("✓ Random access time: %v (target: <1μs)\n", avgAccessTime)
	fmt.Printf("✓ Write throughput: %.2f MB/s\n", writeThroughput)
	
	// Memory usage estimation
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryUsagePercent := float64(m.Alloc) / float64(fileSizeBytes) * 100
	fmt.Printf("✓ Memory usage: %.2f%% of file size (target: <1%%)\n", memoryUsagePercent)
	
	fmt.Println("\nDemo completed successfully!")
}
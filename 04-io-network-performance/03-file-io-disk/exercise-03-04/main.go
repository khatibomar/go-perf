package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// IOProfiler tracks I/O performance metrics
type IOProfiler struct {
	mu              sync.RWMutex
	totalReads      int64
	totalWrites     int64
	bytesRead       int64
	bytesWritten    int64
	readDuration    time.Duration
	writeDuration   time.Duration
	syscallCount    int64
	bufferHits      int64
	bufferMisses    int64
	fileDescriptors map[string]*os.File
	startTime       time.Time
}

// IOStats represents I/O performance statistics
type IOStats struct {
	TotalReads       int64         `json:"total_reads"`
	TotalWrites      int64         `json:"total_writes"`
	BytesRead        int64         `json:"bytes_read"`
	BytesWritten     int64         `json:"bytes_written"`
	ReadThroughput   float64       `json:"read_throughput_mbps"`
	WriteThroughput  float64       `json:"write_throughput_mbps"`
	AvgReadLatency   time.Duration `json:"avg_read_latency"`
	AvgWriteLatency  time.Duration `json:"avg_write_latency"`
	SyscallCount     int64         `json:"syscall_count"`
	BufferHitRatio   float64       `json:"buffer_hit_ratio"`
	IOPS             float64       `json:"iops"`
	Duration         time.Duration `json:"duration"`
	OpenFileCount    int           `json:"open_file_count"`
}

// BufferConfig represents buffer optimization configuration
type BufferConfig struct {
	ReadBufferSize  int  `json:"read_buffer_size"`
	WriteBufferSize int  `json:"write_buffer_size"`
	UseBuffering    bool `json:"use_buffering"`
	AlignToPage     bool `json:"align_to_page"`
	PoolBuffers     bool `json:"pool_buffers"`
}

// OptimizedFileReader provides optimized file reading with profiling
type OptimizedFileReader struct {
	profiler     *IOProfiler
	bufferConfig BufferConfig
	bufferPool   sync.Pool
}

// OptimizedFileWriter provides optimized file writing with profiling
type OptimizedFileWriter struct {
	profiler     *IOProfiler
	bufferConfig BufferConfig
	bufferPool   sync.Pool
}

// DiskIOOptimizer provides comprehensive disk I/O optimization
type DiskIOOptimizer struct {
	profiler *IOProfiler
	reader   *OptimizedFileReader
	writer   *OptimizedFileWriter
	config   BufferConfig
}

// NewIOProfiler creates a new I/O profiler
func NewIOProfiler() *IOProfiler {
	return &IOProfiler{
		fileDescriptors: make(map[string]*os.File),
		startTime:       time.Now(),
	}
}

// NewOptimizedFileReader creates a new optimized file reader
func NewOptimizedFileReader(profiler *IOProfiler, config BufferConfig) *OptimizedFileReader {
	reader := &OptimizedFileReader{
		profiler:     profiler,
		bufferConfig: config,
	}
	
	if config.PoolBuffers {
		reader.bufferPool = sync.Pool{
			New: func() interface{} {
				size := config.ReadBufferSize
				if config.AlignToPage {
					pageSize := os.Getpagesize()
					if size%pageSize != 0 {
						size = ((size / pageSize) + 1) * pageSize
					}
				}
				return make([]byte, size)
			},
		}
	}
	
	return reader
}

// NewOptimizedFileWriter creates a new optimized file writer
func NewOptimizedFileWriter(profiler *IOProfiler, config BufferConfig) *OptimizedFileWriter {
	writer := &OptimizedFileWriter{
		profiler:     profiler,
		bufferConfig: config,
	}
	
	if config.PoolBuffers {
		writer.bufferPool = sync.Pool{
			New: func() interface{} {
				size := config.WriteBufferSize
				if config.AlignToPage {
					pageSize := os.Getpagesize()
					if size%pageSize != 0 {
						size = ((size / pageSize) + 1) * pageSize
					}
				}
				return make([]byte, size)
			},
		}
	}
	
	return writer
}

// NewDiskIOOptimizer creates a new disk I/O optimizer
func NewDiskIOOptimizer(config BufferConfig) *DiskIOOptimizer {
	profiler := NewIOProfiler()
	return &DiskIOOptimizer{
		profiler: profiler,
		reader:   NewOptimizedFileReader(profiler, config),
		writer:   NewOptimizedFileWriter(profiler, config),
		config:   config,
	}
}

// ReadFile reads a file with optimization and profiling
func (r *OptimizedFileReader) ReadFile(filename string) ([]byte, error) {
	start := time.Now()
	defer func() {
		r.profiler.recordRead(time.Since(start))
	}()
	
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	r.profiler.trackFileDescriptor(filename, file)
	defer r.profiler.untrackFileDescriptor(filename)
	
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	
	fileSize := stat.Size()
	var data []byte
	
	if r.bufferConfig.UseBuffering {
		data, err = r.readBuffered(file, fileSize)
	} else {
		data, err = r.readDirect(file, fileSize)
	}
	
	if err != nil {
		return nil, err
	}
	
	r.profiler.recordBytesRead(int64(len(data)))
	return data, nil
}

// readBuffered reads file using buffered I/O
func (r *OptimizedFileReader) readBuffered(file *os.File, fileSize int64) ([]byte, error) {
	bufSize := r.bufferConfig.ReadBufferSize
	if bufSize <= 0 {
		bufSize = 64 * 1024 // Default 64KB
	}
	
	reader := bufio.NewReaderSize(file, bufSize)
	data := make([]byte, fileSize)
	
	_, err := io.ReadFull(reader, data)
	if err != nil && err != io.EOF {
		return nil, err
	}
	
	r.profiler.recordBufferHit()
	return data, nil
}

// readDirect reads file using direct I/O
func (r *OptimizedFileReader) readDirect(file *os.File, fileSize int64) ([]byte, error) {
	var buffer []byte
	
	if r.bufferConfig.PoolBuffers {
		buf := r.bufferPool.Get().([]byte)
		defer r.bufferPool.Put(buf)
		
		if int64(len(buf)) >= fileSize {
			buffer = buf[:fileSize]
			r.profiler.recordBufferHit()
		} else {
			buffer = make([]byte, fileSize)
			r.profiler.recordBufferMiss()
		}
	} else {
		buffer = make([]byte, fileSize)
		r.profiler.recordBufferMiss()
	}
	
	_, err := io.ReadFull(file, buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}
	
	return buffer, nil
}

// WriteFile writes data to a file with optimization and profiling
func (w *OptimizedFileWriter) WriteFile(filename string, data []byte) error {
	start := time.Now()
	defer func() {
		w.profiler.recordWrite(time.Since(start))
	}()
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	w.profiler.trackFileDescriptor(filename, file)
	defer w.profiler.untrackFileDescriptor(filename)
	
	if w.bufferConfig.UseBuffering {
		err = w.writeBuffered(file, data)
	} else {
		err = w.writeDirect(file, data)
	}
	
	if err != nil {
		return err
	}
	
	w.profiler.recordBytesWritten(int64(len(data)))
	return nil
}

// writeBuffered writes data using buffered I/O
func (w *OptimizedFileWriter) writeBuffered(file *os.File, data []byte) error {
	bufSize := w.bufferConfig.WriteBufferSize
	if bufSize <= 0 {
		bufSize = 64 * 1024 // Default 64KB
	}
	
	writer := bufio.NewWriterSize(file, bufSize)
	_, err := writer.Write(data)
	if err != nil {
		return err
	}
	
	w.profiler.recordBufferHit()
	return writer.Flush()
}

// writeDirect writes data using direct I/O
func (w *OptimizedFileWriter) writeDirect(file *os.File, data []byte) error {
	_, err := file.Write(data)
	w.profiler.recordBufferMiss()
	return err
}

// ProcessFiles processes multiple files with optimization
func (d *DiskIOOptimizer) ProcessFiles(ctx context.Context, filenames []string, workerCount int) error {
	semaphore := make(chan struct{}, workerCount)
	var wg sync.WaitGroup
	errorChan := make(chan error, len(filenames))
	
	for _, filename := range filenames {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				errorChan <- ctx.Err()
				return
			}
			
			data, err := d.reader.ReadFile(file)
			if err != nil {
				errorChan <- fmt.Errorf("failed to read %s: %w", file, err)
				return
			}
			
			// Process data (example: convert to uppercase)
			processedData := make([]byte, len(data))
			for i, b := range data {
				if b >= 'a' && b <= 'z' {
					processedData[i] = b - 32
				} else {
					processedData[i] = b
				}
			}
			
			outputFile := file + ".processed"
			if err := d.writer.WriteFile(outputFile, processedData); err != nil {
				errorChan <- fmt.Errorf("failed to write %s: %w", outputFile, err)
			}
		}(filename)
	}
	
	wg.Wait()
	close(errorChan)
	
	for err := range errorChan {
		if err != nil {
			return err
		}
	}
	
	return nil
}

// GetStats returns current I/O statistics
func (p *IOProfiler) GetStats() IOStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	duration := time.Since(p.startTime)
	durationSeconds := duration.Seconds()
	
	var readThroughput, writeThroughput float64
	if durationSeconds > 0 {
		readThroughput = float64(p.bytesRead) / (1024 * 1024) / durationSeconds
		writeThroughput = float64(p.bytesWritten) / (1024 * 1024) / durationSeconds
	}
	
	var avgReadLatency, avgWriteLatency time.Duration
	if p.totalReads > 0 {
		avgReadLatency = p.readDuration / time.Duration(p.totalReads)
	}
	if p.totalWrites > 0 {
		avgWriteLatency = p.writeDuration / time.Duration(p.totalWrites)
	}
	
	var bufferHitRatio float64
	totalBufferOps := p.bufferHits + p.bufferMisses
	if totalBufferOps > 0 {
		bufferHitRatio = float64(p.bufferHits) / float64(totalBufferOps)
	}
	
	var iops float64
	if durationSeconds > 0 {
		iops = float64(p.totalReads+p.totalWrites) / durationSeconds
	}
	
	return IOStats{
		TotalReads:       p.totalReads,
		TotalWrites:      p.totalWrites,
		BytesRead:        p.bytesRead,
		BytesWritten:     p.bytesWritten,
		ReadThroughput:   readThroughput,
		WriteThroughput:  writeThroughput,
		AvgReadLatency:   avgReadLatency,
		AvgWriteLatency:  avgWriteLatency,
		SyscallCount:     p.syscallCount,
		BufferHitRatio:   bufferHitRatio,
		IOPS:             iops,
		Duration:         duration,
		OpenFileCount:    len(p.fileDescriptors),
	}
}

// recordRead records a read operation
func (p *IOProfiler) recordRead(duration time.Duration) {
	atomic.AddInt64(&p.totalReads, 1)
	atomic.AddInt64(&p.syscallCount, 1)
	p.mu.Lock()
	p.readDuration += duration
	p.mu.Unlock()
}

// recordWrite records a write operation
func (p *IOProfiler) recordWrite(duration time.Duration) {
	atomic.AddInt64(&p.totalWrites, 1)
	atomic.AddInt64(&p.syscallCount, 1)
	p.mu.Lock()
	p.writeDuration += duration
	p.mu.Unlock()
}

// recordBytesRead records bytes read
func (p *IOProfiler) recordBytesRead(bytes int64) {
	atomic.AddInt64(&p.bytesRead, bytes)
}

// recordBytesWritten records bytes written
func (p *IOProfiler) recordBytesWritten(bytes int64) {
	atomic.AddInt64(&p.bytesWritten, bytes)
}

// recordBufferHit records a buffer cache hit
func (p *IOProfiler) recordBufferHit() {
	atomic.AddInt64(&p.bufferHits, 1)
}

// recordBufferMiss records a buffer cache miss
func (p *IOProfiler) recordBufferMiss() {
	atomic.AddInt64(&p.bufferMisses, 1)
}

// trackFileDescriptor tracks an open file descriptor
func (p *IOProfiler) trackFileDescriptor(filename string, file *os.File) {
	p.mu.Lock()
	p.fileDescriptors[filename] = file
	p.mu.Unlock()
}

// untrackFileDescriptor removes tracking for a file descriptor
func (p *IOProfiler) untrackFileDescriptor(filename string) {
	p.mu.Lock()
	delete(p.fileDescriptors, filename)
	p.mu.Unlock()
}

// OptimizeBufferSize finds optimal buffer size for given workload
func (d *DiskIOOptimizer) OptimizeBufferSize(testFile string, testSizes []int) (int, error) {
	bestSize := 0
	bestThroughput := 0.0
	
	// Create test data
	testData := make([]byte, 1024*1024) // 1MB test data
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	
	for _, size := range testSizes {
		// Reset profiler
		d.profiler = NewIOProfiler()
		d.config.ReadBufferSize = size
		d.config.WriteBufferSize = size
		d.reader = NewOptimizedFileReader(d.profiler, d.config)
		d.writer = NewOptimizedFileWriter(d.profiler, d.config)
		
		// Write test file
		if err := d.writer.WriteFile(testFile, testData); err != nil {
			return 0, err
		}
		
		// Read test file multiple times
		for i := 0; i < 10; i++ {
			_, err := d.reader.ReadFile(testFile)
			if err != nil {
				return 0, err
			}
		}
		
		stats := d.profiler.GetStats()
		if stats.ReadThroughput > bestThroughput {
			bestThroughput = stats.ReadThroughput
			bestSize = size
		}
		
		// Clean up
		os.Remove(testFile)
	}
	
	return bestSize, nil
}

// CreateTestFiles creates test files for benchmarking
func CreateTestFiles(count int, sizeKB int) ([]string, error) {
	var filenames []string
	data := make([]byte, sizeKB*1024)
	
	// Fill with test data
	for i := range data {
		data[i] = byte(i % 256)
	}
	
	for i := 0; i < count; i++ {
		filename := fmt.Sprintf("test_file_%d.dat", i)
		file, err := os.Create(filename)
		if err != nil {
			return nil, err
		}
		
		_, err = file.Write(data)
		file.Close()
		if err != nil {
			return nil, err
		}
		
		filenames = append(filenames, filename)
	}
	
	return filenames, nil
}

// CleanupTestFiles removes test files
func CleanupTestFiles(filenames []string) {
	for _, filename := range filenames {
		os.Remove(filename)
		os.Remove(filename + ".processed")
	}
}

// DefaultBufferConfig returns default buffer configuration
func DefaultBufferConfig() BufferConfig {
	return BufferConfig{
		ReadBufferSize:  64 * 1024, // 64KB
		WriteBufferSize: 64 * 1024, // 64KB
		UseBuffering:    true,
		AlignToPage:     true,
		PoolBuffers:     true,
	}
}

func main() {
	fmt.Println("Disk I/O Profiling and Optimization Demo")
	fmt.Println("========================================")
	
	// Create optimizer with default configuration
	config := DefaultBufferConfig()
	optimizer := NewDiskIOOptimizer(config)
	
	// Create test files
	fmt.Println("\nCreating test files...")
	filenames, err := CreateTestFiles(100, 10) // 100 files, 10KB each
	if err != nil {
		fmt.Printf("Error creating test files: %v\n", err)
		return
	}
	defer CleanupTestFiles(filenames)
	
	// Test buffer size optimization
	fmt.Println("\nOptimizing buffer size...")
	testSizes := []int{4096, 8192, 16384, 32768, 65536, 131072}
	optimalSize, err := optimizer.OptimizeBufferSize("buffer_test.dat", testSizes)
	if err != nil {
		fmt.Printf("Error optimizing buffer size: %v\n", err)
	} else {
		fmt.Printf("Optimal buffer size: %d bytes\n", optimalSize)
	}
	
	// Update configuration with optimal buffer size
	config.ReadBufferSize = optimalSize
	config.WriteBufferSize = optimalSize
	optimizer = NewDiskIOOptimizer(config)
	
	// Process files with different worker counts
	workerCounts := []int{1, 2, 4, 8}
	
	for _, workers := range workerCounts {
		fmt.Printf("\nProcessing files with %d workers...\n", workers)
		
		// Reset profiler for each test
		optimizer.profiler = NewIOProfiler()
		optimizer.reader = NewOptimizedFileReader(optimizer.profiler, config)
		optimizer.writer = NewOptimizedFileWriter(optimizer.profiler, config)
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		
		start := time.Now()
		err := optimizer.ProcessFiles(ctx, filenames, workers)
		processingTime := time.Since(start)
		cancel()
		
		if err != nil {
			fmt.Printf("Error processing files: %v\n", err)
			continue
		}
		
		stats := optimizer.profiler.GetStats()
		
		fmt.Printf("Processing completed in: %v\n", processingTime)
		fmt.Printf("Total reads: %d\n", stats.TotalReads)
		fmt.Printf("Total writes: %d\n", stats.TotalWrites)
		fmt.Printf("Bytes read: %d (%.2f MB)\n", stats.BytesRead, float64(stats.BytesRead)/(1024*1024))
		fmt.Printf("Bytes written: %d (%.2f MB)\n", stats.BytesWritten, float64(stats.BytesWritten)/(1024*1024))
		fmt.Printf("Read throughput: %.2f MB/s\n", stats.ReadThroughput)
		fmt.Printf("Write throughput: %.2f MB/s\n", stats.WriteThroughput)
		fmt.Printf("Average read latency: %v\n", stats.AvgReadLatency)
		fmt.Printf("Average write latency: %v\n", stats.AvgWriteLatency)
		fmt.Printf("IOPS: %.2f\n", stats.IOPS)
		fmt.Printf("Buffer hit ratio: %.2f%%\n", stats.BufferHitRatio*100)
		fmt.Printf("Syscall count: %d\n", stats.SyscallCount)
		fmt.Printf("Open file descriptors: %d\n", stats.OpenFileCount)
		
		// Memory statistics
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		fmt.Printf("Memory allocated: %.2f MB\n", float64(m.Alloc)/(1024*1024))
		fmt.Printf("Total allocations: %d\n", m.TotalAlloc)
		fmt.Printf("GC cycles: %d\n", m.NumGC)
	}
	
	fmt.Println("\nDisk I/O profiling and optimization completed successfully!")
}
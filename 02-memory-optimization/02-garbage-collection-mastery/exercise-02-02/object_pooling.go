package main

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// PoolManager manages multiple object pools for different types
type PoolManager struct {
	pools map[string]Pool
	stats map[string]*PoolStats
	mu    sync.RWMutex
}

// Pool interface for different pool implementations
type Pool interface {
	Get() interface{}
	Put(interface{})
	Stats() PoolStats
	Reset()
	Size() int
}

// PoolStats tracks pool performance metrics
type PoolStats struct {
	Gets           uint64    // Number of Get() calls
	Puts           uint64    // Number of Put() calls
	Hits           uint64    // Number of successful gets from pool
	Misses         uint64    // Number of gets that required new allocation
	Allocations    uint64    // Total new allocations
	Deallocations  uint64    // Total objects discarded
	CurrentSize    int       // Current pool size
	MaxSize        int       // Maximum pool size reached
	HitRate        float64   // Cache hit rate percentage
	LastReset      time.Time // Last time stats were reset
}

// NewPoolManager creates a new pool manager
func NewPoolManager() *PoolManager {
	return &PoolManager{
		pools: make(map[string]Pool),
		stats: make(map[string]*PoolStats),
	}
}

// RegisterPool registers a pool with the manager
func (pm *PoolManager) RegisterPool(name string, pool Pool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	pm.pools[name] = pool
	pm.stats[name] = &PoolStats{LastReset: time.Now()}
}

// GetPool retrieves a pool by name
func (pm *PoolManager) GetPool(name string) (Pool, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	pool, exists := pm.pools[name]
	return pool, exists
}

// GetAllStats returns statistics for all pools
func (pm *PoolManager) GetAllStats() map[string]PoolStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	stats := make(map[string]PoolStats)
	for name, pool := range pm.pools {
		stats[name] = pool.Stats()
	}
	return stats
}

// BufferPool implements a pool for byte slices with size classes
type BufferPool struct {
	pools    map[int]*sync.Pool // Size class -> pool
	sizes    []int              // Available size classes
	stats    PoolStats
	maxSize  int                // Maximum buffer size to pool
	mu       sync.RWMutex
}

// NewBufferPool creates a new buffer pool with specified size classes
func NewBufferPool(sizes []int, maxSize int) *BufferPool {
	bp := &BufferPool{
		pools:   make(map[int]*sync.Pool),
		sizes:   make([]int, len(sizes)),
		maxSize: maxSize,
		stats:   PoolStats{LastReset: time.Now()},
	}
	
	copy(bp.sizes, sizes)
	
	// Create pools for each size class
	for _, size := range sizes {
		capturedSize := size // Capture for closure
		bp.pools[size] = &sync.Pool{
			New: func() interface{} {
				bp.mu.Lock()
				bp.stats.Allocations++
				bp.mu.Unlock()
				return make([]byte, capturedSize)
			},
		}
	}
	
	return bp
}

// Get retrieves a buffer of at least the specified size
func (bp *BufferPool) Get() interface{} {
	return bp.GetBuffer(1024) // Default size
}

// GetBuffer retrieves a buffer of at least the specified size
func (bp *BufferPool) GetBuffer(size int) []byte {
	bp.mu.Lock()
	bp.stats.Gets++
	bp.mu.Unlock()
	
	// Find the smallest size class that fits
	for _, poolSize := range bp.sizes {
		if poolSize >= size {
			if pool, exists := bp.pools[poolSize]; exists {
				buffer := pool.Get().([]byte)
				bp.mu.Lock()
				bp.stats.Hits++
				bp.mu.Unlock()
				return buffer[:size] // Return slice of requested size
			}
		}
	}
	
	// No suitable pool found, allocate directly
	bp.mu.Lock()
	bp.stats.Misses++
	bp.stats.Allocations++
	bp.mu.Unlock()
	
	return make([]byte, size)
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(obj interface{}) {
	bp.PutBuffer(obj.([]byte))
}

// PutBuffer returns a buffer to the pool
func (bp *BufferPool) PutBuffer(buffer []byte) {
	bp.mu.Lock()
	bp.stats.Puts++
	bp.mu.Unlock()
	
	size := cap(buffer)
	
	// Don't pool buffers that are too large
	if size > bp.maxSize {
		bp.mu.Lock()
		bp.stats.Deallocations++
		bp.mu.Unlock()
		return
	}
	
	// Find appropriate pool
	for _, poolSize := range bp.sizes {
		if poolSize == size {
			if pool, exists := bp.pools[poolSize]; exists {
				// Reset buffer content
				for i := range buffer {
					buffer[i] = 0
				}
				pool.Put(buffer[:poolSize])
				return
			}
		}
	}
	
	// No matching pool, discard
	bp.mu.Lock()
	bp.stats.Deallocations++
	bp.mu.Unlock()
}

// Stats returns pool statistics
func (bp *BufferPool) Stats() PoolStats {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	
	stats := bp.stats
	if stats.Gets > 0 {
		stats.HitRate = float64(stats.Hits) / float64(stats.Gets) * 100
	}
	
	// Calculate current size across all pools
	currentSize := 0
	for _ = range bp.pools {
		// Note: sync.Pool doesn't expose size, this is an approximation
		currentSize += 10 // Estimate
	}
	stats.CurrentSize = currentSize
	
	return stats
}

// Reset resets pool statistics
func (bp *BufferPool) Reset() {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	
	bp.stats = PoolStats{LastReset: time.Now()}
}

// Size returns approximate pool size
func (bp *BufferPool) Size() int {
	return bp.Stats().CurrentSize
}

// StringBuilderPool implements a pool for strings.Builder objects
type StringBuilderPool struct {
	pool     sync.Pool
	stats    PoolStats
	maxSize  int // Maximum builder capacity to pool
	mu       sync.RWMutex
}

// NewStringBuilderPool creates a new string builder pool
func NewStringBuilderPool(maxSize int) *StringBuilderPool {
	sbp := &StringBuilderPool{
		maxSize: maxSize,
		stats:   PoolStats{LastReset: time.Now()},
	}
	
	sbp.pool = sync.Pool{
		New: func() interface{} {
			sbp.mu.Lock()
			sbp.stats.Allocations++
			sbp.mu.Unlock()
			return &strings.Builder{}
		},
	}
	
	return sbp
}

// Get retrieves a string builder from the pool
func (sbp *StringBuilderPool) Get() interface{} {
	sbp.mu.Lock()
	sbp.stats.Gets++
	sbp.stats.Hits++
	sbp.mu.Unlock()
	
	builder := sbp.pool.Get().(*strings.Builder)
	builder.Reset() // Ensure clean state
	return builder
}

// GetBuilder retrieves a string builder from the pool
func (sbp *StringBuilderPool) GetBuilder() *strings.Builder {
	return sbp.Get().(*strings.Builder)
}

// Put returns a string builder to the pool
func (sbp *StringBuilderPool) Put(obj interface{}) {
	sbp.PutBuilder(obj.(*strings.Builder))
}

// PutBuilder returns a string builder to the pool
func (sbp *StringBuilderPool) PutBuilder(builder *strings.Builder) {
	sbp.mu.Lock()
	sbp.stats.Puts++
	sbp.mu.Unlock()
	
	// Don't pool builders that are too large
	if builder.Cap() > sbp.maxSize {
		sbp.mu.Lock()
		sbp.stats.Deallocations++
		sbp.mu.Unlock()
		return
	}
	
	builder.Reset()
	sbp.pool.Put(builder)
}

// Stats returns pool statistics
func (sbp *StringBuilderPool) Stats() PoolStats {
	sbp.mu.RLock()
	defer sbp.mu.RUnlock()
	
	stats := sbp.stats
	if stats.Gets > 0 {
		stats.HitRate = float64(stats.Hits) / float64(stats.Gets) * 100
	}
	return stats
}

// Reset resets pool statistics
func (sbp *StringBuilderPool) Reset() {
	sbp.mu.Lock()
	defer sbp.mu.Unlock()
	
	sbp.stats = PoolStats{LastReset: time.Now()}
}

// Size returns approximate pool size
func (sbp *StringBuilderPool) Size() int {
	return 0 // sync.Pool doesn't expose size
}

// ObjectPool implements a generic object pool
type ObjectPool struct {
	pool    sync.Pool
	stats   PoolStats
	validate func(interface{}) bool // Optional validation function
	reset   func(interface{})       // Optional reset function
	mu      sync.RWMutex
}

// NewObjectPool creates a new generic object pool
func NewObjectPool(factory func() interface{}) *ObjectPool {
	op := &ObjectPool{
		stats: PoolStats{LastReset: time.Now()},
	}
	
	op.pool = sync.Pool{
		New: func() interface{} {
			op.mu.Lock()
			op.stats.Allocations++
			op.mu.Unlock()
			return factory()
		},
	}
	
	return op
}

// SetValidator sets a validation function for objects
func (op *ObjectPool) SetValidator(validate func(interface{}) bool) {
	op.validate = validate
}

// SetReset sets a reset function for objects
func (op *ObjectPool) SetReset(reset func(interface{})) {
	op.reset = reset
}

// Get retrieves an object from the pool
func (op *ObjectPool) Get() interface{} {
	op.mu.Lock()
	op.stats.Gets++
	op.stats.Hits++
	op.mu.Unlock()
	
	obj := op.pool.Get()
	
	// Apply reset function if provided
	if op.reset != nil {
		op.reset(obj)
	}
	
	return obj
}

// Put returns an object to the pool
func (op *ObjectPool) Put(obj interface{}) {
	op.mu.Lock()
	op.stats.Puts++
	op.mu.Unlock()
	
	// Validate object if validator is provided
	if op.validate != nil && !op.validate(obj) {
		op.mu.Lock()
		op.stats.Deallocations++
		op.mu.Unlock()
		return
	}
	
	op.pool.Put(obj)
}

// Stats returns pool statistics
func (op *ObjectPool) Stats() PoolStats {
	op.mu.RLock()
	defer op.mu.RUnlock()
	
	stats := op.stats
	if stats.Gets > 0 {
		stats.HitRate = float64(stats.Hits) / float64(stats.Gets) * 100
	}
	return stats
}

// Reset resets pool statistics
func (op *ObjectPool) Reset() {
	op.mu.Lock()
	defer op.mu.Unlock()
	
	op.stats = PoolStats{LastReset: time.Now()}
}

// Size returns approximate pool size
func (op *ObjectPool) Size() int {
	return 0 // sync.Pool doesn't expose size
}

// WorkerPool implements a pool of reusable worker goroutines
type WorkerPool struct {
	workers   chan chan func()
	workQueue chan func()
	quit      chan bool
	stats     PoolStats
	size      int
	mu        sync.RWMutex
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(size int, queueSize int) *WorkerPool {
	wp := &WorkerPool{
		workers:   make(chan chan func(), size),
		workQueue: make(chan func(), queueSize),
		quit:      make(chan bool),
		size:      size,
		stats:     PoolStats{LastReset: time.Now()},
	}
	
	// Start workers
	for i := 0; i < size; i++ {
		worker := NewWorker(wp.workers, wp.quit)
		worker.Start()
	}
	
	// Start dispatcher
	go wp.dispatch()
	
	return wp
}

// Worker represents a worker goroutine
type Worker struct {
	workerPool chan chan func()
	jobChannel chan func()
	quit       chan bool
}

// NewWorker creates a new worker
func NewWorker(workerPool chan chan func(), quit chan bool) *Worker {
	return &Worker{
		workerPool: workerPool,
		jobChannel: make(chan func()),
		quit:       quit,
	}
}

// Start starts the worker
func (w *Worker) Start() {
	go func() {
		for {
			// Register worker in pool
			w.workerPool <- w.jobChannel
			
			select {
			case job := <-w.jobChannel:
				// Execute job
				job()
			case <-w.quit:
				return
			}
		}
	}()
}

// dispatch dispatches jobs to available workers
func (wp *WorkerPool) dispatch() {
	for {
		select {
		case job := <-wp.workQueue:
			// Get available worker
			worker := <-wp.workers
			// Send job to worker
			worker <- job
			
			wp.mu.Lock()
			wp.stats.Gets++
			wp.stats.Hits++
			wp.mu.Unlock()
			
		case <-wp.quit:
			return
		}
	}
}

// Submit submits a job to the worker pool
func (wp *WorkerPool) Submit(job func()) {
	wp.workQueue <- job
}

// Get retrieves a worker (not applicable for worker pool)
func (wp *WorkerPool) Get() interface{} {
	return nil
}

// Put returns a worker (not applicable for worker pool)
func (wp *WorkerPool) Put(obj interface{}) {
	// Not applicable for worker pool
}

// Stats returns pool statistics
func (wp *WorkerPool) Stats() PoolStats {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	
	stats := wp.stats
	stats.CurrentSize = wp.size
	if stats.Gets > 0 {
		stats.HitRate = float64(stats.Hits) / float64(stats.Gets) * 100
	}
	return stats
}

// Reset resets pool statistics
func (wp *WorkerPool) Reset() {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	wp.stats = PoolStats{LastReset: time.Now()}
}

// Size returns worker pool size
func (wp *WorkerPool) Size() int {
	return wp.size
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop() {
	close(wp.quit)
}

// PoolBenchmark provides benchmarking capabilities for pools
type PoolBenchmark struct {
	manager *PoolManager
}

// NewPoolBenchmark creates a new pool benchmark
func NewPoolBenchmark() *PoolBenchmark {
	return &PoolBenchmark{
		manager: NewPoolManager(),
	}
}

// BenchmarkPool benchmarks a specific pool against direct allocation
func (pb *PoolBenchmark) BenchmarkPool(poolName string, iterations int, workload func(Pool)) BenchmarkResult {
	pool, exists := pb.manager.GetPool(poolName)
	if !exists {
		return BenchmarkResult{Error: fmt.Errorf("pool %s not found", poolName)}
	}
	
	// Reset pool stats
	pool.Reset()
	
	// Benchmark with pool
	var poolStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&poolStats)
	
	poolStart := time.Now()
	workload(pool)
	poolDuration := time.Since(poolStart)
	
	var poolStatsAfter runtime.MemStats
	runtime.ReadMemStats(&poolStatsAfter)
	
	// Benchmark without pool (direct allocation)
	runtime.GC()
	var directStats runtime.MemStats
	runtime.ReadMemStats(&directStats)
	
	directStart := time.Now()
	workload(nil) // Pass nil to indicate direct allocation
	directDuration := time.Since(directStart)
	
	var directStatsAfter runtime.MemStats
	runtime.ReadMemStats(&directStatsAfter)
	
	return BenchmarkResult{
		PoolName:        poolName,
		Iterations:      iterations,
		PoolDuration:    poolDuration,
		DirectDuration:  directDuration,
		Speedup:         float64(directDuration) / float64(poolDuration),
		PoolAllocations: poolStatsAfter.TotalAlloc - poolStats.TotalAlloc,
		DirectAllocations: directStatsAfter.TotalAlloc - directStats.TotalAlloc,
		AllocationReduction: float64(directStatsAfter.TotalAlloc-directStats.TotalAlloc-
			(poolStatsAfter.TotalAlloc-poolStats.TotalAlloc)) / 
			float64(directStatsAfter.TotalAlloc-directStats.TotalAlloc) * 100,
		PoolStats: pool.Stats(),
	}
}

// BenchmarkResult contains the results of a pool benchmark
type BenchmarkResult struct {
	PoolName            string
	Iterations          int
	PoolDuration        time.Duration
	DirectDuration      time.Duration
	Speedup             float64
	PoolAllocations     uint64
	DirectAllocations   uint64
	AllocationReduction float64
	PoolStats           PoolStats
	Error               error
}

// GeneratePoolReport generates a comprehensive pool performance report
func (pb *PoolBenchmark) GeneratePoolReport() string {
	var report strings.Builder
	
	report.WriteString("=== Object Pool Performance Report ===\n\n")
	
	allStats := pb.manager.GetAllStats()
	
	for poolName, stats := range allStats {
		report.WriteString(fmt.Sprintf("Pool: %s\n", poolName))
		report.WriteString(fmt.Sprintf("  Gets: %d\n", stats.Gets))
		report.WriteString(fmt.Sprintf("  Puts: %d\n", stats.Puts))
		report.WriteString(fmt.Sprintf("  Hits: %d\n", stats.Hits))
		report.WriteString(fmt.Sprintf("  Misses: %d\n", stats.Misses))
		report.WriteString(fmt.Sprintf("  Hit Rate: %.2f%%\n", stats.HitRate))
		report.WriteString(fmt.Sprintf("  Allocations: %d\n", stats.Allocations))
		report.WriteString(fmt.Sprintf("  Deallocations: %d\n", stats.Deallocations))
		report.WriteString(fmt.Sprintf("  Current Size: %d\n", stats.CurrentSize))
		report.WriteString("\n")
	}
	
	return report.String()
}

// DemonstrateObjectPooling demonstrates object pooling capabilities
func DemonstrateObjectPooling() {
	fmt.Println("Object Pooling Demonstration")
	fmt.Println("============================")
	
	// Create pool manager
	manager := NewPoolManager()
	
	// Create and register different pools
	bufferPool := NewBufferPool([]int{64, 256, 1024, 4096}, 8192)
	manager.RegisterPool("buffer", bufferPool)
	
	stringBuilderPool := NewStringBuilderPool(4096)
	manager.RegisterPool("stringbuilder", stringBuilderPool)
	
	// Create object pool for custom structs
	type CustomObject struct {
		Data []int
		Name string
	}
	
	customPool := NewObjectPool(func() interface{} {
		return &CustomObject{
			Data: make([]int, 100),
			Name: "",
		}
	})
	
	customPool.SetReset(func(obj interface{}) {
		custom := obj.(*CustomObject)
		for i := range custom.Data {
			custom.Data[i] = 0
		}
		custom.Name = ""
	})
	
	manager.RegisterPool("custom", customPool)
	
	// Create worker pool
	workerPool := NewWorkerPool(4, 100)
	manager.RegisterPool("worker", workerPool)
	defer workerPool.Stop()
	
	// Demonstrate buffer pool
	fmt.Println("Testing Buffer Pool...")
	for i := 0; i < 1000; i++ {
		buffer := bufferPool.GetBuffer(512)
		// Use buffer
		for j := range buffer {
			buffer[j] = byte(j % 256)
		}
		bufferPool.PutBuffer(buffer)
	}
	
	// Demonstrate string builder pool
	fmt.Println("Testing String Builder Pool...")
	for i := 0; i < 1000; i++ {
		builder := stringBuilderPool.GetBuilder()
		for j := 0; j < 10; j++ {
			builder.WriteString(fmt.Sprintf("item_%d_", j))
		}
		result := builder.String()
		_ = result
		stringBuilderPool.PutBuilder(builder)
	}
	
	// Demonstrate custom object pool
	fmt.Println("Testing Custom Object Pool...")
	for i := 0; i < 1000; i++ {
		obj := customPool.Get().(*CustomObject)
		obj.Name = fmt.Sprintf("object_%d", i)
		for j := range obj.Data {
			obj.Data[j] = j
		}
		customPool.Put(obj)
	}
	
	// Demonstrate worker pool
	fmt.Println("Testing Worker Pool...")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		workerPool.Submit(func() {
			defer wg.Done()
			// Simulate work
			time.Sleep(1 * time.Millisecond)
		})
	}
	wg.Wait()
	
	// Generate and display report
	benchmark := &PoolBenchmark{manager: manager}
	report := benchmark.GeneratePoolReport()
	fmt.Println(report)
	
	fmt.Println("Object pooling demonstration completed!")
}
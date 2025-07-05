package main

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ObjectPool interface for type-specific object pools
type ObjectPool interface {
	Get() interface{}
	Put(interface{})
	Stats() PoolStats
	Reset()
}

// PoolStats contains pool statistics
type PoolStats struct {
	Gets        int64
	Puts        int64
	News        int64 // Number of new objects created
	Reuses      int64 // Number of objects reused
	CurrentSize int64 // Current pool size
	MaxSize     int64 // Maximum pool size reached
	HitRate     float64
}

// BufferPool manages []byte buffers with size classes
type BufferPool struct {
	pools map[int]*sync.Pool
	mu    sync.RWMutex
	stats PoolStats
}

// NewBufferPool creates a new buffer pool
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pools: make(map[int]*sync.Pool),
	}
}

// getSizeClass returns the appropriate size class for a buffer
func (bp *BufferPool) getSizeClass(size int) int {
	// Round up to next power of 2
	if size <= 0 {
		return 64 // Minimum size
	}
	
	class := 64
	for class < size {
		class *= 2
	}
	return class
}

// getPool returns a pool for the given size class
func (bp *BufferPool) getPool(sizeClass int) *sync.Pool {
	bp.mu.RLock()
	pool, exists := bp.pools[sizeClass]
	bp.mu.RUnlock()
	
	if !exists {
		bp.mu.Lock()
		if pool, exists = bp.pools[sizeClass]; !exists {
			pool = &sync.Pool{
				New: func() interface{} {
					atomic.AddInt64(&bp.stats.News, 1)
					return make([]byte, 0, sizeClass)
				},
			}
			bp.pools[sizeClass] = pool
		}
		bp.mu.Unlock()
	}
	
	return pool
}

// Get returns a buffer with default size
func (bp *BufferPool) Get() interface{} {
	size := 1024 // Default size
	sizeClass := bp.getSizeClass(size)
	pool := bp.getPool(sizeClass)
	
	buf := pool.Get().([]byte)
	buf = buf[:0] // Reset length but keep capacity
	
	atomic.AddInt64(&bp.stats.Gets, 1)
	if cap(buf) >= size {
		atomic.AddInt64(&bp.stats.Reuses, 1)
	}
	
	return buf
}

// GetBuffer returns a buffer of at least the specified size
func (bp *BufferPool) GetBuffer(size int) []byte {
	sizeClass := bp.getSizeClass(size)
	pool := bp.getPool(sizeClass)
	
	buf := pool.Get().([]byte)
	buf = buf[:0] // Reset length but keep capacity
	
	atomic.AddInt64(&bp.stats.Gets, 1)
	if cap(buf) >= size {
		atomic.AddInt64(&bp.stats.Reuses, 1)
	}
	
	return buf
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(obj interface{}) {
	buf, ok := obj.([]byte)
	if !ok || buf == nil || cap(buf) == 0 {
		return
	}
	
	sizeClass := bp.getSizeClass(cap(buf))
	pool := bp.getPool(sizeClass)
	
	// Only pool if capacity matches size class (avoid fragmentation)
	if cap(buf) == sizeClass {
		pool.Put(buf)
		atomic.AddInt64(&bp.stats.Puts, 1)
	}
}

// Stats returns pool statistics
func (bp *BufferPool) Stats() PoolStats {
	stats := bp.stats
	if stats.Gets > 0 {
		stats.HitRate = float64(stats.Reuses) / float64(stats.Gets) * 100
	}
	return stats
}

// Reset resets the pool
func (bp *BufferPool) Reset() {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	
	for _, pool := range bp.pools {
		*pool = sync.Pool{}
	}
	bp.stats = PoolStats{}
}

// StringBuilderPool manages strings.Builder objects
type StringBuilderPool struct {
	pool  sync.Pool
	stats PoolStats
}

// NewStringBuilderPool creates a new string builder pool
func NewStringBuilderPool(initialCapacity int) *StringBuilderPool {
	sbp := &StringBuilderPool{}
	sbp.pool = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&sbp.stats.News, 1)
			var sb strings.Builder
			sb.Grow(initialCapacity)
			return &sb
		},
	}
	return sbp
}

// Get returns a string builder
func (sbp *StringBuilderPool) Get() interface{} {
	sb := sbp.pool.Get().(*strings.Builder)
	sb.Reset()
	
	atomic.AddInt64(&sbp.stats.Gets, 1)
	atomic.AddInt64(&sbp.stats.Reuses, 1)
	
	return sb
}

// Put returns a string builder to the pool
func (sbp *StringBuilderPool) Put(obj interface{}) {
	sb, ok := obj.(*strings.Builder)
	if !ok || sb == nil {
		return
	}
	
	// Don't pool if it's grown too large
	if sb.Cap() > 64*1024 {
		return
	}
	
	sbp.pool.Put(sb)
	atomic.AddInt64(&sbp.stats.Puts, 1)
}

// Stats returns pool statistics
func (sbp *StringBuilderPool) Stats() PoolStats {
	stats := sbp.stats
	if stats.Gets > 0 {
		stats.HitRate = float64(stats.Reuses) / float64(stats.Gets) * 100
	}
	return stats
}

// Reset resets the pool
func (sbp *StringBuilderPool) Reset() {
	sbp.pool = sync.Pool{}
	sbp.stats = PoolStats{}
}

// WorkerPool manages worker goroutines and their associated resources
type WorkerPool struct {
	workers   chan *Worker
	maxWorkers int
	stats     PoolStats
	mu        sync.Mutex
}

// Worker represents a reusable worker
type Worker struct {
	id     int
	buffer []byte
	tasks  chan func()
	done   chan bool
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(maxWorkers int, bufferSize int) *WorkerPool {
	wp := &WorkerPool{
		workers:    make(chan *Worker, maxWorkers),
		maxWorkers: maxWorkers,
	}
	
	// Pre-create workers
	for i := 0; i < maxWorkers; i++ {
		worker := &Worker{
			id:     i,
			buffer: make([]byte, bufferSize),
			tasks:  make(chan func(), 1),
			done:   make(chan bool, 1),
		}
		wp.workers <- worker
	}
	
	return wp
}

// Get returns a worker from the pool
func (wp *WorkerPool) Get() interface{} {
	select {
	case worker := <-wp.workers:
		atomic.AddInt64(&wp.stats.Gets, 1)
		atomic.AddInt64(&wp.stats.Reuses, 1)
		return worker
	default:
		// No workers available, create a temporary one
		atomic.AddInt64(&wp.stats.Gets, 1)
		atomic.AddInt64(&wp.stats.News, 1)
		return &Worker{
			id:     -1, // Temporary worker
			buffer: make([]byte, 1024),
			tasks:  make(chan func(), 1),
			done:   make(chan bool, 1),
		}
	}
}

// Put returns a worker to the pool
func (wp *WorkerPool) Put(obj interface{}) {
	worker, ok := obj.(*Worker)
	if !ok || worker == nil || worker.id == -1 {
		return // Don't pool temporary workers
	}
	
	// Reset worker state
	worker.buffer = worker.buffer[:0]
	
	select {
	case wp.workers <- worker:
		atomic.AddInt64(&wp.stats.Puts, 1)
	default:
		// Pool is full, discard worker
	}
}

// Stats returns pool statistics
func (wp *WorkerPool) Stats() PoolStats {
	stats := wp.stats
	stats.CurrentSize = int64(len(wp.workers))
	stats.MaxSize = int64(wp.maxWorkers)
	if stats.Gets > 0 {
		stats.HitRate = float64(stats.Reuses) / float64(stats.Gets) * 100
	}
	return stats
}

// Reset resets the pool
func (wp *WorkerPool) Reset() {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	// Drain existing workers
	for len(wp.workers) > 0 {
		<-wp.workers
	}
	
	// Recreate workers
	for i := 0; i < wp.maxWorkers; i++ {
		worker := &Worker{
			id:     i,
			buffer: make([]byte, 1024),
			tasks:  make(chan func(), 1),
			done:   make(chan bool, 1),
		}
		wp.workers <- worker
	}
	
	wp.stats = PoolStats{}
}

// ConnectionPool manages database/network connections
type ConnectionPool struct {
	connections chan *Connection
	maxConns    int
	createConn  func() *Connection
	stats       PoolStats
	mu          sync.Mutex
}

// Connection represents a reusable connection
type Connection struct {
	id       int
	active   bool
	lastUsed time.Time
	data     map[string]interface{}
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxConns int, createFunc func() *Connection) *ConnectionPool {
	cp := &ConnectionPool{
		connections: make(chan *Connection, maxConns),
		maxConns:    maxConns,
		createConn:  createFunc,
	}
	
	// Pre-create some connections
	for i := 0; i < maxConns/2; i++ {
		conn := createFunc()
		conn.id = i
		cp.connections <- conn
	}
	
	return cp
}

// Get returns a connection from the pool
func (cp *ConnectionPool) Get() interface{} {
	select {
	case conn := <-cp.connections:
		conn.active = true
		conn.lastUsed = time.Now()
		atomic.AddInt64(&cp.stats.Gets, 1)
		atomic.AddInt64(&cp.stats.Reuses, 1)
		return conn
	default:
		// No connections available, create a new one
		conn := cp.createConn()
		conn.id = -1 // Temporary connection
		conn.active = true
		conn.lastUsed = time.Now()
		atomic.AddInt64(&cp.stats.Gets, 1)
		atomic.AddInt64(&cp.stats.News, 1)
		return conn
	}
}

// Put returns a connection to the pool
func (cp *ConnectionPool) Put(obj interface{}) {
	conn, ok := obj.(*Connection)
	if !ok || conn == nil {
		return
	}
	
	conn.active = false
	
	// Don't pool temporary connections
	if conn.id == -1 {
		return
	}
	
	// Reset connection state
	conn.data = make(map[string]interface{})
	
	select {
	case cp.connections <- conn:
		atomic.AddInt64(&cp.stats.Puts, 1)
	default:
		// Pool is full, discard connection
	}
}

// Stats returns pool statistics
func (cp *ConnectionPool) Stats() PoolStats {
	stats := cp.stats
	stats.CurrentSize = int64(len(cp.connections))
	stats.MaxSize = int64(cp.maxConns)
	if stats.Gets > 0 {
		stats.HitRate = float64(stats.Reuses) / float64(stats.Gets) * 100
	}
	return stats
}

// Reset resets the pool
func (cp *ConnectionPool) Reset() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	
	// Drain existing connections
	for len(cp.connections) > 0 {
		<-cp.connections
	}
	
	// Recreate connections
	for i := 0; i < cp.maxConns/2; i++ {
		conn := cp.createConn()
		conn.id = i
		cp.connections <- conn
	}
	
	cp.stats = PoolStats{}
}

// PoolManager manages multiple object pools
type PoolManager struct {
	pools map[string]ObjectPool
	mu    sync.RWMutex
}

// NewPoolManager creates a new pool manager
func NewPoolManager() *PoolManager {
	return &PoolManager{
		pools: make(map[string]ObjectPool),
	}
}

// RegisterPool registers a pool with the manager
func (pm *PoolManager) RegisterPool(name string, pool ObjectPool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.pools[name] = pool
}

// GetPool returns a pool by name
func (pm *PoolManager) GetPool(name string) ObjectPool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.pools[name]
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

// PrintAllStats prints statistics for all pools
func (pm *PoolManager) PrintAllStats() {
	fmt.Println("=== Pool Manager Statistics ===")
	fmt.Printf("%-20s %-10s %-10s %-10s %-10s %-10s\n", 
		"Pool", "Gets", "Puts", "News", "Reuses", "Hit Rate")
	fmt.Println("------------------------------------------------------------------------")
	
	stats := pm.GetAllStats()
	for name, stat := range stats {
		fmt.Printf("%-20s %-10d %-10d %-10d %-10d %-10.1f%%\n",
			name, stat.Gets, stat.Puts, stat.News, stat.Reuses, stat.HitRate)
	}
}

// ResetAllPools resets all pools
func (pm *PoolManager) ResetAllPools() {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	for _, pool := range pm.pools {
		pool.Reset()
	}
}

// BenchmarkPools benchmarks different pool implementations
func BenchmarkPools() {
	fmt.Println("=== Object Pool Benchmark ===")
	
	// Create pool manager
	manager := NewPoolManager()
	
	// Register pools
	manager.RegisterPool("BufferPool", NewBufferPool())
	manager.RegisterPool("StringBuilderPool", NewStringBuilderPool(1024))
	manager.RegisterPool("WorkerPool", NewWorkerPool(10, 1024))
	manager.RegisterPool("ConnectionPool", NewConnectionPool(5, func() *Connection {
		return &Connection{
			data: make(map[string]interface{}),
		}
	}))
	
	// Benchmark buffer pool
	fmt.Println("\n--- Buffer Pool Benchmark ---")
	bufferPool := manager.GetPool("BufferPool").(*BufferPool)
	startTime := time.Now()
	
	for i := 0; i < 10000; i++ {
		size := 64 + (i%10)*64 // Varying sizes
		buf := bufferPool.GetBuffer(size)
		
		// Simulate usage
		for j := 0; j < size/8; j++ {
			buf = append(buf, byte(j))
		}
		
		bufferPool.Put(buf)
	}
	
	bufferDuration := time.Since(startTime)
	fmt.Printf("Buffer pool: %d operations in %v (%.0f ops/sec)\n", 
		10000, bufferDuration, 10000.0/bufferDuration.Seconds())
	
	// Benchmark string builder pool
	fmt.Println("\n--- String Builder Pool Benchmark ---")
	sbPool := manager.GetPool("StringBuilderPool").(*StringBuilderPool)
	startTime = time.Now()
	
	for i := 0; i < 10000; i++ {
		sb := sbPool.Get().(*strings.Builder)
		
		// Simulate usage
		for j := 0; j < 10; j++ {
			sb.WriteString("hello world ")
		}
		
		result := sb.String()
		_ = result
		
		sbPool.Put(sb)
	}
	
	sbDuration := time.Since(startTime)
	fmt.Printf("String builder pool: %d operations in %v (%.0f ops/sec)\n", 
		10000, sbDuration, 10000.0/sbDuration.Seconds())
	
	// Benchmark worker pool
	fmt.Println("\n--- Worker Pool Benchmark ---")
	workerPool := manager.GetPool("WorkerPool").(*WorkerPool)
	startTime = time.Now()
	
	for i := 0; i < 1000; i++ {
		worker := workerPool.Get().(*Worker)
		
		// Simulate work
		worker.buffer = append(worker.buffer[:0], []byte("some work data")...)
		
		workerPool.Put(worker)
	}
	
	workerDuration := time.Since(startTime)
	fmt.Printf("Worker pool: %d operations in %v (%.0f ops/sec)\n", 
		1000, workerDuration, 1000.0/workerDuration.Seconds())
	
	// Print all statistics
	fmt.Println()
	manager.PrintAllStats()
}

// CompareWithStandardAllocation compares pools with standard allocation
func CompareWithStandardAllocation() {
	fmt.Println("\n=== Pool vs Standard Allocation Comparison ===")
	
	// Buffer allocation comparison
	fmt.Println("\n--- Buffer Allocation ---")
	
	// Standard allocation
	startTime := time.Now()
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	for i := 0; i < 10000; i++ {
		buf := make([]byte, 0, 1024)
		for j := 0; j < 100; j++ {
			buf = append(buf, byte(j))
		}
		_ = buf
	}
	
	runtime.ReadMemStats(&m2)
	standardDuration := time.Since(startTime)
	standardAllocs := m2.Mallocs - m1.Mallocs
	standardBytes := m2.TotalAlloc - m1.TotalAlloc
	
	// Pool allocation
	bufferPool := NewBufferPool()
	startTime = time.Now()
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	for i := 0; i < 10000; i++ {
		buf := bufferPool.GetBuffer(1024)
		for j := 0; j < 100; j++ {
			buf = append(buf, byte(j))
		}
		bufferPool.Put(buf)
	}
	
	runtime.ReadMemStats(&m2)
	poolDuration := time.Since(startTime)
	poolAllocs := m2.Mallocs - m1.Mallocs
	poolBytes := m2.TotalAlloc - m1.TotalAlloc
	
	fmt.Printf("%-20s %-12s %-12s %-12s %-12s\n", 
		"Method", "Time (ms)", "Speedup", "Allocs", "Bytes")
	fmt.Println("--------------------------------------------------------------------")
	fmt.Printf("%-20s %-12.1f %-12s %-12d %-12d\n", 
		"Standard", float64(standardDuration.Nanoseconds())/1e6, "1.0x", standardAllocs, standardBytes)
	fmt.Printf("%-20s %-12.1f %-12.1fx %-12d %-12d\n", 
		"Pool", float64(poolDuration.Nanoseconds())/1e6, 
		float64(standardDuration)/float64(poolDuration), poolAllocs, poolBytes)
	
	stats := bufferPool.Stats()
	fmt.Printf("Pool hit rate: %.1f%%\n", stats.HitRate)
}

// Removed main function to avoid conflict with memory_pools.go main function
// The main function is now in memory_pools.go which demonstrates both
// object pools and memory allocators
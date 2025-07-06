package main

import (
	"sync"
	"sync/atomic"
)

// BufferPool interface defines the contract for buffer pools
type BufferPool interface {
	Get(size int) []byte
	Put(buf []byte)
	Stats() PoolStats
}

// PoolStats provides metrics about pool performance
type PoolStats struct {
	Hits        int64
	Misses      int64
	Allocations int64
	PoolSize    int64
}

// SyncBufferPool wraps sync.Pool with size optimization
type SyncBufferPool struct {
	pools map[int]*sync.Pool
	stats PoolStats
	mu    sync.RWMutex
}

// NewSyncBufferPool creates a new sync.Pool-based buffer pool
func NewSyncBufferPool() *SyncBufferPool {
	return &SyncBufferPool{
		pools: make(map[int]*sync.Pool),
	}
}

// Get retrieves a buffer of at least the specified size
func (p *SyncBufferPool) Get(size int) []byte {
	sizeClass := p.getSizeClass(size)
	
	p.mu.RLock()
	pool, exists := p.pools[sizeClass]
	p.mu.RUnlock()
	
	if !exists {
		p.mu.Lock()
		if pool, exists = p.pools[sizeClass]; !exists {
			pool = &sync.Pool{
				New: func() interface{} {
					atomic.AddInt64(&p.stats.Allocations, 1)
					return make([]byte, sizeClass)
				},
			}
			p.pools[sizeClass] = pool
		}
		p.mu.Unlock()
	}
	
	buf := pool.Get().([]byte)
	if len(buf) >= size {
		atomic.AddInt64(&p.stats.Hits, 1)
		return buf[:size]
	}
	
	atomic.AddInt64(&p.stats.Misses, 1)
	return make([]byte, size)
}

// Put returns a buffer to the pool
func (p *SyncBufferPool) Put(buf []byte) {
	if cap(buf) == 0 {
		return
	}
	
	sizeClass := p.getSizeClass(cap(buf))
	
	p.mu.RLock()
	pool, exists := p.pools[sizeClass]
	p.mu.RUnlock()
	
	if exists {
		// Reset buffer before returning to pool
		buf = buf[:cap(buf)]
		for i := range buf {
			buf[i] = 0
		}
		pool.Put(buf)
	}
}

// getSizeClass returns the appropriate size class for a given size
func (p *SyncBufferPool) getSizeClass(size int) int {
	// Use power-of-2 size classes
	if size <= 64 {
		return 64
	}
	if size <= 128 {
		return 128
	}
	if size <= 256 {
		return 256
	}
	if size <= 512 {
		return 512
	}
	if size <= 1024 {
		return 1024
	}
	if size <= 2048 {
		return 2048
	}
	if size <= 4096 {
		return 4096
	}
	if size <= 8192 {
		return 8192
	}
	// For larger sizes, round up to nearest 8KB
	return ((size + 8191) / 8192) * 8192
}

// Stats returns current pool statistics
func (p *SyncBufferPool) Stats() PoolStats {
	return PoolStats{
		Hits:        atomic.LoadInt64(&p.stats.Hits),
		Misses:      atomic.LoadInt64(&p.stats.Misses),
		Allocations: atomic.LoadInt64(&p.stats.Allocations),
		PoolSize:    int64(len(p.pools)),
	}
}

// CustomBufferPool implements a custom buffer pool with fixed capacity
type CustomBufferPool struct {
	buffers chan []byte
	size    int
	stats   PoolStats
}

// NewCustomBufferPool creates a new custom buffer pool
func NewCustomBufferPool(bufferSize, poolSize int) *CustomBufferPool {
	pool := &CustomBufferPool{
		buffers: make(chan []byte, poolSize),
		size:    bufferSize,
	}
	
	// Pre-allocate buffers
	for i := 0; i < poolSize; i++ {
		pool.buffers <- make([]byte, bufferSize)
		atomic.AddInt64(&pool.stats.Allocations, 1)
	}
	
	return pool
}

// Get retrieves a buffer from the pool
func (p *CustomBufferPool) Get(size int) []byte {
	select {
	case buf := <-p.buffers:
		atomic.AddInt64(&p.stats.Hits, 1)
		if size <= len(buf) {
			return buf[:size]
		}
		// Buffer too small, put it back and allocate new one
		select {
		case p.buffers <- buf:
		default:
		}
		// Allocate new buffer since existing one is too small
		atomic.AddInt64(&p.stats.Misses, 1)
		atomic.AddInt64(&p.stats.Allocations, 1)
		return make([]byte, size)
	default:
		atomic.AddInt64(&p.stats.Misses, 1)
		atomic.AddInt64(&p.stats.Allocations, 1)
		return make([]byte, size)
	}
}

// Put returns a buffer to the pool
func (p *CustomBufferPool) Put(buf []byte) {
	if cap(buf) != p.size {
		return // Only accept buffers of the correct size
	}
	
	// Reset buffer
	buf = buf[:cap(buf)]
	for i := range buf {
		buf[i] = 0
	}
	
	select {
	case p.buffers <- buf:
		// Successfully returned to pool
	default:
		// Pool is full, discard buffer
	}
}

// Stats returns current pool statistics
func (p *CustomBufferPool) Stats() PoolStats {
	return PoolStats{
		Hits:        atomic.LoadInt64(&p.stats.Hits),
		Misses:      atomic.LoadInt64(&p.stats.Misses),
		Allocations: atomic.LoadInt64(&p.stats.Allocations),
		PoolSize:    int64(len(p.buffers)),
	}
}

// DynamicBufferPool implements a pool that adapts to workload
type DynamicBufferPool struct {
	pools   map[int]*fixedSizePool
	stats   PoolStats
	mu      sync.RWMutex
	maxSize int
}

type fixedSizePool struct {
	buffers chan []byte
	size    int
}

// NewDynamicBufferPool creates a new dynamic buffer pool
func NewDynamicBufferPool(maxPoolSize int) *DynamicBufferPool {
	return &DynamicBufferPool{
		pools:   make(map[int]*fixedSizePool),
		maxSize: maxPoolSize,
	}
}

// Get retrieves a buffer from the appropriate size pool
func (p *DynamicBufferPool) Get(size int) []byte {
	sizeClass := p.getSizeClass(size)
	
	p.mu.RLock()
	pool, exists := p.pools[sizeClass]
	p.mu.RUnlock()
	
	if !exists {
		p.mu.Lock()
		if pool, exists = p.pools[sizeClass]; !exists {
			pool = &fixedSizePool{
				buffers: make(chan []byte, p.maxSize),
				size:    sizeClass,
			}
			p.pools[sizeClass] = pool
		}
		p.mu.Unlock()
	}
	
	select {
	case buf := <-pool.buffers:
		atomic.AddInt64(&p.stats.Hits, 1)
		return buf[:size]
	default:
		atomic.AddInt64(&p.stats.Misses, 1)
		atomic.AddInt64(&p.stats.Allocations, 1)
		return make([]byte, size)
	}
}

// Put returns a buffer to the appropriate pool
func (p *DynamicBufferPool) Put(buf []byte) {
	if cap(buf) == 0 {
		return
	}
	
	sizeClass := p.getSizeClass(cap(buf))
	
	p.mu.RLock()
	pool, exists := p.pools[sizeClass]
	p.mu.RUnlock()
	
	if exists {
		// Reset buffer
		buf = buf[:cap(buf)]
		for i := range buf {
			buf[i] = 0
		}
		
		select {
		case pool.buffers <- buf:
			// Successfully returned to pool
		default:
			// Pool is full, discard buffer
		}
	}
}

// getSizeClass returns the appropriate size class
func (p *DynamicBufferPool) getSizeClass(size int) int {
	// Use power-of-2 size classes up to 4KB, then 4KB increments
	if size <= 64 {
		return 64
	}
	if size <= 128 {
		return 128
	}
	if size <= 256 {
		return 256
	}
	if size <= 512 {
		return 512
	}
	if size <= 1024 {
		return 1024
	}
	if size <= 2048 {
		return 2048
	}
	if size <= 4096 {
		return 4096
	}
	// For larger sizes, round up to nearest 4KB
	return ((size + 4095) / 4096) * 4096
}

// Stats returns current pool statistics
func (p *DynamicBufferPool) Stats() PoolStats {
	p.mu.RLock()
	poolCount := len(p.pools)
	p.mu.RUnlock()
	
	return PoolStats{
		Hits:        atomic.LoadInt64(&p.stats.Hits),
		Misses:      atomic.LoadInt64(&p.stats.Misses),
		Allocations: atomic.LoadInt64(&p.stats.Allocations),
		PoolSize:    int64(poolCount),
	}
}
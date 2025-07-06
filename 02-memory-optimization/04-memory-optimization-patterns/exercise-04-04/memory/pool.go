package memory

import (
	"sync"
)

// BufferPool manages a pool of reusable byte slices
type BufferPool struct {
	pools map[int]*sync.Pool
	mu    sync.RWMutex
}

// NewBufferPool creates a new buffer pool
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pools: make(map[int]*sync.Pool),
	}
}

// Get retrieves a buffer of the specified size
func (bp *BufferPool) Get(size int) []byte {
	// Round up to nearest power of 2 for better pooling
	poolSize := roundUpToPowerOf2(size)

	bp.mu.RLock()
	pool, exists := bp.pools[poolSize]
	bp.mu.RUnlock()

	if !exists {
		bp.mu.Lock()
		// Double-check after acquiring write lock
		if pool, exists = bp.pools[poolSize]; !exists {
			pool = &sync.Pool{
				New: func() interface{} {
					return make([]byte, poolSize)
				},
			}
			bp.pools[poolSize] = pool
		}
		bp.mu.Unlock()
	}

	buf := pool.Get().([]byte)
	return buf[:size] // Return slice of requested size
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf []byte) {
	if buf == nil {
		return
	}

	poolSize := roundUpToPowerOf2(cap(buf))

	bp.mu.RLock()
	pool, exists := bp.pools[poolSize]
	bp.mu.RUnlock()

	if exists {
		// Reset the slice to full capacity before returning to pool
		buf = buf[:cap(buf)]
		pool.Put(buf)
	}
}

// roundUpToPowerOf2 rounds up to the nearest power of 2
func roundUpToPowerOf2(n int) int {
	if n <= 0 {
		return 1
	}

	// Handle small sizes directly
	if n <= 64 {
		return 64
	}
	if n <= 128 {
		return 128
	}
	if n <= 256 {
		return 256
	}
	if n <= 512 {
		return 512
	}
	if n <= 1024 {
		return 1024
	}
	if n <= 2048 {
		return 2048
	}
	if n <= 4096 {
		return 4096
	}
	if n <= 8192 {
		return 8192
	}

	// For larger sizes, use bit manipulation
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++

	return n
}

// Stats returns pool statistics
func (bp *BufferPool) Stats() map[int]int {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	stats := make(map[int]int)
	for size := range bp.pools {
		stats[size] = 0 // We can't easily get pool size from sync.Pool
	}
	return stats
}
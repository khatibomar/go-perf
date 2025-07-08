package buffer

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// AdaptiveBuffer dynamically adjusts its size based on load
type AdaptiveBuffer struct {
	mu           sync.RWMutex
	buffer       []interface{}
	capacity     int64
	size         int64
	head        int64
	tail         int64
	loadFactor   float64
	lastResize   time.Time
	resizeDelay  time.Duration
	minCapacity  int
	maxCapacity  int
	growthFactor float64
	shrinkFactor float64
	metrics      *BufferMetrics
	ctx          context.Context
	cancel       context.CancelFunc
}

// BufferMetrics tracks buffer performance
type BufferMetrics struct {
	SendCount    int64
	ReceiveCount int64
	DropCount    int64
	ResizeCount  int64
	AvgLoadFactor float64
	PeakCapacity int64
	mu           sync.RWMutex
}

// NewAdaptiveBuffer creates a new adaptive buffer
func NewAdaptiveBuffer(initialCapacity, minCapacity, maxCapacity int) *AdaptiveBuffer {
	if initialCapacity < minCapacity {
		initialCapacity = minCapacity
	}
	if initialCapacity > maxCapacity {
		initialCapacity = maxCapacity
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	ab := &AdaptiveBuffer{
		buffer:       make([]interface{}, initialCapacity),
		capacity:     int64(initialCapacity),
		resizeDelay:  time.Millisecond * 100,
		minCapacity:  minCapacity,
		maxCapacity:  maxCapacity,
		growthFactor: 1.5,
		shrinkFactor: 0.7,
		metrics:      &BufferMetrics{},
		ctx:          ctx,
		cancel:       cancel,
		lastResize:   time.Now(),
	}
	
	// Start background resize monitor
	go ab.monitorAndResize()
	
	return ab
}

// Send adds data to the buffer
func (ab *AdaptiveBuffer) Send(data interface{}) bool {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	
	// Check if buffer is full
	if atomic.LoadInt64(&ab.size) >= atomic.LoadInt64(&ab.capacity) {
		// Try to resize if needed
		if ab.tryResize() {
			// Resize successful, try again
			if atomic.LoadInt64(&ab.size) >= atomic.LoadInt64(&ab.capacity) {
				atomic.AddInt64(&ab.metrics.DropCount, 1)
				return false
			}
		} else {
			atomic.AddInt64(&ab.metrics.DropCount, 1)
			return false
		}
	}
	
	// Add to buffer
	tailPos := atomic.LoadInt64(&ab.tail)
	ab.buffer[tailPos] = data
	atomic.StoreInt64(&ab.tail, (tailPos+1)%atomic.LoadInt64(&ab.capacity))
	atomic.AddInt64(&ab.size, 1)
	atomic.AddInt64(&ab.metrics.SendCount, 1)
	
	// Update load factor
	ab.updateLoadFactor()
	
	return true
}

// Receive gets data from the buffer
func (ab *AdaptiveBuffer) Receive() (interface{}, bool) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	
	if atomic.LoadInt64(&ab.size) == 0 {
		return nil, false
	}
	
	headPos := atomic.LoadInt64(&ab.head)
	data := ab.buffer[headPos]
	ab.buffer[headPos] = nil // Help GC
	atomic.StoreInt64(&ab.head, (headPos+1)%atomic.LoadInt64(&ab.capacity))
	atomic.AddInt64(&ab.size, -1)
	atomic.AddInt64(&ab.metrics.ReceiveCount, 1)
	
	// Update load factor
	ab.updateLoadFactor()
	
	return data, true
}

// tryResize attempts to resize the buffer based on current load
func (ab *AdaptiveBuffer) tryResize() bool {
	now := time.Now()
	if now.Sub(ab.lastResize) < ab.resizeDelay {
		return false
	}
	
	currentCapacity := int(atomic.LoadInt64(&ab.capacity))
	currentSize := int(atomic.LoadInt64(&ab.size))
	
	// Determine new capacity
	var newCapacity int
	if ab.loadFactor > 0.8 && currentCapacity < ab.maxCapacity {
		// Grow buffer
		newCapacity = int(float64(currentCapacity) * ab.growthFactor)
		if newCapacity > ab.maxCapacity {
			newCapacity = ab.maxCapacity
		}
	} else if ab.loadFactor < 0.3 && currentCapacity > ab.minCapacity {
		// Shrink buffer
		newCapacity = int(float64(currentCapacity) * ab.shrinkFactor)
		if newCapacity < ab.minCapacity {
			newCapacity = ab.minCapacity
		}
	} else {
		return false
	}
	
	if newCapacity == currentCapacity {
		return false
	}
	
	// Create new buffer and copy data
	newBuffer := make([]interface{}, newCapacity)
	copied := 0
	
	// Copy existing data
	for i := 0; i < currentSize && copied < newCapacity; i++ {
		pos := (int(atomic.LoadInt64(&ab.head)) + i) % currentCapacity
		newBuffer[copied] = ab.buffer[pos]
		copied++
	}
	
	// Update buffer
	ab.buffer = newBuffer
	atomic.StoreInt64(&ab.capacity, int64(newCapacity))
	atomic.StoreInt64(&ab.head, 0)
	atomic.StoreInt64(&ab.tail, int64(copied))
	atomic.StoreInt64(&ab.size, int64(copied))
	ab.lastResize = now
	atomic.AddInt64(&ab.metrics.ResizeCount, 1)
	
	// Update peak capacity
	if int64(newCapacity) > atomic.LoadInt64(&ab.metrics.PeakCapacity) {
		atomic.StoreInt64(&ab.metrics.PeakCapacity, int64(newCapacity))
	}
	
	return true
}

// updateLoadFactor calculates current load factor
func (ab *AdaptiveBuffer) updateLoadFactor() {
	capacity := atomic.LoadInt64(&ab.capacity)
	size := atomic.LoadInt64(&ab.size)
	
	if capacity > 0 {
		ab.loadFactor = float64(size) / float64(capacity)
		
		// Update average load factor (simple moving average)
		ab.metrics.mu.Lock()
		ab.metrics.AvgLoadFactor = (ab.metrics.AvgLoadFactor + ab.loadFactor) / 2
		ab.metrics.mu.Unlock()
	}
}

// monitorAndResize runs background monitoring and resizing
func (ab *AdaptiveBuffer) monitorAndResize() {
	ticker := time.NewTicker(ab.resizeDelay)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ab.mu.Lock()
			ab.tryResize()
			ab.mu.Unlock()
		case <-ab.ctx.Done():
			return
		}
	}
}

// GetMetrics returns current buffer metrics
func (ab *AdaptiveBuffer) GetMetrics() BufferMetrics {
	ab.metrics.mu.RLock()
	defer ab.metrics.mu.RUnlock()
	
	return BufferMetrics{
		SendCount:     atomic.LoadInt64(&ab.metrics.SendCount),
		ReceiveCount:  atomic.LoadInt64(&ab.metrics.ReceiveCount),
		DropCount:     atomic.LoadInt64(&ab.metrics.DropCount),
		ResizeCount:   atomic.LoadInt64(&ab.metrics.ResizeCount),
		AvgLoadFactor: ab.metrics.AvgLoadFactor,
		PeakCapacity:  atomic.LoadInt64(&ab.metrics.PeakCapacity),
	}
}

// GetCapacity returns current buffer capacity
func (ab *AdaptiveBuffer) GetCapacity() int {
	return int(atomic.LoadInt64(&ab.capacity))
}

// GetSize returns current buffer size
func (ab *AdaptiveBuffer) GetSize() int {
	return int(atomic.LoadInt64(&ab.size))
}

// GetLoadFactor returns current load factor
func (ab *AdaptiveBuffer) GetLoadFactor() float64 {
	ab.mu.RLock()
	defer ab.mu.RUnlock()
	return ab.loadFactor
}

// Close shuts down the adaptive buffer
func (ab *AdaptiveBuffer) Close() {
	ab.cancel()
}
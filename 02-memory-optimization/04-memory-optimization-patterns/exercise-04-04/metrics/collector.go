package metrics

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Collector collects and manages pipeline metrics
type Collector struct {
	stats           *Stats
	latencyHistory  []time.Duration
	mu              sync.RWMutex
	startTime       time.Time
}

// Stats holds pipeline statistics
type Stats struct {
	Processed       int64         `json:"processed"`
	Written         int64         `json:"written"`
	Errors          int64         `json:"errors"`
	MemoryPressure  int64         `json:"memory_pressure"`
	AvgLatency      time.Duration `json:"avg_latency"`
	MaxLatency      time.Duration `json:"max_latency"`
	MinLatency      time.Duration `json:"min_latency"`
	Throughput      float64       `json:"throughput"` // items per second
	Uptime          time.Duration `json:"uptime"`
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{
		stats: &Stats{
			MinLatency: time.Duration(^uint64(0) >> 1), // Max duration
		},
		latencyHistory: make([]time.Duration, 0, 1000),
		startTime:      time.Now(),
	}
}

// Start begins metrics collection
func (c *Collector) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.updateDerivedMetrics()
		}
	}
}

// IncrementProcessed increments the processed counter
func (c *Collector) IncrementProcessed() {
	atomic.AddInt64(&c.stats.Processed, 1)
}

// IncrementWritten increments the written counter
func (c *Collector) IncrementWritten() {
	atomic.AddInt64(&c.stats.Written, 1)
}

// IncrementErrors increments the error counter
func (c *Collector) IncrementErrors() {
	atomic.AddInt64(&c.stats.Errors, 1)
}

// IncrementMemoryPressure increments the memory pressure counter
func (c *Collector) IncrementMemoryPressure() {
	atomic.AddInt64(&c.stats.MemoryPressure, 1)
}

// RecordLatency records processing latency
func (c *Collector) RecordLatency(latency time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update min/max latency
	if latency < c.stats.MinLatency {
		c.stats.MinLatency = latency
	}
	if latency > c.stats.MaxLatency {
		c.stats.MaxLatency = latency
	}

	// Add to history (keep last 1000 entries)
	if len(c.latencyHistory) >= 1000 {
		c.latencyHistory = c.latencyHistory[1:]
	}
	c.latencyHistory = append(c.latencyHistory, latency)
}

// updateDerivedMetrics calculates derived metrics like throughput and average latency
func (c *Collector) updateDerivedMetrics() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Calculate average latency
	if len(c.latencyHistory) > 0 {
		var total time.Duration
		for _, latency := range c.latencyHistory {
			total += latency
		}
		c.stats.AvgLatency = total / time.Duration(len(c.latencyHistory))
	}

	// Calculate throughput (items per second)
	uptime := time.Since(c.startTime)
	c.stats.Uptime = uptime
	if uptime.Seconds() > 0 {
		c.stats.Throughput = float64(atomic.LoadInt64(&c.stats.Processed)) / uptime.Seconds()
	}
}

// GetStats returns a copy of current statistics
func (c *Collector) GetStats() *Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Create a copy to avoid race conditions
	stats := *c.stats
	stats.Processed = atomic.LoadInt64(&c.stats.Processed)
	stats.Written = atomic.LoadInt64(&c.stats.Written)
	stats.Errors = atomic.LoadInt64(&c.stats.Errors)
	stats.MemoryPressure = atomic.LoadInt64(&c.stats.MemoryPressure)

	return &stats
}

// Reset resets all metrics
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.StoreInt64(&c.stats.Processed, 0)
	atomic.StoreInt64(&c.stats.Written, 0)
	atomic.StoreInt64(&c.stats.Errors, 0)
	atomic.StoreInt64(&c.stats.MemoryPressure, 0)
	c.stats.AvgLatency = 0
	c.stats.MaxLatency = 0
	c.stats.MinLatency = time.Duration(^uint64(0) >> 1)
	c.stats.Throughput = 0
	c.latencyHistory = c.latencyHistory[:0]
	c.startTime = time.Now()
}
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sync"
	"sync/atomic"
	"time"
)

// CacheLevel represents different cache levels
type CacheLevel int

const (
	L1Cache CacheLevel = iota // Fast in-memory cache
	L2Cache                   // Slower but larger cache
	L3Cache                   // Persistent storage cache
)

// WriteStrategy defines cache write strategies
type WriteStrategy int

const (
	WriteThrough WriteStrategy = iota // Write to cache and storage simultaneously
	WriteBehind                       // Write to cache immediately, storage asynchronously
	WriteAround                       // Write directly to storage, bypass cache
)

// EvictionPolicy defines cache eviction strategies
type EvictionPolicy int

const (
	LRU EvictionPolicy = iota // Least Recently Used
	LFU                       // Least Frequently Used
	TTL                       // Time To Live
)

// CacheItem represents an item in the cache
type CacheItem struct {
	Value       interface{}
	Expiry      time.Time
	AccessTime  time.Time
	AccessCount int64
	Size        int64
	Level       CacheLevel
	Dirty       bool // For write-behind strategy
}

// CacheStats holds cache performance statistics
type CacheStats struct {
	Hits          int64
	Misses        int64
	Evictions     int64
	Size          int64
	MemoryUsage   int64
	Writes        int64
	Reads         int64
	Promotions    int64 // Items promoted from lower to higher cache levels
	Demotions     int64 // Items demoted from higher to lower cache levels
	CoherenceOps  int64 // Cache coherence operations
}

// CacheMetrics provides performance metrics
type CacheMetrics struct {
	Level           CacheLevel
	HitRate         float64
	MissRate        float64
	EvictionRate    float64
	MemoryUsage     int64
	AverageLatency  time.Duration
	Throughput      int64
	PromotionRate   float64
	CoherenceRate   float64
}

// Cache represents a single cache level
type Cache struct {
	data        map[string]*CacheItem
	mu          sync.RWMutex
	maxSize     int
	maxMemory   int64
	ttl         time.Duration
	eviction    EvictionPolicy
	level       CacheLevel
	stats       *CacheStats
	cleanupTick *time.Ticker
	stopCleanup chan bool
}

// NewCache creates a new cache instance
func NewCache(maxSize int, maxMemory int64, ttl time.Duration, eviction EvictionPolicy, level CacheLevel) *Cache {
	c := &Cache{
		data:        make(map[string]*CacheItem),
		maxSize:     maxSize,
		maxMemory:   maxMemory,
		ttl:         ttl,
		eviction:    eviction,
		level:       level,
		stats:       &CacheStats{},
		cleanupTick: time.NewTicker(time.Minute),
		stopCleanup: make(chan bool),
	}

	// Start background cleanup goroutine
	go c.cleanup()

	return c
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.AddInt64(&c.stats.Reads, 1)

	item, exists := c.data[key]
	if !exists {
		atomic.AddInt64(&c.stats.Misses, 1)
		return nil, false
	}

	// Check if item has expired
	if time.Now().After(item.Expiry) {
		delete(c.data, key)
		atomic.AddInt64(&c.stats.Size, -1)
		atomic.AddInt64(&c.stats.MemoryUsage, -item.Size)
		atomic.AddInt64(&c.stats.Misses, 1)
		return nil, false
	}

	// Update access statistics
	atomic.AddInt64(&item.AccessCount, 1)
	item.AccessTime = time.Now()
	atomic.AddInt64(&c.stats.Hits, 1)

	return item.Value, true
}

// Set stores a value in the cache
func (c *Cache) Set(key string, value interface{}) bool {
	size := c.calculateSize(value)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict items
	for (len(c.data) >= c.maxSize || atomic.LoadInt64(&c.stats.MemoryUsage)+size > c.maxMemory) && len(c.data) > 0 {
		if !c.evictOne() {
			break
		}
	}

	// If still no space, reject the item
	if len(c.data) >= c.maxSize || atomic.LoadInt64(&c.stats.MemoryUsage)+size > c.maxMemory {
		return false
	}

	item := &CacheItem{
		Value:       value,
		Expiry:      time.Now().Add(c.ttl),
		AccessTime:  time.Now(),
		AccessCount: 1,
		Size:        size,
		Level:       c.level,
		Dirty:       false,
	}

	c.data[key] = item
	atomic.AddInt64(&c.stats.Size, 1)
	atomic.AddInt64(&c.stats.MemoryUsage, size)
	atomic.AddInt64(&c.stats.Writes, 1)

	return true
}

// Delete removes an item from the cache
func (c *Cache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.data[key]
	if !exists {
		return false
	}

	delete(c.data, key)
	atomic.AddInt64(&c.stats.Size, -1)
	atomic.AddInt64(&c.stats.MemoryUsage, -item.Size)

	return true
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]*CacheItem)
	atomic.StoreInt64(&c.stats.Size, 0)
	atomic.StoreInt64(&c.stats.MemoryUsage, 0)
}

// Stats returns cache statistics
func (c *Cache) Stats() CacheStats {
	return CacheStats{
		Hits:         atomic.LoadInt64(&c.stats.Hits),
		Misses:       atomic.LoadInt64(&c.stats.Misses),
		Evictions:    atomic.LoadInt64(&c.stats.Evictions),
		Size:         atomic.LoadInt64(&c.stats.Size),
		MemoryUsage:  atomic.LoadInt64(&c.stats.MemoryUsage),
		Writes:       atomic.LoadInt64(&c.stats.Writes),
		Reads:        atomic.LoadInt64(&c.stats.Reads),
		Promotions:   atomic.LoadInt64(&c.stats.Promotions),
		Demotions:    atomic.LoadInt64(&c.stats.Demotions),
		CoherenceOps: atomic.LoadInt64(&c.stats.CoherenceOps),
	}
}

// GetMetrics returns performance metrics
func (c *Cache) GetMetrics() CacheMetrics {
	stats := c.Stats()
	total := stats.Hits + stats.Misses

	var hitRate, missRate, evictionRate, promotionRate, coherenceRate float64
	if total > 0 {
		hitRate = float64(stats.Hits) / float64(total)
		missRate = float64(stats.Misses) / float64(total)
		evictionRate = float64(stats.Evictions) / float64(total)
		promotionRate = float64(stats.Promotions) / float64(total)
		coherenceRate = float64(stats.CoherenceOps) / float64(total)
	}

	return CacheMetrics{
		Level:         c.level,
		HitRate:       hitRate,
		MissRate:      missRate,
		EvictionRate:  evictionRate,
		MemoryUsage:   stats.MemoryUsage,
		Throughput:    stats.Reads + stats.Writes,
		PromotionRate: promotionRate,
		CoherenceRate: coherenceRate,
	}
}

// calculateSize estimates the memory size of a value
func (c *Cache) calculateSize(value interface{}) int64 {
	switch v := value.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	case int, int32, int64, float32, float64:
		return 8
	default:
		// For complex types, use JSON serialization size as estimate
		if data, err := json.Marshal(v); err == nil {
			return int64(len(data))
		}
		return 64 // Default estimate
	}
}

// evictOne removes one item based on eviction policy
func (c *Cache) evictOne() bool {
	if len(c.data) == 0 {
		return false
	}

	var victimKey string
	switch c.eviction {
	case LRU:
		victimKey = c.findLRUVictim()
	case LFU:
		victimKey = c.findLFUVictim()
	case TTL:
		victimKey = c.findTTLVictim()
	}

	if victimKey != "" {
		item := c.data[victimKey]
		delete(c.data, victimKey)
		atomic.AddInt64(&c.stats.Evictions, 1)
		atomic.AddInt64(&c.stats.Size, -1)
		atomic.AddInt64(&c.stats.MemoryUsage, -item.Size)
		return true
	}

	return false
}

// findLRUVictim finds the least recently used item
func (c *Cache) findLRUVictim() string {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, item := range c.data {
		if first || item.AccessTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.AccessTime
			first = false
		}
	}

	return oldestKey
}

// findLFUVictim finds the least frequently used item
func (c *Cache) findLFUVictim() string {
	var victimKey string
	var minCount int64 = -1

	for key, item := range c.data {
		if minCount == -1 || item.AccessCount < minCount {
			victimKey = key
			minCount = item.AccessCount
		}
	}

	return victimKey
}

// findTTLVictim finds the item closest to expiration
func (c *Cache) findTTLVictim() string {
	var victimKey string
	var earliestExpiry time.Time

	for key, item := range c.data {
		if victimKey == "" || item.Expiry.Before(earliestExpiry) {
			victimKey = key
			earliestExpiry = item.Expiry
		}
	}

	return victimKey
}

// cleanup removes expired items periodically
func (c *Cache) cleanup() {
	for {
		select {
		case <-c.cleanupTick.C:
			c.removeExpired()
		case <-c.stopCleanup:
			return
		}
	}
}

// removeExpired removes all expired items
func (c *Cache) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.data {
		if now.After(item.Expiry) {
			delete(c.data, key)
			atomic.AddInt64(&c.stats.Size, -1)
			atomic.AddInt64(&c.stats.MemoryUsage, -item.Size)
			atomic.AddInt64(&c.stats.Evictions, 1)
		}
	}
}

// Close stops the cache and cleanup goroutine
func (c *Cache) Close() {
	c.cleanupTick.Stop()
	close(c.stopCleanup)
}

// MultiLevelCache represents a hierarchical caching system
type MultiLevelCache struct {
	caches        []*Cache
	writeStrategy WriteStrategy
	storage       Storage
	coherence     CoherenceProtocol
	mu            sync.RWMutex
	writeQueue    chan WriteOperation
	stopWriter    chan bool
}

// WriteOperation represents a pending write operation
type WriteOperation struct {
	Key       string
	Value     interface{}
	Timestamp time.Time
	Level     CacheLevel
}

// Storage interface for persistent storage
type Storage interface {
	Get(key string) (interface{}, error)
	Set(key string, value interface{}) error
	Delete(key string) error
}

// CoherenceProtocol defines cache coherence operations
type CoherenceProtocol interface {
	Invalidate(key string) error
	Update(key string, value interface{}) error
	Notify(level CacheLevel, key string, operation string) error
}

// MockStorage implements Storage interface for testing
type MockStorage struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		data: make(map[string]interface{}),
	}
}

func (s *MockStorage) Get(key string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, exists := s.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return value, nil
}

func (s *MockStorage) Set(key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = value
	return nil
}

func (s *MockStorage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	return nil
}

// SimpleCoherence implements a basic coherence protocol
type SimpleCoherence struct {
	caches []*Cache
	mu     sync.RWMutex
}

func NewSimpleCoherence(caches []*Cache) *SimpleCoherence {
	return &SimpleCoherence{
		caches: caches,
	}
}

func (sc *SimpleCoherence) Invalidate(key string) error {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	for _, cache := range sc.caches {
		cache.Delete(key)
		atomic.AddInt64(&cache.stats.CoherenceOps, 1)
	}
	return nil
}

func (sc *SimpleCoherence) Update(key string, value interface{}) error {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	for _, cache := range sc.caches {
		cache.Set(key, value)
		atomic.AddInt64(&cache.stats.CoherenceOps, 1)
	}
	return nil
}

func (sc *SimpleCoherence) Notify(level CacheLevel, key string, operation string) error {
	// Simple notification - could be extended for more sophisticated protocols
	log.Printf("Coherence notification: Level %d, Key %s, Operation %s", level, key, operation)
	return nil
}

// NewMultiLevelCache creates a new multi-level cache
func NewMultiLevelCache(caches []*Cache, writeStrategy WriteStrategy, storage Storage) *MultiLevelCache {
	mlc := &MultiLevelCache{
		caches:        caches,
		writeStrategy: writeStrategy,
		storage:       storage,
		coherence:     NewSimpleCoherence(caches),
		writeQueue:    make(chan WriteOperation, 1000),
		stopWriter:    make(chan bool),
	}

	// Start background writer for write-behind strategy
	if writeStrategy == WriteBehind {
		go mlc.backgroundWriter()
	}

	return mlc
}

// Get retrieves a value from the cache hierarchy
func (mlc *MultiLevelCache) Get(ctx context.Context, key string) (interface{}, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start)
		if latency > 5*time.Millisecond {
			log.Printf("Warning: Get operation took %v (target: <5ms)", latency)
		}
	}()

	// Try each cache level in order
	for i, cache := range mlc.caches {
		if value, found := cache.Get(key); found {
			// Promote to higher cache levels
			for j := i - 1; j >= 0; j-- {
				if mlc.caches[j].Set(key, value) {
					atomic.AddInt64(&mlc.caches[j].stats.Promotions, 1)
				}
			}
			return value, nil
		}
	}

	// Try storage if not found in any cache
	if mlc.storage != nil {
		value, err := mlc.storage.Get(key)
		if err == nil {
			// Store in all cache levels
			for _, cache := range mlc.caches {
				cache.Set(key, value)
			}
			return value, nil
		}
	}

	return nil, fmt.Errorf("key not found: %s", key)
}

// Set stores a value using the configured write strategy
func (mlc *MultiLevelCache) Set(ctx context.Context, key string, value interface{}) error {
	start := time.Now()
	defer func() {
		latency := time.Since(start)
		if latency > 5*time.Millisecond {
			log.Printf("Warning: Set operation took %v (target: <5ms)", latency)
		}
	}()

	switch mlc.writeStrategy {
	case WriteThrough:
		return mlc.writeThrough(key, value)
	case WriteBehind:
		return mlc.writeBehind(key, value)
	case WriteAround:
		return mlc.writeAround(key, value)
	default:
		return fmt.Errorf("unknown write strategy: %v", mlc.writeStrategy)
	}
}

// writeThrough writes to cache and storage simultaneously
func (mlc *MultiLevelCache) writeThrough(key string, value interface{}) error {
	// Write to all cache levels
	for _, cache := range mlc.caches {
		cache.Set(key, value)
	}

	// Write to storage
	if mlc.storage != nil {
		return mlc.storage.Set(key, value)
	}

	return nil
}

// writeBehind writes to cache immediately, storage asynchronously
func (mlc *MultiLevelCache) writeBehind(key string, value interface{}) error {
	// Write to all cache levels immediately
	for _, cache := range mlc.caches {
		cache.Set(key, value)
		// Mark as dirty for write-behind
		if item, exists := cache.data[key]; exists {
			item.Dirty = true
		}
	}

	// Queue for async write to storage
	select {
	case mlc.writeQueue <- WriteOperation{
		Key:       key,
		Value:     value,
		Timestamp: time.Now(),
		Level:     L1Cache,
	}:
	default:
		// Queue is full, write synchronously
		if mlc.storage != nil {
			return mlc.storage.Set(key, value)
		}
	}

	return nil
}

// writeAround writes directly to storage, bypassing cache
func (mlc *MultiLevelCache) writeAround(key string, value interface{}) error {
	// Invalidate from all cache levels
	for _, cache := range mlc.caches {
		cache.Delete(key)
	}

	// Write directly to storage
	if mlc.storage != nil {
		return mlc.storage.Set(key, value)
	}

	return nil
}

// backgroundWriter handles async writes for write-behind strategy
func (mlc *MultiLevelCache) backgroundWriter() {
	for {
		select {
		case op := <-mlc.writeQueue:
			if mlc.storage != nil {
				if err := mlc.storage.Set(op.Key, op.Value); err != nil {
					log.Printf("Background write failed for key %s: %v", op.Key, err)
				}
			}
		case <-mlc.stopWriter:
			return
		}
	}
}

// Delete removes a key from all cache levels and storage
func (mlc *MultiLevelCache) Delete(ctx context.Context, key string) error {
	// Remove from all cache levels
	for _, cache := range mlc.caches {
		cache.Delete(key)
	}

	// Remove from storage
	if mlc.storage != nil {
		return mlc.storage.Delete(key)
	}

	return nil
}

// InvalidatePattern invalidates keys matching a pattern
func (mlc *MultiLevelCache) InvalidatePattern(pattern string) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %v", err)
	}

	// Collect keys to invalidate
	var keysToInvalidate []string
	for _, cache := range mlc.caches {
		cache.mu.RLock()
		for key := range cache.data {
			if regex.MatchString(key) {
				keysToInvalidate = append(keysToInvalidate, key)
			}
		}
		cache.mu.RUnlock()
	}

	// Remove duplicates
	keySet := make(map[string]bool)
	for _, key := range keysToInvalidate {
		keySet[key] = true
	}

	// Invalidate each unique key
	for key := range keySet {
		for _, cache := range mlc.caches {
			cache.mu.Lock()
			if item, exists := cache.data[key]; exists {
				delete(cache.data, key)
				atomic.AddInt64(&cache.stats.Size, -1)
				atomic.AddInt64(&cache.stats.MemoryUsage, -item.Size)
				atomic.AddInt64(&cache.stats.CoherenceOps, 1)
			}
			cache.mu.Unlock()
		}
		// Also remove from storage if available
		if mlc.storage != nil {
			mlc.storage.Delete(key)
		}
	}

	return nil
}

// GetStats returns statistics for all cache levels
func (mlc *MultiLevelCache) GetStats() map[CacheLevel]CacheStats {
	stats := make(map[CacheLevel]CacheStats)
	for _, cache := range mlc.caches {
		stats[cache.level] = cache.Stats()
	}
	return stats
}

// GetMetrics returns performance metrics for all cache levels
func (mlc *MultiLevelCache) GetMetrics() map[CacheLevel]CacheMetrics {
	metrics := make(map[CacheLevel]CacheMetrics)
	for _, cache := range mlc.caches {
		metrics[cache.level] = cache.GetMetrics()
	}
	return metrics
}

// Flush ensures all pending writes are completed
func (mlc *MultiLevelCache) Flush() error {
	if mlc.writeStrategy == WriteBehind {
		// Process all pending writes
		for len(mlc.writeQueue) > 0 {
			select {
			case op := <-mlc.writeQueue:
				if mlc.storage != nil {
					if err := mlc.storage.Set(op.Key, op.Value); err != nil {
						return err
					}
				}
			default:
				break
			}
		}
	}
	return nil
}

// Close shuts down the multi-level cache
func (mlc *MultiLevelCache) Close() error {
	// Flush pending writes
	if err := mlc.Flush(); err != nil {
		return err
	}

	// Stop background writer
	if mlc.writeStrategy == WriteBehind {
		close(mlc.stopWriter)
	}

	// Close all cache levels
	for _, cache := range mlc.caches {
		cache.Close()
	}

	return nil
}

// Example usage and demonstration
func main() {
	fmt.Println("=== Multi-Level Caching Architecture Demo ===")

	// Create cache levels
	l1Cache := NewCache(100, 1024*1024, 5*time.Minute, LRU, L1Cache)    // 100 items, 1MB, 5min TTL
	l2Cache := NewCache(1000, 10*1024*1024, 30*time.Minute, LRU, L2Cache) // 1000 items, 10MB, 30min TTL
	l3Cache := NewCache(10000, 100*1024*1024, 2*time.Hour, LFU, L3Cache)  // 10000 items, 100MB, 2h TTL

	caches := []*Cache{l1Cache, l2Cache, l3Cache}
	storage := NewMockStorage()

	// Create multi-level cache with write-through strategy
	mlc := NewMultiLevelCache(caches, WriteThrough, storage)
	defer mlc.Close()

	ctx := context.Background()

	fmt.Println("\n1. Testing cache hierarchy...")

	// Set some values
	keys := []string{"user:1", "user:2", "user:3", "product:1", "product:2"}
	values := []string{"John Doe", "Jane Smith", "Bob Johnson", "Laptop", "Phone"}

	for i, key := range keys {
		err := mlc.Set(ctx, key, values[i])
		if err != nil {
			fmt.Printf("Error setting %s: %v\n", key, err)
		} else {
			fmt.Printf("Set %s = %s\n", key, values[i])
		}
	}

	fmt.Println("\n2. Testing cache retrieval and promotion...")

	// Get values (should promote from lower to higher levels)
	for _, key := range keys {
		value, err := mlc.Get(ctx, key)
		if err != nil {
			fmt.Printf("Error getting %s: %v\n", key, err)
		} else {
			fmt.Printf("Get %s = %v\n", key, value)
		}
	}

	fmt.Println("\n3. Cache statistics by level:")
	metrics := mlc.GetMetrics()
	for level, metric := range metrics {
		levelName := []string{"L1", "L2", "L3"}[level]
		fmt.Printf("%s Cache - Hit Rate: %.2f%%, Memory: %d bytes, Promotions: %.2f%%\n",
			levelName, metric.HitRate*100, metric.MemoryUsage, metric.PromotionRate*100)
	}

	fmt.Println("\n4. Testing write strategies...")

	// Test write-behind strategy
	mlcWriteBehind := NewMultiLevelCache(caches, WriteBehind, storage)
	defer mlcWriteBehind.Close()

	err := mlcWriteBehind.Set(ctx, "async:key", "async value")
	if err != nil {
		fmt.Printf("Error with write-behind: %v\n", err)
	} else {
		fmt.Println("Write-behind operation completed")
	}

	// Test write-around strategy
	mlcWriteAround := NewMultiLevelCache(caches, WriteAround, storage)
	defer mlcWriteAround.Close()

	err = mlcWriteAround.Set(ctx, "direct:key", "direct value")
	if err != nil {
		fmt.Printf("Error with write-around: %v\n", err)
	} else {
		fmt.Println("Write-around operation completed")
	}

	fmt.Println("\n5. Testing pattern invalidation...")

	// Add some test data
	testKeys := []string{"temp:data1", "temp:data2", "temp:data3", "perm:data1"}
	for _, key := range testKeys {
		mlc.Set(ctx, key, "test value")
	}

	// Invalidate temp keys
	err = mlc.InvalidatePattern("^temp:.*")
	if err != nil {
		fmt.Printf("Error invalidating pattern: %v\n", err)
	} else {
		fmt.Println("Successfully invalidated temp:* pattern")
	}

	// Verify invalidation
	for _, key := range testKeys {
		value, err := mlc.Get(ctx, key)
		if err != nil {
			fmt.Printf("%s: not found (invalidated)\n", key)
		} else {
			fmt.Printf("%s: %v (still exists)\n", key, value)
		}
	}

	fmt.Println("\n6. Final performance metrics:")
	finalMetrics := mlc.GetMetrics()
	for level, metric := range finalMetrics {
		levelName := []string{"L1", "L2", "L3"}[level]
		fmt.Printf("%s Cache Performance:\n", levelName)
		fmt.Printf("  Hit Rate: %.2f%% (Target: L1>90%%, L2>95%%)\n", metric.HitRate*100)
		fmt.Printf("  Memory Usage: %d bytes\n", metric.MemoryUsage)
		fmt.Printf("  Throughput: %d ops\n", metric.Throughput)
		fmt.Printf("  Promotion Rate: %.2f%%\n", metric.PromotionRate*100)
		fmt.Printf("  Coherence Rate: %.2f%%\n", metric.CoherenceRate*100)
		fmt.Println()
	}

	fmt.Println("=== Multi-Level Cache Demo Completed Successfully ===")
}
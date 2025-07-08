package main

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// EvictionPolicy defines the cache eviction strategy
type EvictionPolicy int

const (
	LRU EvictionPolicy = iota // Least Recently Used
	LFU                       // Least Frequently Used
	TTL                       // Time To Live based
)

// CacheItem represents a single cache entry
type CacheItem struct {
	Value       interface{}
	Expiry      time.Time
	AccessTime  time.Time
	AccessCount int64
	Size        int64
}

// CacheStats holds cache performance statistics
type CacheStats struct {
	Hits        int64
	Misses      int64
	Evictions   int64
	Size        int64
	MemoryUsage int64
}

// Cache represents a high-performance thread-safe in-memory cache
type Cache struct {
	data     map[string]*CacheItem
	mu       sync.RWMutex
	maxSize  int
	ttl      time.Duration
	eviction EvictionPolicy
	stats    *CacheStats
	cleanup  *time.Ticker
	stopCh   chan struct{}
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	MaxSize         int
	TTL             time.Duration
	EvictionPolicy  EvictionPolicy
	CleanupInterval time.Duration
}

// DefaultCacheConfig returns a default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		MaxSize:         1000000, // 1M entries
		TTL:             5 * time.Minute,
		EvictionPolicy:  LRU,
		CleanupInterval: 1 * time.Minute,
	}
}

// NewCache creates a new cache instance
func NewCache(config CacheConfig) *Cache {
	c := &Cache{
		data:     make(map[string]*CacheItem),
		maxSize:  config.MaxSize,
		ttl:      config.TTL,
		eviction: config.EvictionPolicy,
		stats:    &CacheStats{},
		stopCh:   make(chan struct{}),
	}
	
	// Start background cleanup goroutine only if cleanup interval is reasonable
	if config.CleanupInterval > 0 {
		c.cleanup = time.NewTicker(config.CleanupInterval)
		go c.cleanupExpired()
	}
	
	return c
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, exists := c.data[key]
	c.mu.RUnlock()
	
	if !exists {
		atomic.AddInt64(&c.stats.Misses, 1)
		return nil, false
	}
	
	// Check if item has expired
	if time.Now().After(item.Expiry) {
		c.Delete(key)
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
func (c *Cache) Set(key string, value interface{}) {
	item := &CacheItem{
		Value:       value,
		Expiry:      time.Now().Add(c.ttl),
		AccessTime:  time.Now(),
		AccessCount: 1,
		Size:        c.calculateSize(value),
	}
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check if key already exists
	if existingItem, exists := c.data[key]; exists {
		// Update existing item
		atomic.AddInt64(&c.stats.MemoryUsage, item.Size-existingItem.Size)
		item.AccessCount = existingItem.AccessCount // Preserve access count
		c.data[key] = item
		return
	}
	
	// Check if eviction is needed
	for len(c.data) >= c.maxSize {
		oldSize := len(c.data)
		c.evict()
		// If eviction didn't reduce size, break to avoid infinite loop
		if len(c.data) >= oldSize {
			break
		}
	}
	
	c.data[key] = item
	atomic.AddInt64(&c.stats.Size, 1)
	atomic.AddInt64(&c.stats.MemoryUsage, item.Size)
}

// SetWithTTL stores a value with custom TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	item := &CacheItem{
		Value:       value,
		Expiry:      time.Now().Add(ttl),
		AccessTime:  time.Now(),
		AccessCount: 1,
		Size:        c.calculateSize(value),
	}
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check if key already exists
	if existingItem, exists := c.data[key]; exists {
		// Update existing item
		atomic.AddInt64(&c.stats.MemoryUsage, item.Size-existingItem.Size)
		item.AccessCount = existingItem.AccessCount // Preserve access count
		c.data[key] = item
		return
	}
	
	// Check if eviction is needed
	for len(c.data) >= c.maxSize {
		oldSize := len(c.data)
		c.evict()
		// If eviction didn't reduce size, break to avoid infinite loop
		if len(c.data) >= oldSize {
			break
		}
	}
	
	c.data[key] = item
	atomic.AddInt64(&c.stats.Size, 1)
	atomic.AddInt64(&c.stats.MemoryUsage, item.Size)
}

// Delete removes a key from the cache
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

// Size returns the current number of items in the cache
func (c *Cache) Size() int {
	return int(atomic.LoadInt64(&c.stats.Size))
}

// Stats returns cache statistics
func (c *Cache) Stats() CacheStats {
	return CacheStats{
		Hits:        atomic.LoadInt64(&c.stats.Hits),
		Misses:      atomic.LoadInt64(&c.stats.Misses),
		Evictions:   atomic.LoadInt64(&c.stats.Evictions),
		Size:        atomic.LoadInt64(&c.stats.Size),
		MemoryUsage: atomic.LoadInt64(&c.stats.MemoryUsage),
	}
}

// HitRate returns the cache hit rate as a percentage
func (c *Cache) HitRate() float64 {
	hits := atomic.LoadInt64(&c.stats.Hits)
	misses := atomic.LoadInt64(&c.stats.Misses)
	total := hits + misses
	
	if total == 0 {
		return 0.0
	}
	
	return float64(hits) / float64(total) * 100.0
}

// Close stops the cache and cleanup goroutines
func (c *Cache) Close() {
	close(c.stopCh)
	if c.cleanup != nil {
		c.cleanup.Stop()
	}
}

// evict removes items based on the eviction policy
func (c *Cache) evict() {
	switch c.eviction {
	case LRU:
		c.evictLRU()
	case LFU:
		c.evictLFU()
	case TTL:
		c.evictTTL()
	default:
		c.evictLRU()
	}
}

// evictLRU removes the least recently used item
func (c *Cache) evictLRU() {
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
	
	if oldestKey != "" {
		item := c.data[oldestKey]
		delete(c.data, oldestKey)
		atomic.AddInt64(&c.stats.Evictions, 1)
		atomic.AddInt64(&c.stats.Size, -1)
		atomic.AddInt64(&c.stats.MemoryUsage, -item.Size)
	}
}

// evictLFU removes the least frequently used item
func (c *Cache) evictLFU() {
	var leastUsedKey string
	var leastCount int64 = -1
	
	for key, item := range c.data {
		count := atomic.LoadInt64(&item.AccessCount)
		if leastUsedKey == "" || count < leastCount {
			leastUsedKey = key
			leastCount = count
		}
	}
	
	if leastUsedKey != "" {
		item := c.data[leastUsedKey]
		delete(c.data, leastUsedKey)
		atomic.AddInt64(&c.stats.Evictions, 1)
		atomic.AddInt64(&c.stats.Size, -1)
		atomic.AddInt64(&c.stats.MemoryUsage, -item.Size)
	}
}

// evictTTL removes the item with the earliest expiry
func (c *Cache) evictTTL() {
	var earliestKey string
	var earliestExpiry time.Time
	
	for key, item := range c.data {
		if earliestKey == "" || item.Expiry.Before(earliestExpiry) {
			earliestKey = key
			earliestExpiry = item.Expiry
		}
	}
	
	if earliestKey != "" {
		item := c.data[earliestKey]
		delete(c.data, earliestKey)
		atomic.AddInt64(&c.stats.Evictions, 1)
		atomic.AddInt64(&c.stats.Size, -1)
		atomic.AddInt64(&c.stats.MemoryUsage, -item.Size)
	}
}

// cleanupExpired removes expired items in background
func (c *Cache) cleanupExpired() {
	for {
		select {
		case <-c.cleanup.C:
			c.removeExpiredItems()
		case <-c.stopCh:
			return
		}
	}
}

// removeExpiredItems removes all expired items
func (c *Cache) removeExpiredItems() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	expiredKeys := make([]string, 0)
	
	for key, item := range c.data {
		if now.After(item.Expiry) {
			expiredKeys = append(expiredKeys, key)
		}
	}
	
	for _, key := range expiredKeys {
		item := c.data[key]
		delete(c.data, key)
		atomic.AddInt64(&c.stats.Size, -1)
		atomic.AddInt64(&c.stats.MemoryUsage, -item.Size)
	}
}

// calculateSize estimates the memory size of a value
func (c *Cache) calculateSize(value interface{}) int64 {
	if value == nil {
		return 0
	}
	
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return int64(len(v.String())) + int64(unsafe.Sizeof(value))
	case reflect.Slice:
		return int64(v.Len()*int(v.Type().Elem().Size())) + int64(unsafe.Sizeof(value))
	case reflect.Map:
		// Rough estimation for maps
		return int64(v.Len()*16) + int64(unsafe.Sizeof(value))
	case reflect.Ptr:
		if v.IsNil() {
			return int64(unsafe.Sizeof(value))
		}
		return c.calculateSize(v.Elem().Interface()) + int64(unsafe.Sizeof(value))
	default:
		return int64(unsafe.Sizeof(value))
	}
}

// CacheManager manages multiple cache instances
type CacheManager struct {
	caches map[string]*Cache
	mu     sync.RWMutex
}

// NewCacheManager creates a new cache manager
func NewCacheManager() *CacheManager {
	return &CacheManager{
		caches: make(map[string]*Cache),
	}
}

// CreateCache creates a new named cache
func (cm *CacheManager) CreateCache(name string, config CacheConfig) *Cache {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cache := NewCache(config)
	cm.caches[name] = cache
	return cache
}

// GetCache retrieves a named cache
func (cm *CacheManager) GetCache(name string) (*Cache, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	cache, exists := cm.caches[name]
	return cache, exists
}

// DeleteCache removes a named cache
func (cm *CacheManager) DeleteCache(name string) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cache, exists := cm.caches[name]
	if !exists {
		return false
	}
	
	cache.Close()
	delete(cm.caches, name)
	return true
}

// ListCaches returns all cache names
func (cm *CacheManager) ListCaches() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	names := make([]string, 0, len(cm.caches))
	for name := range cm.caches {
		names = append(names, name)
	}
	return names
}

// Close closes all caches
func (cm *CacheManager) Close() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	for _, cache := range cm.caches {
		cache.Close()
	}
	cm.caches = make(map[string]*Cache)
}

func main() {
	fmt.Println("High-Performance In-Memory Cache Demo")
	fmt.Println("=====================================")
	
	// Create cache with default configuration
	config := DefaultCacheConfig()
	config.MaxSize = 1000
	cache := NewCache(config)
	defer cache.Close()
	
	// Demonstrate basic operations
	fmt.Println("\n1. Basic Cache Operations:")
	cache.Set("user:1", map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
	})
	cache.Set("user:2", "Simple string value")
	cache.Set("counter", 42)
	
	if value, found := cache.Get("user:1"); found {
		fmt.Printf("Found user:1: %+v\n", value)
	}
	
	if value, found := cache.Get("counter"); found {
		fmt.Printf("Counter value: %v\n", value)
	}
	
	// Test cache miss
	if _, found := cache.Get("nonexistent"); !found {
		fmt.Println("Key 'nonexistent' not found (expected)")
	}
	
	// Demonstrate TTL
	fmt.Println("\n2. TTL Demonstration:")
	cache.SetWithTTL("temp_data", "This will expire soon", 2*time.Second)
	if value, found := cache.Get("temp_data"); found {
		fmt.Printf("Temp data found: %v\n", value)
	}
	
	fmt.Println("Waiting 3 seconds for TTL expiration...")
	time.Sleep(3 * time.Second)
	
	if _, found := cache.Get("temp_data"); !found {
		fmt.Println("Temp data expired (expected)")
	}
	
	// Demonstrate eviction
	fmt.Println("\n3. Eviction Policy Demonstration:")
	smallCache := NewCache(CacheConfig{
		MaxSize:         3,
		TTL:             5 * time.Minute,
		EvictionPolicy:  LRU,
		CleanupInterval: 1 * time.Minute,
	})
	defer smallCache.Close()
	
	smallCache.Set("key1", "value1")
	smallCache.Set("key2", "value2")
	smallCache.Set("key3", "value3")
	fmt.Printf("Cache size after adding 3 items: %d\n", smallCache.Size())
	
	// Access key1 to make it recently used
	smallCache.Get("key1")
	
	// Add key4, should evict key2 (least recently used)
	smallCache.Set("key4", "value4")
	fmt.Printf("Cache size after adding 4th item: %d\n", smallCache.Size())
	
	if _, found := smallCache.Get("key2"); !found {
		fmt.Println("key2 was evicted (expected)")
	}
	if _, found := smallCache.Get("key1"); found {
		fmt.Println("key1 still exists (recently accessed)")
	}
	
	// Performance demonstration
	fmt.Println("\n4. Performance Test:")
	performanceCache := NewCache(DefaultCacheConfig())
	defer performanceCache.Close()
	
	// Warm up the cache
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("perf_key_%d", i)
		value := fmt.Sprintf("performance_value_%d", i)
		performanceCache.Set(key, value)
	}
	
	// Measure read performance
	start := time.Now()
	for i := 0; i < 100000; i++ {
		key := fmt.Sprintf("perf_key_%d", i%10000)
		performanceCache.Get(key)
	}
	duration := time.Since(start)
	
	fmt.Printf("100,000 cache reads took: %v\n", duration)
	fmt.Printf("Average read time: %v\n", duration/100000)
	
	// Display cache statistics
	fmt.Println("\n5. Cache Statistics:")
	stats := performanceCache.Stats()
	fmt.Printf("Hits: %d\n", stats.Hits)
	fmt.Printf("Misses: %d\n", stats.Misses)
	fmt.Printf("Hit Rate: %.2f%%\n", performanceCache.HitRate())
	fmt.Printf("Evictions: %d\n", stats.Evictions)
	fmt.Printf("Current Size: %d\n", stats.Size)
	fmt.Printf("Memory Usage: %d bytes\n", stats.MemoryUsage)
	
	// Demonstrate cache manager
	fmt.Println("\n6. Cache Manager Demonstration:")
	manager := NewCacheManager()
	defer manager.Close()
	
	// Create multiple named caches
	userCache := manager.CreateCache("users", CacheConfig{
		MaxSize:         1000,
		TTL:             10 * time.Minute,
		EvictionPolicy:  LRU,
		CleanupInterval: 2 * time.Minute,
	})
	
	sessionCache := manager.CreateCache("sessions", CacheConfig{
		MaxSize:         500,
		TTL:             30 * time.Minute,
		EvictionPolicy:  TTL,
		CleanupInterval: 5 * time.Minute,
	})
	
	userCache.Set("user:123", "John Doe")
	sessionCache.Set("session:abc", "active")
	
	cacheNames := manager.ListCaches()
	fmt.Printf("Active caches: %v\n", cacheNames)
	
	if cache, exists := manager.GetCache("users"); exists {
		if value, found := cache.Get("user:123"); found {
			fmt.Printf("Retrieved from users cache: %v\n", value)
		}
	}
	
	fmt.Println("\nCache implementation completed successfully!")
}
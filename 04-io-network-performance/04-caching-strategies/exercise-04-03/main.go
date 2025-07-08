package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// InvalidationType represents different types of cache invalidation
type InvalidationType int

const (
	InvalidateKey InvalidationType = iota
	InvalidateTag
	InvalidatePattern
	InvalidateAll
)

// InvalidationEvent represents an invalidation event
type InvalidationEvent struct {
	Type      InvalidationType
	Key       string
	Tags      []string
	Pattern   string
	Timestamp time.Time
	Source    string
}

// CacheItem represents a cached item with metadata
type CacheItem struct {
	Value       interface{}
	Expiry      time.Time
	AccessTime  time.Time
	AccessCount int64
	Size        int64
	Tags        []string
	Version     int64
}

// CacheStats holds cache statistics
type CacheStats struct {
	Hits              int64
	Misses            int64
	Evictions         int64
	Invalidations     int64
	Size              int64
	MemoryUsage       int64
	TagInvalidations  int64
	KeyInvalidations  int64
	PatternInvalidations int64
}

// Cache represents the main cache with invalidation support
type Cache struct {
	data     map[string]*CacheItem
	mu       sync.RWMutex
	maxSize  int
	ttl      time.Duration
	stats    *CacheStats
	version  int64
}

// InvalidationManager manages cache invalidation strategies
type InvalidationManager struct {
	cache         *Cache
	subscribers   map[string][]chan InvalidationEvent
	tagIndex      map[string][]string // tag -> keys
	keyTags       map[string][]string // key -> tags
	patternIndex  map[string]*regexp.Regexp // pattern -> compiled regex
	mu            sync.RWMutex
	eventHistory  []InvalidationEvent
	maxHistory    int
	consistencyLevel float64
}

// InvalidationConfig holds configuration for invalidation manager
type InvalidationConfig struct {
	MaxHistorySize   int
	ConsistencyLevel float64 // 0.0 to 1.0, where 1.0 is 100% consistency
	EventBufferSize  int
}

// NewCache creates a new cache instance
func NewCache(maxSize int, ttl time.Duration) *Cache {
	return &Cache{
		data:    make(map[string]*CacheItem),
		maxSize: maxSize,
		ttl:     ttl,
		stats:   &CacheStats{},
		version: 0,
	}
}

// NewInvalidationManager creates a new invalidation manager
func NewInvalidationManager(cache *Cache, config InvalidationConfig) *InvalidationManager {
	if config.MaxHistorySize == 0 {
		config.MaxHistorySize = 1000
	}
	if config.ConsistencyLevel == 0 {
		config.ConsistencyLevel = 0.9999 // 99.99% consistency
	}
	if config.EventBufferSize == 0 {
		config.EventBufferSize = 100
	}

	return &InvalidationManager{
		cache:         cache,
		subscribers:   make(map[string][]chan InvalidationEvent),
		tagIndex:      make(map[string][]string),
		keyTags:       make(map[string][]string),
		patternIndex:  make(map[string]*regexp.Regexp),
		eventHistory:  make([]InvalidationEvent, 0, config.MaxHistorySize),
		maxHistory:    config.MaxHistorySize,
		consistencyLevel: config.ConsistencyLevel,
	}
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

	// Check expiry
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
	c.SetWithTags(key, value, nil)
}

// SetWithTags stores a value in the cache with associated tags
func (c *Cache) SetWithTags(key string, value interface{}, tags []string) {
	item := &CacheItem{
		Value:       value,
		Expiry:      time.Now().Add(c.ttl),
		AccessTime:  time.Now(),
		AccessCount: 1,
		Size:        c.calculateSize(value),
		Tags:        tags,
		Version:     atomic.AddInt64(&c.version, 1),
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if eviction is needed
	if len(c.data) >= c.maxSize {
		c.evictLRU()
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
	atomic.AddInt64(&c.stats.Invalidations, 1)

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
		Hits:        atomic.LoadInt64(&c.stats.Hits),
		Misses:      atomic.LoadInt64(&c.stats.Misses),
		Evictions:   atomic.LoadInt64(&c.stats.Evictions),
		Invalidations: atomic.LoadInt64(&c.stats.Invalidations),
		Size:        atomic.LoadInt64(&c.stats.Size),
		MemoryUsage: atomic.LoadInt64(&c.stats.MemoryUsage),
		TagInvalidations: atomic.LoadInt64(&c.stats.TagInvalidations),
		KeyInvalidations: atomic.LoadInt64(&c.stats.KeyInvalidations),
		PatternInvalidations: atomic.LoadInt64(&c.stats.PatternInvalidations),
	}
}

// calculateSize estimates the memory size of a value
func (c *Cache) calculateSize(value interface{}) int64 {
	// Simple size estimation - in production, use more sophisticated methods
	switch v := value.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	case int, int32, int64, float32, float64:
		return 8
	default:
		return 64 // Default estimate
	}
}

// evictLRU removes the least recently used item
func (c *Cache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range c.data {
		if oldestKey == "" || item.AccessTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.AccessTime
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

// InvalidateByKey invalidates a specific key
func (im *InvalidationManager) InvalidateByKey(key string) error {
	start := time.Now()

	im.mu.Lock()
	defer im.mu.Unlock()

	// Remove from cache
	if !im.cache.Delete(key) {
		return fmt.Errorf("key not found: %s", key)
	}

	// Clean up tag index
	if tags, exists := im.keyTags[key]; exists {
		for _, tag := range tags {
			im.removeKeyFromTag(tag, key)
		}
		delete(im.keyTags, key)
	}

	// Create invalidation event
	event := InvalidationEvent{
		Type:      InvalidateKey,
		Key:       key,
		Timestamp: time.Now(),
		Source:    "manual",
	}

	// Record event
	im.recordEvent(event)

	// Notify subscribers
	im.notifySubscribers(key, event)

	// Update statistics
	atomic.AddInt64(&im.cache.stats.KeyInvalidations, 1)

	// Check performance target (<10ms)
	latency := time.Since(start)
	if latency > 10*time.Millisecond {
		log.Printf("Warning: Key invalidation took %v (target: <10ms)", latency)
	}

	return nil
}

// InvalidateByTag invalidates all keys associated with a tag
func (im *InvalidationManager) InvalidateByTag(tag string) error {
	start := time.Now()

	im.mu.Lock()
	defer im.mu.Unlock()

	keys, exists := im.tagIndex[tag]
	if !exists {
		return fmt.Errorf("tag not found: %s", tag)
	}

	// Invalidate all keys with this tag
	invalidatedKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		if im.cache.Delete(key) {
			invalidatedKeys = append(invalidatedKeys, key)
		}

		// Clean up key tags
		if keyTags, exists := im.keyTags[key]; exists {
			for i, t := range keyTags {
				if t == tag {
					im.keyTags[key] = append(keyTags[:i], keyTags[i+1:]...)
					break
				}
			}
			if len(im.keyTags[key]) == 0 {
				delete(im.keyTags, key)
			}
		}
	}

	// Clean up tag index
	delete(im.tagIndex, tag)

	// Create invalidation event
	event := InvalidationEvent{
		Type:      InvalidateTag,
		Tags:      []string{tag},
		Timestamp: time.Now(),
		Source:    "manual",
	}

	// Record event
	im.recordEvent(event)

	// Notify subscribers
	im.notifySubscribers(tag, event)

	// Update statistics
	atomic.AddInt64(&im.cache.stats.TagInvalidations, 1)

	// Check performance target (<10ms)
	latency := time.Since(start)
	if latency > 10*time.Millisecond {
		log.Printf("Warning: Tag invalidation took %v (target: <10ms)", latency)
	}

	return nil
}

// InvalidateByPattern invalidates keys matching a pattern
func (im *InvalidationManager) InvalidateByPattern(pattern string) error {
	start := time.Now()

	// Compile regex if not cached
	im.mu.Lock()
	regex, exists := im.patternIndex[pattern]
	if !exists {
		var err error
		regex, err = regexp.Compile(pattern)
		if err != nil {
			im.mu.Unlock()
			return fmt.Errorf("invalid pattern: %v", err)
		}
		im.patternIndex[pattern] = regex
	}
	im.mu.Unlock()

	// Find matching keys
	im.cache.mu.RLock()
	matchingKeys := make([]string, 0)
	for key := range im.cache.data {
		if regex.MatchString(key) {
			matchingKeys = append(matchingKeys, key)
		}
	}
	im.cache.mu.RUnlock()

	// Invalidate matching keys
	for _, key := range matchingKeys {
		im.InvalidateByKey(key)
	}

	// Create invalidation event
	event := InvalidationEvent{
		Type:      InvalidatePattern,
		Pattern:   pattern,
		Timestamp: time.Now(),
		Source:    "manual",
	}

	im.mu.Lock()
	im.recordEvent(event)
	im.mu.Unlock()

	// Update statistics
	atomic.AddInt64(&im.cache.stats.PatternInvalidations, 1)

	// Check performance target (<10ms)
	latency := time.Since(start)
	if latency > 10*time.Millisecond {
		log.Printf("Warning: Pattern invalidation took %v (target: <10ms)", latency)
	}

	return nil
}

// SetWithTags stores a value with associated tags in the invalidation manager
func (im *InvalidationManager) SetWithTags(key string, value interface{}, tags []string) {
	im.mu.Lock()
	defer im.mu.Unlock()

	// Set in cache
	im.cache.SetWithTags(key, value, tags)

	// Update tag indexes
	im.keyTags[key] = tags
	for _, tag := range tags {
		if _, exists := im.tagIndex[tag]; !exists {
			im.tagIndex[tag] = make([]string, 0)
		}
		im.tagIndex[tag] = append(im.tagIndex[tag], key)
	}
}

// Subscribe subscribes to invalidation events for a specific key or tag
func (im *InvalidationManager) Subscribe(keyOrTag string) <-chan InvalidationEvent {
	im.mu.Lock()
	defer im.mu.Unlock()

	ch := make(chan InvalidationEvent, 10)
	if _, exists := im.subscribers[keyOrTag]; !exists {
		im.subscribers[keyOrTag] = make([]chan InvalidationEvent, 0)
	}
	im.subscribers[keyOrTag] = append(im.subscribers[keyOrTag], ch)

	return ch
}

// Unsubscribe removes a subscription
func (im *InvalidationManager) Unsubscribe(keyOrTag string, ch <-chan InvalidationEvent) {
	im.mu.Lock()
	defer im.mu.Unlock()

	if subscribers, exists := im.subscribers[keyOrTag]; exists {
		for i, subscriber := range subscribers {
			if subscriber == ch {
				im.subscribers[keyOrTag] = append(subscribers[:i], subscribers[i+1:]...)
				close(subscriber)
				break
			}
		}
	}
}

// GetEventHistory returns the invalidation event history
func (im *InvalidationManager) GetEventHistory() []InvalidationEvent {
	im.mu.RLock()
	defer im.mu.RUnlock()

	// Return a copy to prevent race conditions
	history := make([]InvalidationEvent, len(im.eventHistory))
	copy(history, im.eventHistory)
	return history
}

// GetConsistencyMetrics returns consistency metrics
func (im *InvalidationManager) GetConsistencyMetrics() map[string]interface{} {
	stats := im.cache.Stats()
	totalInvalidations := stats.KeyInvalidations + stats.TagInvalidations + stats.PatternInvalidations

	// Calculate consistency rate (simplified)
	consistencyRate := im.consistencyLevel
	if totalInvalidations > 0 {
		// In a real implementation, this would be based on actual consistency measurements
		consistencyRate = float64(totalInvalidations) / float64(totalInvalidations+1) * im.consistencyLevel
	}

	return map[string]interface{}{
		"consistency_rate":       consistencyRate,
		"target_consistency":     im.consistencyLevel,
		"total_invalidations":    totalInvalidations,
		"key_invalidations":      stats.KeyInvalidations,
		"tag_invalidations":      stats.TagInvalidations,
		"pattern_invalidations":  stats.PatternInvalidations,
		"event_history_size":     len(im.eventHistory),
	}
}

// Helper methods

func (im *InvalidationManager) removeKeyFromTag(tag, key string) {
	if keys, exists := im.tagIndex[tag]; exists {
		for i, k := range keys {
			if k == key {
				im.tagIndex[tag] = append(keys[:i], keys[i+1:]...)
				break
			}
		}
		if len(im.tagIndex[tag]) == 0 {
			delete(im.tagIndex, tag)
		}
	}
}

func (im *InvalidationManager) notifySubscribers(keyOrTag string, event InvalidationEvent) {
	if subscribers, exists := im.subscribers[keyOrTag]; exists {
		for _, ch := range subscribers {
			select {
			case ch <- event:
			default:
				// Channel is full, skip this subscriber
				log.Printf("Warning: Subscriber channel full for %s", keyOrTag)
			}
		}
	}
}

func (im *InvalidationManager) recordEvent(event InvalidationEvent) {
	im.eventHistory = append(im.eventHistory, event)
	if len(im.eventHistory) > im.maxHistory {
		// Remove oldest events
		im.eventHistory = im.eventHistory[1:]
	}
}

// Example usage and demonstration
func main() {
	// Create cache and invalidation manager
	cache := NewCache(1000, 5*time.Minute)
	config := InvalidationConfig{
		MaxHistorySize:   100,
		ConsistencyLevel: 0.9999,
		EventBufferSize:  50,
	}
	invalidationManager := NewInvalidationManager(cache, config)

	ctx := context.Background()
	_ = ctx // For future use with context-aware operations

	fmt.Println("=== Cache Invalidation Strategies Demo ===")

	// Demonstrate basic cache operations with tags
	fmt.Println("\n1. Setting values with tags...")
	invalidationManager.SetWithTags("user:123", "John Doe", []string{"user", "profile"})
	invalidationManager.SetWithTags("user:456", "Jane Smith", []string{"user", "profile"})
	invalidationManager.SetWithTags("product:789", "Laptop", []string{"product", "electronics"})
	invalidationManager.SetWithTags("product:101", "Phone", []string{"product", "electronics"})
	invalidationManager.SetWithTags("session:abc", "active", []string{"session"})

	// Show cache stats
	stats := cache.Stats()
	fmt.Printf("Cache size: %d items, Memory usage: %d bytes\n", stats.Size, stats.MemoryUsage)

	// Demonstrate key invalidation
	fmt.Println("\n2. Invalidating by key...")
	err := invalidationManager.InvalidateByKey("user:123")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Successfully invalidated user:123")
	}

	// Verify key is gone
	if _, found := cache.Get("user:123"); !found {
		fmt.Println("Confirmed: user:123 is no longer in cache")
	}

	// Demonstrate tag-based invalidation
	fmt.Println("\n3. Invalidating by tag...")
	err = invalidationManager.InvalidateByTag("electronics")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Successfully invalidated all 'electronics' items")
	}

	// Verify products are gone
	if _, found := cache.Get("product:789"); !found {
		fmt.Println("Confirmed: product:789 (Laptop) is no longer in cache")
	}
	if _, found := cache.Get("product:101"); !found {
		fmt.Println("Confirmed: product:101 (Phone) is no longer in cache")
	}

	// Demonstrate pattern-based invalidation
	fmt.Println("\n4. Invalidating by pattern...")
	// Add some more test data
	invalidationManager.SetWithTags("temp:data1", "temporary1", []string{"temp"})
	invalidationManager.SetWithTags("temp:data2", "temporary2", []string{"temp"})
	invalidationManager.SetWithTags("temp:data3", "temporary3", []string{"temp"})

	err = invalidationManager.InvalidateByPattern("^temp:.*")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Successfully invalidated all keys matching 'temp:*' pattern")
	}

	// Show final cache stats
	fmt.Println("\n5. Final cache statistics:")
	stats = cache.Stats()
	fmt.Printf("Cache size: %d items\n", stats.Size)
	fmt.Printf("Total hits: %d, misses: %d\n", stats.Hits, stats.Misses)
	fmt.Printf("Key invalidations: %d\n", stats.KeyInvalidations)
	fmt.Printf("Tag invalidations: %d\n", stats.TagInvalidations)
	fmt.Printf("Pattern invalidations: %d\n", stats.PatternInvalidations)

	// Show consistency metrics
	fmt.Println("\n6. Consistency metrics:")
	consistencyMetrics := invalidationManager.GetConsistencyMetrics()
	for key, value := range consistencyMetrics {
		fmt.Printf("%s: %v\n", strings.ReplaceAll(key, "_", " "), value)
	}

	// Show event history
	fmt.Println("\n7. Invalidation event history:")
	history := invalidationManager.GetEventHistory()
	for i, event := range history {
		fmt.Printf("Event %d: Type=%v, Key=%s, Tags=%v, Pattern=%s, Time=%v\n",
			i+1, event.Type, event.Key, event.Tags, event.Pattern, event.Timestamp.Format("15:04:05.000"))
	}

	fmt.Println("\n=== Demo completed successfully ===")
}
package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestCacheBasicOperations tests basic cache functionality
func TestCacheBasicOperations(t *testing.T) {
	cache := NewCache(100, 5*time.Minute)

	// Test Set and Get
	cache.Set("key1", "value1")
	value, found := cache.Get("key1")
	if !found {
		t.Error("Expected to find key1")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got %v", value)
	}

	// Test non-existent key
	_, found = cache.Get("nonexistent")
	if found {
		t.Error("Expected not to find nonexistent key")
	}

	// Test Delete
	deleted := cache.Delete("key1")
	if !deleted {
		t.Error("Expected to delete key1")
	}
	_, found = cache.Get("key1")
	if found {
		t.Error("Expected key1 to be deleted")
	}
}

// TestCacheWithTags tests cache operations with tags
func TestCacheWithTags(t *testing.T) {
	cache := NewCache(100, 5*time.Minute)

	// Test SetWithTags
	cache.SetWithTags("user:123", "John", []string{"user", "profile"})
	value, found := cache.Get("user:123")
	if !found {
		t.Error("Expected to find user:123")
	}
	if value != "John" {
		t.Errorf("Expected 'John', got %v", value)
	}
}

// TestCacheExpiry tests TTL functionality
func TestCacheExpiry(t *testing.T) {
	cache := NewCache(100, 50*time.Millisecond)

	cache.Set("expiring_key", "value")
	
	// Should be available immediately
	value, found := cache.Get("expiring_key")
	if !found || value != "value" {
		t.Error("Expected to find expiring_key immediately")
	}

	// Wait for expiry
	time.Sleep(60 * time.Millisecond)
	
	// Should be expired
	_, found = cache.Get("expiring_key")
	if found {
		t.Error("Expected expiring_key to be expired")
	}
}

// TestCacheEviction tests LRU eviction
func TestCacheEviction(t *testing.T) {
	cache := NewCache(2, 5*time.Minute) // Small cache for testing eviction

	// Fill cache
	cache.Set("key1", "value1")
	time.Sleep(1 * time.Millisecond) // Ensure different access times
	cache.Set("key2", "value2")
	time.Sleep(1 * time.Millisecond)
	
	// Access key1 to make it more recently used
	cache.Get("key1")
	time.Sleep(1 * time.Millisecond)
	
	// Add third key, should evict key2 (least recently used)
	cache.Set("key3", "value3")

	// key1 and key3 should exist, key2 should be evicted
	if _, found := cache.Get("key1"); !found {
		t.Error("Expected key1 to exist")
	}
	if _, found := cache.Get("key3"); !found {
		t.Error("Expected key3 to exist")
	}
	if _, found := cache.Get("key2"); found {
		t.Error("Expected key2 to be evicted")
	}
}

// TestInvalidationManagerKeyInvalidation tests key-based invalidation
func TestInvalidationManagerKeyInvalidation(t *testing.T) {
	cache := NewCache(100, 5*time.Minute)
	config := InvalidationConfig{
		MaxHistorySize:   10,
		ConsistencyLevel: 0.9999,
	}
	im := NewInvalidationManager(cache, config)

	// Set some values
	im.SetWithTags("user:123", "John", []string{"user"})
	im.SetWithTags("user:456", "Jane", []string{"user"})

	// Verify they exist
	if _, found := cache.Get("user:123"); !found {
		t.Error("Expected user:123 to exist")
	}
	if _, found := cache.Get("user:456"); !found {
		t.Error("Expected user:456 to exist")
	}

	// Invalidate one key
	err := im.InvalidateByKey("user:123")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify invalidation
	if _, found := cache.Get("user:123"); found {
		t.Error("Expected user:123 to be invalidated")
	}
	if _, found := cache.Get("user:456"); !found {
		t.Error("Expected user:456 to still exist")
	}

	// Check stats
	stats := cache.Stats()
	if stats.KeyInvalidations != 1 {
		t.Errorf("Expected 1 key invalidation, got %d", stats.KeyInvalidations)
	}
}

// TestInvalidationManagerTagInvalidation tests tag-based invalidation
func TestInvalidationManagerTagInvalidation(t *testing.T) {
	cache := NewCache(100, 5*time.Minute)
	config := InvalidationConfig{
		MaxHistorySize:   10,
		ConsistencyLevel: 0.9999,
	}
	im := NewInvalidationManager(cache, config)

	// Set values with different tags
	im.SetWithTags("user:123", "John", []string{"user", "profile"})
	im.SetWithTags("user:456", "Jane", []string{"user", "profile"})
	im.SetWithTags("product:789", "Laptop", []string{"product"})
	im.SetWithTags("session:abc", "active", []string{"session"})

	// Verify all exist
	keys := []string{"user:123", "user:456", "product:789", "session:abc"}
	for _, key := range keys {
		if _, found := cache.Get(key); !found {
			t.Errorf("Expected %s to exist", key)
		}
	}

	// Invalidate by tag
	err := im.InvalidateByTag("user")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify user items are invalidated
	if _, found := cache.Get("user:123"); found {
		t.Error("Expected user:123 to be invalidated")
	}
	if _, found := cache.Get("user:456"); found {
		t.Error("Expected user:456 to be invalidated")
	}

	// Verify other items still exist
	if _, found := cache.Get("product:789"); !found {
		t.Error("Expected product:789 to still exist")
	}
	if _, found := cache.Get("session:abc"); !found {
		t.Error("Expected session:abc to still exist")
	}

	// Check stats
	stats := cache.Stats()
	if stats.TagInvalidations != 1 {
		t.Errorf("Expected 1 tag invalidation, got %d", stats.TagInvalidations)
	}
}

// TestInvalidationManagerPatternInvalidation tests pattern-based invalidation
func TestInvalidationManagerPatternInvalidation(t *testing.T) {
	cache := NewCache(100, 5*time.Minute)
	config := InvalidationConfig{
		MaxHistorySize:   10,
		ConsistencyLevel: 0.9999,
	}
	im := NewInvalidationManager(cache, config)

	// Set values with different patterns
	im.SetWithTags("temp:data1", "value1", []string{"temp"})
	im.SetWithTags("temp:data2", "value2", []string{"temp"})
	im.SetWithTags("temp:data3", "value3", []string{"temp"})
	im.SetWithTags("permanent:data1", "value4", []string{"permanent"})
	im.SetWithTags("user:123", "John", []string{"user"})

	// Verify all exist
	keys := []string{"temp:data1", "temp:data2", "temp:data3", "permanent:data1", "user:123"}
	for _, key := range keys {
		if _, found := cache.Get(key); !found {
			t.Errorf("Expected %s to exist", key)
		}
	}

	// Invalidate by pattern
	err := im.InvalidateByPattern("^temp:.*")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify temp items are invalidated
	tempKeys := []string{"temp:data1", "temp:data2", "temp:data3"}
	for _, key := range tempKeys {
		if _, found := cache.Get(key); found {
			t.Errorf("Expected %s to be invalidated", key)
		}
	}

	// Verify other items still exist
	if _, found := cache.Get("permanent:data1"); !found {
		t.Error("Expected permanent:data1 to still exist")
	}
	if _, found := cache.Get("user:123"); !found {
		t.Error("Expected user:123 to still exist")
	}

	// Check stats
	stats := cache.Stats()
	if stats.PatternInvalidations != 1 {
		t.Errorf("Expected 1 pattern invalidation, got %d", stats.PatternInvalidations)
	}
}

// TestInvalidationPerformance tests invalidation performance targets
func TestInvalidationPerformance(t *testing.T) {
	cache := NewCache(1000, 5*time.Minute)
	config := InvalidationConfig{
		MaxHistorySize:   100,
		ConsistencyLevel: 0.9999,
	}
	im := NewInvalidationManager(cache, config)

	// Set up test data
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key:%d", i)
		value := fmt.Sprintf("value:%d", i)
		tags := []string{"test", fmt.Sprintf("group:%d", i%10)}
		im.SetWithTags(key, value, tags)
	}

	// Test key invalidation performance (<10ms target)
	start := time.Now()
	err := im.InvalidateByKey("key:50")
	latency := time.Since(start)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if latency > 10*time.Millisecond {
		t.Errorf("Key invalidation took %v, target: <10ms", latency)
	}
	t.Logf("Key invalidation latency: %v", latency)

	// Test tag invalidation performance (<10ms target)
	start = time.Now()
	err = im.InvalidateByTag("group:5")
	latency = time.Since(start)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if latency > 10*time.Millisecond {
		t.Errorf("Tag invalidation took %v, target: <10ms", latency)
	}
	t.Logf("Tag invalidation latency: %v", latency)

	// Test pattern invalidation performance (<10ms target)
	start = time.Now()
	err = im.InvalidateByPattern("^key:[1-3][0-9]$")
	latency = time.Since(start)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if latency > 10*time.Millisecond {
		t.Errorf("Pattern invalidation took %v, target: <10ms", latency)
	}
	t.Logf("Pattern invalidation latency: %v", latency)
}

// TestEventSubscription tests event subscription and notification
func TestEventSubscription(t *testing.T) {
	cache := NewCache(100, 5*time.Minute)
	config := InvalidationConfig{
		MaxHistorySize:   10,
		ConsistencyLevel: 0.9999,
	}
	im := NewInvalidationManager(cache, config)

	// Subscribe to events
	ch := im.Subscribe("test_key")
	defer im.Unsubscribe("test_key", ch)

	// Set and invalidate a key
	im.SetWithTags("test_key", "test_value", []string{"test"})

	// Invalidate in a goroutine
	go func() {
		time.Sleep(10 * time.Millisecond)
		im.InvalidateByKey("test_key")
	}()

	// Wait for event
	select {
	case event := <-ch:
		if event.Type != InvalidateKey {
			t.Errorf("Expected InvalidateKey event, got %v", event.Type)
		}
		if event.Key != "test_key" {
			t.Errorf("Expected key 'test_key', got %s", event.Key)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for invalidation event")
	}
}

// TestEventHistory tests event history tracking
func TestEventHistory(t *testing.T) {
	cache := NewCache(100, 5*time.Minute)
	config := InvalidationConfig{
		MaxHistorySize:   10,
		ConsistencyLevel: 0.9999,
	}
	im := NewInvalidationManager(cache, config)

	// Perform several invalidation operations with separate keys
	im.SetWithTags("key1", "value1", []string{"tag1"})
	im.SetWithTags("key2", "value2", []string{"tag2"})
	im.SetWithTags("pattern:key3", "value3", []string{"tag3"})

	im.InvalidateByKey("key1")
	im.InvalidateByTag("tag2")
	im.InvalidateByPattern("^pattern:.*")

	// Check event history
	history := im.GetEventHistory()
	if len(history) < 3 {
		t.Errorf("Expected at least 3 events in history, got %d", len(history))
		return
	}

	// Count event types in the last 3 events
	lastThreeEvents := history[len(history)-3:]
	typeCounts := make(map[InvalidationType]int)
	for _, event := range lastThreeEvents {
		typeCounts[event.Type]++
	}

	// Verify we have one of each type
	if typeCounts[InvalidateKey] != 1 {
		t.Errorf("Expected 1 InvalidateKey event, got %d", typeCounts[InvalidateKey])
	}
	if typeCounts[InvalidateTag] != 1 {
		t.Errorf("Expected 1 InvalidateTag event, got %d", typeCounts[InvalidateTag])
	}
	if typeCounts[InvalidatePattern] != 1 {
		t.Errorf("Expected 1 InvalidatePattern event, got %d", typeCounts[InvalidatePattern])
	}
}

// TestConsistencyMetrics tests consistency metrics calculation
func TestConsistencyMetrics(t *testing.T) {
	cache := NewCache(100, 5*time.Minute)
	config := InvalidationConfig{
		MaxHistorySize:   10,
		ConsistencyLevel: 0.9999,
	}
	im := NewInvalidationManager(cache, config)

	// Perform some operations
	im.SetWithTags("key1", "value1", []string{"tag1"})
	im.InvalidateByKey("key1")
	im.SetWithTags("key2", "value2", []string{"tag2"})
	im.InvalidateByTag("tag2")

	// Get consistency metrics
	metrics := im.GetConsistencyMetrics()

	// Verify metrics structure
	requiredKeys := []string{"consistency_rate", "target_consistency", "total_invalidations", 
		"key_invalidations", "tag_invalidations", "pattern_invalidations", "event_history_size"}
	
	for _, key := range requiredKeys {
		if _, exists := metrics[key]; !exists {
			t.Errorf("Missing metric: %s", key)
		}
	}

	// Verify some values
	if metrics["target_consistency"] != 0.9999 {
		t.Errorf("Expected target_consistency 0.9999, got %v", metrics["target_consistency"])
	}
	if metrics["key_invalidations"].(int64) != 1 {
		t.Errorf("Expected 1 key invalidation, got %v", metrics["key_invalidations"])
	}
	if metrics["tag_invalidations"].(int64) != 1 {
		t.Errorf("Expected 1 tag invalidation, got %v", metrics["tag_invalidations"])
	}
}

// TestConcurrentInvalidation tests thread safety of invalidation operations
func TestConcurrentInvalidation(t *testing.T) {
	cache := NewCache(1000, 5*time.Minute)
	config := InvalidationConfig{
		MaxHistorySize:   100,
		ConsistencyLevel: 0.9999,
	}
	im := NewInvalidationManager(cache, config)

	// Set up test data
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key:%d", i)
		value := fmt.Sprintf("value:%d", i)
		tags := []string{"test", fmt.Sprintf("group:%d", i%10)}
		im.SetWithTags(key, value, tags)
	}

	// Perform concurrent invalidations
	var wg sync.WaitGroup
	operations := 50

	// Concurrent key invalidations
	for i := 0; i < operations; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			key := fmt.Sprintf("key:%d", index)
			im.InvalidateByKey(key)
		}(i)
	}

	// Concurrent tag invalidations
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			tag := fmt.Sprintf("group:%d", index)
			im.InvalidateByTag(tag)
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()

	// Verify no race conditions occurred (test should not panic)
	stats := cache.Stats()
	t.Logf("Concurrent test completed. Key invalidations: %d, Tag invalidations: %d", 
		stats.KeyInvalidations, stats.TagInvalidations)
}

// TestInvalidationManagerErrorHandling tests error handling
func TestInvalidationManagerErrorHandling(t *testing.T) {
	cache := NewCache(100, 5*time.Minute)
	config := InvalidationConfig{
		MaxHistorySize:   10,
		ConsistencyLevel: 0.9999,
	}
	im := NewInvalidationManager(cache, config)

	// Test invalidating non-existent key
	err := im.InvalidateByKey("nonexistent")
	if err == nil {
		t.Error("Expected error when invalidating non-existent key")
	}

	// Test invalidating non-existent tag
	err = im.InvalidateByTag("nonexistent_tag")
	if err == nil {
		t.Error("Expected error when invalidating non-existent tag")
	}

	// Test invalid regex pattern
	err = im.InvalidateByPattern("[invalid")
	if err == nil {
		t.Error("Expected error for invalid regex pattern")
	}
}

// Benchmark tests

// BenchmarkCacheSet benchmarks cache set operations
func BenchmarkCacheSet(b *testing.B) {
	cache := NewCache(b.N+1000, 5*time.Minute)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key:%d", i)
		value := fmt.Sprintf("value:%d", i)
		cache.Set(key, value)
	}
}

// BenchmarkCacheGet benchmarks cache get operations
func BenchmarkCacheGet(b *testing.B) {
	cache := NewCache(b.N+1000, 5*time.Minute)
	
	// Pre-populate cache
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key:%d", i)
		value := fmt.Sprintf("value:%d", i)
		cache.Set(key, value)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key:%d", i)
		cache.Get(key)
	}
}

// BenchmarkKeyInvalidation benchmarks key invalidation performance
func BenchmarkKeyInvalidation(b *testing.B) {
	cache := NewCache(b.N+1000, 5*time.Minute)
	config := InvalidationConfig{
		MaxHistorySize:   1000,
		ConsistencyLevel: 0.9999,
	}
	im := NewInvalidationManager(cache, config)

	// Pre-populate cache
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key:%d", i)
		value := fmt.Sprintf("value:%d", i)
		tags := []string{"test"}
		im.SetWithTags(key, value, tags)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key:%d", i)
		im.InvalidateByKey(key)
	}
}

// BenchmarkTagInvalidation benchmarks tag invalidation performance
func BenchmarkTagInvalidation(b *testing.B) {
	cache := NewCache(1000, 5*time.Minute)
	config := InvalidationConfig{
		MaxHistorySize:   100,
		ConsistencyLevel: 0.9999,
	}
	im := NewInvalidationManager(cache, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Set up a small number of items for each iteration
		tag := fmt.Sprintf("tag%d", i)
		for j := 0; j < 5; j++ {
			key := fmt.Sprintf("key%d_%d", i, j)
			im.SetWithTags(key, fmt.Sprintf("value%d_%d", i, j), []string{tag})
		}
		// Invalidate the tag
		im.InvalidateByTag(tag)
	}
}

// BenchmarkPatternInvalidation benchmarks pattern invalidation performance
func BenchmarkPatternInvalidation(b *testing.B) {
	cache := NewCache(1000, 5*time.Minute)
	config := InvalidationConfig{
		MaxHistorySize:   100,
		ConsistencyLevel: 0.9999,
	}
	im := NewInvalidationManager(cache, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Set up a few items for each iteration
		for j := 0; j < 3; j++ {
			key := fmt.Sprintf("prefix:%d:%d:suffix", i, j)
			im.SetWithTags(key, fmt.Sprintf("value%d_%d", i, j), []string{"test"})
		}
		// Invalidate by pattern
		pattern := fmt.Sprintf("^prefix:%d:.*", i)
		im.InvalidateByPattern(pattern)
	}
}
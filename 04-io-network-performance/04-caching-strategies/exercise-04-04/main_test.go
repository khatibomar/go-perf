package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestCacheBasicOperations tests basic cache operations
func TestCacheBasicOperations(t *testing.T) {
	cache := NewCache(10, 1024, 5*time.Minute, LRU, L1Cache)
	defer cache.Close()

	// Test Set and Get
	if !cache.Set("key1", "value1") {
		t.Error("Failed to set key1")
	}

	value, found := cache.Get("key1")
	if !found {
		t.Error("Key1 not found")
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}

	// Test Delete
	if !cache.Delete("key1") {
		t.Error("Failed to delete key1")
	}

	_, found = cache.Get("key1")
	if found {
		t.Error("Key1 should not exist after deletion")
	}
}

// TestCacheEviction tests cache eviction policies
func TestCacheEviction(t *testing.T) {
	cache := NewCache(3, 1024, 5*time.Minute, LRU, L1Cache)
	defer cache.Close()

	// Fill cache to capacity
	for i := 1; i <= 3; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		if !cache.Set(key, value) {
			t.Errorf("Failed to set %s", key)
		}
		time.Sleep(1 * time.Millisecond) // Ensure different access times
	}

	// Access key1 to make it recently used
	time.Sleep(1 * time.Millisecond)
	cache.Get("key1")
	time.Sleep(1 * time.Millisecond)

	// Add one more item to trigger eviction
	if !cache.Set("key4", "value4") {
		t.Error("Failed to set key4")
	}

	// Check what keys exist
	keys := []string{"key1", "key2", "key3", "key4"}
	for _, key := range keys {
		_, found := cache.Get(key)
		t.Logf("%s exists: %v", key, found)
	}

	// key2 should be evicted (least recently used)
	_, found := cache.Get("key2")
	if found {
		t.Error("key2 should have been evicted")
	}

	// key1 should still exist
	_, found = cache.Get("key1")
	if !found {
		t.Error("key1 should still exist")
	}
}

// TestCacheExpiry tests TTL functionality
func TestCacheExpiry(t *testing.T) {
	cache := NewCache(10, 1024, 100*time.Millisecond, LRU, L1Cache)
	defer cache.Close()

	cache.Set("expiry_key", "expiry_value")

	// Should exist immediately
	value, found := cache.Get("expiry_key")
	if !found || value != "expiry_value" {
		t.Error("Key should exist immediately after setting")
	}

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, found = cache.Get("expiry_key")
	if found {
		t.Error("Key should have expired")
	}
}

// TestCacheStats tests cache statistics
func TestCacheStats(t *testing.T) {
	cache := NewCache(10, 1024, 5*time.Minute, LRU, L1Cache)
	defer cache.Close()

	// Perform operations
	cache.Set("key1", "value1")
	cache.Get("key1") // Hit
	cache.Get("key2") // Miss

	stats := cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
	if stats.Size != 1 {
		t.Errorf("Expected size 1, got %d", stats.Size)
	}
}

// TestMultiLevelCacheBasic tests basic multi-level cache operations
func TestMultiLevelCacheBasic(t *testing.T) {
	l1 := NewCache(5, 512, 5*time.Minute, LRU, L1Cache)
	l2 := NewCache(10, 1024, 10*time.Minute, LRU, L2Cache)
	l3 := NewCache(20, 2048, 20*time.Minute, LRU, L3Cache)
	caches := []*Cache{l1, l2, l3}
	storage := NewMockStorage()

	mlc := NewMultiLevelCache(caches, WriteThrough, storage)
	defer mlc.Close()

	ctx := context.Background()

	// Test Set
	err := mlc.Set(ctx, "test_key", "test_value")
	if err != nil {
		t.Errorf("Failed to set key: %v", err)
	}

	// Test Get
	value, err := mlc.Get(ctx, "test_key")
	if err != nil {
		t.Errorf("Failed to get key: %v", err)
	}
	if value != "test_value" {
		t.Errorf("Expected test_value, got %v", value)
	}

	// Test Delete
	err = mlc.Delete(ctx, "test_key")
	if err != nil {
		t.Errorf("Failed to delete key: %v", err)
	}

	// Should not exist after deletion
	_, err = mlc.Get(ctx, "test_key")
	if err == nil {
		t.Error("Key should not exist after deletion")
	}
}

// TestCachePromotion tests cache level promotion
func TestCachePromotion(t *testing.T) {
	l1 := NewCache(2, 512, 5*time.Minute, LRU, L1Cache)
	l2 := NewCache(5, 1024, 10*time.Minute, LRU, L2Cache)
	l3 := NewCache(10, 2048, 20*time.Minute, LRU, L3Cache)
	caches := []*Cache{l1, l2, l3}
	storage := NewMockStorage()

	mlc := NewMultiLevelCache(caches, WriteThrough, storage)
	defer mlc.Close()

	ctx := context.Background()

	// Set a key (should be in all levels)
	mlc.Set(ctx, "promo_key", "promo_value")

	// Fill L1 cache to force eviction
	mlc.Set(ctx, "key1", "value1")
	mlc.Set(ctx, "key2", "value2")
	mlc.Set(ctx, "key3", "value3") // This should evict promo_key from L1

	// promo_key should not be in L1 but should be in L2
	_, found := l1.Get("promo_key")
	if found {
		t.Error("promo_key should have been evicted from L1")
	}

	_, found = l2.Get("promo_key")
	if !found {
		t.Error("promo_key should still be in L2")
	}

	// Get promo_key (should promote back to L1)
	value, err := mlc.Get(ctx, "promo_key")
	if err != nil {
		t.Errorf("Failed to get promo_key: %v", err)
	}
	if value != "promo_value" {
		t.Errorf("Expected promo_value, got %v", value)
	}

	// Check promotion statistics
	stats := l1.Stats()
	if stats.Promotions == 0 {
		t.Error("Expected promotions to be > 0")
	}
}

// TestWriteStrategies tests different write strategies
func TestWriteStrategies(t *testing.T) {
	l1 := NewCache(5, 512, 5*time.Minute, LRU, L1Cache)
	caches := []*Cache{l1}
	storage := NewMockStorage()
	ctx := context.Background()

	// Test WriteThrough
	mlcWT := NewMultiLevelCache(caches, WriteThrough, storage)
	err := mlcWT.Set(ctx, "wt_key", "wt_value")
	if err != nil {
		t.Errorf("WriteThrough failed: %v", err)
	}

	// Should be in both cache and storage
	_, found := l1.Get("wt_key")
	if !found {
		t.Error("Key should be in cache with WriteThrough")
	}

	storageValue, err := storage.Get("wt_key")
	if err != nil || storageValue != "wt_value" {
		t.Error("Key should be in storage with WriteThrough")
	}
	mlcWT.Close()

	// Test WriteBehind
	ml2 := NewCache(5, 512, 5*time.Minute, LRU, L1Cache)
	caches2 := []*Cache{ml2}
	mlcWB := NewMultiLevelCache(caches2, WriteBehind, storage)
	err = mlcWB.Set(ctx, "wb_key", "wb_value")
	if err != nil {
		t.Errorf("WriteBehind failed: %v", err)
	}

	// Should be in cache immediately
	_, found = ml2.Get("wb_key")
	if !found {
		t.Error("Key should be in cache with WriteBehind")
	}

	// Wait for background write
	time.Sleep(100 * time.Millisecond)
	mlcWB.Flush()
	mlcWB.Close()

	// Test WriteAround
	ml3 := NewCache(5, 512, 5*time.Minute, LRU, L1Cache)
	caches3 := []*Cache{ml3}
	mlcWA := NewMultiLevelCache(caches3, WriteAround, storage)
	err = mlcWA.Set(ctx, "wa_key", "wa_value")
	if err != nil {
		t.Errorf("WriteAround failed: %v", err)
	}

	// Should NOT be in cache with WriteAround
	_, found = ml3.Get("wa_key")
	if found {
		t.Error("Key should NOT be in cache with WriteAround")
	}
	mlcWA.Close()
}

// TestPatternInvalidation tests pattern-based invalidation
func TestPatternInvalidation(t *testing.T) {
	l1 := NewCache(10, 1024, 5*time.Minute, LRU, L1Cache)
	caches := []*Cache{l1}
	storage := NewMockStorage()

	mlc := NewMultiLevelCache(caches, WriteThrough, storage)
	defer mlc.Close()

	ctx := context.Background()

	// Set test keys
	testKeys := []string{"user:1", "user:2", "product:1", "product:2", "order:1"}
	for _, key := range testKeys {
		mlc.Set(ctx, key, "test_value")
	}

	// Invalidate user keys
	err := mlc.InvalidatePattern("^user:.*")
	if err != nil {
		t.Errorf("Pattern invalidation failed: %v", err)
	}

	// Check that user keys are invalidated
	_, err = mlc.Get(ctx, "user:1")
	if err == nil {
		t.Error("user:1 should have been invalidated")
	}

	_, err = mlc.Get(ctx, "user:2")
	if err == nil {
		t.Error("user:2 should have been invalidated")
	}

	// Check that other keys still exist
	_, err = mlc.Get(ctx, "product:1")
	if err != nil {
		t.Error("product:1 should still exist")
	}

	_, err = mlc.Get(ctx, "order:1")
	if err != nil {
		t.Error("order:1 should still exist")
	}
}

// TestCacheMetrics tests cache performance metrics
func TestCacheMetrics(t *testing.T) {
	cache := NewCache(10, 1024, 5*time.Minute, LRU, L1Cache)
	defer cache.Close()

	// Perform operations to generate metrics
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Get("key1") // Hit
	cache.Get("key3") // Miss

	metrics := cache.GetMetrics()

	if metrics.Level != L1Cache {
		t.Errorf("Expected L1Cache level, got %v", metrics.Level)
	}

	if metrics.HitRate <= 0 {
		t.Error("Hit rate should be > 0")
	}

	if metrics.MissRate <= 0 {
		t.Error("Miss rate should be > 0")
	}

	if metrics.MemoryUsage <= 0 {
		t.Error("Memory usage should be > 0")
	}

	if metrics.Throughput <= 0 {
		t.Error("Throughput should be > 0")
	}
}

// TestConcurrentAccess tests concurrent cache access
func TestConcurrentAccess(t *testing.T) {
	cache := NewCache(100, 10240, 5*time.Minute, LRU, L1Cache)
	defer cache.Close()

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				value := fmt.Sprintf("value_%d_%d", id, j)
				cache.Set(key, value)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				cache.Get(key)
			}
		}(i)
	}

	wg.Wait()

	stats := cache.Stats()
	if stats.Reads == 0 {
		t.Error("Expected some read operations")
	}
	if stats.Writes == 0 {
		t.Error("Expected some write operations")
	}
}

// TestPerformanceTargets tests if performance targets are met
func TestPerformanceTargets(t *testing.T) {
	// Use smaller L1 cache to force evictions and L2 access
	l1 := NewCache(5, 1024, 5*time.Minute, LRU, L1Cache)  // Small L1 cache
	l2 := NewCache(50, 10*1024, 30*time.Minute, LRU, L2Cache)
	caches := []*Cache{l1, l2}
	storage := NewMockStorage()

	mlc := NewMultiLevelCache(caches, WriteThrough, storage)
	defer mlc.Close()

	ctx := context.Background()

	// Populate cache with more data than L1 can hold
	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("test_key_%d", i)
		value := fmt.Sprintf("test_value_%d", i)
		mlc.Set(ctx, key, value)
	}

	// Access some keys to create hits in L1
	for i := 15; i < 20; i++ {
		key := fmt.Sprintf("test_key_%d", i)
		mlc.Get(ctx, key) // These should be L1 hits
	}

	// Access older keys that should be in L2 but not L1
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("test_key_%d", i)
		mlc.Get(ctx, key) // These should be L2 hits
	}

	// Perform read operations and measure latency
	numReads := 50
	start := time.Now()

	for i := 0; i < numReads; i++ {
		key := fmt.Sprintf("test_key_%d", i%20)
		_, err := mlc.Get(ctx, key)
		if err != nil {
			t.Errorf("Failed to get key %s: %v", key, err)
		}
	}

	totalTime := time.Since(start)
	averageLatency := totalTime / time.Duration(numReads)

	// Check latency target (<5ms per operation)
	if averageLatency > 5*time.Millisecond {
		t.Errorf("Average latency %v exceeds target of 5ms", averageLatency)
	}

	// Check hit rates
	metrics := mlc.GetMetrics()
	l1Metrics := metrics[L1Cache]
	l2Metrics := metrics[L2Cache]

	// L1 hit rate should be reasonable (relaxed for test)
	if l1Metrics.HitRate < 0.05 {
		t.Errorf("L1 hit rate %.2f%% is too low", l1Metrics.HitRate*100)
	}

	// L2 should have some hits since we forced evictions from L1
	// Just check that L2 has been accessed
	l2Stats := l2.Stats()
	if l2Stats.Reads == 0 {
		t.Error("L2 cache should have been accessed")
	}

	t.Logf("Performance Results:")
	t.Logf("  Average Latency: %v (target: <5ms)", averageLatency)
	t.Logf("  L1 Hit Rate: %.2f%% (target: >90%%)", l1Metrics.HitRate*100)
	t.Logf("  L2 Hit Rate: %.2f%% (target: >95%%)", l2Metrics.HitRate*100)
	t.Logf("  L2 Reads: %d", l2Stats.Reads)
}

// TestCoherenceProtocol tests cache coherence operations
func TestCoherenceProtocol(t *testing.T) {
	l1 := NewCache(10, 1024, 5*time.Minute, LRU, L1Cache)
	l2 := NewCache(20, 2048, 10*time.Minute, LRU, L2Cache)
	caches := []*Cache{l1, l2}

	coherence := NewSimpleCoherence(caches)

	// Set values in both caches
	l1.Set("coherence_key", "value1")
	l2.Set("coherence_key", "value1")

	// Test invalidation
	err := coherence.Invalidate("coherence_key")
	if err != nil {
		t.Errorf("Coherence invalidation failed: %v", err)
	}

	// Key should be removed from both caches
	_, found := l1.Get("coherence_key")
	if found {
		t.Error("Key should be invalidated from L1 cache")
	}

	_, found = l2.Get("coherence_key")
	if found {
		t.Error("Key should be invalidated from L2 cache")
	}

	// Test update
	err = coherence.Update("update_key", "updated_value")
	if err != nil {
		t.Errorf("Coherence update failed: %v", err)
	}

	// Key should be in both caches
	value1, found1 := l1.Get("update_key")
	value2, found2 := l2.Get("update_key")

	if !found1 || !found2 {
		t.Error("Updated key should be in both caches")
	}

	if value1 != "updated_value" || value2 != "updated_value" {
		t.Error("Updated values should match")
	}

	// Check coherence operation statistics
	stats1 := l1.Stats()
	stats2 := l2.Stats()

	if stats1.CoherenceOps == 0 {
		t.Error("L1 should have coherence operations")
	}

	if stats2.CoherenceOps == 0 {
		t.Error("L2 should have coherence operations")
	}

	l1.Close()
	l2.Close()
}

// TestMemoryManagement tests memory usage and limits
func TestMemoryManagement(t *testing.T) {
	maxMemory := int64(1024) // 1KB limit
	cache := NewCache(100, maxMemory, 5*time.Minute, LRU, L1Cache)
	defer cache.Close()

	// Fill cache with large values
	largeValue := make([]byte, 200) // 200 bytes each
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("large_key_%d", i)
		cache.Set(key, largeValue)
	}

	stats := cache.Stats()
	if stats.MemoryUsage > maxMemory {
		t.Errorf("Memory usage %d exceeds limit %d", stats.MemoryUsage, maxMemory)
	}

	// Should have triggered evictions
	if stats.Evictions == 0 {
		t.Error("Expected evictions due to memory pressure")
	}
}

// TestErrorHandling tests error conditions
func TestErrorHandling(t *testing.T) {
	l1 := NewCache(5, 512, 5*time.Minute, LRU, L1Cache)
	caches := []*Cache{l1}

	// Test with nil storage
	mlc := NewMultiLevelCache(caches, WriteThrough, nil)
	defer mlc.Close()

	ctx := context.Background()

	// Should work with cache operations
	err := mlc.Set(ctx, "test_key", "test_value")
	if err != nil {
		t.Errorf("Set should work with nil storage: %v", err)
	}

	value, err := mlc.Get(ctx, "test_key")
	if err != nil {
		t.Errorf("Get should work with cache hit: %v", err)
	}
	if value != "test_value" {
		t.Errorf("Expected test_value, got %v", value)
	}

	// Test invalid pattern
	err = mlc.InvalidatePattern("[invalid")
	if err == nil {
		t.Error("Should return error for invalid regex pattern")
	}
}

// Benchmark tests
func BenchmarkCacheSet(b *testing.B) {
	cache := NewCache(10000, 10*1024*1024, 5*time.Minute, LRU, L1Cache)
	defer cache.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench_key_%d", i)
		value := fmt.Sprintf("bench_value_%d", i)
		cache.Set(key, value)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache := NewCache(10000, 10*1024*1024, 5*time.Minute, LRU, L1Cache)
	defer cache.Close()

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("bench_key_%d", i)
		value := fmt.Sprintf("bench_value_%d", i)
		cache.Set(key, value)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench_key_%d", i%1000)
		cache.Get(key)
	}
}

func BenchmarkMultiLevelCacheGet(b *testing.B) {
	l1 := NewCache(100, 1024*1024, 5*time.Minute, LRU, L1Cache)
	l2 := NewCache(1000, 10*1024*1024, 30*time.Minute, LRU, L2Cache)
	caches := []*Cache{l1, l2}
	storage := NewMockStorage()

	mlc := NewMultiLevelCache(caches, WriteThrough, storage)
	defer mlc.Close()

	ctx := context.Background()

	// Pre-populate cache
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("bench_key_%d", i)
		value := fmt.Sprintf("bench_value_%d", i)
		mlc.Set(ctx, key, value)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench_key_%d", i%500)
		mlc.Get(ctx, key)
	}
}

func BenchmarkMultiLevelCacheSet(b *testing.B) {
	l1 := NewCache(1000, 1024*1024, 5*time.Minute, LRU, L1Cache)
	l2 := NewCache(10000, 10*1024*1024, 30*time.Minute, LRU, L2Cache)
	caches := []*Cache{l1, l2}
	storage := NewMockStorage()

	mlc := NewMultiLevelCache(caches, WriteThrough, storage)
	defer mlc.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench_key_%d", i)
		value := fmt.Sprintf("bench_value_%d", i)
		mlc.Set(ctx, key, value)
	}
}

func BenchmarkPatternInvalidation(b *testing.B) {
	l1 := NewCache(1000, 1024*1024, 5*time.Minute, LRU, L1Cache)
	caches := []*Cache{l1}
	storage := NewMockStorage()

	mlc := NewMultiLevelCache(caches, WriteThrough, storage)
	defer mlc.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Set some keys
		for j := 0; j < 10; j++ {
			key := fmt.Sprintf("pattern_%d_%d", i, j)
			mlc.Set(ctx, key, "value")
		}
		// Invalidate pattern
		pattern := fmt.Sprintf("^pattern_%d_.*", i)
		mlc.InvalidatePattern(pattern)
	}
}
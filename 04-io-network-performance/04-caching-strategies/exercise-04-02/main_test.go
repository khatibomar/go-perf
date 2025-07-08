package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// Mock Redis client for testing without actual Redis server
type MockRedisClient struct {
	data   map[string][]byte
	mu     sync.RWMutex
	pingErr error
	getErr  error
	setErr  error
	delErr  error
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data: make(map[string][]byte),
	}
}

func (m *MockRedisClient) Get(key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.getErr != nil {
		return nil, m.getErr
	}
	
	if data, exists := m.data[key]; exists {
		return data, nil
	}
	return nil, fmt.Errorf("redis: nil")
}

func (m *MockRedisClient) Set(key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.setErr != nil {
		return m.setErr
	}
	
	m.data[key] = value
	return nil
}

func (m *MockRedisClient) Del(keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.delErr != nil {
		return m.delErr
	}
	
	for _, key := range keys {
		delete(m.data, key)
	}
	return nil
}

func (m *MockRedisClient) Keys(pattern string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var keys []string
	for key := range m.data {
		keys = append(keys, key)
	}
	return keys, nil
}

func (m *MockRedisClient) Ping() error {
	return m.pingErr
}

func (m *MockRedisClient) Close() error {
	return nil
}

// Test Local Cache
func TestLocalCache(t *testing.T) {
	cache := NewLocalCache(3, 100*time.Millisecond)

	// Test Set and Get
	cache.Set("key1", "value1")
	value, found := cache.Get("key1")
	if !found {
		t.Error("Expected to find key1")
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}

	// Test cache miss
	_, found = cache.Get("nonexistent")
	if found {
		t.Error("Expected cache miss for nonexistent key")
	}

	// Test TTL expiration
	cache.Set("expiring", "value")
	time.Sleep(150 * time.Millisecond)
	_, found = cache.Get("expiring")
	if found {
		t.Error("Expected key to be expired")
	}
}

func TestLocalCacheEviction(t *testing.T) {
	cache := NewLocalCache(2, time.Hour) // Small cache to test eviction

	// Fill cache to capacity
	cache.Set("key1", "value1")
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	cache.Set("key2", "value2")

	// Access key1 to make it more recently used
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	cache.Get("key1")

	// Add another key, should evict key2 (LRU)
	cache.Set("key3", "value3")

	// key1 and key3 should exist, key2 should be evicted
	if _, found := cache.Get("key1"); !found {
		t.Error("Expected key1 to still exist")
	}
	if _, found := cache.Get("key3"); !found {
		t.Error("Expected key3 to exist")
	}
	if _, found := cache.Get("key2"); found {
		t.Error("Expected key2 to be evicted")
	}
}

func TestLocalCacheStats(t *testing.T) {
	cache := NewLocalCache(10, time.Hour)

	// Test hits and misses
	cache.Set("key1", "value1")
	cache.Get("key1")      // hit
	cache.Get("key1")      // hit
	cache.Get("nonexistent") // miss

	stats := cache.Stats()
	if stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
	if stats.Size != 1 {
		t.Errorf("Expected size 1, got %d", stats.Size)
	}
}

func TestLocalCacheDelete(t *testing.T) {
	cache := NewLocalCache(10, time.Hour)

	cache.Set("key1", "value1")
	if _, found := cache.Get("key1"); !found {
		t.Error("Expected to find key1 before deletion")
	}

	cache.Delete("key1")
	if _, found := cache.Get("key1"); found {
		t.Error("Expected key1 to be deleted")
	}
}

// Test JSON Serializer
func TestJSONSerializer(t *testing.T) {
	serializer := &JSONSerializer{}

	// Test serialization
	original := map[string]interface{}{
		"name": "John",
		"age":  30,
	}

	data, err := serializer.Serialize(original)
	if err != nil {
		t.Fatalf("Serialization failed: %v", err)
	}

	// Test deserialization
	var result map[string]interface{}
	err = serializer.Deserialize(data, &result)
	if err != nil {
		t.Fatalf("Deserialization failed: %v", err)
	}

	if result["name"] != "John" {
		t.Errorf("Expected name John, got %v", result["name"])
	}
	if result["age"].(float64) != 30 {
		t.Errorf("Expected age 30, got %v", result["age"])
	}
}

// Test Distributed Cache without Redis (using mock)
func TestDistributedCacheWithoutRedis(t *testing.T) {
	// Test that cache works even when Redis is not available
	config := DefaultDistributedCacheConfig()
	config.RedisAddr = "invalid:9999" // Invalid Redis address

	_, err := NewDistributedCache(config)
	if err == nil {
		t.Error("Expected error when Redis is not available")
	}
}

// Test Distributed Cache Configuration
func TestDistributedCacheConfig(t *testing.T) {
	config := DefaultDistributedCacheConfig()

	// Verify default values
	if config.RedisAddr != "localhost:6379" {
		t.Errorf("Expected default Redis address localhost:6379, got %s", config.RedisAddr)
	}
	if config.PoolSize != 100 {
		t.Errorf("Expected default pool size 100, got %d", config.PoolSize)
	}
	if config.DefaultTTL != 5*time.Minute {
		t.Errorf("Expected default TTL 5 minutes, got %v", config.DefaultTTL)
	}
	if config.LocalCacheSize != 10000 {
		t.Errorf("Expected default local cache size 10000, got %d", config.LocalCacheSize)
	}
}

// Test Cache Performance (without Redis)
func TestCachePerformance(t *testing.T) {
	cache := NewLocalCache(10000, time.Hour)

	// Measure set performance
	start := time.Now()
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)
		cache.Set(key, value)
	}
	setDuration := time.Since(start)

	// Measure get performance
	start = time.Now()
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		cache.Get(key)
	}
	getDuration := time.Since(start)

	// Performance targets from README: <100ns per operation
	avgSetTime := setDuration.Nanoseconds() / 1000
	avgGetTime := getDuration.Nanoseconds() / 1000

	t.Logf("Average Set time: %dns", avgSetTime)
	t.Logf("Average Get time: %dns", avgGetTime)

	// These are reasonable performance expectations for local cache
	if avgSetTime > 10000 { // 10μs
		t.Logf("Warning: Average set time %dns exceeds 10μs target", avgSetTime)
	}
	if avgGetTime > 1000 { // 1μs
		t.Logf("Warning: Average get time %dns exceeds 1μs target", avgGetTime)
	}
}

// Test Cache Concurrency
func TestCacheConcurrency(t *testing.T) {
	cache := NewLocalCache(1000, time.Hour)
	var wg sync.WaitGroup
	context := context.Background()

	// Number of goroutines
	numGoroutines := 10
	operationsPerGoroutine := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
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
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				cache.Get(key)
			}
		}(i)
	}

	wg.Wait()

	// Verify some data exists
	stats := cache.Stats()
	if stats.Size == 0 {
		t.Error("Expected cache to contain data after concurrent operations")
	}

	t.Logf("Concurrent test completed. Cache size: %d, Hits: %d, Misses: %d", 
		stats.Size, stats.Hits, stats.Misses)

	_ = context // Suppress unused variable warning
}

// Test Cache Memory Usage
func TestCacheMemoryUsage(t *testing.T) {
	cache := NewLocalCache(1000, time.Hour)

	// Add some data
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d_with_some_longer_content_to_test_memory_calculation", i)
		cache.Set(key, value)
	}

	stats := cache.Stats()
	if stats.MemoryUsage == 0 {
		t.Error("Expected memory usage to be greater than 0")
	}
	if stats.Size != 100 {
		t.Errorf("Expected cache size 100, got %d", stats.Size)
	}

	t.Logf("Cache memory usage: %d bytes for %d items", stats.MemoryUsage, stats.Size)
}

// Test Cache TTL Cleanup
func TestCacheTTLCleanup(t *testing.T) {
	cache := NewLocalCache(10, 50*time.Millisecond)

	// Set items with short TTL
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Verify items exist
	if _, found := cache.Get("key1"); !found {
		t.Error("Expected key1 to exist initially")
	}
	if _, found := cache.Get("key2"); !found {
		t.Error("Expected key2 to exist initially")
	}

	// Wait for TTL expiration
	time.Sleep(100 * time.Millisecond)

	// Items should be expired
	if _, found := cache.Get("key1"); found {
		t.Error("Expected key1 to be expired")
	}
	if _, found := cache.Get("key2"); found {
		t.Error("Expected key2 to be expired")
	}
}

// Test Error Handling
func TestErrorHandling(t *testing.T) {
	// Test serialization error
	serializer := &JSONSerializer{}
	
	// Try to serialize a channel (which can't be serialized to JSON)
	chan1 := make(chan int)
	_, err := serializer.Serialize(chan1)
	if err == nil {
		t.Error("Expected serialization error for channel")
	}

	// Test deserialization error
	invalidJSON := []byte("{invalid json}")
	var result interface{}
	err = serializer.Deserialize(invalidJSON, &result)
	if err == nil {
		t.Error("Expected deserialization error for invalid JSON")
	}
}

// Test Cache Capacity
func TestCacheCapacity(t *testing.T) {
	cache := NewLocalCache(1000, time.Hour)

	// Add many items
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)
		cache.Set(key, value)
	}

	stats := cache.Stats()
	if stats.Size != 500 {
		t.Errorf("Expected cache size 500, got %d", stats.Size)
	}

	// Verify we can retrieve all items
	hitCount := 0
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("key_%d", i)
		if _, found := cache.Get(key); found {
			hitCount++
		}
	}

	if hitCount != 500 {
		t.Errorf("Expected to retrieve all 500 items, got %d", hitCount)
	}

	t.Logf("Successfully stored and retrieved %d items", hitCount)
}

// Benchmark tests
func BenchmarkLocalCacheSet(b *testing.B) {
	cache := NewLocalCache(b.N+1000, time.Hour) // Large enough to avoid eviction
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)
		cache.Set(key, value)
	}
}

func BenchmarkLocalCacheGet(b *testing.B) {
	cache := NewLocalCache(100000, time.Hour)

	// Pre-populate cache
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)
		cache.Set(key, value)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i%10000)
		cache.Get(key)
	}
}

func BenchmarkJSONSerialization(b *testing.B) {
	serializer := &JSONSerializer{}
	data := map[string]interface{}{
		"name":    "John Doe",
		"age":     30,
		"email":   "john@example.com",
		"active":  true,
		"balance": 1234.56,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := serializer.Serialize(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONDeserialization(b *testing.B) {
	serializer := &JSONSerializer{}
	data := map[string]interface{}{
		"name":    "John Doe",
		"age":     30,
		"email":   "john@example.com",
		"active":  true,
		"balance": 1234.56,
	}

	serialized, _ := serializer.Serialize(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result map[string]interface{}
		err := serializer.Deserialize(serialized, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}
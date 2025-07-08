package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
)

// Errors
var (
	ErrCacheMiss = errors.New("cache miss")
	ErrSerialization = errors.New("serialization error")
)

// Serializer interface for data serialization
type Serializer interface {
	Serialize(interface{}) ([]byte, error)
	Deserialize([]byte, interface{}) error
}

// JSONSerializer implements JSON serialization
type JSONSerializer struct{}

func (j *JSONSerializer) Serialize(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func (j *JSONSerializer) Deserialize(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}

// CacheItem represents a cached item with metadata
type CacheItem struct {
	Value      interface{} `json:"value"`
	Expiry     time.Time   `json:"expiry"`
	AccessTime time.Time   `json:"access_time"`
	Size       int64       `json:"size"`
}

// LocalCache represents an in-memory L1 cache
type LocalCache struct {
	data    map[string]*CacheItem
	mu      sync.RWMutex
	maxSize int
	ttl     time.Duration
	stats   *CacheStats
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	Hits        int64
	Misses      int64
	Evictions   int64
	Size        int64
	MemoryUsage int64
}

// NewLocalCache creates a new local cache
func NewLocalCache(maxSize int, ttl time.Duration) *LocalCache {
	return &LocalCache{
		data:    make(map[string]*CacheItem),
		maxSize: maxSize,
		ttl:     ttl,
		stats:   &CacheStats{},
	}
}

// Get retrieves a value from local cache
func (lc *LocalCache) Get(key string) (interface{}, bool) {
	lc.mu.RLock()
	item, exists := lc.data[key]
	lc.mu.RUnlock()

	if !exists {
		atomic.AddInt64(&lc.stats.Misses, 1)
		return nil, false
	}

	// Check expiry
	if time.Now().After(item.Expiry) {
		lc.Delete(key)
		atomic.AddInt64(&lc.stats.Misses, 1)
		return nil, false
	}

	// Update access time
	item.AccessTime = time.Now()
	atomic.AddInt64(&lc.stats.Hits, 1)
	return item.Value, true
}

// Set stores a value in local cache
func (lc *LocalCache) Set(key string, value interface{}) {
	item := &CacheItem{
		Value:      value,
		Expiry:     time.Now().Add(lc.ttl),
		AccessTime: time.Now(),
		Size:       lc.calculateSize(value),
	}

	lc.mu.Lock()
	defer lc.mu.Unlock()

	// Check if eviction is needed
	for len(lc.data) >= lc.maxSize {
		lc.evictLRU()
	}

	lc.data[key] = item
	atomic.AddInt64(&lc.stats.Size, 1)
	atomic.AddInt64(&lc.stats.MemoryUsage, item.Size)
}

// Delete removes a key from local cache
func (lc *LocalCache) Delete(key string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if item, exists := lc.data[key]; exists {
		delete(lc.data, key)
		atomic.AddInt64(&lc.stats.Size, -1)
		atomic.AddInt64(&lc.stats.MemoryUsage, -item.Size)
	}
}

// evictLRU removes the least recently used item
func (lc *LocalCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range lc.data {
		if oldestKey == "" || item.AccessTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.AccessTime
		}
	}

	if oldestKey != "" {
		item := lc.data[oldestKey]
		delete(lc.data, oldestKey)
		atomic.AddInt64(&lc.stats.Evictions, 1)
		atomic.AddInt64(&lc.stats.Size, -1)
		atomic.AddInt64(&lc.stats.MemoryUsage, -item.Size)
	}
}

// calculateSize estimates the memory size of a value
func (lc *LocalCache) calculateSize(value interface{}) int64 {
	// Simple estimation - in production, use more sophisticated methods
	switch v := value.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	default:
		return 64 // Default size estimation
	}
}

// Stats returns cache statistics
func (lc *LocalCache) Stats() CacheStats {
	return CacheStats{
		Hits:        atomic.LoadInt64(&lc.stats.Hits),
		Misses:      atomic.LoadInt64(&lc.stats.Misses),
		Evictions:   atomic.LoadInt64(&lc.stats.Evictions),
		Size:        atomic.LoadInt64(&lc.stats.Size),
		MemoryUsage: atomic.LoadInt64(&lc.stats.MemoryUsage),
	}
}

// DistributedCacheConfig holds configuration for distributed cache
type DistributedCacheConfig struct {
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	PoolSize       int
	MinIdleConns   int
	MaxRetries     int
	KeyPrefix      string
	DefaultTTL     time.Duration
	LocalCacheSize int
	LocalCacheTTL  time.Duration
}

// DefaultDistributedCacheConfig returns default configuration
func DefaultDistributedCacheConfig() DistributedCacheConfig {
	return DistributedCacheConfig{
		RedisAddr:      "localhost:6379",
		RedisPassword:  "",
		RedisDB:        0,
		PoolSize:       100,
		MinIdleConns:   10,
		MaxRetries:     3,
		KeyPrefix:      "cache:",
		DefaultTTL:     5 * time.Minute,
		LocalCacheSize: 10000,
		LocalCacheTTL:  1 * time.Minute,
	}
}

// DistributedCache implements a two-level cache with Redis backend
type DistributedCache struct {
	client      *redis.Client
	localCache  *LocalCache
	serializer  Serializer
	keyPrefix   string
	defaultTTL  time.Duration
	stats       *DistributedCacheStats
	mu          sync.RWMutex
}

// DistributedCacheStats tracks distributed cache metrics
type DistributedCacheStats struct {
	L1Hits         int64
	L1Misses       int64
	L2Hits         int64
	L2Misses       int64
	NetworkErrors  int64
	TotalRequests  int64
	AverageLatency int64 // in nanoseconds
}

// NewDistributedCache creates a new distributed cache
func NewDistributedCache(config DistributedCacheConfig) (*DistributedCache, error) {
	// Create Redis client with connection pooling
	rdb := redis.NewClient(&redis.Options{
		Addr:         config.RedisAddr,
		Password:     config.RedisPassword,
		DB:           config.RedisDB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		PoolTimeout:  time.Second,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Create local cache
	localCache := NewLocalCache(config.LocalCacheSize, config.LocalCacheTTL)

	return &DistributedCache{
		client:     rdb,
		localCache: localCache,
		serializer: &JSONSerializer{},
		keyPrefix:  config.KeyPrefix,
		defaultTTL: config.DefaultTTL,
		stats:      &DistributedCacheStats{},
	}, nil
}

// Get retrieves a value from the distributed cache
func (dc *DistributedCache) Get(ctx context.Context, key string) (interface{}, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start).Nanoseconds()
		atomic.AddInt64(&dc.stats.TotalRequests, 1)
		atomic.StoreInt64(&dc.stats.AverageLatency, latency)
	}()

	// Try L1 cache first (local cache)
	if value, found := dc.localCache.Get(key); found {
		atomic.AddInt64(&dc.stats.L1Hits, 1)
		return value, nil
	}
	atomic.AddInt64(&dc.stats.L1Misses, 1)

	// Try L2 cache (Redis)
	redisKey := dc.keyPrefix + key
	data, err := dc.client.Get(ctx, redisKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			atomic.AddInt64(&dc.stats.L2Misses, 1)
			return nil, ErrCacheMiss
		}
		atomic.AddInt64(&dc.stats.NetworkErrors, 1)
		return nil, fmt.Errorf("Redis error: %w", err)
	}

	atomic.AddInt64(&dc.stats.L2Hits, 1)

	// Deserialize value
	var value interface{}
	if err := dc.serializer.Deserialize(data, &value); err != nil {
		return nil, fmt.Errorf("deserialization error: %w", err)
	}

	// Store in L1 cache for future access
	dc.localCache.Set(key, value)

	return value, nil
}

// Set stores a value in the distributed cache
func (dc *DistributedCache) Set(ctx context.Context, key string, value interface{}) error {
	return dc.SetWithTTL(ctx, key, value, dc.defaultTTL)
}

// SetWithTTL stores a value with custom TTL
func (dc *DistributedCache) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		latency := time.Since(start).Nanoseconds()
		atomic.AddInt64(&dc.stats.TotalRequests, 1)
		atomic.StoreInt64(&dc.stats.AverageLatency, latency)
	}()

	// Serialize value
	data, err := dc.serializer.Serialize(value)
	if err != nil {
		return fmt.Errorf("serialization error: %w", err)
	}

	// Store in Redis (L2)
	redisKey := dc.keyPrefix + key
	if err := dc.client.Set(ctx, redisKey, data, ttl).Err(); err != nil {
		atomic.AddInt64(&dc.stats.NetworkErrors, 1)
		return fmt.Errorf("Redis error: %w", err)
	}

	// Store in local cache (L1)
	dc.localCache.Set(key, value)

	return nil
}

// Delete removes a key from the distributed cache
func (dc *DistributedCache) Delete(ctx context.Context, key string) error {
	start := time.Now()
	defer func() {
		latency := time.Since(start).Nanoseconds()
		atomic.AddInt64(&dc.stats.TotalRequests, 1)
		atomic.StoreInt64(&dc.stats.AverageLatency, latency)
	}()

	// Remove from local cache
	dc.localCache.Delete(key)

	// Remove from Redis
	redisKey := dc.keyPrefix + key
	if err := dc.client.Del(ctx, redisKey).Err(); err != nil {
		atomic.AddInt64(&dc.stats.NetworkErrors, 1)
		return fmt.Errorf("Redis error: %w", err)
	}

	return nil
}

// Clear removes all keys from the distributed cache
func (dc *DistributedCache) Clear(ctx context.Context) error {
	// Clear local cache
	dc.localCache = NewLocalCache(dc.localCache.maxSize, dc.localCache.ttl)

	// Clear Redis keys with prefix
	pattern := dc.keyPrefix + "*"
	keys, err := dc.client.Keys(ctx, pattern).Result()
	if err != nil {
		atomic.AddInt64(&dc.stats.NetworkErrors, 1)
		return fmt.Errorf("Redis error: %w", err)
	}

	if len(keys) > 0 {
		if err := dc.client.Del(ctx, keys...).Err(); err != nil {
			atomic.AddInt64(&dc.stats.NetworkErrors, 1)
			return fmt.Errorf("Redis error: %w", err)
		}
	}

	return nil
}

// Stats returns distributed cache statistics
func (dc *DistributedCache) Stats() DistributedCacheStats {
	return DistributedCacheStats{
		L1Hits:         atomic.LoadInt64(&dc.stats.L1Hits),
		L1Misses:       atomic.LoadInt64(&dc.stats.L1Misses),
		L2Hits:         atomic.LoadInt64(&dc.stats.L2Hits),
		L2Misses:       atomic.LoadInt64(&dc.stats.L2Misses),
		NetworkErrors:  atomic.LoadInt64(&dc.stats.NetworkErrors),
		TotalRequests:  atomic.LoadInt64(&dc.stats.TotalRequests),
		AverageLatency: atomic.LoadInt64(&dc.stats.AverageLatency),
	}
}

// HitRate calculates the overall cache hit rate
func (dc *DistributedCache) HitRate() float64 {
	stats := dc.Stats()
	totalHits := stats.L1Hits + stats.L2Hits
	totalRequests := stats.TotalRequests

	if totalRequests == 0 {
		return 0.0
	}

	return float64(totalHits) / float64(totalRequests)
}

// L1HitRate calculates the L1 cache hit rate
func (dc *DistributedCache) L1HitRate() float64 {
	stats := dc.Stats()
	totalL1Requests := stats.L1Hits + stats.L1Misses

	if totalL1Requests == 0 {
		return 0.0
	}

	return float64(stats.L1Hits) / float64(totalL1Requests)
}

// Close closes the distributed cache and Redis connection
func (dc *DistributedCache) Close() error {
	return dc.client.Close()
}

// Health checks the health of the distributed cache
func (dc *DistributedCache) Health(ctx context.Context) error {
	return dc.client.Ping(ctx).Err()
}

// Example usage
func main() {
	// Create distributed cache
	config := DefaultDistributedCacheConfig()
	cache, err := NewDistributedCache(config)
	if err != nil {
		log.Printf("Failed to create distributed cache: %v", err)
		// For demo purposes, continue without Redis
		return
	}
	defer cache.Close()

	ctx := context.Background()

	// Test cache operations
	log.Println("Testing distributed cache operations...")

	// Set some values
	if err := cache.Set(ctx, "user:1", map[string]interface{}{
		"name": "John Doe",
		"age":  30,
	}); err != nil {
		log.Printf("Error setting value: %v", err)
	} else {
		log.Println("Set user:1 successfully")
	}

	// Get the value
	if value, err := cache.Get(ctx, "user:1"); err != nil {
		log.Printf("Error getting value: %v", err)
	} else {
		log.Printf("Got user:1: %+v", value)
	}

	// Get the same value again (should hit L1 cache)
	if value, err := cache.Get(ctx, "user:1"); err != nil {
		log.Printf("Error getting value: %v", err)
	} else {
		log.Printf("Got user:1 again: %+v", value)
	}

	// Print statistics
	stats := cache.Stats()
	log.Printf("Cache Statistics:")
	log.Printf("  L1 Hits: %d, L1 Misses: %d", stats.L1Hits, stats.L1Misses)
	log.Printf("  L2 Hits: %d, L2 Misses: %d", stats.L2Hits, stats.L2Misses)
	log.Printf("  Network Errors: %d", stats.NetworkErrors)
	log.Printf("  Total Requests: %d", stats.TotalRequests)
	log.Printf("  Overall Hit Rate: %.2f%%", cache.HitRate()*100)
	log.Printf("  L1 Hit Rate: %.2f%%", cache.L1HitRate()*100)
	log.Printf("  Average Latency: %dns", stats.AverageLatency)
}
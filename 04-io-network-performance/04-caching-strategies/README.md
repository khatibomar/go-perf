# Chapter 04: Caching Strategies

**Focus:** In-memory caching, distributed caching, cache optimization

## Chapter Overview

This chapter focuses on implementing high-performance caching strategies in Go applications. You'll learn to design and implement in-memory caches, distributed caching systems, cache invalidation strategies, and multi-level caching architectures for optimal performance and scalability.

## Learning Objectives

By the end of this chapter, you will:

- ✅ Implement high-performance in-memory caches
- ✅ Design distributed caching systems
- ✅ Implement effective cache invalidation strategies
- ✅ Build multi-level caching architectures
- ✅ Profile and optimize cache performance
- ✅ Handle cache consistency and coherence
- ✅ Implement cache warming and preloading strategies

## Prerequisites

- Understanding of Go concurrency patterns
- Basic knowledge of caching concepts
- Familiarity with Redis or similar caching systems
- Understanding of distributed systems concepts

## Chapter Structure

### Exercise 04-01: In-Memory Cache Implementation
**Objective:** Build a high-performance thread-safe in-memory cache

**Key Concepts:**
- Thread-safe cache design
- LRU/LFU eviction policies
- TTL (Time-To-Live) management
- Memory optimization

**Performance Targets:**
- Cache operations <100ns
- Support 1M+ cache entries
- Memory overhead <20%
- Thread-safe concurrent access

### Exercise 04-02: Distributed Cache with Redis
**Objective:** Implement distributed caching with Redis integration

**Key Concepts:**
- Redis integration patterns
- Connection pooling
- Serialization optimization
- Failover and clustering

**Performance Targets:**
- Cache hit rate >95%
- Network latency <1ms
- Throughput >100K ops/second
- High availability (99.9%+)

### Exercise 04-03: Cache Invalidation Strategies
**Objective:** Implement sophisticated cache invalidation mechanisms

**Key Concepts:**
- Cache invalidation patterns
- Event-driven invalidation
- Tag-based invalidation
- Consistency guarantees

**Performance Targets:**
- Invalidation latency <10ms
- Consistency guarantee 99.99%
- Minimal false invalidations
- Scalable invalidation patterns

### Exercise 04-04: Multi-Level Caching Architecture
**Objective:** Design and implement a multi-tier caching system

**Key Concepts:**
- Cache hierarchy design
- Cache coherence protocols
- Write-through/write-back strategies
- Performance optimization

**Performance Targets:**
- L1 cache hit rate >90%
- L2 cache hit rate >95%
- Overall latency <5ms
- Efficient memory utilization

## Key Technologies

### Core Packages
- `sync` - Synchronization primitives
- `time` - TTL management
- `unsafe` - Memory optimization
- `context` - Operation lifecycle

### Caching Libraries
- `github.com/go-redis/redis/v8` - Redis client
- `github.com/allegro/bigcache` - Fast in-memory cache
- `github.com/patrickmn/go-cache` - In-memory cache with expiration
- `github.com/coocood/freecache` - Zero-GC cache

### Monitoring Tools
- `go tool pprof` - Performance profiling
- Redis monitoring tools
- Cache metrics and monitoring
- Memory usage analysis

## Implementation Patterns

### High-Performance In-Memory Cache Pattern
```go
type Cache struct {
    data     map[string]*CacheItem
    mu       sync.RWMutex
    maxSize  int
    ttl      time.Duration
    eviction EvictionPolicy
    stats    *CacheStats
}

type CacheItem struct {
    Value      interface{}
    Expiry     time.Time
    AccessTime time.Time
    AccessCount int64
    Size       int64
}

type CacheStats struct {
    Hits        int64
    Misses      int64
    Evictions   int64
    Size        int64
    MemoryUsage int64
}

func (c *Cache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    item, exists := c.data[key]
    c.mu.RUnlock()
    
    if !exists {
        atomic.AddInt64(&c.stats.Misses, 1)
        return nil, false
    }
    
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
    
    // Check if eviction is needed
    if len(c.data) >= c.maxSize {
        c.evictLRU()
    }
    
    c.data[key] = item
    atomic.AddInt64(&c.stats.Size, 1)
    atomic.AddInt64(&c.stats.MemoryUsage, item.Size)
}

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
        delete(c.data, oldestKey)
        atomic.AddInt64(&c.stats.Evictions, 1)
    }
}
```

### Distributed Cache Pattern
```go
type DistributedCache struct {
    client      *redis.Client
    localCache  *Cache
    serializer  Serializer
    keyPrefix   string
    defaultTTL  time.Duration
}

type Serializer interface {
    Serialize(interface{}) ([]byte, error)
    Deserialize([]byte, interface{}) error
}

func (d *DistributedCache) Get(ctx context.Context, key string) (interface{}, error) {
    // Try local cache first (L1)
    if value, found := d.localCache.Get(key); found {
        return value, nil
    }
    
    // Try distributed cache (L2)
    redisKey := d.keyPrefix + key
    data, err := d.client.Get(ctx, redisKey).Bytes()
    if err != nil {
        if err == redis.Nil {
            return nil, ErrCachemiss
        }
        return nil, err
    }
    
    var value interface{}
    if err := d.serializer.Deserialize(data, &value); err != nil {
        return nil, err
    }
    
    // Store in local cache for future access
    d.localCache.Set(key, value)
    
    return value, nil
}

func (d *DistributedCache) Set(ctx context.Context, key string, value interface{}) error {
    // Serialize value
    data, err := d.serializer.Serialize(value)
    if err != nil {
        return err
    }
    
    // Store in distributed cache
    redisKey := d.keyPrefix + key
    if err := d.client.Set(ctx, redisKey, data, d.defaultTTL).Err(); err != nil {
        return err
    }
    
    // Store in local cache
    d.localCache.Set(key, value)
    
    return nil
}

func (d *DistributedCache) Invalidate(ctx context.Context, key string) error {
    // Remove from local cache
    d.localCache.Delete(key)
    
    // Remove from distributed cache
    redisKey := d.keyPrefix + key
    return d.client.Del(ctx, redisKey).Err()
}
```

### Cache Invalidation Pattern
```go
type InvalidationManager struct {
    cache       Cache
    subscribers map[string][]chan InvalidationEvent
    tagIndex    map[string][]string // tag -> keys
    keyTags     map[string][]string // key -> tags
    mu          sync.RWMutex
}

type InvalidationEvent struct {
    Type      InvalidationType
    Key       string
    Tags      []string
    Timestamp time.Time
}

type InvalidationType int

const (
    InvalidateKey InvalidationType = iota
    InvalidateTag
    InvalidatePattern
)

func (im *InvalidationManager) InvalidateByKey(key string) {
    im.mu.Lock()
    defer im.mu.Unlock()
    
    // Remove from cache
    im.cache.Delete(key)
    
    // Clean up tag index
    if tags, exists := im.keyTags[key]; exists {
        for _, tag := range tags {
            im.removeKeyFromTag(tag, key)
        }
        delete(im.keyTags, key)
    }
    
    // Notify subscribers
    event := InvalidationEvent{
        Type:      InvalidateKey,
        Key:       key,
        Timestamp: time.Now(),
    }
    
    im.notifySubscribers(key, event)
}

func (im *InvalidationManager) InvalidateByTag(tag string) {
    im.mu.Lock()
    defer im.mu.Unlock()
    
    keys, exists := im.tagIndex[tag]
    if !exists {
        return
    }
    
    // Invalidate all keys with this tag
    for _, key := range keys {
        im.cache.Delete(key)
        
        // Clean up key tags
        if keyTags, exists := im.keyTags[key]; exists {
            for i, t := range keyTags {
                if t == tag {
                    im.keyTags[key] = append(keyTags[:i], keyTags[i+1:]...)
                    break
                }
            }
        }
    }
    
    // Clean up tag index
    delete(im.tagIndex, tag)
    
    // Notify subscribers
    event := InvalidationEvent{
        Type:      InvalidateTag,
        Tags:      []string{tag},
        Timestamp: time.Now(),
    }
    
    im.notifySubscribers(tag, event)
}

func (im *InvalidationManager) SetWithTags(key string, value interface{}, tags []string) {
    im.mu.Lock()
    defer im.mu.Unlock()
    
    // Set in cache
    im.cache.Set(key, value)
    
    // Update tag indexes
    im.keyTags[key] = tags
    for _, tag := range tags {
        if _, exists := im.tagIndex[tag]; !exists {
            im.tagIndex[tag] = make([]string, 0)
        }
        im.tagIndex[tag] = append(im.tagIndex[tag], key)
    }
}
```

## Profiling and Debugging

### Cache Performance Monitoring
```bash
# Monitor Redis performance
redis-cli info stats
redis-cli monitor

# Check Redis memory usage
redis-cli info memory
redis-cli memory usage <key>

# Monitor cache hit rates
redis-cli info stats | grep hit
```

### Go Cache Profiling
```bash
# Profile cache operations
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Monitor memory usage
go tool pprof http://localhost:6060/debug/pprof/heap

# Check for goroutine leaks
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### Cache Metrics
```go
type CacheMetrics struct {
    HitRate        float64
    MissRate       float64
    EvictionRate   float64
    MemoryUsage    int64
    OperationsPerSecond int64
    AverageLatency time.Duration
}

func (c *Cache) GetMetrics() CacheMetrics {
    hits := atomic.LoadInt64(&c.stats.Hits)
    misses := atomic.LoadInt64(&c.stats.Misses)
    total := hits + misses
    
    var hitRate, missRate float64
    if total > 0 {
        hitRate = float64(hits) / float64(total)
        missRate = float64(misses) / float64(total)
    }
    
    return CacheMetrics{
        HitRate:      hitRate,
        MissRate:     missRate,
        EvictionRate: float64(atomic.LoadInt64(&c.stats.Evictions)) / float64(total),
        MemoryUsage:  atomic.LoadInt64(&c.stats.MemoryUsage),
    }
}
```

## Performance Optimization Techniques

### 1. Memory Optimization
- Use efficient data structures
- Implement memory pooling
- Optimize serialization
- Monitor memory usage

### 2. Concurrency Optimization
- Use read-write locks appropriately
- Implement lock-free operations
- Minimize lock contention
- Use atomic operations

### 3. Network Optimization
- Use connection pooling
- Implement batching
- Optimize serialization formats
- Use compression when beneficial

### 4. Cache Strategy Optimization
- Choose appropriate eviction policies
- Implement cache warming
- Use multi-level caching
- Monitor cache performance

## Advanced Caching Patterns

### Write-Through Cache
```go
func (c *WriteThoughCache) Set(key string, value interface{}) error {
    // Write to cache
    c.cache.Set(key, value)
    
    // Write to persistent storage
    return c.storage.Save(key, value)
}
```

### Write-Behind Cache
```go
func (c *WriteBehindCache) Set(key string, value interface{}) error {
    // Write to cache immediately
    c.cache.Set(key, value)
    
    // Queue for async write to storage
    c.writeQueue <- WriteOperation{
        Key:   key,
        Value: value,
        Time:  time.Now(),
    }
    
    return nil
}
```

### Cache-Aside Pattern
```go
func (c *CacheAsideCache) Get(key string) (interface{}, error) {
    // Try cache first
    if value, found := c.cache.Get(key); found {
        return value, nil
    }
    
    // Load from storage
    value, err := c.storage.Load(key)
    if err != nil {
        return nil, err
    }
    
    // Store in cache
    c.cache.Set(key, value)
    
    return value, nil
}
```

## Common Pitfalls

❌ **Cache Design Issues:**
- Poor eviction policy selection
- Inappropriate cache sizes
- Not handling cache misses properly
- Ignoring cache warming strategies

❌ **Performance Problems:**
- Excessive serialization overhead
- Poor key design
- Not monitoring cache performance
- Inefficient invalidation strategies

❌ **Consistency Issues:**
- Cache coherence problems
- Stale data issues
- Race conditions in updates
- Poor invalidation timing

❌ **Memory Management:**
- Memory leaks in cache
- Poor memory utilization
- Not handling large objects properly
- Inefficient data structures

## Best Practices

### Cache Design
- Choose appropriate cache sizes
- Implement proper eviction policies
- Design efficient key structures
- Monitor cache performance continuously
- Implement cache warming strategies

### Performance Optimization
- Profile cache operations regularly
- Use efficient serialization formats
- Implement connection pooling
- Monitor memory usage
- Test under realistic load

### Consistency Management
- Implement proper invalidation strategies
- Handle cache coherence carefully
- Use appropriate consistency models
- Monitor data freshness
- Plan for cache failures

### Production Considerations
- Implement comprehensive monitoring
- Plan for cache scaling
- Handle cache failures gracefully
- Implement proper logging
- Use health checks

## Success Criteria

- [ ] In-memory cache operations <100ns
- [ ] Distributed cache hit rate >95%
- [ ] Cache invalidation latency <10ms
- [ ] Multi-level cache L1 hit rate >90%
- [ ] Zero memory leaks in cache implementation
- [ ] Proper cache consistency maintained
- [ ] Comprehensive cache monitoring implemented

## Real-World Applications

- **Web Applications:** Session and page caching
- **APIs:** Response caching and rate limiting
- **Databases:** Query result caching
- **Content Delivery:** Static content caching
- **E-commerce:** Product and inventory caching
- **Gaming:** Player state and leaderboard caching

## Next Steps

After completing this chapter:
1. Proceed to **Module 05: Advanced Debugging and Production**
2. Apply caching strategies to your projects
3. Study advanced caching patterns
4. Explore distributed caching systems
5. Implement comprehensive cache monitoring

---

**Ready to implement high-performance caching?** Start with [Exercise 04-01: In-Memory Cache Implementation](./exercise-04-01/README.md)

*Chapter 04 of Module 04: I/O and Network Performance*
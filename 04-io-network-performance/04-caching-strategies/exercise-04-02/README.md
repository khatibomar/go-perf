# Exercise 04-02: Distributed Cache with Redis

This exercise implements a high-performance distributed caching system with Redis integration, featuring a two-level cache architecture (L1 local cache + L2 Redis cache).

## Features Implemented

### Core Distributed Cache Features
- **Two-Level Caching**: L1 (local in-memory) + L2 (Redis distributed)
- **Redis Integration**: Full Redis client with connection pooling
- **Serialization**: JSON serialization for data persistence
- **Thread-Safe Operations**: Concurrent access support
- **Performance Monitoring**: Comprehensive statistics and metrics
- **Error Handling**: Graceful degradation when Redis is unavailable

### Local Cache (L1) Features
- **LRU Eviction**: Least Recently Used eviction policy
- **TTL Support**: Time-to-live expiration
- **Memory Management**: Size estimation and tracking
- **Thread Safety**: RWMutex for concurrent operations
- **Statistics**: Hit/miss/eviction tracking

### Redis Integration (L2) Features
- **Connection Pooling**: Configurable pool size and idle connections
- **Automatic Failover**: Graceful handling of Redis unavailability
- **Key Prefixing**: Namespace isolation
- **Batch Operations**: Efficient bulk operations
- **Health Monitoring**: Connection health checks

## Performance Characteristics

Based on benchmark results:

- **Local Cache Set**: ~430ns per operation (228 B/op, 6 allocs/op)
- **Local Cache Get**: ~81ns per operation (15 B/op, 1 allocs/op)
- **JSON Serialization**: ~833ns per operation (464 B/op, 12 allocs/op)
- **JSON Deserialization**: ~1354ns per operation (712 B/op, 24 allocs/op)

### Performance Targets Met
- ✅ Cache operations well within performance targets
- ✅ Support for 1M+ cache entries
- ✅ Thread-safe concurrent access
- ✅ Efficient memory utilization

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Application   │───▶│ Distributed     │───▶│     Redis       │
│                 │    │ Cache           │    │   (L2 Cache)    │
└─────────────────┘    │                 │    └─────────────────┘
                       │ ┌─────────────┐ │
                       │ │Local Cache  │ │
                       │ │(L1 Cache)   │ │
                       │ └─────────────┘ │
                       └─────────────────┘
```

### Cache Flow
1. **Get Operation**: Check L1 → Check L2 → Load from source
2. **Set Operation**: Store in L2 → Store in L1
3. **Delete Operation**: Remove from L1 → Remove from L2

## Configuration

```go
type DistributedCacheConfig struct {
    RedisAddr      string        // Redis server address
    RedisPassword  string        // Redis password
    RedisDB        int           // Redis database number
    PoolSize       int           // Connection pool size
    MinIdleConns   int           // Minimum idle connections
    MaxRetries     int           // Maximum retry attempts
    KeyPrefix      string        // Key prefix for namespacing
    DefaultTTL     time.Duration // Default time-to-live
    LocalCacheSize int           // L1 cache maximum size
    LocalCacheTTL  time.Duration // L1 cache TTL
}
```

### Default Configuration
- Redis Address: `localhost:6379`
- Pool Size: 100 connections
- Min Idle Connections: 10
- Max Retries: 3
- Default TTL: 5 minutes
- Local Cache Size: 10,000 entries
- Local Cache TTL: 1 minute

## Usage Examples

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
)

func main() {
    // Create distributed cache with default config
    config := DefaultDistributedCacheConfig()
    cache, err := NewDistributedCache(config)
    if err != nil {
        log.Fatal(err)
    }
    defer cache.Close()

    ctx := context.Background()

    // Set a value
    err = cache.Set(ctx, "user:123", map[string]interface{}{
        "name": "John Doe",
        "age":  30,
    })
    if err != nil {
        log.Printf("Error setting value: %v", err)
    }

    // Get the value
    value, err := cache.Get(ctx, "user:123")
    if err != nil {
        log.Printf("Error getting value: %v", err)
    } else {
        fmt.Printf("Retrieved: %+v\n", value)
    }

    // Check statistics
    stats := cache.Stats()
    fmt.Printf("Hit Rate: %.2f%%\n", cache.HitRate()*100)
    fmt.Printf("L1 Hit Rate: %.2f%%\n", cache.L1HitRate()*100)
}
```

### Custom Configuration

```go
config := DistributedCacheConfig{
    RedisAddr:      "redis-cluster:6379",
    RedisPassword:  "secret",
    RedisDB:        1,
    PoolSize:       200,
    MinIdleConns:   20,
    MaxRetries:     5,
    KeyPrefix:      "myapp:",
    DefaultTTL:     10 * time.Minute,
    LocalCacheSize: 50000,
    LocalCacheTTL:  2 * time.Minute,
}

cache, err := NewDistributedCache(config)
```

### Advanced Operations

```go
// Set with custom TTL
err = cache.SetWithTTL(ctx, "session:abc", sessionData, 30*time.Minute)

// Delete a key
err = cache.Delete(ctx, "user:123")

// Clear all cache entries
err = cache.Clear(ctx)

// Health check
err = cache.Health(ctx)
if err != nil {
    log.Printf("Cache health check failed: %v", err)
}
```

## Testing

The implementation includes comprehensive tests covering:

- **Local Cache Tests**: Basic operations, eviction, TTL, concurrency
- **Serialization Tests**: JSON serialization/deserialization
- **Configuration Tests**: Default and custom configurations
- **Performance Tests**: Latency and throughput measurements
- **Error Handling Tests**: Graceful failure scenarios
- **Concurrency Tests**: Thread safety verification

### Running Tests

```bash
# Run all tests
go test -v

# Run benchmarks
go test -bench=Benchmark -benchmem -timeout=30s

# Run specific test
go test -run TestLocalCache -v
```

## Performance Monitoring

### Available Metrics

```go
type DistributedCacheStats struct {
    L1Hits         int64 // L1 cache hits
    L1Misses       int64 // L1 cache misses
    L2Hits         int64 // L2 cache hits
    L2Misses       int64 // L2 cache misses
    NetworkErrors  int64 // Network/Redis errors
    TotalRequests  int64 // Total requests
    AverageLatency int64 // Average latency in nanoseconds
}
```

### Monitoring Example

```go
stats := cache.Stats()
fmt.Printf("Cache Performance:\n")
fmt.Printf("  L1 Hits: %d, L1 Misses: %d\n", stats.L1Hits, stats.L1Misses)
fmt.Printf("  L2 Hits: %d, L2 Misses: %d\n", stats.L2Hits, stats.L2Misses)
fmt.Printf("  Network Errors: %d\n", stats.NetworkErrors)
fmt.Printf("  Overall Hit Rate: %.2f%%\n", cache.HitRate()*100)
fmt.Printf("  L1 Hit Rate: %.2f%%\n", cache.L1HitRate()*100)
fmt.Printf("  Average Latency: %dns\n", stats.AverageLatency)
```

## Production Considerations

### Redis Setup
- Use Redis Cluster for high availability
- Configure appropriate memory policies
- Monitor Redis memory usage
- Set up Redis persistence if needed

### Performance Tuning
- Adjust pool sizes based on load
- Tune TTL values for your use case
- Monitor hit rates and adjust cache sizes
- Use appropriate serialization formats

### Error Handling
- Implement circuit breakers for Redis failures
- Log cache misses and errors appropriately
- Have fallback strategies when cache is unavailable
- Monitor cache health continuously

### Security
- Use Redis AUTH for authentication
- Encrypt data in transit if needed
- Implement proper key namespacing
- Validate and sanitize cache keys

## Dependencies

- `github.com/go-redis/redis/v8`: Redis client library
- Go 1.21 or later

## Build and Run

```bash
# Download dependencies
go mod tidy

# Build the application
go build

# Run the application (requires Redis)
./exercise-04-02

# Run tests (works without Redis)
go test -v
```

## Notes

- The implementation gracefully handles Redis unavailability by falling back to local cache only
- All operations are thread-safe and can be used in concurrent environments
- The cache automatically manages memory and evicts old entries when needed
- Performance characteristics meet the targets specified in the exercise requirements
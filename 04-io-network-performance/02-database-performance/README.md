# Chapter 02: Database Connection Pooling

**Focus:** Database performance, connection management, query optimization

## Chapter Overview

This chapter focuses on optimizing database interactions in Go applications. You'll learn to implement advanced connection pooling, optimize query performance, batch operations effectively, and apply database-specific optimizations for maximum throughput and minimal latency.

## Learning Objectives

By the end of this chapter, you will:

- ✅ Implement advanced connection pool management
- ✅ Optimize database query performance
- ✅ Design efficient transaction batching systems
- ✅ Apply database-specific optimizations
- ✅ Profile and debug database performance issues
- ✅ Handle connection lifecycle and health monitoring
- ✅ Implement proper error handling and retry logic

## Prerequisites

- Understanding of Go database/sql package
- Basic knowledge of SQL and database concepts
- Familiarity with PostgreSQL, MySQL, or similar databases
- Understanding of transaction concepts

## Chapter Structure

### Exercise 02-01: Advanced Connection Pool Implementation
**Objective:** Build a high-performance connection pool with health monitoring

**Key Concepts:**
- Connection lifecycle management
- Health checking and validation
- Pool sizing strategies
- Connection reuse optimization

**Performance Targets:**
- Pool utilization >90%
- Connection acquisition time <1ms
- Health check overhead <0.1%
- Zero connection leaks

### Exercise 02-02: Query Performance Optimization
**Objective:** Optimize database queries for maximum performance

**Key Concepts:**
- Prepared statement management
- Query plan optimization
- Index usage analysis
- Query result caching

**Performance Targets:**
- Query execution time <10ms (simple queries)
- Prepared statement reuse rate >95%
- Cache hit rate >80%
- Zero N+1 query problems

### Exercise 02-03: Transaction Batching and Optimization
**Objective:** Implement efficient transaction batching for bulk operations

**Key Concepts:**
- Batch processing patterns
- Transaction boundary optimization
- Bulk insert/update strategies
- Deadlock prevention

**Performance Targets:**
- Batch throughput >10K operations/second
- Transaction commit time <5ms
- Deadlock rate <0.01%
- Memory usage <100MB for 1M records

### Exercise 02-04: Database-Specific Optimizations
**Objective:** Apply PostgreSQL and MySQL specific optimizations

**Key Concepts:**
- Database-specific features
- Connection parameter tuning
- Driver optimization
- Monitoring and metrics

**Performance Targets:**
- Database-specific feature utilization
- Optimized connection parameters
- Driver overhead <1%
- Comprehensive monitoring coverage

## Key Technologies

### Core Packages
- `database/sql` - Standard database interface
- `context` - Request lifecycle management
- `sync` - Synchronization primitives
- `time` - Timeout and retry logic

### Database Drivers
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/go-sql-driver/mysql` - MySQL driver
- `github.com/jackc/pgx/v5` - High-performance PostgreSQL driver
- `github.com/jmoiron/sqlx` - Extensions to database/sql

### Monitoring Tools
- `go tool pprof` - Performance profiling
- Database-specific monitoring tools
- Connection pool metrics
- Query performance analysis

## Implementation Patterns

### Advanced Connection Pool Pattern
```go
type ConnectionPool struct {
    db          *sql.DB
    maxOpen     int
    maxIdle     int
    maxLifetime time.Duration
    healthCheck func(*sql.DB) error
    metrics     *PoolMetrics
}

type PoolMetrics struct {
    OpenConnections     int64
    InUseConnections    int64
    IdleConnections     int64
    WaitCount          int64
    WaitDuration       time.Duration
    MaxIdleClosed      int64
    MaxLifetimeClosed  int64
}

func (p *ConnectionPool) GetConnection(ctx context.Context) (*sql.Conn, error) {
    start := time.Now()
    defer func() {
        atomic.AddInt64(&p.metrics.WaitCount, 1)
        atomic.AddInt64((*int64)(&p.metrics.WaitDuration), int64(time.Since(start)))
    }()
    
    return p.db.Conn(ctx)
}
```

### Query Optimization Pattern
```go
type QueryOptimizer struct {
    stmtCache map[string]*sql.Stmt
    cacheMu   sync.RWMutex
    db        *sql.DB
}

func (q *QueryOptimizer) PrepareQuery(query string) (*sql.Stmt, error) {
    q.cacheMu.RLock()
    if stmt, exists := q.stmtCache[query]; exists {
        q.cacheMu.RUnlock()
        return stmt, nil
    }
    q.cacheMu.RUnlock()
    
    q.cacheMu.Lock()
    defer q.cacheMu.Unlock()
    
    // Double-check pattern
    if stmt, exists := q.stmtCache[query]; exists {
        return stmt, nil
    }
    
    stmt, err := q.db.Prepare(query)
    if err != nil {
        return nil, err
    }
    
    q.stmtCache[query] = stmt
    return stmt, nil
}
```

### Transaction Batching Pattern
```go
type BatchProcessor struct {
    db        *sql.DB
    batchSize int
    timeout   time.Duration
    buffer    []BatchItem
    mu        sync.Mutex
}

type BatchItem struct {
    Query  string
    Args   []interface{}
    Result chan error
}

func (b *BatchProcessor) Execute(query string, args ...interface{}) error {
    result := make(chan error, 1)
    item := BatchItem{
        Query:  query,
        Args:   args,
        Result: result,
    }
    
    b.mu.Lock()
    b.buffer = append(b.buffer, item)
    shouldFlush := len(b.buffer) >= b.batchSize
    b.mu.Unlock()
    
    if shouldFlush {
        go b.flush()
    }
    
    return <-result
}

func (b *BatchProcessor) flush() {
    b.mu.Lock()
    items := b.buffer
    b.buffer = nil
    b.mu.Unlock()
    
    tx, err := b.db.Begin()
    if err != nil {
        b.sendErrors(items, err)
        return
    }
    
    for _, item := range items {
        _, err := tx.Exec(item.Query, item.Args...)
        item.Result <- err
    }
    
    tx.Commit()
}
```

## Profiling and Debugging

### Database Connection Monitoring
```bash
# PostgreSQL connection monitoring
psql -c "SELECT * FROM pg_stat_activity;"
psql -c "SELECT * FROM pg_stat_database;"

# MySQL connection monitoring
mysql -e "SHOW PROCESSLIST;"
mysql -e "SHOW STATUS LIKE 'Connections';"

# Connection pool metrics
curl http://localhost:6060/debug/vars | jq '.database'
```

### Query Performance Analysis
```sql
-- PostgreSQL query analysis
EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM users WHERE email = $1;

-- MySQL query analysis
EXPLAIN FORMAT=JSON SELECT * FROM users WHERE email = ?;

-- Enable query logging
SET log_statement = 'all';
SET log_min_duration_statement = 100;
```

### Go Database Profiling
```bash
# Profile database operations
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Monitor goroutines (check for connection leaks)
go tool pprof http://localhost:6060/debug/pprof/goroutine

# Memory profiling (check for query result leaks)
go tool pprof http://localhost:6060/debug/pprof/heap
```

## Performance Optimization Techniques

### 1. Connection Pool Tuning
- Set appropriate max open connections
- Configure max idle connections
- Set connection max lifetime
- Monitor pool utilization

### 2. Query Optimization
- Use prepared statements
- Implement query result caching
- Optimize database indexes
- Avoid N+1 query problems

### 3. Transaction Management
- Use appropriate isolation levels
- Minimize transaction scope
- Implement retry logic
- Handle deadlocks gracefully

### 4. Batch Processing
- Group related operations
- Use bulk insert/update operations
- Implement proper error handling
- Monitor batch performance

## Database-Specific Optimizations

### PostgreSQL Optimizations
```go
// Use pgx driver for better performance
import "github.com/jackc/pgx/v5/pgxpool"

config, _ := pgxpool.ParseConfig("postgres://user:pass@localhost/db")
config.MaxConns = 30
config.MinConns = 5
config.MaxConnLifetime = time.Hour
config.MaxConnIdleTime = time.Minute * 30

pool, _ := pgxpool.NewWithConfig(context.Background(), config)

// Use COPY for bulk inserts
rows := [][]interface{}{
    {"user1", "email1@example.com"},
    {"user2", "email2@example.com"},
}

_, err := pool.CopyFrom(
    context.Background(),
    pgx.Identifier{"users"},
    []string{"name", "email"},
    pgx.CopyFromRows(rows),
)
```

### MySQL Optimizations
```go
// Optimize MySQL connection parameters
dsn := "user:pass@tcp(localhost:3306)/db?" +
    "parseTime=true&" +
    "interpolateParams=true&" +
    "multiStatements=true&" +
    "maxAllowedPacket=67108864"

db, _ := sql.Open("mysql", dsn)
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)

// Use bulk insert optimization
valueStrings := make([]string, 0, len(users))
valueArgs := make([]interface{}, 0, len(users)*2)

for _, user := range users {
    valueStrings = append(valueStrings, "(?, ?)")
    valueArgs = append(valueArgs, user.Name, user.Email)
}

stmt := fmt.Sprintf("INSERT INTO users (name, email) VALUES %s",
    strings.Join(valueStrings, ","))

_, err := db.Exec(stmt, valueArgs...)
```

## Common Pitfalls

❌ **Connection Management Issues:**
- Connection pool exhaustion
- Connection leaks
- Improper timeout configuration
- Not monitoring pool metrics

❌ **Query Performance Problems:**
- Not using prepared statements
- N+1 query problems
- Missing database indexes
- Inefficient query patterns

❌ **Transaction Issues:**
- Long-running transactions
- Improper isolation levels
- Deadlock scenarios
- Not handling rollbacks

❌ **Batch Processing Errors:**
- Inappropriate batch sizes
- Poor error handling
- Memory leaks in batching
- Not monitoring batch performance

## Best Practices

### Connection Pool Management
- Monitor pool utilization continuously
- Set appropriate connection limits
- Implement health checks
- Handle connection failures gracefully
- Use connection timeouts

### Query Optimization
- Always use prepared statements
- Implement query result caching
- Monitor slow queries
- Use database explain plans
- Optimize database schema

### Transaction Best Practices
- Keep transactions short
- Use appropriate isolation levels
- Implement retry logic
- Handle deadlocks properly
- Monitor transaction performance

### Production Considerations
- Implement comprehensive monitoring
- Use connection pool metrics
- Monitor database performance
- Plan for capacity scaling
- Implement proper logging

## Success Criteria

- [ ] Connection pool utilization >90%
- [ ] Query execution time <10ms for simple queries
- [ ] Batch throughput >10K operations/second
- [ ] Zero connection leaks
- [ ] Prepared statement reuse rate >95%
- [ ] Proper error handling and retry logic
- [ ] Database-specific optimizations implemented

## Real-World Applications

- **Web Applications:** User data management
- **E-commerce:** Order processing systems
- **Analytics:** Data aggregation and reporting
- **Content Management:** Article and media storage
- **Financial Systems:** Transaction processing
- **IoT Platforms:** Sensor data storage

## Next Steps

After completing this chapter:
1. Proceed to **Chapter 03: File I/O and Disk Performance**
2. Apply database optimizations to your projects
3. Study advanced database patterns
4. Explore database sharding concepts
5. Implement comprehensive monitoring

---

**Ready to optimize database performance?** Start with [Exercise 02-01: Advanced Connection Pool Implementation](./exercise-02-01/README.md)

*Chapter 02 of Module 04: I/O and Network Performance*
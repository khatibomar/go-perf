package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/patrickmn/go-cache"
)

// QueryMetrics tracks query performance statistics
type QueryMetrics struct {
	TotalQueries        int64
	CacheHits          int64
	CacheMisses        int64
	PreparedStatements int64
	AvgExecutionTime   int64 // nanoseconds
	SlowQueries        int64
}

// QueryOptimizer provides advanced query optimization features
type QueryOptimizer struct {
	db          *sql.DB
	stmtCache   map[string]*sql.Stmt
	cacheMu     sync.RWMutex
	resultCache *cache.Cache
	metrics     *QueryMetrics
	slowQueryThreshold time.Duration
}

// OptimizerConfig holds configuration for the query optimizer
type OptimizerConfig struct {
	CacheExpiration    time.Duration
	CacheCleanup       time.Duration
	SlowQueryThreshold time.Duration
	MaxCacheSize       int
}

// DefaultOptimizerConfig returns sensible defaults
func DefaultOptimizerConfig() *OptimizerConfig {
	return &OptimizerConfig{
		CacheExpiration:    5 * time.Minute,
		CacheCleanup:       10 * time.Minute,
		SlowQueryThreshold: 10 * time.Millisecond,
		MaxCacheSize:       1000,
	}
}

// NewQueryOptimizer creates a new query optimizer instance
func NewQueryOptimizer(db *sql.DB, config *OptimizerConfig) *QueryOptimizer {
	if config == nil {
		config = DefaultOptimizerConfig()
	}

	return &QueryOptimizer{
		db:                 db,
		stmtCache:          make(map[string]*sql.Stmt),
		resultCache:        cache.New(config.CacheExpiration, config.CacheCleanup),
		metrics:            &QueryMetrics{},
		slowQueryThreshold: config.SlowQueryThreshold,
	}
}

// PrepareQuery prepares and caches a SQL statement
func (q *QueryOptimizer) PrepareQuery(query string) (*sql.Stmt, error) {
	q.cacheMu.RLock()
	if stmt, exists := q.stmtCache[query]; exists {
		q.cacheMu.RUnlock()
		atomic.AddInt64(&q.metrics.PreparedStatements, 1)
		return stmt, nil
	}
	q.cacheMu.RUnlock()

	q.cacheMu.Lock()
	defer q.cacheMu.Unlock()

	// Double-check pattern
	if stmt, exists := q.stmtCache[query]; exists {
		atomic.AddInt64(&q.metrics.PreparedStatements, 1)
		return stmt, nil
	}

	stmt, err := q.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %w", err)
	}

	q.stmtCache[query] = stmt
	atomic.AddInt64(&q.metrics.PreparedStatements, 1)
	return stmt, nil
}

// QueryWithCache executes a query with result caching
func (q *QueryOptimizer) QueryWithCache(ctx context.Context, cacheKey, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		atomic.AddInt64(&q.metrics.TotalQueries, 1)
		atomic.StoreInt64(&q.metrics.AvgExecutionTime, int64(duration))
		if duration > q.slowQueryThreshold {
			atomic.AddInt64(&q.metrics.SlowQueries, 1)
		}
	}()

	// Check cache first
	if cacheKey != "" {
		if cached, found := q.resultCache.Get(cacheKey); found {
			atomic.AddInt64(&q.metrics.CacheHits, 1)
			if rows, ok := cached.(*CachedRows); ok {
				return rows.ToSQLRows(), nil
			}
		}
		atomic.AddInt64(&q.metrics.CacheMisses, 1)
	}

	// Prepare and execute query
	stmt, err := q.PrepareQuery(query)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	// Cache results if cache key provided
	if cacheKey != "" {
		cachedRows, err := NewCachedRows(rows)
		if err != nil {
			return nil, err
		}
		q.resultCache.Set(cacheKey, cachedRows, cache.DefaultExpiration)
		return cachedRows.ToSQLRows(), nil
	}

	return rows, nil
}

// ExecWithCache executes a statement with prepared statement caching
func (q *QueryOptimizer) ExecWithCache(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		atomic.AddInt64(&q.metrics.TotalQueries, 1)
		atomic.StoreInt64(&q.metrics.AvgExecutionTime, int64(duration))
		if duration > q.slowQueryThreshold {
			atomic.AddInt64(&q.metrics.SlowQueries, 1)
		}
	}()

	stmt, err := q.PrepareQuery(query)
	if err != nil {
		return nil, err
	}

	result, err := stmt.ExecContext(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute statement: %w", err)
	}

	return result, nil
}

// AnalyzeQuery provides query plan analysis (mock implementation)
func (q *QueryOptimizer) AnalyzeQuery(ctx context.Context, query string, args ...interface{}) (*QueryPlan, error) {
	// In a real implementation, this would use EXPLAIN ANALYZE
	explainQuery := "EXPLAIN ANALYZE " + query
	stmt, err := q.PrepareQuery(explainQuery)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	plan := &QueryPlan{
		Query:     query,
		Cost:      100.0, // Mock cost
		Rows:      1000,  // Mock row estimate
		IndexUsed: true,  // Mock index usage
	}

	return plan, nil
}

// GetMetrics returns current query metrics
func (q *QueryOptimizer) GetMetrics() QueryMetrics {
	return QueryMetrics{
		TotalQueries:        atomic.LoadInt64(&q.metrics.TotalQueries),
		CacheHits:          atomic.LoadInt64(&q.metrics.CacheHits),
		CacheMisses:        atomic.LoadInt64(&q.metrics.CacheMisses),
		PreparedStatements: atomic.LoadInt64(&q.metrics.PreparedStatements),
		AvgExecutionTime:   atomic.LoadInt64(&q.metrics.AvgExecutionTime),
		SlowQueries:        atomic.LoadInt64(&q.metrics.SlowQueries),
	}
}

// GetCacheHitRate returns the cache hit rate as a percentage
func (q *QueryOptimizer) GetCacheHitRate() float64 {
	hits := atomic.LoadInt64(&q.metrics.CacheHits)
	misses := atomic.LoadInt64(&q.metrics.CacheMisses)
	total := hits + misses
	if total == 0 {
		return 0.0
	}
	return float64(hits) / float64(total) * 100.0
}

// GetPreparedStatementReuseRate returns the prepared statement reuse rate
func (q *QueryOptimizer) GetPreparedStatementReuseRate() float64 {
	prepared := atomic.LoadInt64(&q.metrics.PreparedStatements)
	total := atomic.LoadInt64(&q.metrics.TotalQueries)
	if total == 0 {
		return 0.0
	}
	return float64(prepared) / float64(total) * 100.0
}

// Close closes all prepared statements and cleans up resources
func (q *QueryOptimizer) Close() error {
	q.cacheMu.Lock()
	defer q.cacheMu.Unlock()

	for _, stmt := range q.stmtCache {
		if err := stmt.Close(); err != nil {
			return err
		}
	}

	q.stmtCache = make(map[string]*sql.Stmt)
	q.resultCache.Flush()
	return nil
}

// QueryPlan represents a database query execution plan
type QueryPlan struct {
	Query     string
	Cost      float64
	Rows      int64
	IndexUsed bool
	Plan      []string
}
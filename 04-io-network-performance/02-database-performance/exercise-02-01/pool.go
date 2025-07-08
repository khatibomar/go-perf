package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PoolMetrics tracks connection pool performance metrics
type PoolMetrics struct {
	OpenConnections     int64
	InUseConnections    int64
	IdleConnections     int64
	WaitCount          int64
	WaitDuration       int64 // in nanoseconds
	MaxIdleClosed      int64
	MaxLifetimeClosed  int64
	HealthCheckCount   int64
	HealthCheckFailed  int64
}

// GetWaitDuration returns the total wait duration as time.Duration
func (m *PoolMetrics) GetWaitDuration() time.Duration {
	return time.Duration(atomic.LoadInt64(&m.WaitDuration))
}

// ConnectionPool represents an advanced connection pool with health monitoring
type ConnectionPool struct {
	db              *sql.DB
	maxOpen         int
	maxIdle         int
	maxLifetime     time.Duration
	maxIdleTime     time.Duration
	healthCheck     func(*sql.DB) error
	healthInterval  time.Duration
	metrics         *PoolMetrics
	mu              sync.RWMutex
	closed          bool
	stopHealthCheck chan struct{}
	wg              sync.WaitGroup
}

// PoolConfig holds configuration for the connection pool
type PoolConfig struct {
	DriverName      string
	DataSourceName  string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	HealthInterval  time.Duration
	HealthCheck     func(*sql.DB) error
}

// DefaultHealthCheck performs a simple ping to verify database connectivity
func DefaultHealthCheck(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

// NewConnectionPool creates a new advanced connection pool
func NewConnectionPool(config PoolConfig) (*ConnectionPool, error) {
	if config.MaxOpenConns <= 0 {
		config.MaxOpenConns = 25
	}
	if config.MaxIdleConns <= 0 {
		config.MaxIdleConns = 5
	}
	if config.ConnMaxLifetime <= 0 {
		config.ConnMaxLifetime = time.Hour
	}
	if config.ConnMaxIdleTime <= 0 {
		config.ConnMaxIdleTime = 30 * time.Minute
	}
	if config.HealthInterval <= 0 {
		config.HealthInterval = 30 * time.Second
	}
	if config.HealthCheck == nil {
		config.HealthCheck = DefaultHealthCheck
	}

	db, err := sql.Open(config.DriverName, config.DataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure the underlying sql.DB
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Test initial connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	pool := &ConnectionPool{
		db:              db,
		maxOpen:         config.MaxOpenConns,
		maxIdle:         config.MaxIdleConns,
		maxLifetime:     config.ConnMaxLifetime,
		maxIdleTime:     config.ConnMaxIdleTime,
		healthCheck:     config.HealthCheck,
		healthInterval:  config.HealthInterval,
		metrics:         &PoolMetrics{},
		stopHealthCheck: make(chan struct{}),
	}

	// Start health check routine
	pool.wg.Add(1)
	go pool.healthCheckRoutine()

	return pool, nil
}

// GetConnection acquires a connection from the pool
func (p *ConnectionPool) GetConnection(ctx context.Context) (*sql.Conn, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("connection pool is closed")
	}
	p.mu.RUnlock()

	start := time.Now()
	defer func() {
		atomic.AddInt64(&p.metrics.WaitCount, 1)
		atomic.AddInt64(&p.metrics.WaitDuration, int64(time.Since(start)))
	}()

	conn, err := p.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}

	return conn, nil
}

// Execute executes a query with automatic connection management
func (p *ConnectionPool) Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	conn, err := p.GetConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	return conn.ExecContext(ctx, query, args...)
}

// Query executes a query and returns rows with automatic connection management
func (p *ConnectionPool) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns at most one row
func (p *ConnectionPool) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return p.db.QueryRowContext(ctx, query, args...)
}

// BeginTx starts a transaction
func (p *ConnectionPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("connection pool is closed")
	}
	p.mu.RUnlock()

	return p.db.BeginTx(ctx, opts)
}

// GetMetrics returns a copy of the current pool metrics
func (p *ConnectionPool) GetMetrics() PoolMetrics {
	stats := p.db.Stats()
	
	// Update metrics from sql.DB stats
	atomic.StoreInt64(&p.metrics.OpenConnections, int64(stats.OpenConnections))
	atomic.StoreInt64(&p.metrics.InUseConnections, int64(stats.InUse))
	atomic.StoreInt64(&p.metrics.IdleConnections, int64(stats.Idle))
	atomic.StoreInt64(&p.metrics.MaxIdleClosed, stats.MaxIdleClosed)
	atomic.StoreInt64(&p.metrics.MaxLifetimeClosed, stats.MaxLifetimeClosed)

	return PoolMetrics{
		OpenConnections:     atomic.LoadInt64(&p.metrics.OpenConnections),
		InUseConnections:    atomic.LoadInt64(&p.metrics.InUseConnections),
		IdleConnections:     atomic.LoadInt64(&p.metrics.IdleConnections),
		WaitCount:          atomic.LoadInt64(&p.metrics.WaitCount),
		WaitDuration:       atomic.LoadInt64(&p.metrics.WaitDuration),
		MaxIdleClosed:      atomic.LoadInt64(&p.metrics.MaxIdleClosed),
		MaxLifetimeClosed:  atomic.LoadInt64(&p.metrics.MaxLifetimeClosed),
		HealthCheckCount:   atomic.LoadInt64(&p.metrics.HealthCheckCount),
		HealthCheckFailed:  atomic.LoadInt64(&p.metrics.HealthCheckFailed),
	}
}

// GetUtilization returns the current pool utilization percentage
func (p *ConnectionPool) GetUtilization() float64 {
	stats := p.db.Stats()
	if p.maxOpen == 0 {
		return 0
	}
	return float64(stats.InUse) / float64(p.maxOpen) * 100
}

// healthCheckRoutine runs periodic health checks
func (p *ConnectionPool) healthCheckRoutine() {
	defer p.wg.Done()
	ticker := time.NewTicker(p.healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			atomic.AddInt64(&p.metrics.HealthCheckCount, 1)
			if err := p.healthCheck(p.db); err != nil {
				atomic.AddInt64(&p.metrics.HealthCheckFailed, 1)
				// Log error in production, for now we'll just increment counter
			}
		case <-p.stopHealthCheck:
			return
		}
	}
}

// Close closes the connection pool and all its connections
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	// Stop health check routine
	close(p.stopHealthCheck)
	p.wg.Wait()

	// Close the database
	return p.db.Close()
}

// Ping verifies the database connection is still alive
func (p *ConnectionPool) Ping(ctx context.Context) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("connection pool is closed")
	}
	p.mu.RUnlock()

	return p.db.PingContext(ctx)
}
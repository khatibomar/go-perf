package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// MockDriver implements a simple mock database driver for testing
type MockDriver struct{}

func (d MockDriver) Open(name string) (driver.Conn, error) {
	return &MockConn{}, nil
}

type MockConn struct{}

func (c *MockConn) Prepare(query string) (driver.Stmt, error) {
	return &MockStmt{}, nil
}

func (c *MockConn) Close() error {
	return nil
}

func (c *MockConn) Begin() (driver.Tx, error) {
	return &MockTx{}, nil
}

type MockStmt struct{}

func (s *MockStmt) Close() error {
	return nil
}

func (s *MockStmt) NumInput() int {
	return -1 // -1 means the driver doesn't know the number of placeholders
}

func (s *MockStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &MockResult{}, nil
}

func (s *MockStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &MockRows{}, nil
}

type MockResult struct{}

func (r *MockResult) LastInsertId() (int64, error) {
	return 1, nil
}

func (r *MockResult) RowsAffected() (int64, error) {
	return 1, nil
}

type MockRows struct {
	closed bool
}

func (r *MockRows) Columns() []string {
	return []string{"id", "name"}
}

func (r *MockRows) Close() error {
	r.closed = true
	return nil
}

func (r *MockRows) Next(dest []driver.Value) error {
	if r.closed {
		return fmt.Errorf("rows closed")
	}
	return fmt.Errorf("no more rows")
}

type MockTx struct{}

func (t *MockTx) Commit() error {
	return nil
}

func (t *MockTx) Rollback() error {
	return nil
}

func init() {
	sql.Register("mock", &MockDriver{})
}

func TestNewConnectionPool(t *testing.T) {
	config := PoolConfig{
		DriverName:      "mock",
		DataSourceName:  "test",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
		HealthInterval:  10 * time.Second,
	}

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	if pool.maxOpen != 10 {
		t.Errorf("Expected maxOpen to be 10, got %d", pool.maxOpen)
	}
	if pool.maxIdle != 5 {
		t.Errorf("Expected maxIdle to be 5, got %d", pool.maxIdle)
	}
}

func TestConnectionPoolDefaults(t *testing.T) {
	config := PoolConfig{
		DriverName:     "mock",
		DataSourceName: "test",
	}

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	if pool.maxOpen != 25 {
		t.Errorf("Expected default maxOpen to be 25, got %d", pool.maxOpen)
	}
	if pool.maxIdle != 5 {
		t.Errorf("Expected default maxIdle to be 5, got %d", pool.maxIdle)
	}
	if pool.maxLifetime != time.Hour {
		t.Errorf("Expected default maxLifetime to be 1 hour, got %v", pool.maxLifetime)
	}
}

func TestGetConnection(t *testing.T) {
	config := PoolConfig{
		DriverName:     "mock",
		DataSourceName: "test",
		MaxOpenConns:   5,
		MaxIdleConns:   2,
	}

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	conn, err := pool.GetConnection(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}
	defer conn.Close()

	// Verify metrics are updated
	metrics := pool.GetMetrics()
	if metrics.WaitCount == 0 {
		t.Error("Expected WaitCount to be incremented")
	}
}

func TestConnectionPoolMetrics(t *testing.T) {
	config := PoolConfig{
		DriverName:     "mock",
		DataSourceName: "test",
		MaxOpenConns:   3,
		MaxIdleConns:   1,
	}

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Get multiple connections
	conns := make([]*sql.Conn, 3)
	for i := 0; i < 3; i++ {
		conns[i], err = pool.GetConnection(ctx)
		if err != nil {
			t.Fatalf("Failed to get connection %d: %v", i, err)
		}
	}

	// Check metrics
	metrics := pool.GetMetrics()
	if metrics.WaitCount != 3 {
		t.Errorf("Expected WaitCount to be 3, got %d", metrics.WaitCount)
	}

	// Check utilization
	utilization := pool.GetUtilization()
	if utilization <= 0 {
		t.Errorf("Expected utilization to be > 0, got %f", utilization)
	}

	// Close connections
	for _, conn := range conns {
		conn.Close()
	}
}

func TestHealthCheck(t *testing.T) {
	failingHealthCheck := func(db *sql.DB) error {
		return fmt.Errorf("health check failed")
	}

	config := PoolConfig{
		DriverName:     "mock",
		DataSourceName: "test",
		HealthInterval: 100 * time.Millisecond,
		HealthCheck:    failingHealthCheck,
	}

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	// Wait for health checks to run
	time.Sleep(250 * time.Millisecond)

	metrics := pool.GetMetrics()
	if metrics.HealthCheckCount == 0 {
		t.Error("Expected health checks to have run")
	}
	if metrics.HealthCheckFailed == 0 {
		t.Error("Expected health checks to have failed")
	}
}

func TestConnectionPoolClose(t *testing.T) {
	config := PoolConfig{
		DriverName:     "mock",
		DataSourceName: "test",
	}

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}

	err = pool.Close()
	if err != nil {
		t.Fatalf("Failed to close connection pool: %v", err)
	}

	// Verify pool is closed
	ctx := context.Background()
	_, err = pool.GetConnection(ctx)
	if err == nil {
		t.Error("Expected error when getting connection from closed pool")
	}
}

func TestConcurrentConnections(t *testing.T) {
	config := PoolConfig{
		DriverName:     "mock",
		DataSourceName: "test",
		MaxOpenConns:   10,
		MaxIdleConns:   5,
	}

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	var wg sync.WaitGroup
	errorChan := make(chan error, 20)

	// Launch 20 goroutines to get connections concurrently
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := pool.GetConnection(ctx)
			if err != nil {
				errorChan <- err
				return
			}
			defer conn.Close()

			// Simulate some work
			time.Sleep(10 * time.Millisecond)
		}()
	}

	wg.Wait()
	close(errorChan)

	// Check for errors
	for err := range errorChan {
		t.Errorf("Concurrent connection error: %v", err)
	}

	// Verify metrics
	metrics := pool.GetMetrics()
	if metrics.WaitCount != 20 {
		t.Errorf("Expected WaitCount to be 20, got %d", metrics.WaitCount)
	}
}

func TestExecuteQuery(t *testing.T) {
	config := PoolConfig{
		DriverName:     "mock",
		DataSourceName: "test",
	}

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	_, err = pool.Execute(ctx, "INSERT INTO test (name) VALUES (?)", "test")
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}
}

func TestTransaction(t *testing.T) {
	config := PoolConfig{
		DriverName:     "mock",
		DataSourceName: "test",
	}

	pool, err := NewConnectionPool(config)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	tx, err := pool.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}
}

// Benchmark tests
func BenchmarkGetConnection(b *testing.B) {
	config := PoolConfig{
		DriverName:     "mock",
		DataSourceName: "test",
		MaxOpenConns:   25,
		MaxIdleConns:   10,
	}

	pool, err := NewConnectionPool(config)
	if err != nil {
		b.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, err := pool.GetConnection(ctx)
			if err != nil {
				b.Errorf("Failed to get connection: %v", err)
				continue
			}
			conn.Close()
		}
	})
}

func BenchmarkExecuteQuery(b *testing.B) {
	config := PoolConfig{
		DriverName:     "mock",
		DataSourceName: "test",
		MaxOpenConns:   25,
		MaxIdleConns:   10,
	}

	pool, err := NewConnectionPool(config)
	if err != nil {
		b.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := pool.Execute(ctx, "SELECT 1", nil)
			if err != nil {
				b.Errorf("Failed to execute query: %v", err)
			}
		}
	})
}

func BenchmarkConcurrentConnections(b *testing.B) {
	config := PoolConfig{
		DriverName:     "mock",
		DataSourceName: "test",
		MaxOpenConns:   50,
		MaxIdleConns:   20,
	}

	pool, err := NewConnectionPool(config)
	if err != nil {
		b.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, err := pool.GetConnection(ctx)
			if err != nil {
				b.Errorf("Failed to get connection: %v", err)
				continue
			}
			// Simulate work
			_, err = conn.ExecContext(ctx, "SELECT 1")
			if err != nil {
				b.Errorf("Failed to execute query: %v", err)
			}
			conn.Close()
		}
	})
}

// Test with sqlmock for more realistic database testing
func TestWithSQLMock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Mock ping for health check
	mock.ExpectPing()

	// Create a pool using the mock database
	pool := &ConnectionPool{
		db:              db,
		maxOpen:         10,
		maxIdle:         5,
		maxLifetime:     time.Hour,
		maxIdleTime:     30 * time.Minute,
		healthCheck:     DefaultHealthCheck,
		healthInterval:  time.Minute,
		metrics:         &PoolMetrics{},
		stopHealthCheck: make(chan struct{}),
	}

	// Test ping
	ctx := context.Background()
	err = pool.Ping(ctx)
	if err != nil {
		t.Fatalf("Failed to ping: %v", err)
	}

	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}
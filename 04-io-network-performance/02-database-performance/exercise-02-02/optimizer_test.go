package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// MockDriver implements driver.Driver for testing
type MockDriver struct{}

func (d MockDriver) Open(name string) (driver.Conn, error) {
	return &MockConn{}, nil
}

// MockConn implements driver.Conn
type MockConn struct{}

func (c *MockConn) Prepare(query string) (driver.Stmt, error) {
	return &MockStmt{query: query}, nil
}

func (c *MockConn) Close() error {
	return nil
}

func (c *MockConn) Begin() (driver.Tx, error) {
	return &MockTx{}, nil
}

// MockStmt implements driver.Stmt
type MockStmt struct {
	query string
}

func (s *MockStmt) Close() error {
	return nil
}

func (s *MockStmt) NumInput() int {
	return -1 // Unknown number of placeholders
}

func (s *MockStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &MockResult{}, nil
}

func (s *MockStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &MockDriverRows{}, nil
}

// MockResult implements driver.Result
type MockResult struct{}

func (r *MockResult) LastInsertId() (int64, error) {
	return 1, nil
}

func (r *MockResult) RowsAffected() (int64, error) {
	return 1, nil
}

// MockDriverRows implements driver.Rows
type MockDriverRows struct {
	current int
}

func (r *MockDriverRows) Columns() []string {
	return []string{"id", "name", "email"}
}

func (r *MockDriverRows) Close() error {
	return nil
}

func (r *MockDriverRows) Next(dest []driver.Value) error {
	if r.current >= 2 {
		return io.EOF
	}
	r.current++
	dest[0] = int64(r.current)
	dest[1] = "user" + string(rune('0'+r.current))
	dest[2] = "user" + string(rune('0'+r.current)) + "@example.com"
	return nil
}

// MockTx implements driver.Tx
type MockTx struct{}

func (t *MockTx) Commit() error {
	return nil
}

func (t *MockTx) Rollback() error {
	return nil
}

func TestQueryOptimizerDefaults(t *testing.T) {
	sql.Register("mock", MockDriver{})
	db, err := sql.Open("mock", "test")
	if err != nil {
		t.Fatalf("Failed to open mock database: %v", err)
	}
	defer db.Close()

	optimizer := NewQueryOptimizer(db, nil)
	if optimizer == nil {
		t.Fatal("Expected optimizer to be created")
	}

	if optimizer.db != db {
		t.Error("Expected optimizer to use provided database")
	}

	if optimizer.stmtCache == nil {
		t.Error("Expected statement cache to be initialized")
	}

	if optimizer.resultCache == nil {
		t.Error("Expected result cache to be initialized")
	}

	if optimizer.metrics == nil {
		t.Error("Expected metrics to be initialized")
	}
}

func TestQueryOptimizerCustomConfig(t *testing.T) {
	sql.Register("mock2", MockDriver{})
	db, err := sql.Open("mock2", "test")
	if err != nil {
		t.Fatalf("Failed to open mock database: %v", err)
	}
	defer db.Close()

	config := &OptimizerConfig{
		CacheExpiration:    1 * time.Minute,
		CacheCleanup:       2 * time.Minute,
		SlowQueryThreshold: 5 * time.Millisecond,
		MaxCacheSize:       500,
	}

	optimizer := NewQueryOptimizer(db, config)
	if optimizer.slowQueryThreshold != config.SlowQueryThreshold {
		t.Errorf("Expected slow query threshold %v, got %v", config.SlowQueryThreshold, optimizer.slowQueryThreshold)
	}
}

func TestPrepareQuery(t *testing.T) {
	sql.Register("mock3", MockDriver{})
	db, err := sql.Open("mock3", "test")
	if err != nil {
		t.Fatalf("Failed to open mock database: %v", err)
	}
	defer db.Close()

	optimizer := NewQueryOptimizer(db, nil)
	defer optimizer.Close()

	query := "SELECT * FROM users WHERE id = ?"

	// First call should prepare the statement
	stmt1, err := optimizer.PrepareQuery(query)
	if err != nil {
		t.Fatalf("Failed to prepare query: %v", err)
	}

	// Second call should return cached statement
	stmt2, err := optimizer.PrepareQuery(query)
	if err != nil {
		t.Fatalf("Failed to prepare query: %v", err)
	}

	if stmt1 != stmt2 {
		t.Error("Expected same statement instance from cache")
	}

	metrics := optimizer.GetMetrics()
	if metrics.PreparedStatements != 2 {
		t.Errorf("Expected 2 prepared statements, got %d", metrics.PreparedStatements)
	}
}

func TestExecWithCache(t *testing.T) {
	sql.Register("mock4", MockDriver{})
	db, err := sql.Open("mock4", "test")
	if err != nil {
		t.Fatalf("Failed to open mock database: %v", err)
	}
	defer db.Close()

	optimizer := NewQueryOptimizer(db, nil)
	defer optimizer.Close()

	ctx := context.Background()
	query := "INSERT INTO users (name, email) VALUES (?, ?)"

	result, err := optimizer.ExecWithCache(ctx, query, "John Doe", "john@example.com")
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("Failed to get rows affected: %v", err)
	}

	if rowsAffected != 1 {
		t.Errorf("Expected 1 row affected, got %d", rowsAffected)
	}

	metrics := optimizer.GetMetrics()
	if metrics.TotalQueries != 1 {
		t.Errorf("Expected 1 total query, got %d", metrics.TotalQueries)
	}
}

func TestQueryWithCacheMiss(t *testing.T) {
	sql.Register("mock5", MockDriver{})
	db, err := sql.Open("mock5", "test")
	if err != nil {
		t.Fatalf("Failed to open mock database: %v", err)
	}
	defer db.Close()

	optimizer := NewQueryOptimizer(db, nil)
	defer optimizer.Close()

	ctx := context.Background()
	query := "SELECT * FROM users WHERE id = ?"
	cacheKey := "user:1"

	_, err = optimizer.QueryWithCache(ctx, cacheKey, query, 1)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	metrics := optimizer.GetMetrics()
	if metrics.CacheMisses != 1 {
		t.Errorf("Expected 1 cache miss, got %d", metrics.CacheMisses)
	}

	if metrics.CacheHits != 0 {
		t.Errorf("Expected 0 cache hits, got %d", metrics.CacheHits)
	}
}

func TestAnalyzeQuery(t *testing.T) {
	sql.Register("mock6", MockDriver{})
	db, err := sql.Open("mock6", "test")
	if err != nil {
		t.Fatalf("Failed to open mock database: %v", err)
	}
	defer db.Close()

	optimizer := NewQueryOptimizer(db, nil)
	defer optimizer.Close()

	ctx := context.Background()
	query := "SELECT * FROM users WHERE id = ?"

	plan, err := optimizer.AnalyzeQuery(ctx, query, 1)
	if err != nil {
		t.Fatalf("Failed to analyze query: %v", err)
	}

	if plan.Query != query {
		t.Errorf("Expected query %s, got %s", query, plan.Query)
	}

	if plan.Cost <= 0 {
		t.Error("Expected positive cost")
	}

	if plan.Rows <= 0 {
		t.Error("Expected positive row estimate")
	}
}

func TestGetCacheHitRate(t *testing.T) {
	sql.Register("mock7", MockDriver{})
	db, err := sql.Open("mock7", "test")
	if err != nil {
		t.Fatalf("Failed to open mock database: %v", err)
	}
	defer db.Close()

	optimizer := NewQueryOptimizer(db, nil)
	defer optimizer.Close()

	// Initially should be 0%
	hitRate := optimizer.GetCacheHitRate()
	if hitRate != 0.0 {
		t.Errorf("Expected 0%% hit rate, got %.2f%%", hitRate)
	}

	// Simulate cache misses
	ctx := context.Background()
	query := "SELECT * FROM users WHERE id = ?"
	optimizer.QueryWithCache(ctx, "key1", query, 1)
	optimizer.QueryWithCache(ctx, "key2", query, 2)

	hitRate = optimizer.GetCacheHitRate()
	if hitRate != 0.0 {
		t.Errorf("Expected 0%% hit rate after misses, got %.2f%%", hitRate)
	}
}

func TestGetPreparedStatementReuseRate(t *testing.T) {
	sql.Register("mock8", MockDriver{})
	db, err := sql.Open("mock8", "test")
	if err != nil {
		t.Fatalf("Failed to open mock database: %v", err)
	}
	defer db.Close()

	optimizer := NewQueryOptimizer(db, nil)
	defer optimizer.Close()

	// Initially should be 0%
	reuseRate := optimizer.GetPreparedStatementReuseRate()
	if reuseRate != 0.0 {
		t.Errorf("Expected 0%% reuse rate, got %.2f%%", reuseRate)
	}

	// Execute same query multiple times
	ctx := context.Background()
	query := "SELECT * FROM users WHERE id = ?"
	optimizer.ExecWithCache(ctx, query, 1)
	optimizer.ExecWithCache(ctx, query, 2)

	reuseRate = optimizer.GetPreparedStatementReuseRate()
	if reuseRate != 100.0 {
		t.Errorf("Expected 100%% reuse rate, got %.2f%%", reuseRate)
	}
}

func TestOptimizerClose(t *testing.T) {
	sql.Register("mock9", MockDriver{})
	db, err := sql.Open("mock9", "test")
	if err != nil {
		t.Fatalf("Failed to open mock database: %v", err)
	}
	defer db.Close()

	optimizer := NewQueryOptimizer(db, nil)

	// Prepare some statements
	optimizer.PrepareQuery("SELECT * FROM users")
	optimizer.PrepareQuery("SELECT * FROM orders")

	err = optimizer.Close()
	if err != nil {
		t.Fatalf("Failed to close optimizer: %v", err)
	}

	// Cache should be empty after close
	if len(optimizer.stmtCache) != 0 {
		t.Error("Expected statement cache to be empty after close")
	}
}

var benchOnce sync.Once

// Benchmark tests
func BenchmarkPrepareQuery(b *testing.B) {
	benchOnce.Do(func() {
		sql.Register("mockbench", MockDriver{})
	})
	db, err := sql.Open("mockbench", "test")
	if err != nil {
		b.Fatalf("Failed to open mock database: %v", err)
	}
	defer db.Close()

	optimizer := NewQueryOptimizer(db, nil)
	defer optimizer.Close()

	query := "SELECT * FROM users WHERE id = ?"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := optimizer.PrepareQuery(query)
		if err != nil {
			b.Fatalf("Failed to prepare query: %v", err)
		}
	}
}

func BenchmarkExecWithCache(b *testing.B) {
	benchOnce.Do(func() {
		sql.Register("mockbench", MockDriver{})
	})
	db, err := sql.Open("mockbench", "test")
	if err != nil {
		b.Fatalf("Failed to open mock database: %v", err)
	}
	defer db.Close()

	optimizer := NewQueryOptimizer(db, nil)
	defer optimizer.Close()

	ctx := context.Background()
	query := "INSERT INTO users (name, email) VALUES (?, ?)"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := optimizer.ExecWithCache(ctx, query, "user", "user@example.com")
		if err != nil {
			b.Fatalf("Failed to execute query: %v", err)
		}
	}
}

func BenchmarkQueryWithCache(b *testing.B) {
	benchOnce.Do(func() {
		sql.Register("mockbench", MockDriver{})
	})
	db, err := sql.Open("mockbench", "test")
	if err != nil {
		b.Fatalf("Failed to open mock database: %v", err)
	}
	defer db.Close()

	optimizer := NewQueryOptimizer(db, nil)
	defer optimizer.Close()

	ctx := context.Background()
	query := "SELECT * FROM users WHERE id = ?"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cacheKey := "user:1"
		_, err := optimizer.QueryWithCache(ctx, cacheKey, query, 1)
		if err != nil {
			b.Fatalf("Failed to execute query: %v", err)
		}
		// Don't close rows in benchmark to avoid mock driver issues
	}
}

// Test with go-sqlmock for more realistic testing
func TestWithSQLMock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	optimizer := NewQueryOptimizer(db, nil)
	defer optimizer.Close()

	// Set up expectations
	mock.ExpectPrepare("SELECT \\* FROM users WHERE id = \\?").ExpectQuery().WithArgs(1).WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "email"}).AddRow(1, "John Doe", "john@example.com"),
	)

	ctx := context.Background()
	query := "SELECT * FROM users WHERE id = ?"

	rows, err := optimizer.QueryWithCache(ctx, "user:1", query, 1)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}
	if rows != nil {
		// Don't close rows here as it may cause issues with mock
		// rows.Close()
	}

	// Verify expectations
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}
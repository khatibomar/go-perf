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

// MockDriver for testing
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
	return -1
}

func (s *MockStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &MockResult{}, nil
}

func (s *MockStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &MockRows{}, nil
}

type MockTx struct{}

func (tx *MockTx) Commit() error {
	return nil
}

func (tx *MockTx) Rollback() error {
	return nil
}

type MockResult struct{}

func (r *MockResult) LastInsertId() (int64, error) {
	return 1, nil
}

func (r *MockResult) RowsAffected() (int64, error) {
	return 1, nil
}

type MockRows struct{}

func (r *MockRows) Columns() []string {
	return []string{"id"}
}

func (r *MockRows) Close() error {
	return nil
}

func (r *MockRows) Next(dest []driver.Value) error {
	return fmt.Errorf("no more rows")
}

var registerOnce sync.Once

func setupMockDB() (*sql.DB, sqlmock.Sqlmock, error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, err
	}
	return db, mock, nil
}

func TestBatchProcessorDefaults(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	bp := NewBatchProcessor(db, nil)
	if bp == nil {
		t.Fatal("Expected non-nil BatchProcessor")
	}
	
	if bp.config.BatchSize != 1000 {
		t.Errorf("Expected default batch size 1000, got %d", bp.config.BatchSize)
	}
	
	if bp.config.FlushInterval != 100*time.Millisecond {
		t.Errorf("Expected default flush interval 100ms, got %v", bp.config.FlushInterval)
	}
	
	bp.Close()
}

func TestBatchProcessorCustomConfig(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	config := &BatchConfig{
		BatchSize:     500,
		FlushInterval: 50 * time.Millisecond,
		MaxRetries:    5,
	}
	
	bp := NewBatchProcessor(db, config)
	if bp.config.BatchSize != 500 {
		t.Errorf("Expected batch size 500, got %d", bp.config.BatchSize)
	}
	
	bp.Close()
}

func TestBatchProcessorExecute(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	
	bp := NewBatchProcessor(db, &BatchConfig{
		BatchSize:     1,
		FlushInterval: 10 * time.Millisecond,
	})
	defer bp.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	result, err := bp.Execute(ctx, "INSERT INTO users (name) VALUES (?)", "John")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	
	if result.Error != nil {
		t.Errorf("Expected no error, got %v", result.Error)
	}
	
	if result.RowsAffected != 1 {
		t.Errorf("Expected 1 row affected, got %d", result.RowsAffected)
	}
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestBatchProcessorExecuteWithPriority(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations for multiple operations
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()
	
	bp := NewBatchProcessor(db, &BatchConfig{
		BatchSize:     2,
		FlushInterval: 10 * time.Millisecond,
	})
	defer bp.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Execute low priority operation first
	var wg sync.WaitGroup
	wg.Add(2)
	
	var result1, result2 *BatchResult
	var err1, err2 error
	
	go func() {
		defer wg.Done()
		result1, err1 = bp.ExecuteWithPriority(ctx, 1, "INSERT INTO users (name) VALUES (?)", "Low Priority")
	}()
	
	go func() {
		defer wg.Done()
		result2, err2 = bp.ExecuteWithPriority(ctx, 10, "INSERT INTO users (name) VALUES (?)", "High Priority")
	}()
	
	wg.Wait()
	
	if err1 != nil {
		t.Errorf("Execute 1 failed: %v", err1)
	}
	if err2 != nil {
		t.Errorf("Execute 2 failed: %v", err2)
	}
	
	if result1 != nil && result1.Error != nil {
		t.Errorf("Expected no error for result1, got %v", result1.Error)
	}
	if result2 != nil && result2.Error != nil {
		t.Errorf("Expected no error for result2, got %v", result2.Error)
	}
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestBatchProcessorFlush(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	
	bp := NewBatchProcessor(db, &BatchConfig{
		BatchSize:     1000, // Large batch size to prevent auto-flush
		FlushInterval: 1 * time.Hour, // Long interval to prevent timer flush
	})
	defer bp.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Add an operation but don't trigger auto-flush
	go func() {
		bp.Execute(ctx, "INSERT INTO users (name) VALUES (?)", "John")
	}()
	
	// Wait a bit to ensure the operation is buffered
	time.Sleep(10 * time.Millisecond)
	
	// Manually flush
	err = bp.Flush()
	if err != nil {
		t.Errorf("Flush failed: %v", err)
	}
	
	// Wait for flush to complete
	time.Sleep(100 * time.Millisecond)
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestBatchProcessorMetrics(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	
	bp := NewBatchProcessor(db, &BatchConfig{
		BatchSize:     1,
		FlushInterval: 10 * time.Millisecond,
		EnableMetrics: true,
	})
	defer bp.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Execute an operation
	_, err = bp.Execute(ctx, "INSERT INTO users (name) VALUES (?)", "John")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	
	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	
	// Check metrics
	metrics := bp.GetMetrics()
	if metrics.TotalBatches == 0 {
		t.Error("Expected at least 1 batch to be processed")
	}
	
	if metrics.TotalOperations == 0 {
		t.Error("Expected at least 1 operation to be processed")
	}
	
	if metrics.SuccessfulBatches == 0 {
		t.Error("Expected at least 1 successful batch")
	}
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestBatchProcessorThroughput(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	bp := NewBatchProcessor(db, &BatchConfig{
		BatchSize:     100,
		FlushInterval: 10 * time.Millisecond,
		EnableMetrics: true,
	})
	defer bp.Close()
	
	// Initial throughput should be 0
	throughput := bp.GetThroughput()
	if throughput != 0 {
		t.Errorf("Expected initial throughput 0, got %d", throughput)
	}
	
	// Average commit time should be 0 initially
	avgTime := bp.GetAverageCommitTime()
	if avgTime != 0 {
		t.Errorf("Expected initial average commit time 0, got %v", avgTime)
	}
	
	// Deadlock rate should be 0 initially
	deadlockRate := bp.GetDeadlockRate()
	if deadlockRate != 0 {
		t.Errorf("Expected initial deadlock rate 0, got %f", deadlockRate)
	}
}

func TestBatchProcessorClose(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	bp := NewBatchProcessor(db, nil)
	
	// Close should succeed
	err = bp.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
	
	// Second close should fail
	err = bp.Close()
	if err == nil {
		t.Error("Expected error on second close")
	}
}

func TestBatchProcessorConcurrency(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations for multiple operations
	for i := 0; i < 10; i++ {
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(int64(i+1), 1))
		mock.ExpectCommit()
	}
	
	bp := NewBatchProcessor(db, &BatchConfig{
		BatchSize:          1,
		FlushInterval:      10 * time.Millisecond,
		MaxConcurrentBatch: 5,
	})
	defer bp.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Execute multiple operations concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := bp.Execute(ctx, "INSERT INTO users (name) VALUES (?)", fmt.Sprintf("User%d", id))
			if err != nil {
				t.Errorf("Execute %d failed: %v", id, err)
			}
		}(i)
	}
	
	wg.Wait()
	
	// Wait for all operations to complete
	time.Sleep(200 * time.Millisecond)
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func BenchmarkBatchProcessor(b *testing.B) {
	registerOnce.Do(func() {
		sql.Register("mockbench", MockDriver{})
	})
	
	db, err := sql.Open("mockbench", "test")
	if err != nil {
		b.Fatalf("Failed to open mock DB: %v", err)
	}
	defer db.Close()
	
	bp := NewBatchProcessor(db, &BatchConfig{
		BatchSize:     1000,
		FlushInterval: 10 * time.Millisecond,
	})
	defer bp.Close()
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bp.Execute(ctx, "INSERT INTO users (name) VALUES (?)", "BenchUser")
		}
	})
}

func BenchmarkBatchProcessorWithPriority(b *testing.B) {
	registerOnce.Do(func() {
		sql.Register("mockbench2", MockDriver{})
	})
	
	db, err := sql.Open("mockbench2", "test")
	if err != nil {
		b.Fatalf("Failed to open mock DB: %v", err)
	}
	defer db.Close()
	
	bp := NewBatchProcessor(db, &BatchConfig{
		BatchSize:     1000,
		FlushInterval: 10 * time.Millisecond,
	})
	defer bp.Close()
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bp.ExecuteWithPriority(ctx, 5, "INSERT INTO users (name) VALUES (?)", "BenchUser")
		}
	})
}
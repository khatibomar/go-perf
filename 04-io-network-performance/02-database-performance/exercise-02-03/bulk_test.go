package main

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestBulkProcessorDefaults(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	bp := NewBulkProcessor(db, nil)
	if bp == nil {
		t.Fatal("Expected non-nil BulkProcessor")
	}
	
	config := bp.GetConfig()
	if config.ChunkSize != 1000 {
		t.Errorf("Expected default chunk size 1000, got %d", config.ChunkSize)
	}
	
	if config.MaxConcurrency != 5 {
		t.Errorf("Expected default max concurrency 5, got %d", config.MaxConcurrency)
	}
	
	if config.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", config.Timeout)
	}
}

func TestBulkProcessorCustomConfig(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	config := &BulkConfig{
		ChunkSize:      500,
		MaxConcurrency: 3,
		Timeout:        15 * time.Second,
		RetryAttempts:  5,
	}
	
	bp := NewBulkProcessor(db, config)
	if bp.config.ChunkSize != 500 {
		t.Errorf("Expected chunk size 500, got %d", bp.config.ChunkSize)
	}
	
	if bp.config.MaxConcurrency != 3 {
		t.Errorf("Expected max concurrency 3, got %d", bp.config.MaxConcurrency)
	}
}

func TestBulkInsert(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations for bulk insert
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(1, 3))
	
	bp := NewBulkProcessor(db, &BulkConfig{
		ChunkSize:       3,
		MaxConcurrency:  1,
		OptimizeQueries: true,
	})
	
	ctx := context.Background()
	columns := []string{"name", "email"}
	data := [][]interface{}{
		{"John", "john@example.com"},
		{"Jane", "jane@example.com"},
		{"Bob", "bob@example.com"},
	}
	
	result, err := bp.BulkInsert(ctx, "users", columns, data)
	if err != nil {
		t.Fatalf("BulkInsert failed: %v", err)
	}
	
	if result.TotalRows != 3 {
		t.Errorf("Expected 3 total rows, got %d", result.TotalRows)
	}
	
	if result.AffectedRows != 3 {
		t.Errorf("Expected 3 affected rows, got %d", result.AffectedRows)
	}
	
	if result.ProcessedChunks != 1 {
		t.Errorf("Expected 1 processed chunk, got %d", result.ProcessedChunks)
	}
	
	if result.FailedChunks != 0 {
		t.Errorf("Expected 0 failed chunks, got %d", result.FailedChunks)
	}
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestBulkInsertEmptyData(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	bp := NewBulkProcessor(db, nil)
	
	ctx := context.Background()
	columns := []string{"name", "email"}
	data := [][]interface{}{}
	
	result, err := bp.BulkInsert(ctx, "users", columns, data)
	if err != nil {
		t.Fatalf("BulkInsert failed: %v", err)
	}
	
	if result.TotalRows != 0 {
		t.Errorf("Expected 0 total rows, got %d", result.TotalRows)
	}
	
	if result.AffectedRows != 0 {
		t.Errorf("Expected 0 affected rows, got %d", result.AffectedRows)
	}
}

func TestBulkInsertMultipleChunks(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations for multiple chunks
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(1, 2))
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(3, 2))
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(5, 1))
	
	bp := NewBulkProcessor(db, &BulkConfig{
		ChunkSize:       2, // Small chunk size to force multiple chunks
		MaxConcurrency:  1,
		OptimizeQueries: true,
	})
	
	ctx := context.Background()
	columns := []string{"name", "email"}
	data := [][]interface{}{
		{"John", "john@example.com"},
		{"Jane", "jane@example.com"},
		{"Bob", "bob@example.com"},
		{"Alice", "alice@example.com"},
		{"Charlie", "charlie@example.com"},
	}
	
	result, err := bp.BulkInsert(ctx, "users", columns, data)
	if err != nil {
		t.Fatalf("BulkInsert failed: %v", err)
	}
	
	if result.TotalRows != 5 {
		t.Errorf("Expected 5 total rows, got %d", result.TotalRows)
	}
	
	if result.AffectedRows != 5 {
		t.Errorf("Expected 5 affected rows, got %d", result.AffectedRows)
	}
	
	if result.ProcessedChunks != 3 {
		t.Errorf("Expected 3 processed chunks, got %d", result.ProcessedChunks)
	}
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestBulkUpdate(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations for bulk update
	mock.ExpectExec("UPDATE users SET").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE users SET").WillReturnResult(sqlmock.NewResult(0, 1))
	
	bp := NewBulkProcessor(db, &BulkConfig{
		ChunkSize:      10,
		MaxConcurrency: 1,
		UseTransaction: false,
	})
	
	ctx := context.Background()
	updates := []BulkUpdateItem{
		{
			SetClause:   "name = ?, email = ?",
			SetValues:   []interface{}{"John Updated", "john.updated@example.com"},
			WhereClause: "id = ?",
			WhereValues: []interface{}{1},
		},
		{
			SetClause:   "name = ?, email = ?",
			SetValues:   []interface{}{"Jane Updated", "jane.updated@example.com"},
			WhereClause: "id = ?",
			WhereValues: []interface{}{2},
		},
	}
	
	result, err := bp.BulkUpdate(ctx, "users", updates)
	if err != nil {
		t.Fatalf("BulkUpdate failed: %v", err)
	}
	
	if result.TotalRows != 2 {
		t.Errorf("Expected 2 total rows, got %d", result.TotalRows)
	}
	
	if result.AffectedRows != 2 {
		t.Errorf("Expected 2 affected rows, got %d", result.AffectedRows)
	}
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestBulkDelete(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations for bulk delete with IN clause
	mock.ExpectExec("DELETE FROM users WHERE id IN").WillReturnResult(sqlmock.NewResult(0, 3))
	
	bp := NewBulkProcessor(db, &BulkConfig{
		ChunkSize:       10,
		MaxConcurrency:  1,
		OptimizeQueries: true,
	})
	
	ctx := context.Background()
	values := []interface{}{1, 2, 3}
	
	result, err := bp.BulkDelete(ctx, "users", "id", values)
	if err != nil {
		t.Fatalf("BulkDelete failed: %v", err)
	}
	
	if result.TotalRows != 3 {
		t.Errorf("Expected 3 total rows, got %d", result.TotalRows)
	}
	
	if result.AffectedRows != 3 {
		t.Errorf("Expected 3 affected rows, got %d", result.AffectedRows)
	}
	
	if result.ProcessedChunks != 1 {
		t.Errorf("Expected 1 processed chunk, got %d", result.ProcessedChunks)
	}
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestBulkUpsert(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations for upsert
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(1, 2))
	
	bp := NewBulkProcessor(db, &BulkConfig{
		ChunkSize:      10,
		MaxConcurrency: 1,
	})
	
	ctx := context.Background()
	columns := []string{"id", "name", "email"}
	data := [][]interface{}{
		{1, "John", "john@example.com"},
		{2, "Jane", "jane@example.com"},
	}
	conflictColumns := []string{"id"}
	
	result, err := bp.BulkUpsert(ctx, "users", columns, data, conflictColumns)
	if err != nil {
		t.Fatalf("BulkUpsert failed: %v", err)
	}
	
	if result.TotalRows != 2 {
		t.Errorf("Expected 2 total rows, got %d", result.TotalRows)
	}
	
	if result.AffectedRows != 2 {
		t.Errorf("Expected 2 affected rows, got %d", result.AffectedRows)
	}
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestBulkProcessorSetConfig(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	bp := NewBulkProcessor(db, nil)
	
	newConfig := &BulkConfig{
		ChunkSize:      2000,
		MaxConcurrency: 10,
		Timeout:        60 * time.Second,
	}
	
	bp.SetConfig(newConfig)
	
	config := bp.GetConfig()
	if config.ChunkSize != 2000 {
		t.Errorf("Expected chunk size 2000, got %d", config.ChunkSize)
	}
	
	if config.MaxConcurrency != 10 {
		t.Errorf("Expected max concurrency 10, got %d", config.MaxConcurrency)
	}
	
	if config.Timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", config.Timeout)
	}
}

func TestGenerateInsertQuery(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	bp := NewBulkProcessor(db, &BulkConfig{OptimizeQueries: true})
	
	columns := []string{"name", "email"}
	query := bp.generateInsertQuery("users", columns, 2)
	
	expected := "INSERT INTO users (name,email) VALUES (?,?),(?,?)"
	if query != expected {
		t.Errorf("Expected query %q, got %q", expected, query)
	}
}

func TestGenerateUpsertQuery(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	bp := NewBulkProcessor(db, nil)
	
	columns := []string{"id", "name", "email"}
	conflictColumns := []string{"id"}
	query := bp.generateUpsertQuery("users", columns, conflictColumns, 1)
	
	// Should contain INSERT ... ON CONFLICT ... DO UPDATE
	if !contains(query, "INSERT INTO users") {
		t.Error("Query should contain INSERT INTO users")
	}
	if !contains(query, "ON CONFLICT") {
		t.Error("Query should contain ON CONFLICT")
	}
	if !contains(query, "DO UPDATE SET") {
		t.Error("Query should contain DO UPDATE SET")
	}
}

func TestChunkData(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	bp := NewBulkProcessor(db, nil)
	
	data := [][]interface{}{
		{1, "John"},
		{2, "Jane"},
		{3, "Bob"},
		{4, "Alice"},
		{5, "Charlie"},
	}
	
	chunks := bp.chunkData(data, 2)
	
	if len(chunks) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(chunks))
	}
	
	if len(chunks[0]) != 2 {
		t.Errorf("Expected first chunk size 2, got %d", len(chunks[0]))
	}
	
	if len(chunks[1]) != 2 {
		t.Errorf("Expected second chunk size 2, got %d", len(chunks[1]))
	}
	
	if len(chunks[2]) != 1 {
		t.Errorf("Expected third chunk size 1, got %d", len(chunks[2]))
	}
}

func TestChunkValues(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	bp := NewBulkProcessor(db, nil)
	
	values := []interface{}{1, 2, 3, 4, 5}
	chunks := bp.chunkValues(values, 2)
	
	if len(chunks) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(chunks))
	}
	
	if len(chunks[0]) != 2 {
		t.Errorf("Expected first chunk size 2, got %d", len(chunks[0]))
	}
	
	if len(chunks[1]) != 2 {
		t.Errorf("Expected second chunk size 2, got %d", len(chunks[1]))
	}
	
	if len(chunks[2]) != 1 {
		t.Errorf("Expected third chunk size 1, got %d", len(chunks[2]))
	}
}

func BenchmarkBulkInsert(b *testing.B) {
	registerOnce.Do(func() {
		sql.Register("mockbulk", MockDriver{})
	})
	
	db, err := sql.Open("mockbulk", "test")
	if err != nil {
		b.Fatalf("Failed to open mock DB: %v", err)
	}
	defer db.Close()
	
	bp := NewBulkProcessor(db, &BulkConfig{
		ChunkSize:       1000,
		MaxConcurrency:  5,
		OptimizeQueries: true,
	})
	
	ctx := context.Background()
	columns := []string{"name", "email"}
	
	// Prepare test data
	data := make([][]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		data[i] = []interface{}{fmt.Sprintf("User%d", i), fmt.Sprintf("user%d@example.com", i)}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.BulkInsert(ctx, "users", columns, data)
	}
}

func BenchmarkBulkDelete(b *testing.B) {
	registerOnce.Do(func() {
		sql.Register("mockbulk2", MockDriver{})
	})
	
	db, err := sql.Open("mockbulk2", "test")
	if err != nil {
		b.Fatalf("Failed to open mock DB: %v", err)
	}
	defer db.Close()
	
	bp := NewBulkProcessor(db, &BulkConfig{
		ChunkSize:       1000,
		MaxConcurrency:  5,
		OptimizeQueries: true,
	})
	
	ctx := context.Background()
	
	// Prepare test data
	values := make([]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		values[i] = i + 1
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.BulkDelete(ctx, "users", "id", values)
	}
}
package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewDeadlockDetector(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	dd := NewDeadlockDetector(db, nil)
	if dd == nil {
		t.Fatal("Expected non-nil DeadlockDetector")
	}
	
	if !dd.IsPreventionEnabled() {
		t.Error("Expected prevention to be enabled by default")
	}
	
	if dd.GetDeadlockCount() != 0 {
		t.Error("Expected initial deadlock count to be 0")
	}
	
	if dd.GetActiveTransactionCount() != 0 {
		t.Error("Expected initial active transaction count to be 0")
	}
}

func TestSetResourceOrder(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	dd := NewDeadlockDetector(db, nil)
	
	resources := []string{"table_a", "table_b", "table_c"}
	dd.SetResourceOrder(resources)
	
	// Test resource ordering
	testResources := []string{"table_c", "table_a", "table_b"}
	sorted := dd.sortResourcesByOrder(testResources)
	
	expected := []string{"table_a", "table_b", "table_c"}
	for i, resource := range sorted {
		if resource != expected[i] {
			t.Errorf("Expected resource %s at position %d, got %s", expected[i], i, resource)
		}
	}
}

func TestAddResourceOrder(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	dd := NewDeadlockDetector(db, nil)
	
	dd.AddResourceOrder("new_table", 100)
	
	// Test that new resource is ordered correctly
	testResources := []string{"new_table", "users"} // users has order 0 by default
	sorted := dd.sortResourcesByOrder(testResources)
	
	// users should come first (order 0), then new_table (order 100)
	if sorted[0] != "users" {
		t.Errorf("Expected 'users' first, got %s", sorted[0])
	}
	if sorted[1] != "new_table" {
		t.Errorf("Expected 'new_table' second, got %s", sorted[1])
	}
}

func TestBeginTransaction(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations
	mock.ExpectBegin()
	
	dd := NewDeadlockDetector(db, nil)
	
	ctx := context.Background()
	resources := []string{"users", "accounts"}
	
	tx, txInfo, err := dd.BeginTransaction(ctx, resources)
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}
	
	if tx == nil {
		t.Error("Expected non-nil transaction")
	}
	
	if txInfo == nil {
		t.Error("Expected non-nil transaction info")
	}
	
	if txInfo.ID == "" {
		t.Error("Expected non-empty transaction ID")
	}
	
	if len(txInfo.Resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(txInfo.Resources))
	}
	
	// Check that resources are sorted
	if txInfo.Resources[0] != "accounts" || txInfo.Resources[1] != "users" {
		t.Errorf("Expected sorted resources [accounts, users], got %v", txInfo.Resources)
	}
	
	// Check that transaction is tracked
	if dd.GetActiveTransactionCount() != 1 {
		t.Error("Expected 1 active transaction")
	}
	
	// Clean up
	mock.ExpectCommit()
	dd.CommitTransaction(tx, txInfo)
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestCommitTransaction(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations
	mock.ExpectBegin()
	mock.ExpectCommit()
	
	dd := NewDeadlockDetector(db, nil)
	
	ctx := context.Background()
	resources := []string{"users"}
	
	tx, txInfo, err := dd.BeginTransaction(ctx, resources)
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}
	
	// Verify transaction is tracked
	if dd.GetActiveTransactionCount() != 1 {
		t.Error("Expected 1 active transaction before commit")
	}
	
	err = dd.CommitTransaction(tx, txInfo)
	if err != nil {
		t.Fatalf("CommitTransaction failed: %v", err)
	}
	
	// Verify transaction is no longer tracked
	if dd.GetActiveTransactionCount() != 0 {
		t.Error("Expected 0 active transactions after commit")
	}
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestRollbackTransaction(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations
	mock.ExpectBegin()
	mock.ExpectRollback()
	
	dd := NewDeadlockDetector(db, nil)
	
	ctx := context.Background()
	resources := []string{"users"}
	
	tx, txInfo, err := dd.BeginTransaction(ctx, resources)
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}
	
	// Verify transaction is tracked
	if dd.GetActiveTransactionCount() != 1 {
		t.Error("Expected 1 active transaction before rollback")
	}
	
	err = dd.RollbackTransaction(tx, txInfo)
	if err != nil {
		t.Fatalf("RollbackTransaction failed: %v", err)
	}
	
	// Verify transaction is no longer tracked
	if dd.GetActiveTransactionCount() != 0 {
		t.Error("Expected 0 active transactions after rollback")
	}
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestExecuteWithDeadlockPrevention(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	
	dd := NewDeadlockDetector(db, nil)
	
	ctx := context.Background()
	resources := []string{"users"}
	
	err = dd.ExecuteWithDeadlockPrevention(ctx, resources, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO users (name) VALUES (?)", "John")
		return err
	})
	
	if err != nil {
		t.Fatalf("ExecuteWithDeadlockPrevention failed: %v", err)
	}
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestExecuteWithDeadlockPreventionRetry(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations for retry scenario
	// First attempt fails with deadlock
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").WillReturnError(fmt.Errorf("deadlock detected"))
	mock.ExpectRollback()
	
	// Second attempt succeeds
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	
	dd := NewDeadlockDetector(db, nil)
	
	ctx := context.Background()
	resources := []string{"users"}
	
	err = dd.ExecuteWithDeadlockPrevention(ctx, resources, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO users (name) VALUES (?)", "John")
		return err
	})
	
	if err != nil {
		t.Fatalf("ExecuteWithDeadlockPrevention failed: %v", err)
	}
	
	// Check that deadlock was counted
	if dd.GetDeadlockCount() != 1 {
		t.Errorf("Expected 1 deadlock, got %d", dd.GetDeadlockCount())
	}
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestIsDeadlockError(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	dd := NewDeadlockDetector(db, nil)
	
	testCases := []struct {
		error    error
		expected bool
	}{
		{nil, false},
		{fmt.Errorf("deadlock detected"), true},
		{fmt.Errorf("lock timeout exceeded"), true},
		{fmt.Errorf("lock wait timeout"), true},
		{fmt.Errorf("serialization failure"), true},
		{fmt.Errorf("could not serialize access"), true},
		{fmt.Errorf("connection refused"), false},
		{fmt.Errorf("syntax error"), false},
	}
	
	for _, tc := range testCases {
		result := dd.isDeadlockError(tc.error)
		if result != tc.expected {
			t.Errorf("For error %v, expected %v, got %v", tc.error, tc.expected, result)
		}
	}
}

func TestDetectDeadlocks(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations for two transactions
	mock.ExpectBegin()
	mock.ExpectBegin()
	
	dd := NewDeadlockDetector(db, nil)
	
	ctx := context.Background()
	
	// Create two transactions with overlapping resources
	tx1, txInfo1, err := dd.BeginTransaction(ctx, []string{"users", "accounts"})
	if err != nil {
		t.Fatalf("BeginTransaction 1 failed: %v", err)
	}
	
	// Wait a bit to ensure different start times
	time.Sleep(1 * time.Millisecond)
	
	tx2, txInfo2, err := dd.BeginTransaction(ctx, []string{"accounts", "orders"})
	if err != nil {
		t.Fatalf("BeginTransaction 2 failed: %v", err)
	}
	
	// Detect deadlocks
	deadlocks := dd.DetectDeadlocks()
	
	// Should detect potential deadlock due to overlapping resources
	if len(deadlocks) == 0 {
		t.Error("Expected to detect potential deadlock")
	}
	
	// Clean up
	mock.ExpectRollback()
	mock.ExpectRollback()
	dd.RollbackTransaction(tx1, txInfo1)
	dd.RollbackTransaction(tx2, txInfo2)
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetActiveTransactions(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations
	mock.ExpectBegin()
	mock.ExpectBegin()
	
	dd := NewDeadlockDetector(db, nil)
	
	ctx := context.Background()
	
	// Initially no active transactions
	transactions := dd.GetActiveTransactions()
	if len(transactions) != 0 {
		t.Errorf("Expected 0 active transactions, got %d", len(transactions))
	}
	
	// Create two transactions
	tx1, txInfo1, err := dd.BeginTransaction(ctx, []string{"users"})
	if err != nil {
		t.Fatalf("BeginTransaction 1 failed: %v", err)
	}
	
	tx2, txInfo2, err := dd.BeginTransaction(ctx, []string{"accounts"})
	if err != nil {
		t.Fatalf("BeginTransaction 2 failed: %v", err)
	}
	
	// Should have 2 active transactions
	transactions = dd.GetActiveTransactions()
	if len(transactions) != 2 {
		t.Errorf("Expected 2 active transactions, got %d", len(transactions))
	}
	
	// Clean up
	mock.ExpectRollback()
	mock.ExpectRollback()
	dd.RollbackTransaction(tx1, txInfo1)
	dd.RollbackTransaction(tx2, txInfo2)
	
	// Should have 0 active transactions
	transactions = dd.GetActiveTransactions()
	if len(transactions) != 0 {
		t.Errorf("Expected 0 active transactions after cleanup, got %d", len(transactions))
	}
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestCleanupExpiredTransactions(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations
	mock.ExpectBegin()
	
	dd := NewDeadlockDetector(db, nil)
	
	ctx := context.Background()
	
	// Create a transaction
	tx, txInfo, err := dd.BeginTransaction(ctx, []string{"users"})
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}
	
	// Manually set start time to past
	txInfo.StartTime = time.Now().Add(-10 * time.Minute)
	
	// Cleanup expired transactions (older than 5 minutes)
	expiredCount := dd.CleanupExpiredTransactions(5 * time.Minute)
	
	if expiredCount != 1 {
		t.Errorf("Expected 1 expired transaction, got %d", expiredCount)
	}
	
	// Should have 0 active transactions after cleanup
	if dd.GetActiveTransactionCount() != 0 {
		t.Error("Expected 0 active transactions after cleanup")
	}
	
	// Clean up the actual transaction (it's already removed from tracking)
	mock.ExpectRollback()
	tx.Rollback()
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetStats(t *testing.T) {
	db, mock, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	// Set up mock expectations
	mock.ExpectBegin()
	
	dd := NewDeadlockDetector(db, nil)
	
	ctx := context.Background()
	
	// Create a transaction
	tx, txInfo, err := dd.BeginTransaction(ctx, []string{"users"})
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}
	
	// Get stats
	stats := dd.GetStats()
	
	if stats.ActiveTransactions != 1 {
		t.Errorf("Expected 1 active transaction, got %d", stats.ActiveTransactions)
	}
	
	if stats.TotalDeadlocks != 0 {
		t.Errorf("Expected 0 total deadlocks, got %d", stats.TotalDeadlocks)
	}
	
	if !stats.ResourceOrderingEnabled {
		t.Error("Expected resource ordering to be enabled")
	}
	
	if stats.AverageTransactionAge <= 0 {
		t.Error("Expected positive average transaction age")
	}
	
	// Clean up
	mock.ExpectRollback()
	dd.RollbackTransaction(tx, txInfo)
	
	// Verify all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestSetPreventionEnabled(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	dd := NewDeadlockDetector(db, nil)
	
	// Initially enabled
	if !dd.IsPreventionEnabled() {
		t.Error("Expected prevention to be enabled initially")
	}
	
	// Disable prevention
	dd.SetPreventionEnabled(false)
	if dd.IsPreventionEnabled() {
		t.Error("Expected prevention to be disabled")
	}
	
	// Re-enable prevention
	dd.SetPreventionEnabled(true)
	if !dd.IsPreventionEnabled() {
		t.Error("Expected prevention to be enabled")
	}
}

func TestGenerateTransactionID(t *testing.T) {
	id1 := generateTransactionID()
	id2 := generateTransactionID()
	
	if id1 == "" {
		t.Error("Expected non-empty transaction ID")
	}
	
	if id2 == "" {
		t.Error("Expected non-empty transaction ID")
	}
	
	if id1 == id2 {
		t.Error("Expected different transaction IDs")
	}
	
	if !strings.HasPrefix(id1, "tx_") {
		t.Error("Expected transaction ID to start with 'tx_'")
	}
}

func TestSortResourcesByOrderAlphabetical(t *testing.T) {
	db, _, err := setupMockDB()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer db.Close()
	
	dd := NewDeadlockDetector(db, nil)
	
	// Clear existing resource order
	dd.SetResourceOrder([]string{})
	
	// Test alphabetical sorting for resources without defined order
	resources := []string{"zebra", "apple", "banana"}
	sorted := dd.sortResourcesByOrder(resources)
	
	expected := []string{"apple", "banana", "zebra"}
	for i, resource := range sorted {
		if resource != expected[i] {
			t.Errorf("Expected resource %s at position %d, got %s", expected[i], i, resource)
		}
	}
}

func BenchmarkBeginTransaction(b *testing.B) {
	registerOnce.Do(func() {
		sql.Register("mockdeadlock", MockDriver{})
	})
	
	db, err := sql.Open("mockdeadlock", "test")
	if err != nil {
		b.Fatalf("Failed to open mock DB: %v", err)
	}
	defer db.Close()
	
	dd := NewDeadlockDetector(db, nil)
	ctx := context.Background()
	resources := []string{"users", "accounts", "orders"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, txInfo, err := dd.BeginTransaction(ctx, resources)
		if err != nil {
			b.Fatalf("BeginTransaction failed: %v", err)
		}
		// Immediately rollback to clean up
		tx.Rollback()
		dd.activeTransactions.Delete(txInfo.ID)
		txInfo.Cancel()
	}
}

func BenchmarkSortResourcesByOrder(b *testing.B) {
	registerOnce.Do(func() {
		sql.Register("mockdeadlock2", MockDriver{})
	})
	
	db, err := sql.Open("mockdeadlock2", "test")
	if err != nil {
		b.Fatalf("Failed to open mock DB: %v", err)
	}
	defer db.Close()
	
	dd := NewDeadlockDetector(db, nil)
	resources := []string{"orders", "users", "accounts", "products", "inventory"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dd.sortResourcesByOrder(resources)
	}
}
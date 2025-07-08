package main

import (
	"testing"
)

// Test basic configuration and initialization
func TestBasicFunctionality(t *testing.T) {
	// Test BatchConfig defaults
	config := DefaultBatchConfig()
	if config.BatchSize != 1000 {
		t.Errorf("Expected default batch size 1000, got %d", config.BatchSize)
	}
	
	// Test BulkConfig defaults
	bulkConfig := DefaultBulkConfig()
	if bulkConfig.ChunkSize != 1000 {
		t.Errorf("Expected default chunk size 1000, got %d", bulkConfig.ChunkSize)
	}
	
	// Test DeadlockConfig defaults
	deadlockConfig := DefaultDeadlockConfig()
	if deadlockConfig.Strategy != ResourceOrdering {
		t.Errorf("Expected default strategy ResourceOrdering, got %v", deadlockConfig.Strategy)
	}
}

func TestBulkOperationTypes(t *testing.T) {
	// Test BulkOperation constants
	if BulkInsert != 0 {
		t.Errorf("Expected BulkInsert to be 0, got %d", BulkInsert)
	}
	if BulkUpdate != 1 {
		t.Errorf("Expected BulkUpdate to be 1, got %d", BulkUpdate)
	}
	if BulkDelete != 2 {
		t.Errorf("Expected BulkDelete to be 2, got %d", BulkDelete)
	}
	if BulkUpsert != 3 {
		t.Errorf("Expected BulkUpsert to be 3, got %d", BulkUpsert)
	}
}

func TestDeadlockStrategy(t *testing.T) {
	// Test DeadlockStrategy constants
	if ResourceOrdering != 0 {
		t.Errorf("Expected ResourceOrdering to be 0, got %d", ResourceOrdering)
	}
	if TimeoutBased != 1 {
		t.Errorf("Expected TimeoutBased to be 1, got %d", TimeoutBased)
	}
	if WaitDie != 2 {
		t.Errorf("Expected WaitDie to be 2, got %d", WaitDie)
	}
	if WoundWait != 3 {
		t.Errorf("Expected WoundWait to be 3, got %d", WoundWait)
	}
}

func TestConfigValidation(t *testing.T) {
	// Test that configs have reasonable defaults
	config := DefaultBatchConfig()
	if config.FlushInterval <= 0 {
		t.Error("FlushInterval should be positive")
	}
	if config.MaxRetries < 0 {
		t.Error("MaxRetries should be non-negative")
	}
	if config.TransactionTimeout <= 0 {
		t.Error("TransactionTimeout should be positive")
	}
	
	bulkConfig := DefaultBulkConfig()
	if bulkConfig.MaxConcurrency <= 0 {
		t.Error("MaxConcurrency should be positive")
	}
	if bulkConfig.Timeout <= 0 {
		t.Error("Timeout should be positive")
	}
	
	deadlockConfig := DefaultDeadlockConfig()
	if deadlockConfig.DetectionInterval <= 0 {
		t.Error("DetectionInterval should be positive")
	}
	if deadlockConfig.TransactionTimeout <= 0 {
		t.Error("TransactionTimeout should be positive")
	}
}

func TestUtilityFunctions(t *testing.T) {
	// Test contains function
	if !contains("hello world", "world") {
		t.Error("contains should find 'world' in 'hello world'")
	}
	if contains("hello", "world") {
		t.Error("contains should not find 'world' in 'hello'")
	}
	
	// Test indexOfSubstring function
	index := indexOfSubstring("hello world", "world")
	if index != 6 {
		t.Errorf("Expected index 6, got %d", index)
	}
	
	index = indexOfSubstring("hello", "world")
	if index != -1 {
		t.Errorf("Expected index -1, got %d", index)
	}
}

// Benchmark basic operations without database
func BenchmarkConfigCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultBatchConfig()
		_ = DefaultBulkConfig()
		_ = DefaultDeadlockConfig()
	}
}

func BenchmarkStringOperations(b *testing.B) {
	for i := 0; i < b.N; i++ {
		contains("this is a test string for benchmarking", "test")
		indexOfSubstring("this is a test string for benchmarking", "test")
	}
}
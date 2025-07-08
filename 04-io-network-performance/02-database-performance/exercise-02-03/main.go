package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

func main() {
	fmt.Println("=== Exercise 02-03: Transaction Batching and Optimization ===")
	fmt.Println()
	
	// For demonstration purposes, we'll use a mock database
	// In production, you would connect to a real database
	db := setupDemoDatabase()
	if db != nil {
		defer db.Close()
		demonstrateTransactionBatching(db)
	} else {
		fmt.Println("Note: Using mock demonstrations since no real database is connected.")
		demonstrateWithoutDatabase()
	}
	
	fmt.Println("\n=== Performance Optimization Techniques ===")
	demonstrateOptimizationTechniques()
	
	fmt.Println("\n=== How to Use with Real Database ===")
	showRealDatabaseUsage()
	
	fmt.Println("\n=== Running Tests and Benchmarks ===")
	fmt.Println("To run tests: go test -v")
	fmt.Println("To run benchmarks: go test -bench=. -benchmem")
	fmt.Println("To run specific benchmark: go test -bench=BenchmarkBatchProcessor -benchmem")
}

func setupDemoDatabase() *sql.DB {
	// Try to connect to a demo database
	// This will fail gracefully if no database is available
	connStr := "postgres://demo:demo@localhost/demo?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil
	}
	
	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil
	}
	
	return db
}

func demonstrateTransactionBatching(db *sql.DB) {
	fmt.Println("=== Transaction Batching Demonstration ===")
	
	// Create tables for demonstration
	createTestTables(db)
	
	// 1. Demonstrate Batch Processing
	fmt.Println("\n1. Batch Processing:")
	demonstrateBasicBatching(db)
	
	// 2. Demonstrate Bulk Operations
	fmt.Println("\n2. Bulk Operations:")
	demonstrateBasicBulkOperations(db)
	
	// 3. Demonstrate Deadlock Prevention
	fmt.Println("\n3. Deadlock Prevention:")
	demonstrateBasicDeadlockPrevention(db)
	
	// 4. Performance Metrics
	fmt.Println("\n4. Performance Metrics:")
	demonstratePerformanceMetrics(db)
}

func demonstrateWithoutDatabase() {
	fmt.Println("=== Mock Demonstration (No Database Required) ===")
	
	// Demonstrate configuration and basic usage patterns
	fmt.Println("\n1. Batch Processor Configuration:")
	config := DefaultBatchConfig()
	fmt.Printf("   - Batch Size: %d\n", config.BatchSize)
	fmt.Printf("   - Flush Interval: %v\n", config.FlushInterval)
	fmt.Printf("   - Max Retries: %d\n", config.MaxRetries)
	fmt.Printf("   - Max Concurrent Batches: %d\n", config.MaxConcurrentBatch)
	
	fmt.Println("\n2. Bulk Operations Configuration:")
	bulkConfig := DefaultBulkConfig()
	fmt.Printf("   - Chunk Size: %d\n", bulkConfig.ChunkSize)
	fmt.Printf("   - Max Concurrency: %d\n", bulkConfig.MaxConcurrency)
	fmt.Printf("   - Timeout: %v\n", bulkConfig.Timeout)
	fmt.Printf("   - Retry Attempts: %d\n", bulkConfig.RetryAttempts)
	
	fmt.Println("\n3. Deadlock Prevention Configuration:")
	deadlockConfig := DefaultDeadlockConfig()
	fmt.Printf("   - Strategy: %v\n", deadlockConfig.Strategy)
	fmt.Printf("   - Detection Interval: %v\n", deadlockConfig.DetectionInterval)
	fmt.Printf("   - Transaction Timeout: %v\n", deadlockConfig.TransactionTimeout)
	fmt.Printf("   - Max Retries: %d\n", deadlockConfig.MaxRetries)
}

func createTestTables(db *sql.DB) {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			amount DECIMAL(10,2) NOT NULL,
			status VARCHAR(20) DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			price DECIMAL(10,2) NOT NULL,
			stock INTEGER DEFAULT 0
		)`,
	}
	
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			log.Printf("Warning: Could not create table: %v", err)
		}
	}
}

func demonstrateBasicBatching(db *sql.DB) {
	// Create batch processor
	config := &BatchConfig{
		BatchSize:     10,
		FlushInterval: 100 * time.Millisecond,
		MaxRetries:    3,
	}
	
	bp := NewBatchProcessor(db, config)
	defer bp.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Execute multiple operations
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			result, err := bp.Execute(ctx, 
				"INSERT INTO users (name, email) VALUES ($1, $2)",
				fmt.Sprintf("User%d", id),
				fmt.Sprintf("user%d@example.com", id))
			
			if err != nil {
				log.Printf("Batch operation %d failed: %v", id, err)
			} else if result.Error != nil {
				log.Printf("Batch operation %d error: %v", id, result.Error)
			} else {
				fmt.Printf("   Batch operation %d: %d rows affected in %v\n", 
					id, result.RowsAffected, result.Duration)
			}
		}(i)
	}
	
	wg.Wait()
	
	// Show metrics
	metrics := bp.GetMetrics()
	fmt.Printf("   Total Batches: %d\n", metrics.TotalBatches)
	fmt.Printf("   Total Operations: %d\n", metrics.TotalOperations)
	fmt.Printf("   Successful Batches: %d\n", metrics.SuccessfulBatches)
	fmt.Printf("   Average Batch Size: %d\n", metrics.AverageBatchSize)
	fmt.Printf("   Throughput: %d ops/sec\n", bp.GetThroughput())
}

func demonstrateBasicBulkOperations(db *sql.DB) {
	// Create bulk processor
	config := &BulkConfig{
		ChunkSize:       100,
		MaxConcurrency:  3,
		OptimizeQueries: true,
	}
	
	bulkProcessor := NewBulkProcessor(db, config)
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Prepare bulk insert data
	columns := []string{"name", "price", "stock"}
	data := make([][]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		data[i] = []interface{}{
			fmt.Sprintf("Product%d", i),
			float64(rand.Intn(10000)) / 100.0, // Random price
			rand.Intn(1000),                   // Random stock
		}
	}
	
	// Perform bulk insert
	start := time.Now()
	result, err := bulkProcessor.BulkInsert(ctx, "products", columns, data)
	duration := time.Since(start)
	
	if err != nil {
		log.Printf("Bulk insert failed: %v", err)
		return
	}
	
	fmt.Printf("   Bulk Insert Results:\n")
	fmt.Printf("   - Total Rows: %d\n", result.TotalRows)
	fmt.Printf("   - Affected Rows: %d\n", result.AffectedRows)
	fmt.Printf("   - Processed Chunks: %d\n", result.ProcessedChunks)
	fmt.Printf("   - Failed Chunks: %d\n", result.FailedChunks)
	fmt.Printf("   - Duration: %v\n", duration)
	fmt.Printf("   - Throughput: %.2f rows/sec\n", float64(result.AffectedRows)/duration.Seconds())
	
	// Demonstrate bulk update
	updates := []BulkUpdateItem{
		{
			SetClause:   "price = $1",
			SetValues:   []interface{}{99.99},
			WhereClause: "id = $1",
			WhereValues: []interface{}{1},
		},
		{
			SetClause:   "stock = $1",
			SetValues:   []interface{}{0},
			WhereClause: "id = $1",
			WhereValues: []interface{}{2},
		},
	}
	
	updateResult, err := bulkProcessor.BulkUpdate(ctx, "products", updates)
	if err != nil {
		log.Printf("Bulk update failed: %v", err)
	} else {
		fmt.Printf("   Bulk Update: %d rows affected\n", updateResult.AffectedRows)
	}
}

func demonstrateBasicDeadlockPrevention(db *sql.DB) {
	// Create deadlock detector
	config := DefaultDeadlockConfig()
	dd := NewDeadlockDetector(db, config)
	
	// Set resource ordering to prevent deadlocks
	resources := []string{"users", "orders", "products"}
	dd.SetResourceOrder(resources)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Simulate concurrent transactions that could cause deadlocks
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			// Use different resource orders to test prevention
			txResources := []string{"orders", "users"} // Will be sorted automatically
			
			err := dd.ExecuteWithDeadlockPrevention(ctx, txResources, func(tx *sql.Tx) error {
				// Simulate some work
				time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
				
				// Execute some operations
				_, err := tx.Exec("UPDATE users SET name = $1 WHERE id = $2", 
					fmt.Sprintf("UpdatedUser%d", id), id+1)
				if err != nil {
					return err
				}
				
				_, err = tx.Exec("UPDATE orders SET status = $1 WHERE user_id = $2", 
					"processed", id+1)
				return err
			})
			
			if err != nil {
				log.Printf("Transaction %d failed: %v", id, err)
			} else {
				fmt.Printf("   Transaction %d completed successfully\n", id)
			}
		}(i)
	}
	
	wg.Wait()
	
	// Show deadlock statistics
	stats := dd.GetStats()
	fmt.Printf("   Deadlock Statistics:\n")
	fmt.Printf("   - Total Deadlocks: %d\n", stats.TotalDeadlocks)
	fmt.Printf("   - Active Transactions: %d\n", stats.ActiveTransactions)
	fmt.Printf("   - Resource Ordering Enabled: %v\n", stats.ResourceOrderingEnabled)
}

func demonstratePerformanceMetrics(db *sql.DB) {
	// Create processors with metrics enabled
	batchConfig := &BatchConfig{
		BatchSize:     50,
		FlushInterval: 50 * time.Millisecond,
		EnableMetrics: true,
	}
	
	bp := NewBatchProcessor(db, batchConfig)
	defer bp.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Perform operations and measure performance
	start := time.Now()
	var wg sync.WaitGroup
	
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			bp.Execute(ctx, "INSERT INTO orders (user_id, amount) VALUES ($1, $2)", 
				id%10+1, float64(rand.Intn(10000))/100.0)
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	// Display comprehensive metrics
	metrics := bp.GetMetrics()
	fmt.Printf("   Performance Metrics:\n")
	fmt.Printf("   - Total Duration: %v\n", duration)
	fmt.Printf("   - Total Batches: %d\n", metrics.TotalBatches)
	fmt.Printf("   - Total Operations: %d\n", metrics.TotalOperations)
	fmt.Printf("   - Successful Batches: %d\n", metrics.SuccessfulBatches)
	fmt.Printf("   - Failed Batches: %d\n", metrics.FailedBatches)
	fmt.Printf("   - Average Batch Size: %d\n", metrics.AverageBatchSize)
	fmt.Printf("   - Average Commit Time: %v\n", bp.GetAverageCommitTime())
	fmt.Printf("   - Throughput: %d ops/sec\n", bp.GetThroughput())
	fmt.Printf("   - Deadlock Rate: %.2f%%\n", bp.GetDeadlockRate())
	fmt.Printf("   - Overall Throughput: %.2f ops/sec\n", 
		float64(metrics.TotalOperations)/duration.Seconds())
}

func demonstrateOptimizationTechniques() {
	fmt.Println("\n1. Batch Size Optimization:")
	fmt.Println("   - Small batches (10-100): Lower latency, higher overhead")
	fmt.Println("   - Medium batches (100-1000): Balanced performance")
	fmt.Println("   - Large batches (1000+): Higher throughput, higher latency")
	
	fmt.Println("\n2. Concurrency Control:")
	fmt.Println("   - Limit concurrent batches to prevent resource exhaustion")
	fmt.Println("   - Use semaphores to control parallel execution")
	fmt.Println("   - Monitor system resources (CPU, memory, connections)")
	
	fmt.Println("\n3. Query Optimization:")
	fmt.Println("   - Use multi-row INSERT statements")
	fmt.Println("   - Leverage UPSERT for conflict resolution")
	fmt.Println("   - Use IN clauses for bulk DELETE operations")
	fmt.Println("   - Prepare statements for repeated operations")
	
	fmt.Println("\n4. Deadlock Prevention:")
	fmt.Println("   - Consistent resource ordering")
	fmt.Println("   - Timeout-based detection")
	fmt.Println("   - Retry with exponential backoff")
	fmt.Println("   - Transaction isolation level optimization")
	
	fmt.Println("\n5. Memory Management:")
	fmt.Println("   - Chunk large datasets")
	fmt.Println("   - Stream processing for huge datasets")
	fmt.Println("   - Connection pooling")
	fmt.Println("   - Garbage collection optimization")
}

func showRealDatabaseUsage() {
	fmt.Println(`
// Connect to your database
db, err := sql.Open("postgres", "your-connection-string")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// 1. Batch Processing
config := &BatchConfig{
    BatchSize:     1000,
    FlushInterval: 100 * time.Millisecond,
    MaxRetries:    3,
}
bp := NewBatchProcessor(db, config)
defer bp.Close()

// Execute operations
result, err := bp.Execute(ctx, "INSERT INTO users (name) VALUES (?)", "John")

// 2. Bulk Operations
bulkProcessor := NewBulkProcessor(db, nil)
columns := []string{"name", "email"}
data := [][]interface{}{
    {"John", "john@example.com"},
    {"Jane", "jane@example.com"},
}
result, err := bulkProcessor.BulkInsert(ctx, "users", columns, data)

// 3. Deadlock Prevention
dd := NewDeadlockDetector(db, nil)
dd.SetResourceOrder([]string{"users", "orders", "products"})
err := dd.ExecuteWithDeadlockPrevention(ctx, []string{"users"}, func(tx *sql.Tx) error {
    // Your transaction logic here
    return nil
})`)
}
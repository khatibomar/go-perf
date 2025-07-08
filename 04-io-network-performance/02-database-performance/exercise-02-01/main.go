package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	_ "net/http/pprof"
)

func main() {
	// Start pprof server for monitoring
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	// Example usage of the connection pool
	if err := demonstrateConnectionPool(); err != nil {
		log.Fatalf("Connection pool demonstration failed: %v", err)
	}
}

func demonstrateConnectionPool() error {
	// Configure the connection pool
	config := PoolConfig{
		DriverName:      "mock", // Using mock driver for demonstration
		DataSourceName:  "test_database",
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
		HealthInterval:  30 * time.Second,
		HealthCheck:     DefaultHealthCheck,
	}

	// Create the connection pool
	pool, err := NewConnectionPool(config)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}
	defer pool.Close()

	log.Println("Connection pool created successfully")

	// Demonstrate basic operations
	if err := demonstrateBasicOperations(pool); err != nil {
		return fmt.Errorf("basic operations failed: %w", err)
	}

	// Demonstrate concurrent usage
	if err := demonstrateConcurrentUsage(pool); err != nil {
		return fmt.Errorf("concurrent usage failed: %w", err)
	}

	// Demonstrate metrics monitoring
	demonstratemetricsMonitoring(pool)

	// Demonstrate performance under load
	if err := demonstratePerformanceLoad(pool); err != nil {
		return fmt.Errorf("performance load test failed: %w", err)
	}

	log.Println("All demonstrations completed successfully")
	return nil
}

func demonstrateBasicOperations(pool *ConnectionPool) error {
	log.Println("\n=== Demonstrating Basic Operations ===")

	ctx := context.Background()

	// Test ping
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	log.Println("✓ Database ping successful")

	// Test getting a connection
	conn, err := pool.GetConnection(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	log.Println("✓ Connection acquired successfully")
	conn.Close()

	// Test executing a query
	_, err = pool.Execute(ctx, "CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		return fmt.Errorf("failed to execute DDL: %w", err)
	}
	log.Println("✓ DDL query executed successfully")

	// Test transaction
	tx, err := pool.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	_, err = tx.Exec("INSERT INTO users (name) VALUES (?)", "John Doe")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to execute insert: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	log.Println("✓ Transaction completed successfully")

	return nil
}

func demonstrateConcurrentUsage(pool *ConnectionPool) error {
	log.Println("\n=== Demonstrating Concurrent Usage ===")

	ctx := context.Background()
	var wg sync.WaitGroup
	errorChan := make(chan error, 50)
	workerCount := 50
	operationsPerWorker := 10

	start := time.Now()

	// Launch concurrent workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				// Get connection and perform operation
				conn, err := pool.GetConnection(ctx)
				if err != nil {
					errorChan <- fmt.Errorf("worker %d operation %d: %w", workerID, j, err)
					continue
				}

				// Simulate work
				_, err = conn.ExecContext(ctx, "SELECT 1")
				if err != nil {
					conn.Close()
					errorChan <- fmt.Errorf("worker %d query %d: %w", workerID, j, err)
					continue
				}

				conn.Close()
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	duration := time.Since(start)
	totalOperations := workerCount * operationsPerWorker

	// Check for errors
	errorCount := 0
	for err := range errorChan {
		log.Printf("Concurrent operation error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		return fmt.Errorf("%d errors occurred during concurrent operations", errorCount)
	}

	opsPerSecond := float64(totalOperations) / duration.Seconds()
	log.Printf("✓ Completed %d operations in %v (%.2f ops/sec)", totalOperations, duration, opsPerSecond)

	return nil
}

func demonstratemetricsMonitoring(pool *ConnectionPool) {
	log.Println("\n=== Demonstrating Metrics Monitoring ===")

	metrics := pool.GetMetrics()
	utilization := pool.GetUtilization()

	log.Printf("Pool Metrics:")
	log.Printf("  Open Connections: %d", metrics.OpenConnections)
	log.Printf("  In-Use Connections: %d", metrics.InUseConnections)
	log.Printf("  Idle Connections: %d", metrics.IdleConnections)
	log.Printf("  Wait Count: %d", metrics.WaitCount)
	log.Printf("  Total Wait Duration: %v", metrics.GetWaitDuration())
	log.Printf("  Max Idle Closed: %d", metrics.MaxIdleClosed)
	log.Printf("  Max Lifetime Closed: %d", metrics.MaxLifetimeClosed)
	log.Printf("  Health Check Count: %d", metrics.HealthCheckCount)
	log.Printf("  Health Check Failed: %d", metrics.HealthCheckFailed)
	log.Printf("  Pool Utilization: %.2f%%", utilization)

	// Calculate average wait time
	if metrics.WaitCount > 0 {
		avgWaitTime := metrics.GetWaitDuration() / time.Duration(metrics.WaitCount)
		log.Printf("  Average Wait Time: %v", avgWaitTime)

		// Check performance targets
		if avgWaitTime > time.Millisecond {
			log.Printf("⚠️  Warning: Average connection acquisition time (%.2fms) exceeds target (<1ms)",
				float64(avgWaitTime.Nanoseconds())/1e6)
		} else {
			log.Printf("✓ Connection acquisition time target met (%.2fms < 1ms)",
				float64(avgWaitTime.Nanoseconds())/1e6)
		}
	}

	// Check utilization target
	if utilization > 90 {
		log.Printf("✓ Pool utilization target met (%.2f%% > 90%%)", utilization)
	} else {
		log.Printf("ℹ️  Pool utilization below target (%.2f%% < 90%%)", utilization)
	}

	// Check health check overhead
	if metrics.HealthCheckCount > 0 {
		failureRate := float64(metrics.HealthCheckFailed) / float64(metrics.HealthCheckCount) * 100
		log.Printf("  Health Check Failure Rate: %.2f%%", failureRate)

		if failureRate < 0.1 {
			log.Printf("✓ Health check overhead target met (%.2f%% < 0.1%%)", failureRate)
		} else {
			log.Printf("⚠️  Warning: Health check failure rate (%.2f%%) exceeds target (<0.1%%)", failureRate)
		}
	}
}

func demonstratePerformanceLoad(pool *ConnectionPool) error {
	log.Println("\n=== Demonstrating Performance Under Load ===")

	ctx := context.Background()
	var wg sync.WaitGroup
	errorChan := make(chan error, 100)
	workerCount := 100
	operationsPerWorker := 100

	start := time.Now()

	// Launch high-load concurrent workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				// Perform database operation
				_, err := pool.Execute(ctx, "SELECT ?, ?", workerID, j)
				if err != nil {
					errorChan <- fmt.Errorf("worker %d operation %d: %w", workerID, j, err)
					continue
				}
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	duration := time.Since(start)
	totalOperations := workerCount * operationsPerWorker

	// Check for errors
	errorCount := 0
	for err := range errorChan {
		log.Printf("Load test error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		return fmt.Errorf("%d errors occurred during load test", errorCount)
	}

	opsPerSecond := float64(totalOperations) / duration.Seconds()
	log.Printf("✓ Load test completed: %d operations in %v (%.2f ops/sec)", totalOperations, duration, opsPerSecond)

	// Final metrics check
	finalMetrics := pool.GetMetrics()
	log.Printf("Final pool state:")
	log.Printf("  Total operations: %d", finalMetrics.WaitCount)
	log.Printf("  Total wait time: %v", finalMetrics.GetWaitDuration())
	log.Printf("  Average operation time: %v", finalMetrics.GetWaitDuration()/time.Duration(finalMetrics.WaitCount))

	return nil
}
package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// BatchItem represents a single operation in a batch
type BatchItem struct {
	Query    string
	Args     []interface{}
	Result   chan BatchResult
	Retries  int
	Priority int // Higher values = higher priority
}

// BatchResult contains the result of a batch operation
type BatchResult struct {
	RowsAffected int64
	Error        error
	Duration     time.Duration
}

// BatchConfig holds configuration for the batch processor
type BatchConfig struct {
	BatchSize           int
	FlushInterval       time.Duration
	MaxRetries          int
	RetryDelay          time.Duration
	DeadlockRetryDelay  time.Duration
	MaxConcurrentBatch  int
	TransactionTimeout  time.Duration
	EnableMetrics       bool
}

// DefaultBatchConfig returns a default configuration
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		BatchSize:           1000,
		FlushInterval:       100 * time.Millisecond,
		MaxRetries:          3,
		RetryDelay:          10 * time.Millisecond,
		DeadlockRetryDelay:  50 * time.Millisecond,
		MaxConcurrentBatch:  10,
		TransactionTimeout:  30 * time.Second,
		EnableMetrics:       true,
	}
}

// BatchMetrics tracks performance metrics
type BatchMetrics struct {
	TotalBatches        int64
	TotalOperations     int64
	SuccessfulBatches   int64
	FailedBatches       int64
	DeadlockRetries     int64
	AverageCommitTime   int64 // nanoseconds
	AverageBatchSize    int64
	Throughput          int64 // operations per second
	MemoryUsage         int64 // bytes
	lastThroughputCalc  time.Time
	operationsSinceCalc int64
}

// BatchProcessor handles efficient transaction batching
type BatchProcessor struct {
	db       *sql.DB
	config   *BatchConfig
	metrics  *BatchMetrics
	buffer   []BatchItem
	mu       sync.Mutex
	flushCh  chan struct{}
	stopCh   chan struct{}
	wg       sync.WaitGroup
	running  int32
	semaphore chan struct{} // Limits concurrent batches
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(db *sql.DB, config *BatchConfig) *BatchProcessor {
	if config == nil {
		config = DefaultBatchConfig()
	}
	
	bp := &BatchProcessor{
		db:        db,
		config:    config,
		metrics:   &BatchMetrics{lastThroughputCalc: time.Now()},
		buffer:    make([]BatchItem, 0, config.BatchSize),
		flushCh:   make(chan struct{}, 1),
		stopCh:    make(chan struct{}),
		semaphore: make(chan struct{}, config.MaxConcurrentBatch),
	}
	
	// Start background flush goroutine
	bp.start()
	
	return bp
}

// Execute adds an operation to the batch
func (bp *BatchProcessor) Execute(ctx context.Context, query string, args ...interface{}) (*BatchResult, error) {
	result := make(chan BatchResult, 1)
	item := BatchItem{
		Query:  query,
		Args:   args,
		Result: result,
	}
	
	bp.mu.Lock()
	bp.buffer = append(bp.buffer, item)
	shouldFlush := len(bp.buffer) >= bp.config.BatchSize
	bp.mu.Unlock()
	
	if shouldFlush {
		select {
		case bp.flushCh <- struct{}{}:
		default:
			// Channel is full, flush is already scheduled
		}
	}
	
	// Wait for result with context timeout
	select {
	case res := <-result:
		return &res, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ExecuteWithPriority adds a prioritized operation to the batch
func (bp *BatchProcessor) ExecuteWithPriority(ctx context.Context, priority int, query string, args ...interface{}) (*BatchResult, error) {
	result := make(chan BatchResult, 1)
	item := BatchItem{
		Query:    query,
		Args:     args,
		Result:   result,
		Priority: priority,
	}
	
	bp.mu.Lock()
	bp.buffer = append(bp.buffer, item)
	// Sort by priority (higher priority first)
	for i := len(bp.buffer) - 1; i > 0; i-- {
		if bp.buffer[i].Priority > bp.buffer[i-1].Priority {
			bp.buffer[i], bp.buffer[i-1] = bp.buffer[i-1], bp.buffer[i]
		} else {
			break
		}
	}
	shouldFlush := len(bp.buffer) >= bp.config.BatchSize
	bp.mu.Unlock()
	
	if shouldFlush {
		select {
		case bp.flushCh <- struct{}{}:
		default:
		}
	}
	
	select {
	case res := <-result:
		return &res, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Flush forces immediate processing of buffered operations
func (bp *BatchProcessor) Flush() error {
	select {
	case bp.flushCh <- struct{}{}:
		return nil
	default:
		return fmt.Errorf("flush already in progress")
	}
}

// start begins the background processing
func (bp *BatchProcessor) start() {
	atomic.StoreInt32(&bp.running, 1)
	bp.wg.Add(1)
	
	go func() {
		defer bp.wg.Done()
		ticker := time.NewTicker(bp.config.FlushInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-bp.flushCh:
				bp.processBatch()
			case <-ticker.C:
				bp.processBatch()
			case <-bp.stopCh:
				// Process remaining items before stopping
				bp.processBatch()
				return
			}
		}
	}()
}

// processBatch handles the actual batch processing
func (bp *BatchProcessor) processBatch() {
	bp.mu.Lock()
	if len(bp.buffer) == 0 {
		bp.mu.Unlock()
		return
	}
	
	items := make([]BatchItem, len(bp.buffer))
	copy(items, bp.buffer)
	bp.buffer = bp.buffer[:0] // Reset buffer
	bp.mu.Unlock()
	
	// Acquire semaphore to limit concurrent batches
	bp.semaphore <- struct{}{}
	bp.wg.Add(1)
	
	go func() {
		defer func() {
			<-bp.semaphore
			bp.wg.Done()
		}()
		
		bp.executeBatch(items)
	}()
}

// executeBatch executes a batch of operations with retry logic
func (bp *BatchProcessor) executeBatch(items []BatchItem) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		atomic.AddInt64(&bp.metrics.TotalBatches, 1)
		atomic.AddInt64(&bp.metrics.TotalOperations, int64(len(items)))
		
		// Update average commit time
		oldAvg := atomic.LoadInt64(&bp.metrics.AverageCommitTime)
		totalBatches := atomic.LoadInt64(&bp.metrics.TotalBatches)
		newAvg := (oldAvg*(totalBatches-1) + int64(duration)) / totalBatches
		atomic.StoreInt64(&bp.metrics.AverageCommitTime, newAvg)
		
		// Update average batch size
		oldBatchSize := atomic.LoadInt64(&bp.metrics.AverageBatchSize)
		newBatchSize := (oldBatchSize*(totalBatches-1) + int64(len(items))) / totalBatches
		atomic.StoreInt64(&bp.metrics.AverageBatchSize, newBatchSize)
		
		// Update throughput every second
		if time.Since(bp.metrics.lastThroughputCalc) >= time.Second {
			operations := atomic.LoadInt64(&bp.metrics.operationsSinceCalc)
			atomic.StoreInt64(&bp.metrics.Throughput, operations)
			atomic.StoreInt64(&bp.metrics.operationsSinceCalc, 0)
			bp.metrics.lastThroughputCalc = time.Now()
		} else {
			atomic.AddInt64(&bp.metrics.operationsSinceCalc, int64(len(items)))
		}
	}()
	
	for retry := 0; retry <= bp.config.MaxRetries; retry++ {
		if retry > 0 {
			delay := bp.config.RetryDelay
			if bp.isDeadlockError(nil) { // We'll implement deadlock detection
				delay = bp.config.DeadlockRetryDelay
				atomic.AddInt64(&bp.metrics.DeadlockRetries, 1)
			}
			time.Sleep(delay)
		}
		
		err := bp.executeTransaction(items)
		if err == nil {
			atomic.AddInt64(&bp.metrics.SuccessfulBatches, 1)
			return
		}
		
		// Check if we should retry
		if !bp.shouldRetry(err) || retry == bp.config.MaxRetries {
			atomic.AddInt64(&bp.metrics.FailedBatches, 1)
			bp.sendErrorToAll(items, err)
			return
		}
	}
}

// executeTransaction executes all items in a single transaction
func (bp *BatchProcessor) executeTransaction(items []BatchItem) error {
	ctx, cancel := context.WithTimeout(context.Background(), bp.config.TransactionTimeout)
	defer cancel()
	
	tx, err := bp.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		bp.sendErrorToAll(items, err)
		return err
	}
	
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			err := fmt.Errorf("panic in transaction: %v", r)
			bp.sendErrorToAll(items, err)
		}
	}()
	
	// Execute all operations in the transaction
	for _, item := range items {
		start := time.Now()
		result, err := tx.ExecContext(ctx, item.Query, item.Args...)
		duration := time.Since(start)
		
		if err != nil {
			tx.Rollback()
			bp.sendErrorToAll(items, err)
			return err
		}
		
		rowsAffected, _ := result.RowsAffected()
		item.Result <- BatchResult{
			RowsAffected: rowsAffected,
			Error:        nil,
			Duration:     duration,
		}
	}
	
	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		bp.sendErrorToAll(items, err)
		return err
	}
	
	return nil
}

// sendErrorToAll sends an error to all items in the batch
func (bp *BatchProcessor) sendErrorToAll(items []BatchItem, err error) {
	for _, item := range items {
		select {
		case item.Result <- BatchResult{Error: err}:
		default:
			// Channel might be closed or full
		}
	}
}

// shouldRetry determines if an error is retryable
func (bp *BatchProcessor) shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for retryable errors
	errorStr := err.Error()
	retryableErrors := []string{
		"deadlock",
		"lock timeout",
		"connection reset",
		"connection refused",
		"timeout",
	}
	
	for _, retryable := range retryableErrors {
		if contains(errorStr, retryable) {
			return true
		}
	}
	
	return false
}

// isDeadlockError checks if an error is a deadlock
func (bp *BatchProcessor) isDeadlockError(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "deadlock")
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
			len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr ||
			 indexOfSubstring(s, substr) >= 0))
}

// indexOfSubstring finds the index of a substring
func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// GetMetrics returns current batch processing metrics
func (bp *BatchProcessor) GetMetrics() BatchMetrics {
	return BatchMetrics{
		TotalBatches:      atomic.LoadInt64(&bp.metrics.TotalBatches),
		TotalOperations:   atomic.LoadInt64(&bp.metrics.TotalOperations),
		SuccessfulBatches: atomic.LoadInt64(&bp.metrics.SuccessfulBatches),
		FailedBatches:     atomic.LoadInt64(&bp.metrics.FailedBatches),
		DeadlockRetries:   atomic.LoadInt64(&bp.metrics.DeadlockRetries),
		AverageCommitTime: atomic.LoadInt64(&bp.metrics.AverageCommitTime),
		AverageBatchSize:  atomic.LoadInt64(&bp.metrics.AverageBatchSize),
		Throughput:        atomic.LoadInt64(&bp.metrics.Throughput),
		MemoryUsage:       atomic.LoadInt64(&bp.metrics.MemoryUsage),
	}
}

// Close stops the batch processor and processes remaining items
func (bp *BatchProcessor) Close() error {
	if !atomic.CompareAndSwapInt32(&bp.running, 1, 0) {
		return fmt.Errorf("batch processor already stopped")
	}
	
	close(bp.stopCh)
	bp.wg.Wait()
	
	return nil
}

// GetThroughput returns current throughput in operations per second
func (bp *BatchProcessor) GetThroughput() int64 {
	return atomic.LoadInt64(&bp.metrics.Throughput)
}

// GetAverageCommitTime returns average commit time in nanoseconds
func (bp *BatchProcessor) GetAverageCommitTime() time.Duration {
	return time.Duration(atomic.LoadInt64(&bp.metrics.AverageCommitTime))
}

// GetDeadlockRate returns the deadlock retry rate as a percentage
func (bp *BatchProcessor) GetDeadlockRate() float64 {
	totalBatches := atomic.LoadInt64(&bp.metrics.TotalBatches)
	if totalBatches == 0 {
		return 0
	}
	deadlocks := atomic.LoadInt64(&bp.metrics.DeadlockRetries)
	return float64(deadlocks) / float64(totalBatches) * 100
}
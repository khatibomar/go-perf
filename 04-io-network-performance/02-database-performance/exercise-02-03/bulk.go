package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"
)

// BulkOperation represents different types of bulk operations
type BulkOperation int

const (
	BulkInsert BulkOperation = iota
	BulkUpdate
	BulkDelete
	BulkUpsert
)

// BulkConfig holds configuration for bulk operations
type BulkConfig struct {
	ChunkSize       int           // Number of records per chunk
	MaxConcurrency  int           // Maximum concurrent chunks
	Timeout         time.Duration // Operation timeout
	RetryAttempts   int           // Number of retry attempts
	RetryDelay      time.Duration // Delay between retries
	UseTransaction  bool          // Whether to use transactions
	OptimizeQueries bool          // Whether to optimize SQL queries
}

// DefaultBulkConfig returns default bulk operation configuration
func DefaultBulkConfig() *BulkConfig {
	return &BulkConfig{
		ChunkSize:       1000,
		MaxConcurrency:  5,
		Timeout:         30 * time.Second,
		RetryAttempts:   3,
		RetryDelay:      100 * time.Millisecond,
		UseTransaction:  true,
		OptimizeQueries: true,
	}
}

// BulkResult contains the result of a bulk operation
type BulkResult struct {
	TotalRows       int64
	AffectedRows    int64
	ProcessedChunks int
	FailedChunks    int
	Duration        time.Duration
	Errors          []error
}

// BulkProcessor handles bulk database operations
type BulkProcessor struct {
	db     *sql.DB
	config *BulkConfig
	mu     sync.RWMutex
}

// NewBulkProcessor creates a new bulk processor
func NewBulkProcessor(db *sql.DB, config *BulkConfig) *BulkProcessor {
	if config == nil {
		config = DefaultBulkConfig()
	}
	
	return &BulkProcessor{
		db:     db,
		config: config,
	}
}

// BulkInsert performs bulk insert operations
func (bp *BulkProcessor) BulkInsert(ctx context.Context, tableName string, columns []string, data [][]interface{}) (*BulkResult, error) {
	if len(data) == 0 {
		return &BulkResult{}, nil
	}
	
	start := time.Now()
	result := &BulkResult{
		TotalRows: int64(len(data)),
		Errors:    make([]error, 0),
	}
	
	// Generate optimized insert query
	query := bp.generateInsertQuery(tableName, columns, bp.config.ChunkSize)
	
	// Process data in chunks
	chunks := bp.chunkData(data, bp.config.ChunkSize)
	result.ProcessedChunks = len(chunks)
	
	// Process chunks concurrently
	semaphore := make(chan struct{}, bp.config.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	for _, chunk := range chunks {
		wg.Add(1)
		go func(chunkData [][]interface{}) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			affected, err := bp.executeInsertChunk(ctx, query, columns, chunkData)
			
			mu.Lock()
			if err != nil {
				result.FailedChunks++
				result.Errors = append(result.Errors, err)
			} else {
				result.AffectedRows += affected
			}
			mu.Unlock()
		}(chunk)
	}
	
	wg.Wait()
	result.Duration = time.Since(start)
	
	return result, nil
}

// BulkUpdate performs bulk update operations
func (bp *BulkProcessor) BulkUpdate(ctx context.Context, tableName string, updates []BulkUpdateItem) (*BulkResult, error) {
	if len(updates) == 0 {
		return &BulkResult{}, nil
	}
	
	start := time.Now()
	result := &BulkResult{
		TotalRows: int64(len(updates)),
		Errors:    make([]error, 0),
	}
	
	// Group updates by their SET clauses for optimization
	groupedUpdates := bp.groupUpdatesByPattern(updates)
	
	for pattern, items := range groupedUpdates {
		chunks := bp.chunkUpdateData(items, bp.config.ChunkSize)
		result.ProcessedChunks += len(chunks)
		
		// Process chunks for this pattern
		for _, chunk := range chunks {
			affected, err := bp.executeUpdateChunk(ctx, tableName, pattern, chunk)
			if err != nil {
				result.FailedChunks++
				result.Errors = append(result.Errors, err)
			} else {
				result.AffectedRows += affected
			}
		}
	}
	
	result.Duration = time.Since(start)
	return result, nil
}

// BulkDelete performs bulk delete operations
func (bp *BulkProcessor) BulkDelete(ctx context.Context, tableName string, whereColumn string, values []interface{}) (*BulkResult, error) {
	if len(values) == 0 {
		return &BulkResult{}, nil
	}
	
	start := time.Now()
	result := &BulkResult{
		TotalRows: int64(len(values)),
		Errors:    make([]error, 0),
	}
	
	// Process values in chunks
	chunks := bp.chunkValues(values, bp.config.ChunkSize)
	result.ProcessedChunks = len(chunks)
	
	for _, chunk := range chunks {
		affected, err := bp.executeDeleteChunk(ctx, tableName, whereColumn, chunk)
		if err != nil {
			result.FailedChunks++
			result.Errors = append(result.Errors, err)
		} else {
			result.AffectedRows += affected
		}
	}
	
	result.Duration = time.Since(start)
	return result, nil
}

// BulkUpsert performs bulk upsert (insert or update) operations
func (bp *BulkProcessor) BulkUpsert(ctx context.Context, tableName string, columns []string, data [][]interface{}, conflictColumns []string) (*BulkResult, error) {
	if len(data) == 0 {
		return &BulkResult{}, nil
	}
	
	start := time.Now()
	result := &BulkResult{
		TotalRows: int64(len(data)),
		Errors:    make([]error, 0),
	}
	
	// Generate upsert query (PostgreSQL style - can be adapted for other DBs)
	query := bp.generateUpsertQuery(tableName, columns, conflictColumns, bp.config.ChunkSize)
	
	// Process data in chunks
	chunks := bp.chunkData(data, bp.config.ChunkSize)
	result.ProcessedChunks = len(chunks)
	
	for _, chunk := range chunks {
		affected, err := bp.executeUpsertChunk(ctx, query, columns, chunk)
		if err != nil {
			result.FailedChunks++
			result.Errors = append(result.Errors, err)
		} else {
			result.AffectedRows += affected
		}
	}
	
	result.Duration = time.Since(start)
	return result, nil
}

// BulkUpdateItem represents a single update operation
type BulkUpdateItem struct {
	SetClause   string        // e.g., "name = ?, age = ?"
	SetValues   []interface{} // Values for SET clause
	WhereClause string        // e.g., "id = ?"
	WhereValues []interface{} // Values for WHERE clause
}

// generateInsertQuery creates an optimized bulk insert query
func (bp *BulkProcessor) generateInsertQuery(tableName string, columns []string, chunkSize int) string {
	if !bp.config.OptimizeQueries {
		// Simple single-row insert
		placeholders := strings.Repeat("?,", len(columns))
		placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma
		return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			tableName, strings.Join(columns, ","), placeholders)
	}
	
	// Multi-row insert for better performance
	columnList := strings.Join(columns, ",")
	placeholders := strings.Repeat("?,", len(columns))
	placeholders = placeholders[:len(placeholders)-1]
	
	valueGroups := make([]string, chunkSize)
	for i := 0; i < chunkSize; i++ {
		valueGroups[i] = fmt.Sprintf("(%s)", placeholders)
	}
	
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		tableName, columnList, strings.Join(valueGroups, ","))
}

// generateUpsertQuery creates an optimized upsert query
func (bp *BulkProcessor) generateUpsertQuery(tableName string, columns []string, conflictColumns []string, chunkSize int) string {
	columnList := strings.Join(columns, ",")
	placeholders := strings.Repeat("?,", len(columns))
	placeholders = placeholders[:len(placeholders)-1]
	
	valueGroups := make([]string, chunkSize)
	for i := 0; i < chunkSize; i++ {
		valueGroups[i] = fmt.Sprintf("(%s)", placeholders)
	}
	
	// Create UPDATE clause for conflict resolution
	updateClauses := make([]string, 0, len(columns))
	for _, col := range columns {
		// Skip conflict columns in UPDATE
		isConflictCol := false
		for _, conflictCol := range conflictColumns {
			if col == conflictCol {
				isConflictCol = true
				break
			}
		}
		if !isConflictCol {
			updateClauses = append(updateClauses, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
		}
	}
	
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES %s ON CONFLICT (%s) DO UPDATE SET %s",
		tableName, columnList, strings.Join(valueGroups, ","),
		strings.Join(conflictColumns, ","), strings.Join(updateClauses, ","))
}

// executeInsertChunk executes a chunk of insert operations
func (bp *BulkProcessor) executeInsertChunk(ctx context.Context, query string, columns []string, data [][]interface{}) (int64, error) {
	if bp.config.OptimizeQueries && len(data) > 1 {
		// Multi-row insert
		args := make([]interface{}, 0, len(data)*len(columns))
		for _, row := range data {
			args = append(args, row...)
		}
		
		return bp.executeWithRetry(ctx, query, args)
	} else {
		// Single-row inserts
		var totalAffected int64
		for _, row := range data {
			affected, err := bp.executeWithRetry(ctx, query, row)
			if err != nil {
				return totalAffected, err
			}
			totalAffected += affected
		}
		return totalAffected, nil
	}
}

// executeUpdateChunk executes a chunk of update operations
func (bp *BulkProcessor) executeUpdateChunk(ctx context.Context, tableName string, pattern BulkUpdatePattern, items []BulkUpdateItem) (int64, error) {
	if bp.config.UseTransaction && len(items) > 1 {
		return bp.executeUpdateChunkInTransaction(ctx, tableName, pattern, items)
	}
	
	var totalAffected int64
	for _, item := range items {
		query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", tableName, item.SetClause, item.WhereClause)
		args := append(item.SetValues, item.WhereValues...)
		
		affected, err := bp.executeWithRetry(ctx, query, args)
		if err != nil {
			return totalAffected, err
		}
		totalAffected += affected
	}
	
	return totalAffected, nil
}

// executeUpdateChunkInTransaction executes updates in a transaction
func (bp *BulkProcessor) executeUpdateChunkInTransaction(ctx context.Context, tableName string, pattern BulkUpdatePattern, items []BulkUpdateItem) (int64, error) {
	tx, err := bp.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	
	var totalAffected int64
	for _, item := range items {
		query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", tableName, item.SetClause, item.WhereClause)
		args := append(item.SetValues, item.WhereValues...)
		
		result, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return totalAffected, err
		}
		
		affected, _ := result.RowsAffected()
		totalAffected += affected
	}
	
	err = tx.Commit()
	if err != nil {
		return totalAffected, err
	}
	
	return totalAffected, nil
}

// executeDeleteChunk executes a chunk of delete operations
func (bp *BulkProcessor) executeDeleteChunk(ctx context.Context, tableName string, whereColumn string, values []interface{}) (int64, error) {
	if bp.config.OptimizeQueries && len(values) > 1 {
		// Use IN clause for better performance
		placeholders := strings.Repeat("?,", len(values))
		placeholders = placeholders[:len(placeholders)-1]
		
		query := fmt.Sprintf("DELETE FROM %s WHERE %s IN (%s)", tableName, whereColumn, placeholders)
		return bp.executeWithRetry(ctx, query, values)
	} else {
		// Individual deletes
		var totalAffected int64
		query := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", tableName, whereColumn)
		
		for _, value := range values {
			affected, err := bp.executeWithRetry(ctx, query, []interface{}{value})
			if err != nil {
				return totalAffected, err
			}
			totalAffected += affected
		}
		return totalAffected, nil
	}
}

// executeUpsertChunk executes a chunk of upsert operations
func (bp *BulkProcessor) executeUpsertChunk(ctx context.Context, query string, columns []string, data [][]interface{}) (int64, error) {
	args := make([]interface{}, 0, len(data)*len(columns))
	for _, row := range data {
		args = append(args, row...)
	}
	
	return bp.executeWithRetry(ctx, query, args)
}

// executeWithRetry executes a query with retry logic
func (bp *BulkProcessor) executeWithRetry(ctx context.Context, query string, args []interface{}) (int64, error) {
	var lastErr error
	
	for attempt := 0; attempt <= bp.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(bp.config.RetryDelay):
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		}
		
		result, err := bp.db.ExecContext(ctx, query, args...)
		if err == nil {
			affected, _ := result.RowsAffected()
			return affected, nil
		}
		
		lastErr = err
		
		// Check if error is retryable
		if !bp.isRetryableError(err) {
			break
		}
	}
	
	return 0, lastErr
}

// isRetryableError determines if an error is retryable
func (bp *BulkProcessor) isRetryableError(err error) bool {
	errorStr := strings.ToLower(err.Error())
	retryableErrors := []string{
		"deadlock",
		"lock timeout",
		"connection reset",
		"timeout",
		"temporary",
	}
	
	for _, retryable := range retryableErrors {
		if strings.Contains(errorStr, retryable) {
			return true
		}
	}
	
	return false
}

// BulkUpdatePattern represents a pattern for grouping similar updates
type BulkUpdatePattern struct {
	SetClause   string
	WhereClause string
}

// groupUpdatesByPattern groups updates by their SET and WHERE patterns
func (bp *BulkProcessor) groupUpdatesByPattern(updates []BulkUpdateItem) map[BulkUpdatePattern][]BulkUpdateItem {
	groups := make(map[BulkUpdatePattern][]BulkUpdateItem)
	
	for _, update := range updates {
		pattern := BulkUpdatePattern{
			SetClause:   update.SetClause,
			WhereClause: update.WhereClause,
		}
		
		groups[pattern] = append(groups[pattern], update)
	}
	
	return groups
}

// chunkData splits data into chunks of specified size
func (bp *BulkProcessor) chunkData(data [][]interface{}, chunkSize int) [][][]interface{} {
	if chunkSize <= 0 {
		chunkSize = 1000
	}
	
	var chunks [][][]interface{}
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	
	return chunks
}

// chunkUpdateData splits update data into chunks
func (bp *BulkProcessor) chunkUpdateData(data []BulkUpdateItem, chunkSize int) [][]BulkUpdateItem {
	if chunkSize <= 0 {
		chunkSize = 1000
	}
	
	var chunks [][]BulkUpdateItem
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	
	return chunks
}

// chunkValues splits values into chunks
func (bp *BulkProcessor) chunkValues(values []interface{}, chunkSize int) [][]interface{} {
	if chunkSize <= 0 {
		chunkSize = 1000
	}
	
	var chunks [][]interface{}
	for i := 0; i < len(values); i += chunkSize {
		end := i + chunkSize
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[i:end])
	}
	
	return chunks
}

// SetConfig updates the bulk processor configuration
func (bp *BulkProcessor) SetConfig(config *BulkConfig) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.config = config
}

// GetConfig returns the current configuration
func (bp *BulkProcessor) GetConfig() *BulkConfig {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	return bp.config
}
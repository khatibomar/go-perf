package main

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DeadlockDetector provides deadlock detection and prevention
type DeadlockDetector struct {
	db                *sql.DB
	lockOrder         map[string]int    // Resource ordering for deadlock prevention
	activeTransactions sync.Map          // Track active transactions
	deadlockCount     int64             // Count of detected deadlocks
	preventionEnabled bool              // Whether prevention is enabled
	mu                sync.RWMutex
}

// TransactionInfo holds information about an active transaction
type TransactionInfo struct {
	ID        string
	StartTime time.Time
	Resources []string // Resources locked by this transaction
	Context   context.Context
	Cancel    context.CancelFunc
}

// DeadlockPreventionStrategy defines different prevention strategies
type DeadlockPreventionStrategy int

const (
	ResourceOrdering DeadlockPreventionStrategy = iota
	TimeoutBased
	WaitDie
	WoundWait
)

// DeadlockConfig holds configuration for deadlock detection and prevention
type DeadlockConfig struct {
	Strategy           DeadlockPreventionStrategy
	DetectionInterval  time.Duration
	TransactionTimeout time.Duration
	MaxRetries         int
	RetryDelay         time.Duration
	EnableLogging      bool
}

// DefaultDeadlockConfig returns default deadlock configuration
func DefaultDeadlockConfig() *DeadlockConfig {
	return &DeadlockConfig{
		Strategy:           ResourceOrdering,
		DetectionInterval:  100 * time.Millisecond,
		TransactionTimeout: 30 * time.Second,
		MaxRetries:         3,
		RetryDelay:         50 * time.Millisecond,
		EnableLogging:      true,
	}
}

// NewDeadlockDetector creates a new deadlock detector
func NewDeadlockDetector(db *sql.DB, config *DeadlockConfig) *DeadlockDetector {
	if config == nil {
		config = DefaultDeadlockConfig()
	}
	
	dd := &DeadlockDetector{
		db:                db,
		lockOrder:         make(map[string]int),
		preventionEnabled: true,
	}
	
	// Initialize default resource ordering
	dd.initializeResourceOrdering()
	
	return dd
}

// initializeResourceOrdering sets up default resource ordering
func (dd *DeadlockDetector) initializeResourceOrdering() {
	// Common table ordering to prevent deadlocks
	defaultTables := []string{
		"users",
		"accounts",
		"transactions",
		"orders",
		"products",
		"inventory",
		"audit_log",
	}
	
	for i, table := range defaultTables {
		dd.lockOrder[table] = i
	}
}

// SetResourceOrder sets the locking order for resources
func (dd *DeadlockDetector) SetResourceOrder(resources []string) {
	dd.mu.Lock()
	defer dd.mu.Unlock()
	
	dd.lockOrder = make(map[string]int)
	for i, resource := range resources {
		dd.lockOrder[resource] = i
	}
}

// AddResourceOrder adds a resource to the ordering
func (dd *DeadlockDetector) AddResourceOrder(resource string, order int) {
	dd.mu.Lock()
	defer dd.mu.Unlock()
	dd.lockOrder[resource] = order
}

// BeginTransaction starts a new transaction with deadlock prevention
func (dd *DeadlockDetector) BeginTransaction(ctx context.Context, resources []string) (*sql.Tx, *TransactionInfo, error) {
	// Sort resources by their defined order to prevent deadlocks
	orderedResources := dd.sortResourcesByOrder(resources)
	
	// Create transaction context with timeout
	txCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	
	tx, err := dd.db.BeginTx(txCtx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		cancel()
		return nil, nil, err
	}
	
	// Create transaction info
	txInfo := &TransactionInfo{
		ID:        generateTransactionID(),
		StartTime: time.Now(),
		Resources: orderedResources,
		Context:   txCtx,
		Cancel:    cancel,
	}
	
	// Register transaction
	dd.activeTransactions.Store(txInfo.ID, txInfo)
	
	return tx, txInfo, nil
}

// CommitTransaction commits a transaction and cleans up tracking
func (dd *DeadlockDetector) CommitTransaction(tx *sql.Tx, txInfo *TransactionInfo) error {
	defer func() {
		dd.activeTransactions.Delete(txInfo.ID)
		txInfo.Cancel()
	}()
	
	return tx.Commit()
}

// RollbackTransaction rolls back a transaction and cleans up tracking
func (dd *DeadlockDetector) RollbackTransaction(tx *sql.Tx, txInfo *TransactionInfo) error {
	defer func() {
		dd.activeTransactions.Delete(txInfo.ID)
		txInfo.Cancel()
	}()
	
	return tx.Rollback()
}

// ExecuteWithDeadlockPrevention executes a function with deadlock prevention
func (dd *DeadlockDetector) ExecuteWithDeadlockPrevention(ctx context.Context, resources []string, fn func(*sql.Tx) error) error {
	var lastErr error
	
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := time.Duration(attempt*attempt) * 50 * time.Millisecond
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		
		tx, txInfo, err := dd.BeginTransaction(ctx, resources)
		if err != nil {
			lastErr = err
			continue
		}
		
		err = fn(tx)
		if err != nil {
			dd.RollbackTransaction(tx, txInfo)
			
			// Check if it's a deadlock error
			if dd.isDeadlockError(err) {
				atomic.AddInt64(&dd.deadlockCount, 1)
				lastErr = err
				continue // Retry
			}
			
			return err // Non-retryable error
		}
		
		err = dd.CommitTransaction(tx, txInfo)
		if err != nil {
			if dd.isDeadlockError(err) {
				atomic.AddInt64(&dd.deadlockCount, 1)
				lastErr = err
				continue // Retry
			}
			return err
		}
		
		return nil // Success
	}
	
	return fmt.Errorf("transaction failed after retries: %w", lastErr)
}

// sortResourcesByOrder sorts resources according to the defined order
func (dd *DeadlockDetector) sortResourcesByOrder(resources []string) []string {
	dd.mu.RLock()
	defer dd.mu.RUnlock()
	
	// Create a copy to avoid modifying the original slice
	sorted := make([]string, len(resources))
	copy(sorted, resources)
	
	sort.Slice(sorted, func(i, j int) bool {
		orderI, existsI := dd.lockOrder[sorted[i]]
		orderJ, existsJ := dd.lockOrder[sorted[j]]
		
		// If both have defined orders, sort by order
		if existsI && existsJ {
			return orderI < orderJ
		}
		
		// If only one has a defined order, it comes first
		if existsI {
			return true
		}
		if existsJ {
			return false
		}
		
		// If neither has a defined order, sort alphabetically
		return sorted[i] < sorted[j]
	})
	
	return sorted
}

// isDeadlockError checks if an error is a deadlock
func (dd *DeadlockDetector) isDeadlockError(err error) bool {
	if err == nil {
		return false
	}
	
	errorStr := strings.ToLower(err.Error())
	deadlockKeywords := []string{
		"deadlock",
		"lock timeout",
		"lock wait timeout",
		"serialization failure",
		"could not serialize",
	}
	
	for _, keyword := range deadlockKeywords {
		if strings.Contains(errorStr, keyword) {
			return true
		}
	}
	
	return false
}

// DetectDeadlocks performs active deadlock detection
func (dd *DeadlockDetector) DetectDeadlocks() []string {
	var deadlocks []string
	
	// Get all active transactions
	var transactions []*TransactionInfo
	dd.activeTransactions.Range(func(key, value interface{}) bool {
		if txInfo, ok := value.(*TransactionInfo); ok {
			transactions = append(transactions, txInfo)
		}
		return true
	})
	
	// Check for potential deadlocks using wait-for graph analysis
	for i := 0; i < len(transactions); i++ {
		for j := i + 1; j < len(transactions); j++ {
			if dd.hasCircularDependency(transactions[i], transactions[j]) {
				deadlock := fmt.Sprintf("Potential deadlock between transactions %s and %s",
					transactions[i].ID, transactions[j].ID)
				deadlocks = append(deadlocks, deadlock)
			}
		}
	}
	
	return deadlocks
}

// hasCircularDependency checks if two transactions have circular dependency
func (dd *DeadlockDetector) hasCircularDependency(tx1, tx2 *TransactionInfo) bool {
	// Simple heuristic: check if transactions have overlapping resources
	// and different acquisition orders
	resources1 := make(map[string]bool)
	for _, resource := range tx1.Resources {
		resources1[resource] = true
	}
	
	commonResources := 0
	for _, resource := range tx2.Resources {
		if resources1[resource] {
			commonResources++
		}
	}
	
	// If they share resources and started at different times,
	// there might be a deadlock risk
	return commonResources > 0 && tx1.StartTime.Before(tx2.StartTime)
}

// ResolveDeadlock attempts to resolve a detected deadlock
func (dd *DeadlockDetector) ResolveDeadlock(deadlockInfo string) error {
	// Extract transaction IDs from deadlock info
	// This is a simplified implementation
	parts := strings.Split(deadlockInfo, " ")
	if len(parts) < 6 {
		return fmt.Errorf("invalid deadlock info format")
	}
	
	tx1ID := parts[4]
	tx2ID := parts[6]
	
	// Get transaction info
	tx1Value, exists1 := dd.activeTransactions.Load(tx1ID)
	tx2Value, exists2 := dd.activeTransactions.Load(tx2ID)
	
	if !exists1 || !exists2 {
		return fmt.Errorf("transactions not found")
	}
	
	tx1Info := tx1Value.(*TransactionInfo)
	tx2Info := tx2Value.(*TransactionInfo)
	
	// Use youngest transaction wins strategy
	if tx1Info.StartTime.After(tx2Info.StartTime) {
		// Cancel tx1 (younger transaction)
		tx1Info.Cancel()
		dd.activeTransactions.Delete(tx1ID)
	} else {
		// Cancel tx2 (younger transaction)
		tx2Info.Cancel()
		dd.activeTransactions.Delete(tx2ID)
	}
	
	return nil
}

// GetDeadlockCount returns the number of detected deadlocks
func (dd *DeadlockDetector) GetDeadlockCount() int64 {
	return atomic.LoadInt64(&dd.deadlockCount)
}

// GetActiveTransactionCount returns the number of active transactions
func (dd *DeadlockDetector) GetActiveTransactionCount() int {
	count := 0
	dd.activeTransactions.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// GetActiveTransactions returns information about active transactions
func (dd *DeadlockDetector) GetActiveTransactions() []*TransactionInfo {
	var transactions []*TransactionInfo
	dd.activeTransactions.Range(func(key, value interface{}) bool {
		if txInfo, ok := value.(*TransactionInfo); ok {
			transactions = append(transactions, txInfo)
		}
		return true
	})
	return transactions
}

// CleanupExpiredTransactions removes expired transactions
func (dd *DeadlockDetector) CleanupExpiredTransactions(maxAge time.Duration) int {
	now := time.Now()
	expiredCount := 0
	
	dd.activeTransactions.Range(func(key, value interface{}) bool {
		if txInfo, ok := value.(*TransactionInfo); ok {
			if now.Sub(txInfo.StartTime) > maxAge {
				txInfo.Cancel()
				dd.activeTransactions.Delete(key)
				expiredCount++
			}
		}
		return true
	})
	
	return expiredCount
}

// StartDeadlockMonitor starts a background goroutine to monitor for deadlocks
func (dd *DeadlockDetector) StartDeadlockMonitor(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				// Detect deadlocks
				deadlocks := dd.DetectDeadlocks()
				for _, deadlock := range deadlocks {
					// Attempt to resolve deadlock
					dd.ResolveDeadlock(deadlock)
				}
				
				// Cleanup expired transactions
				dd.CleanupExpiredTransactions(5 * time.Minute)
				
			case <-ctx.Done():
				return
			}
		}
	}()
}

// generateTransactionID generates a unique transaction ID
func generateTransactionID() string {
	return fmt.Sprintf("tx_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}

// DeadlockStats provides statistics about deadlock detection
type DeadlockStats struct {
	TotalDeadlocks      int64
	ActiveTransactions  int
	AverageTransactionAge time.Duration
	ResourceOrderingEnabled bool
}

// GetStats returns deadlock detection statistics
func (dd *DeadlockDetector) GetStats() DeadlockStats {
	activeTransactions := dd.GetActiveTransactions()
	
	var totalAge time.Duration
	now := time.Now()
	for _, tx := range activeTransactions {
		totalAge += now.Sub(tx.StartTime)
	}
	
	var avgAge time.Duration
	if len(activeTransactions) > 0 {
		avgAge = totalAge / time.Duration(len(activeTransactions))
	}
	
	return DeadlockStats{
		TotalDeadlocks:      dd.GetDeadlockCount(),
		ActiveTransactions:  len(activeTransactions),
		AverageTransactionAge: avgAge,
		ResourceOrderingEnabled: dd.preventionEnabled,
	}
}

// SetPreventionEnabled enables or disables deadlock prevention
func (dd *DeadlockDetector) SetPreventionEnabled(enabled bool) {
	dd.mu.Lock()
	defer dd.mu.Unlock()
	dd.preventionEnabled = enabled
}

// IsPreventionEnabled returns whether deadlock prevention is enabled
func (dd *DeadlockDetector) IsPreventionEnabled() bool {
	dd.mu.RLock()
	defer dd.mu.RUnlock()
	return dd.preventionEnabled
}
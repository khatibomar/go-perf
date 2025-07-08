package correctness

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Operation represents a queue operation
type Operation struct {
	Type      string      // "enqueue" or "dequeue"
	Value     interface{} // Value for enqueue, result for dequeue
	Success   bool        // Whether operation succeeded
	StartTime time.Time
	EndTime   time.Time
	ThreadID  int
}

// History represents a sequence of operations
type History []Operation

// LinearizabilityChecker checks if a history is linearizable
type LinearizabilityChecker struct {
	history History
	mu      sync.RWMutex
}

// NewLinearizabilityChecker creates a new linearizability checker
func NewLinearizabilityChecker() *LinearizabilityChecker {
	return &LinearizabilityChecker{
		history: make(History, 0),
	}
}

// RecordOperation records an operation in the history
func (lc *LinearizabilityChecker) RecordOperation(op Operation) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.history = append(lc.history, op)
}

// CheckLinearizability checks if the recorded history is linearizable
func (lc *LinearizabilityChecker) CheckLinearizability() (bool, error) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	
	if len(lc.history) == 0 {
		return true, nil
	}
	
	// Sort operations by start time
	sortedHistory := make(History, len(lc.history))
	copy(sortedHistory, lc.history)
	sort.Slice(sortedHistory, func(i, j int) bool {
		return sortedHistory[i].StartTime.Before(sortedHistory[j].StartTime)
	})
	
	// Try to find a valid linearization
	return lc.findLinearization(sortedHistory, []interface{}{}, 0)
}

// findLinearization recursively tries to find a valid linearization
func (lc *LinearizabilityChecker) findLinearization(remaining History, queueState []interface{}, index int) (bool, error) {
	if len(remaining) == 0 {
		return true, nil
	}
	
	// Try each remaining operation as the next in the linearization
	for i, op := range remaining {
		// Check if this operation can be linearized at this point
		if lc.canLinearizeOperation(op, queueState, remaining[:i], remaining[i+1:]) {
			// Apply the operation to get new state
			newState, valid := lc.applyOperation(op, queueState)
			if !valid {
				continue
			}
			
			// Create new remaining list without this operation
			newRemaining := make(History, 0, len(remaining)-1)
			newRemaining = append(newRemaining, remaining[:i]...)
			newRemaining = append(newRemaining, remaining[i+1:]...)
			
			// Recursively check the rest
			if valid, err := lc.findLinearization(newRemaining, newState, index+1); valid {
				return true, err
			} else if err != nil {
				return false, err
			}
		}
	}
	
	return false, nil
}

// canLinearizeOperation checks if an operation can be linearized at this point
func (lc *LinearizabilityChecker) canLinearizeOperation(op Operation, queueState []interface{}, before, after History) bool {
	// Check temporal constraints - operation must not start after any operation that ends before it starts
	for _, other := range before {
		if other.EndTime.Before(op.StartTime) {
			// This operation must come after 'other' in any valid linearization
			return false
		}
	}
	
	for _, other := range after {
		if op.EndTime.Before(other.StartTime) {
			// This operation must come before 'other' in any valid linearization
			// This is okay for current position
		}
	}
	
	return true
}

// applyOperation applies an operation to the queue state
func (lc *LinearizabilityChecker) applyOperation(op Operation, queueState []interface{}) ([]interface{}, bool) {
	newState := make([]interface{}, len(queueState))
	copy(newState, queueState)
	
	switch op.Type {
	case "enqueue":
		if op.Success {
			newState = append(newState, op.Value)
			return newState, true
		}
		// Failed enqueue (e.g., queue full) - state unchanged
		return newState, true
		
	case "dequeue":
		if op.Success {
			if len(newState) == 0 {
				// Cannot dequeue from empty queue
				return nil, false
			}
			// Check if dequeued value matches expected
			if newState[0] != op.Value {
				return nil, false
			}
			newState = newState[1:]
			return newState, true
		} else {
			// Failed dequeue - queue should be empty
			if len(newState) != 0 {
				return nil, false
			}
			return newState, true
		}
	}
	
	return nil, false
}

// Reset clears the recorded history
func (lc *LinearizabilityChecker) Reset() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.history = lc.history[:0]
}

// GetHistory returns a copy of the current history
func (lc *LinearizabilityChecker) GetHistory() History {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	
	history := make(History, len(lc.history))
	copy(history, lc.history)
	return history
}

// AnalyzeHistory provides detailed analysis of the operation history
func (lc *LinearizabilityChecker) AnalyzeHistory() HistoryAnalysis {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	
	analysis := HistoryAnalysis{
		TotalOperations: len(lc.history),
		ThreadStats:     make(map[int]ThreadStats),
	}
	
	var totalDuration time.Duration
	enqueueCount := 0
	dequeueCount := 0
	successfulOps := 0
	
	for _, op := range lc.history {
		duration := op.EndTime.Sub(op.StartTime)
		totalDuration += duration
		
		if op.Success {
			successfulOps++
		}
		
		switch op.Type {
		case "enqueue":
			enqueueCount++
		case "dequeue":
			dequeueCount++
		}
		
		// Update thread stats
		stats := analysis.ThreadStats[op.ThreadID]
		stats.OperationCount++
		stats.TotalDuration += duration
		if duration > stats.MaxDuration {
			stats.MaxDuration = duration
		}
		if stats.MinDuration == 0 || duration < stats.MinDuration {
			stats.MinDuration = duration
		}
		analysis.ThreadStats[op.ThreadID] = stats
	}
	
	analysis.EnqueueCount = enqueueCount
	analysis.DequeueCount = dequeueCount
	analysis.SuccessfulOperations = successfulOps
	analysis.SuccessRate = float64(successfulOps) / float64(len(lc.history))
	
	if len(lc.history) > 0 {
		analysis.AverageLatency = totalDuration / time.Duration(len(lc.history))
		analysis.StartTime = lc.history[0].StartTime
		analysis.EndTime = lc.history[len(lc.history)-1].EndTime
		analysis.TotalTestDuration = analysis.EndTime.Sub(analysis.StartTime)
	}
	
	return analysis
}

// HistoryAnalysis contains analysis results of operation history
type HistoryAnalysis struct {
	TotalOperations     int
	EnqueueCount        int
	DequeueCount        int
	SuccessfulOperations int
	SuccessRate         float64
	AverageLatency      time.Duration
	StartTime           time.Time
	EndTime             time.Time
	TotalTestDuration   time.Duration
	ThreadStats         map[int]ThreadStats
}

// ThreadStats contains per-thread statistics
type ThreadStats struct {
	OperationCount int
	TotalDuration  time.Duration
	MinDuration    time.Duration
	MaxDuration    time.Duration
}

// String returns a string representation of the analysis
func (ha HistoryAnalysis) String() string {
	return fmt.Sprintf(
		"History Analysis:\n"+
		"  Total Operations: %d\n"+
		"  Enqueues: %d, Dequeues: %d\n"+
		"  Successful: %d (%.2f%%)\n"+
		"  Average Latency: %v\n"+
		"  Total Test Duration: %v\n"+
		"  Threads: %d\n",
		ha.TotalOperations,
		ha.EnqueueCount, ha.DequeueCount,
		ha.SuccessfulOperations, ha.SuccessRate*100,
		ha.AverageLatency,
		ha.TotalTestDuration,
		len(ha.ThreadStats),
	)
}

// ValidateQueueProperties validates basic queue properties
func (lc *LinearizabilityChecker) ValidateQueueProperties() []string {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	
	var violations []string
	
	// Check FIFO property by tracking enqueue/dequeue pairs
	enqueued := make([]interface{}, 0)
	dequeued := make([]interface{}, 0)
	
	for _, op := range lc.history {
		if op.Success {
			switch op.Type {
			case "enqueue":
				enqueued = append(enqueued, op.Value)
			case "dequeue":
				dequeued = append(dequeued, op.Value)
			}
		}
	}
	
	// Check if dequeued values follow FIFO order
	minLen := len(enqueued)
	if len(dequeued) < minLen {
		minLen = len(dequeued)
	}
	
	for i := 0; i < minLen; i++ {
		if enqueued[i] != dequeued[i] {
			violations = append(violations, fmt.Sprintf(
				"FIFO violation: expected %v at position %d, got %v",
				enqueued[i], i, dequeued[i]))
			break
		}
	}
	
	return violations
}
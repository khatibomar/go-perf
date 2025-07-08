package workers

import (
	"context"
	"sync"
	"time"
)

// BatchWorker processes work items in batches for improved efficiency
type BatchWorker struct {
	*BasicWorker
	batchSize     int
	batchTimeout  time.Duration
	batchBuffer   []WorkItem
	batchMu       sync.Mutex
	batchTimer    *time.Timer
	batchStats    BatchStats
	batchStatsMu  sync.RWMutex
	processBatch  func([]WorkItem) error
}

// BatchStats tracks batch processing statistics
type BatchStats struct {
	BatchesProcessed    int64
	ItemsInBatches      int64
	AverageBatchSize    float64
	AverageBatchTime    time.Duration
	TotalBatchTime      time.Duration
	BatchTimeouts       int64
	OptimalBatchSize    int
	BatchEfficiency     float64
}

// BatchWorkItem represents work that can be batched
type BatchWorkItem struct {
	ctx               context.Context
	priority          int
	estimatedDuration time.Duration
	batchFunc         func() error
	batchKey          string // Key for grouping batchable items
	batchable         bool
	data              interface{}
}

// NewBatchWorker creates a new batch processing worker
func NewBatchWorker(id string, queueSize int, batchSize int, batchTimeout time.Duration) *BatchWorker {
	basicWorker := NewBasicWorker(id, queueSize)
	
	bw := &BatchWorker{
		BasicWorker:  basicWorker,
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		batchBuffer:  make([]WorkItem, 0, batchSize),
		processBatch: defaultBatchProcessor,
	}
	
	return bw
}

// Start begins the batch worker with batch processing timer
func (w *BatchWorker) Start(ctx context.Context) error {
	err := w.BasicWorker.Start(ctx)
	if err != nil {
		return err
	}
	
	// Start batch timer
	w.batchTimer = time.NewTimer(w.batchTimeout)
	go w.batchTimerHandler()
	
	return nil
}

// Submit adds work to the batch buffer
func (w *BatchWorker) Submit(work WorkItem) error {
	w.batchMu.Lock()
	defer w.batchMu.Unlock()
	
	w.batchBuffer = append(w.batchBuffer, work)
	
	// Process batch if it's full
	if len(w.batchBuffer) >= w.batchSize {
		w.processBatchBuffer()
	}
	
	return nil
}

// SubmitBatchWork submits work specifically designed for batching
func (w *BatchWorker) SubmitBatchWork(batchFunc func() error, batchKey string, priority int, data interface{}) error {
	workItem := &BatchWorkItem{
		ctx:               context.Background(),
		priority:          priority,
		estimatedDuration: time.Millisecond * 50,
		batchFunc:         batchFunc,
		batchKey:          batchKey,
		batchable:         true,
		data:              data,
	}
	
	return w.Submit(workItem)
}

// batchTimerHandler handles batch timeout events
func (w *BatchWorker) batchTimerHandler() {
	for {
		select {
		case <-w.batchTimer.C:
			w.batchMu.Lock()
			if len(w.batchBuffer) > 0 {
				w.processBatchBuffer()
				w.batchStatsMu.Lock()
				w.batchStats.BatchTimeouts++
				w.batchStatsMu.Unlock()
			}
			w.batchTimer.Reset(w.batchTimeout)
			w.batchMu.Unlock()
		case <-w.ctx.Done():
			return
		}
	}
}

// processBatchBuffer processes the current batch buffer
func (w *BatchWorker) processBatchBuffer() {
	if len(w.batchBuffer) == 0 {
		return
	}
	
	batch := make([]WorkItem, len(w.batchBuffer))
	copy(batch, w.batchBuffer)
	w.batchBuffer = w.batchBuffer[:0] // Clear buffer
	
	// Process batch asynchronously
	go w.processBatchAsync(batch)
}

// processBatchAsync processes a batch of work items
func (w *BatchWorker) processBatchAsync(batch []WorkItem) {
	start := time.Now()
	
	w.statsMu.Lock()
	w.stats.TasksInProgress += int64(len(batch))
	w.statsMu.Unlock()
	
	// Group batchable items by batch key
	batchGroups := w.groupBatchableItems(batch)
	
	// Process each group
	var wg sync.WaitGroup
	for _, group := range batchGroups {
		wg.Add(1)
		go func(items []WorkItem) {
			defer wg.Done()
			w.processGroup(items)
		}(group)
	}
	
	wg.Wait()
	
	duration := time.Since(start)
	
	// Update statistics
	w.updateBatchStats(len(batch), duration)
	
	w.statsMu.Lock()
	w.stats.TasksInProgress -= int64(len(batch))
	w.stats.TasksProcessed += int64(len(batch))
	w.stats.TotalExecutionTime += duration
	w.stats.AverageExecutionTime = w.stats.TotalExecutionTime / time.Duration(w.stats.TasksProcessed)
	w.stats.LastActivity = time.Now()
	w.statsMu.Unlock()
}

// groupBatchableItems groups work items by their batch key
func (w *BatchWorker) groupBatchableItems(batch []WorkItem) [][]WorkItem {
	groups := make(map[string][]WorkItem)
	var ungrouped []WorkItem
	
	for _, item := range batch {
		if batchItem, ok := item.(*BatchWorkItem); ok && batchItem.batchable {
			key := batchItem.batchKey
			groups[key] = append(groups[key], item)
		} else {
			ungrouped = append(ungrouped, item)
		}
	}
	
	// Convert map to slice
	result := make([][]WorkItem, 0, len(groups)+1)
	for _, group := range groups {
		result = append(result, group)
	}
	
	// Add ungrouped items as individual groups
	for _, item := range ungrouped {
		result = append(result, []WorkItem{item})
	}
	
	return result
}

// processGroup processes a group of related work items
func (w *BatchWorker) processGroup(items []WorkItem) {
	if len(items) == 1 {
		// Single item, process normally
		err := items[0].Execute()
		if err != nil {
			w.statsMu.Lock()
			w.stats.ErrorCount++
			w.statsMu.Unlock()
		}
		return
	}
	
	// Multiple items, use batch processor
	err := w.processBatch(items)
	if err != nil {
		w.statsMu.Lock()
		w.stats.ErrorCount++
		w.statsMu.Unlock()
	}
}

// updateBatchStats updates batch processing statistics
func (w *BatchWorker) updateBatchStats(batchSize int, duration time.Duration) {
	w.batchStatsMu.Lock()
	defer w.batchStatsMu.Unlock()
	
	w.batchStats.BatchesProcessed++
	w.batchStats.ItemsInBatches += int64(batchSize)
	w.batchStats.TotalBatchTime += duration
	
	// Calculate averages
	w.batchStats.AverageBatchSize = float64(w.batchStats.ItemsInBatches) / float64(w.batchStats.BatchesProcessed)
	w.batchStats.AverageBatchTime = w.batchStats.TotalBatchTime / time.Duration(w.batchStats.BatchesProcessed)
	
	// Update optimal batch size based on efficiency
	w.updateOptimalBatchSize(batchSize, duration)
	
	// Calculate batch efficiency (items per second)
	if duration > 0 {
		itemsPerSecond := float64(batchSize) / duration.Seconds()
		w.batchStats.BatchEfficiency = itemsPerSecond
	}
}

// updateOptimalBatchSize adjusts the optimal batch size based on performance
func (w *BatchWorker) updateOptimalBatchSize(batchSize int, duration time.Duration) {
	// Simple heuristic: if processing time per item is decreasing, batch size is good
	if duration > 0 {
		timePerItem := duration / time.Duration(batchSize)
		
		// If this is more efficient than current optimal, update it
		if w.batchStats.OptimalBatchSize == 0 {
			w.batchStats.OptimalBatchSize = batchSize
		} else {
			optimalTimePerItem := w.batchStats.AverageBatchTime / time.Duration(w.batchStats.OptimalBatchSize)
			if timePerItem < optimalTimePerItem {
				w.batchStats.OptimalBatchSize = batchSize
			}
		}
	}
}

// SetBatchProcessor sets a custom batch processing function
func (w *BatchWorker) SetBatchProcessor(processor func([]WorkItem) error) {
	w.processBatch = processor
}

// SetBatchSize updates the batch size
func (w *BatchWorker) SetBatchSize(size int) {
	w.batchMu.Lock()
	defer w.batchMu.Unlock()
	w.batchSize = size
}

// SetBatchTimeout updates the batch timeout
func (w *BatchWorker) SetBatchTimeout(timeout time.Duration) {
	w.batchTimeout = timeout
	if w.batchTimer != nil {
		w.batchTimer.Reset(timeout)
	}
}

// GetBatchStats returns batch processing statistics
func (w *BatchWorker) GetBatchStats() BatchWorkerStats {
	w.statsMu.RLock()
	w.batchStatsMu.RLock()
	defer w.statsMu.RUnlock()
	defer w.batchStatsMu.RUnlock()
	
	return BatchWorkerStats{
		WorkerStats:    w.stats,
		BatchStats:     w.batchStats,
		BatchSize:      w.batchSize,
		BatchTimeout:   w.batchTimeout,
		BufferSize:     len(w.batchBuffer),
	}
}

// FlushBatch forces processing of the current batch buffer
func (w *BatchWorker) FlushBatch() {
	w.batchMu.Lock()
	defer w.batchMu.Unlock()
	
	if len(w.batchBuffer) > 0 {
		w.processBatchBuffer()
	}
}

// BatchWorkerStats extends WorkerStats with batch-specific metrics
type BatchWorkerStats struct {
	WorkerStats
	BatchStats
	BatchSize    int
	BatchTimeout time.Duration
	BufferSize   int
}

// defaultBatchProcessor is the default batch processing function
func defaultBatchProcessor(items []WorkItem) error {
	// Process all items in the batch
	for _, item := range items {
		if err := item.Execute(); err != nil {
			return err // Return on first error
		}
	}
	return nil
}

// Implementation of WorkItem interface for BatchWorkItem
func (b *BatchWorkItem) Execute() error {
	return b.batchFunc()
}

func (b *BatchWorkItem) Priority() int {
	return b.priority
}

func (b *BatchWorkItem) EstimatedDuration() time.Duration {
	return b.estimatedDuration
}

func (b *BatchWorkItem) Context() context.Context {
	return b.ctx
}

// GetBatchKey returns the batch grouping key
func (b *BatchWorkItem) GetBatchKey() string {
	return b.batchKey
}

// IsBatchable returns whether this item can be batched
func (b *BatchWorkItem) IsBatchable() bool {
	return b.batchable
}

// GetData returns the work item's data
func (b *BatchWorkItem) GetData() interface{} {
	return b.data
}

// DatabaseBatchTask creates a database batch work item for testing
func DatabaseBatchTask(query string, params []interface{}, priority int) *BatchWorkItem {
	return &BatchWorkItem{
		ctx:      context.Background(),
		priority: priority,
		estimatedDuration: time.Millisecond * 10,
		batchFunc: func() error {
			// Simulate database operation
			time.Sleep(time.Millisecond * 5)
			return nil
		},
		batchKey:  "database",
		batchable: true,
		data:      map[string]interface{}{"query": query, "params": params},
	}
}

// EmailBatchTask creates an email batch work item for testing
func EmailBatchTask(recipient string, subject string, priority int) *BatchWorkItem {
	return &BatchWorkItem{
		ctx:      context.Background(),
		priority: priority,
		estimatedDuration: time.Millisecond * 20,
		batchFunc: func() error {
			// Simulate email sending
			time.Sleep(time.Millisecond * 10)
			return nil
		},
		batchKey:  "email",
		batchable: true,
		data:      map[string]interface{}{"recipient": recipient, "subject": subject},
	}
}
package workers

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

// IOWorker is optimized for I/O-bound tasks
type IOWorker struct {
	*BasicWorker
	maxConcurrentIO int
	ioSemaphore     chan struct{}
	connectionPool  *ConnectionPool
	ioStats         IOStats
	ioStatsMu       sync.RWMutex
}

// IOStats tracks I/O-specific statistics
type IOStats struct {
	NetworkRequests   int64
	FileOperations    int64
	DatabaseQueries   int64
	AverageIOLatency  time.Duration
	TotalIOTime       time.Duration
	IOErrors          int64
	ConnectionsActive int64
	ConnectionsTotal  int64
}

// ConnectionPool manages reusable connections
type ConnectionPool struct {
	httpClient    *http.Client
	maxIdleConns  int
	idleTimeout   time.Duration
	connections   map[string]*Connection
	mu            sync.RWMutex
}

// Connection represents a reusable connection
type Connection struct {
	ID          string
	LastUsed    time.Time
	InUse       bool
	Connections int64
}

// IOWorkItem represents I/O-bound work
type IOWorkItem struct {
	ctx               context.Context
	priority          int
	estimatedDuration time.Duration
	ioFunc            func() error
	ioType            IOType
	retryable        bool
	maxRetries        int
	timeout           time.Duration
}

// IOType represents different types of I/O operations
type IOType int

const (
	IOTypeNetwork IOType = iota
	IOTypeFile
	IOTypeDatabase
	IOTypeCache
)

// NewIOWorker creates a new I/O-optimized worker
func NewIOWorker(id string, queueSize int, maxConcurrentIO int) *IOWorker {
	basicWorker := NewBasicWorker(id, queueSize)
	
	// Create HTTP client with optimized settings for I/O
	httpClient := &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
	
	return &IOWorker{
		BasicWorker:     basicWorker,
		maxConcurrentIO: maxConcurrentIO,
		ioSemaphore:     make(chan struct{}, maxConcurrentIO),
		connectionPool: &ConnectionPool{
			httpClient:   httpClient,
			maxIdleConns: 100,
			idleTimeout:  time.Minute * 5,
			connections:  make(map[string]*Connection),
		},
	}
}

// SubmitIOWork submits I/O-bound work
func (w *IOWorker) SubmitIOWork(ioFunc func() error, ioType IOType, priority int, timeout time.Duration) error {
	workItem := &IOWorkItem{
		ctx:               context.Background(),
		priority:          priority,
		estimatedDuration: timeout,
		ioFunc:            ioFunc,
		ioType:            ioType,
		retryable:         true,
		maxRetries:        3,
		timeout:           timeout,
	}
	
	return w.Submit(workItem)
}

// SubmitHTTPRequest submits an HTTP request work item
func (w *IOWorker) SubmitHTTPRequest(url string, method string, body io.Reader, priority int) error {
	workItem := &IOWorkItem{
		ctx:      context.Background(),
		priority: priority,
		estimatedDuration: time.Second * 5,
		ioFunc: func() error {
			req, err := http.NewRequest(method, url, body)
			if err != nil {
				return err
			}
			
			resp, err := w.connectionPool.httpClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			
			// Read response body to complete the request
			_, err = io.ReadAll(resp.Body)
			return err
		},
		ioType:     IOTypeNetwork,
		retryable:  true,
		maxRetries: 3,
		timeout:    time.Second * 30,
	}
	
	return w.Submit(workItem)
}

// processWork overrides basic worker to optimize for I/O tasks
func (w *IOWorker) processWork(work WorkItem) {
	// Acquire I/O semaphore to limit concurrent I/O operations
	w.ioSemaphore <- struct{}{}
	defer func() { <-w.ioSemaphore }()
	
	start := time.Now()
	
	w.statsMu.Lock()
	w.stats.TasksInProgress++
	w.statsMu.Unlock()
	
	// Execute with retry logic for I/O work
	var err error
	if ioWork, ok := work.(*IOWorkItem); ok {
		err = w.executeWithRetry(ioWork)
		w.updateIOStats(ioWork.ioType, time.Since(start), err)
	} else {
		err = work.Execute()
	}
	
	duration := time.Since(start)
	
	w.statsMu.Lock()
	w.stats.TasksInProgress--
	w.stats.TasksProcessed++
	w.stats.TotalExecutionTime += duration
	w.stats.AverageExecutionTime = w.stats.TotalExecutionTime / time.Duration(w.stats.TasksProcessed)
	w.stats.LastActivity = time.Now()
	
	if err != nil {
		w.stats.ErrorCount++
	}
	
	w.statsMu.Unlock()
}

// executeWithRetry executes I/O work with retry logic
func (w *IOWorker) executeWithRetry(ioWork *IOWorkItem) error {
	var lastErr error
	
	for attempt := 0; attempt <= ioWork.maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(ioWork.ctx, ioWork.timeout)
		
		done := make(chan error, 1)
		go func() {
			done <- ioWork.ioFunc()
		}()
		
		select {
		case err := <-done:
			cancel()
			if err == nil {
				return nil // Success
			}
			lastErr = err
			
			// Don't retry if not retryable or on last attempt
			if !ioWork.retryable || attempt == ioWork.maxRetries {
				return err
			}
			
			// Exponential backoff
			backoff := time.Duration(attempt+1) * time.Millisecond * 100
			time.Sleep(backoff)
			
		case <-ctx.Done():
			cancel()
			return ctx.Err()
		}
	}
	
	return lastErr
}

// updateIOStats updates I/O-specific statistics
func (w *IOWorker) updateIOStats(ioType IOType, duration time.Duration, err error) {
	w.ioStatsMu.Lock()
	defer w.ioStatsMu.Unlock()
	
	switch ioType {
	case IOTypeNetwork:
		w.ioStats.NetworkRequests++
	case IOTypeFile:
		w.ioStats.FileOperations++
	case IOTypeDatabase:
		w.ioStats.DatabaseQueries++
	}
	
	w.ioStats.TotalIOTime += duration
	totalOps := w.ioStats.NetworkRequests + w.ioStats.FileOperations + w.ioStats.DatabaseQueries
	if totalOps > 0 {
		w.ioStats.AverageIOLatency = w.ioStats.TotalIOTime / time.Duration(totalOps)
	}
	
	if err != nil {
		w.ioStats.IOErrors++
	}
}

// GetIOStats returns I/O-specific statistics
func (w *IOWorker) GetIOStats() IOWorkerStats {
	w.statsMu.RLock()
	w.ioStatsMu.RLock()
	defer w.statsMu.RUnlock()
	defer w.ioStatsMu.RUnlock()
	
	return IOWorkerStats{
		WorkerStats:        w.stats,
		IOStats:            w.ioStats,
		MaxConcurrentIO:    w.maxConcurrentIO,
		CurrentConcurrentIO: w.maxConcurrentIO - len(w.ioSemaphore),
	}
}

// SetMaxConcurrentIO updates the maximum concurrent I/O operations
func (w *IOWorker) SetMaxConcurrentIO(max int) {
	w.maxConcurrentIO = max
	w.ioSemaphore = make(chan struct{}, max)
}

// IOWorkerStats extends WorkerStats with I/O-specific metrics
type IOWorkerStats struct {
	WorkerStats
	IOStats
	MaxConcurrentIO     int
	CurrentConcurrentIO int
}

// Implementation of WorkItem interface for IOWorkItem
func (i *IOWorkItem) Execute() error {
	return i.ioFunc()
}

func (i *IOWorkItem) Priority() int {
	return i.priority
}

func (i *IOWorkItem) EstimatedDuration() time.Duration {
	return i.estimatedDuration
}

func (i *IOWorkItem) Context() context.Context {
	return i.ctx
}

// GetIOType returns the I/O operation type
func (i *IOWorkItem) GetIOType() IOType {
	return i.ioType
}

// IsRetryable returns whether this work item can be retried
func (i *IOWorkItem) IsRetryable() bool {
	return i.retryable
}

// FileReadTask creates a file reading work item for testing
func FileReadTask(filename string, priority int) *IOWorkItem {
	return &IOWorkItem{
		ctx:      context.Background(),
		priority: priority,
		estimatedDuration: time.Millisecond * 100,
		ioFunc: func() error {
			// Simulate file reading
			time.Sleep(time.Millisecond * 50) // Simulate I/O delay
			return nil
		},
		ioType:     IOTypeFile,
		retryable:  true,
		maxRetries: 3,
		timeout:    time.Second * 5,
	}
}

// DatabaseQueryTask creates a database query work item for testing
func DatabaseQueryTask(query string, priority int) *IOWorkItem {
	return &IOWorkItem{
		ctx:      context.Background(),
		priority: priority,
		estimatedDuration: time.Millisecond * 200,
		ioFunc: func() error {
			// Simulate database query
			time.Sleep(time.Millisecond * 100) // Simulate query execution
			return nil
		},
		ioType:     IOTypeDatabase,
		retryable:  true,
		maxRetries: 2,
		timeout:    time.Second * 10,
	}
}
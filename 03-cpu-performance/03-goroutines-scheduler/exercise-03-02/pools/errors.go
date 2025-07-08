package pools

import (
	"errors"
	"time"
)

// Common pool errors
var (
	ErrPoolFull         = errors.New("pool queue is full")
	ErrPoolClosed       = errors.New("pool is closed")
	ErrShutdownTimeout  = errors.New("shutdown timeout exceeded")
	ErrInvalidWorker    = errors.New("invalid worker ID")
	ErrInvalidPriority  = errors.New("invalid priority level")
	ErrPoolNotStarted   = errors.New("pool not started")
	ErrAlreadyStarted   = errors.New("pool already started")
	
	// Additional pool errors
	ErrPoolShutdown     = errors.New("pool is shutting down")
	ErrQueueFull        = errors.New("task queue is full")
	ErrInvalidWorkers   = errors.New("invalid number of workers")
	ErrTimeout          = errors.New("operation timed out")
	
	// Worker errors
	ErrWorkerAlreadyRunning = errors.New("worker is already running")
	ErrWorkerStopTimeout    = errors.New("worker stop timeout")
	ErrWorkerNotRunning     = errors.New("worker is not running")
	ErrWorkerQueueFull      = errors.New("worker queue is full")
)

// Pool interface defines common pool operations
type Pool interface {
	// Start initializes and starts the pool
	Start() error
	
	// Submit adds a task to the pool
	Submit(task func()) error
	
	// Shutdown gracefully shuts down the pool
	Shutdown(timeout time.Duration) error
	
	// GetMetrics returns current pool metrics
	GetMetrics() PoolMetrics
	
	// QueueDepth returns the current queue depth
	QueueDepth() int
	
	// WorkerCount returns the number of workers
	WorkerCount() int
}

// PriorityPool interface extends Pool for priority-based pools
type PriorityPoolInterface interface {
	// Start initializes and starts the pool
	Start() error
	
	// Submit with priority
	Submit(task func(), priority int) error
	
	// Shutdown gracefully shuts down the pool
	Shutdown(timeout time.Duration) error
	
	// GetMetrics returns current pool metrics
	GetMetrics() PoolMetrics
	
	// QueueDepth returns the current queue depth
	QueueDepth() int
	
	// WorkerCount returns the number of workers
	WorkerCount() int
	
	// GetPriorityQueueDepths returns queue depths for each priority level
	GetPriorityQueueDepths() []int
}

// WorkStealingPoolInterface extends Pool for work-stealing pools
type WorkStealingPoolInterface interface {
	Pool
	
	// SubmitToWorker submits a task to a specific worker
	SubmitToWorker(workerID int, task func()) error
	
	// GetWorkerStats returns statistics for all workers
	GetWorkerStats() []WorkerStats
	
	// GetWorkerQueueDepth returns queue depth for a specific worker
	GetWorkerQueueDepth(workerID int) int
}

// DynamicPoolInterface extends Pool for dynamic scaling pools
type DynamicPoolInterface interface {
	Pool
	
	// SetScalingParameters updates scaling thresholds
	SetScalingParameters(upThreshold, downThreshold float64, interval time.Duration)
}
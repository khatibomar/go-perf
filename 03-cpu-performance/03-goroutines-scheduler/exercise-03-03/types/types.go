package types

import (
	"context"
	"sync"
	"time"
)

// Work represents a unit of work to be processed
type Work interface {
	Execute() error
	GetID() string
	GetAffinityKey() string
	Hash() int
	GetPriority() int
	GetSize() int64
}

// Worker represents a worker that can process work
type Worker interface {
	Submit(work Work) error
	SubmitBatch(works []Work) error
	Start(ctx context.Context) error
	Stop() error
	GetID() string
}

// LoadAwareWorker extends Worker with load information
type LoadAwareWorker interface {
	Worker
	GetLoad() float64
	GetCapacity() int
	GetQueueDepth() int
	GetQueueSize() int
}

// Distributor represents a work distribution strategy
type Distributor interface {
	Distribute(work Work) error
	DistributeBatch(works []Work) error
	AddWorker(worker Worker) error
	RemoveWorker(workerID string) error
	GetMetrics() *DistributorMetrics
}

// LoadBalancer handles dynamic load balancing
type LoadBalancer interface {
	Balance() error
	GetRecommendations() []BalanceRecommendation
	SetThresholds(high, low float64)
}

// Monitor tracks performance metrics
type Monitor interface {
	Start(ctx context.Context)
	Stop()
	GetMetrics() *Metrics
	RecordDistribution(workerID string, latency time.Duration)
	RecordExecution(workID string, duration time.Duration)
}

// SimpleWork implements the Work interface
type SimpleWork struct {
	ID          string
	AffinityKey string
	Priority    int
	Size        int64
	Task        func() error
}

func (w *SimpleWork) Execute() error {
	if w.Task != nil {
		return w.Task()
	}
	return nil
}

func (w *SimpleWork) GetID() string {
	return w.ID
}

func (w *SimpleWork) GetAffinityKey() string {
	return w.AffinityKey
}

func (w *SimpleWork) Hash() int {
	hash := 0
	for _, b := range []byte(w.ID) {
		hash = hash*31 + int(b)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

func (w *SimpleWork) GetPriority() int {
	return w.Priority
}

func (w *SimpleWork) GetSize() int64 {
	return w.Size
}

// Metrics structures
type DistributorMetrics struct {
	TotalDistributed int64
	DistributionTime time.Duration
	WorkerCount      int
	LoadBalance      float64
	Mu               sync.RWMutex
}

type WorkerMetrics struct {
	WorkerID        string
	TasksProcessed  int64
	AverageLoad     float64
	QueueDepth      int
	ProcessingTime  time.Duration
	LastActivity    time.Time
	Mu              sync.RWMutex
}

type Metrics struct {
	Distributors map[string]*DistributorMetrics
	Workers      map[string]*WorkerMetrics
	Overall      *OverallMetrics
	Mu           sync.RWMutex
}

type OverallMetrics struct {
	TotalTasks      int64
	Throughput      float64
	AverageLatency  time.Duration
	P99Latency      time.Duration
	StartTime       time.Time
	Mu              sync.RWMutex
}

type BalanceRecommendation struct {
	Action     string // "scale_up", "scale_down", "redistribute"
	WorkerID   string
	Reason     string
	Priority   int
	EstimatedImpact float64
}

type CoordinatorMetrics struct {
	StealAttempts   int64
	SuccessfulSteals int64
	StealLatency    time.Duration
	Mu              sync.RWMutex
}
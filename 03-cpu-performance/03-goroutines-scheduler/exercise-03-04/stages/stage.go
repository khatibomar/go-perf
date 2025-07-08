package stages

import (
	"context"
	"time"
)

// Stage represents a processing stage in a pipeline
type Stage interface {
	// Process transforms input data to output data
	Process(data interface{}) (interface{}, error)
	
	// GetName returns the stage name
	GetName() string
	
	// GetMetrics returns stage performance metrics
	GetMetrics() *StageMetrics
	
	// Start initializes the stage
	Start(ctx context.Context) error
	
	// Stop gracefully shuts down the stage
	Stop() error
}

// StageMetrics holds performance metrics for a stage
type StageMetrics struct {
	ProcessedCount int64
	ErrorCount     int64
	AvgLatency     time.Duration
	MaxLatency     time.Duration
	MinLatency     time.Duration
	Throughput     float64
	QueueLength    int
	QueueCapacity  int
	TargetLatency  time.Duration
}

// BaseStage provides common functionality for all stages
type BaseStage struct {
	name    string
	metrics *StageMetrics
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewBaseStage creates a new base stage
func NewBaseStage(name string) *BaseStage {
	ctx, cancel := context.WithCancel(context.Background())
	return &BaseStage{
		name:   name,
		metrics: &StageMetrics{
			MinLatency:    time.Hour, // Initialize to high value
			TargetLatency: 1 * time.Millisecond,
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// GetName returns the stage name
func (s *BaseStage) GetName() string {
	return s.name
}

// GetMetrics returns stage metrics
func (s *BaseStage) GetMetrics() *StageMetrics {
	return s.metrics
}

// Start initializes the stage
func (s *BaseStage) Start(ctx context.Context) error {
	s.ctx = ctx
	return nil
}

// Stop gracefully shuts down the stage
func (s *BaseStage) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

// UpdateMetrics updates stage metrics with processing time
func (s *BaseStage) UpdateMetrics(duration time.Duration, success bool) {
	if success {
		s.metrics.ProcessedCount++
	} else {
		s.metrics.ErrorCount++
	}
	
	if duration > s.metrics.MaxLatency {
		s.metrics.MaxLatency = duration
	}
	
	if duration < s.metrics.MinLatency {
		s.metrics.MinLatency = duration
	}
	
	// Simple moving average for latency
	total := s.metrics.ProcessedCount + s.metrics.ErrorCount
	if total > 0 {
		s.metrics.AvgLatency = time.Duration(
			(int64(s.metrics.AvgLatency)*(total-1) + int64(duration)) / total,
		)
	}
}
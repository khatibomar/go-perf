package stages

import (
	"sync"
	"time"
)

// AggregateFunc defines an aggregation function
type AggregateFunc func([]interface{}) (interface{}, error)

// AggregateStage collects and aggregates data
type AggregateStage struct {
	*BaseStage
	aggregateFunc AggregateFunc
	batchSize     int
	timeout       time.Duration
	buffer        []interface{}
	mu            sync.Mutex
	lastFlush     time.Time
}

// NewAggregateStage creates a new aggregate stage
func NewAggregateStage(name string, aggregateFunc AggregateFunc, batchSize int, timeout time.Duration) *AggregateStage {
	return &AggregateStage{
		BaseStage:     NewBaseStage(name),
		aggregateFunc: aggregateFunc,
		batchSize:     batchSize,
		timeout:       timeout,
		buffer:        make([]interface{}, 0, batchSize),
		lastFlush:     time.Now(),
	}
}

// Process adds data to the buffer and returns aggregated result when ready
func (s *AggregateStage) Process(data interface{}) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		s.UpdateMetrics(duration, true)
	}()
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.buffer = append(s.buffer, data)
	
	// Check if we should flush based on size or timeout
	shouldFlush := len(s.buffer) >= s.batchSize ||
		time.Since(s.lastFlush) >= s.timeout
	
	if shouldFlush && len(s.buffer) > 0 {
		result, err := s.aggregateFunc(s.buffer)
		s.buffer = s.buffer[:0] // Reset buffer
		s.lastFlush = time.Now()
		return result, err
	}
	
	// Return nil to indicate no output yet
	return nil, nil
}

// Flush forces aggregation of current buffer
func (s *AggregateStage) Flush() (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if len(s.buffer) == 0 {
		return nil, nil
	}
	
	result, err := s.aggregateFunc(s.buffer)
	s.buffer = s.buffer[:0]
	s.lastFlush = time.Now()
	return result, err
}

// SumAggregateStage sums numeric values
type SumAggregateStage struct {
	*BaseStage
	batchSize int
	timeout   time.Duration
	buffer    []float64
	mu        sync.Mutex
	lastFlush time.Time
}

// NewSumAggregateStage creates a sum aggregation stage
func NewSumAggregateStage(name string, batchSize int, timeout time.Duration) *SumAggregateStage {
	return &SumAggregateStage{
		BaseStage: NewBaseStage(name),
		batchSize: batchSize,
		timeout:   timeout,
		buffer:    make([]float64, 0, batchSize),
		lastFlush: time.Now(),
	}
}

// Process adds numeric data to the buffer and returns sum when ready
func (s *SumAggregateStage) Process(data interface{}) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		s.UpdateMetrics(duration, true)
	}()
	
	// Convert to float64
	var value float64
	switch v := data.(type) {
	case int:
		value = float64(v)
	case int64:
		value = float64(v)
	case float64:
		value = v
	case float32:
		value = float64(v)
	default:
		return data, nil // Pass through non-numeric data
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.buffer = append(s.buffer, value)
	
	// Check if we should flush
	shouldFlush := len(s.buffer) >= s.batchSize ||
		time.Since(s.lastFlush) >= s.timeout
	
	if shouldFlush && len(s.buffer) > 0 {
		sum := 0.0
		for _, v := range s.buffer {
			sum += v
		}
		s.buffer = s.buffer[:0]
		s.lastFlush = time.Now()
		return sum, nil
	}
	
	return nil, nil
}

// CountAggregateStage counts items
type CountAggregateStage struct {
	*BaseStage
	batchSize int
	timeout   time.Duration
	count     int64
	mu        sync.Mutex
	lastFlush time.Time
}

// NewCountAggregateStage creates a count aggregation stage
func NewCountAggregateStage(name string, batchSize int, timeout time.Duration) *CountAggregateStage {
	return &CountAggregateStage{
		BaseStage: NewBaseStage(name),
		batchSize: batchSize,
		timeout:   timeout,
		lastFlush: time.Now(),
	}
}

// Process increments count and returns count when ready
func (s *CountAggregateStage) Process(data interface{}) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		s.UpdateMetrics(duration, true)
	}()
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.count++
	
	// Check if we should flush
	shouldFlush := s.count >= int64(s.batchSize) ||
		time.Since(s.lastFlush) >= s.timeout
	
	if shouldFlush && s.count > 0 {
		result := s.count
		s.count = 0
		s.lastFlush = time.Now()
		return result, nil
	}
	
	return nil, nil
}
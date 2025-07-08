package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"exercise-03-04/stages"
	"exercise-03-04/buffer"
	"exercise-03-04/backpressure"
)

// LinearPipeline implements a simple linear pipeline
type LinearPipeline struct {
	mu               sync.RWMutex
	name             string
	stages           []stages.Stage
	buffers          []*buffer.AdaptiveBuffer
	backpressureCtrl *backpressure.BackpressureController
	metrics          *PipelineMetrics
	ctx              context.Context
	cancel           context.CancelFunc
	running          bool
	inputChan        chan interface{}
	outputChan       chan interface{}
	errorChan        chan error
	wg               sync.WaitGroup
	bufferSize       int
}

// PipelineMetrics tracks pipeline performance
type PipelineMetrics struct {
	ProcessedCount   int64
	ErrorCount       int64
	Throughput       float64
	Latency          time.Duration
	StartTime        time.Time
	LastProcessTime  time.Time
	StageMetrics     map[string]*stages.StageMetrics
	mu               sync.RWMutex
}

// NewLinearPipeline creates a new linear pipeline
func NewLinearPipeline(name string, bufferSize int) *LinearPipeline {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &LinearPipeline{
		name:        name,
		stages:      make([]stages.Stage, 0),
		buffers:     make([]*buffer.AdaptiveBuffer, 0),
		metrics:     &PipelineMetrics{StageMetrics: make(map[string]*stages.StageMetrics)},
		ctx:         ctx,
		cancel:      cancel,
		inputChan:   make(chan interface{}, bufferSize),
		outputChan:  make(chan interface{}, bufferSize),
		errorChan:   make(chan error, bufferSize),
		bufferSize:  bufferSize,
	}
}

// AddStage adds a stage to the pipeline
func (lp *LinearPipeline) AddStage(stage stages.Stage) {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	
	lp.stages = append(lp.stages, stage)
	
	// Add buffer between stages (except for the last stage)
	if len(lp.stages) > 1 {
		buf := buffer.NewAdaptiveBuffer(lp.bufferSize, lp.bufferSize/4, lp.bufferSize*2)
		lp.buffers = append(lp.buffers, buf)
	}
	
	// Initialize stage metrics
	lp.metrics.StageMetrics[stage.GetName()] = &stages.StageMetrics{}
}

// SetBackpressureController sets the backpressure controller
func (lp *LinearPipeline) SetBackpressureController(ctrl *backpressure.BackpressureController) {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	lp.backpressureCtrl = ctrl
}

// Start starts the pipeline
func (lp *LinearPipeline) Start() error {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	
	if lp.running {
		return fmt.Errorf("pipeline %s is already running", lp.name)
	}
	
	if len(lp.stages) == 0 {
		return fmt.Errorf("pipeline %s has no stages", lp.name)
	}
	
	// Start all stages
	for _, stage := range lp.stages {
		if err := stage.Start(lp.ctx); err != nil {
			return fmt.Errorf("failed to start stage %s: %w", stage.GetName(), err)
		}
	}
	
	// Start pipeline workers
	for i := 0; i < len(lp.stages); i++ {
		lp.wg.Add(1)
		go lp.runStage(i)
	}
	
	lp.running = true
	lp.metrics.StartTime = time.Now()
	
	return nil
}

// Stop stops the pipeline
func (lp *LinearPipeline) Stop() error {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	
	if !lp.running {
		return nil
	}
	
	// Cancel context to signal shutdown
	lp.cancel()
	
	// Close input channel
	close(lp.inputChan)
	
	// Wait for all workers to finish
	lp.wg.Wait()
	
	// Stop all stages
	for _, stage := range lp.stages {
		if err := stage.Stop(); err != nil {
			// Log error but continue stopping other stages
			continue
		}
	}
	
	// Close buffers
	for _, buf := range lp.buffers {
		buf.Close()
	}
	
	// Close output and error channels
	close(lp.outputChan)
	close(lp.errorChan)
	
	lp.running = false
	return nil
}

// Process sends data into the pipeline
func (lp *LinearPipeline) Process(data interface{}) error {
	lp.mu.RLock()
	running := lp.running
	lp.mu.RUnlock()
	
	if !running {
		return fmt.Errorf("pipeline %s is not running", lp.name)
	}
	
	// Check backpressure
	if lp.backpressureCtrl != nil {
		load := lp.calculateCurrentLoad()
		action := lp.backpressureCtrl.CheckAndApply(load)
		
		switch action {
		case backpressure.ActionReject:
			return fmt.Errorf("request rejected due to high load")
		case backpressure.ActionDrop:
			// Silently drop the request
			return nil
		}
	}
	
	select {
	case lp.inputChan <- data:
		return nil
	case <-lp.ctx.Done():
		return lp.ctx.Err()
	default:
		return fmt.Errorf("pipeline input buffer is full")
	}
}

// GetOutput returns the output channel
func (lp *LinearPipeline) GetOutput() <-chan interface{} {
	return lp.outputChan
}

// GetErrors returns the error channel
func (lp *LinearPipeline) GetErrors() <-chan error {
	return lp.errorChan
}

// runStage runs a specific stage in the pipeline
func (lp *LinearPipeline) runStage(stageIndex int) {
	defer lp.wg.Done()
	
	stage := lp.stages[stageIndex]
	
	// Determine input source
	var inputChan <-chan interface{}
	if stageIndex == 0 {
		inputChan = lp.inputChan
	} else {
		// Create a channel from the previous buffer
		bufferChan := make(chan interface{}, lp.bufferSize)
		go lp.bufferToChannel(lp.buffers[stageIndex-1], bufferChan)
		inputChan = bufferChan
	}
	
	// Process data through the stage
	for {
		select {
		case data, ok := <-inputChan:
			if !ok {
				return
			}
			
			start := time.Now()
			result, err := stage.Process(data)
			processTime := time.Since(start)
			
			// Update metrics
			lp.updateStageMetrics(stage.GetName(), processTime, err == nil)
			
			if err != nil {
				select {
				case lp.errorChan <- fmt.Errorf("stage %s error: %w", stage.GetName(), err):
				case <-lp.ctx.Done():
					return
				}
				continue
			}
			
			// Skip nil results (filtered data)
			if result == nil {
				continue
			}
			
			// Send result to next stage or output
			if stageIndex == len(lp.stages)-1 {
				// Last stage - send to output
				select {
				case lp.outputChan <- result:
				case <-lp.ctx.Done():
					return
				}
			} else {
				// Send to next stage buffer
				if !lp.buffers[stageIndex].Send(result) {
					// Buffer full - apply backpressure or drop
					select {
					case lp.errorChan <- fmt.Errorf("buffer overflow at stage %s", stage.GetName()):
					case <-lp.ctx.Done():
						return
					}
				}
			}
			
		case <-lp.ctx.Done():
			return
		}
	}
}

// bufferToChannel converts buffer output to channel
func (lp *LinearPipeline) bufferToChannel(buf *buffer.AdaptiveBuffer, outputChan chan<- interface{}) {
	defer close(outputChan)
	
	for {
		select {
		case <-lp.ctx.Done():
			return
		default:
			if data, ok := buf.Receive(); ok {
				select {
				case outputChan <- data:
				case <-lp.ctx.Done():
					return
				}
			} else {
				// No data available, brief sleep to avoid busy waiting
				time.Sleep(time.Microsecond * 100)
			}
		}
	}
}

// calculateCurrentLoad calculates the current pipeline load
func (lp *LinearPipeline) calculateCurrentLoad() float64 {
	if len(lp.buffers) == 0 {
		return 0.0
	}
	
	totalLoad := 0.0
	for _, buf := range lp.buffers {
		totalLoad += buf.GetLoadFactor()
	}
	
	return totalLoad / float64(len(lp.buffers))
}

// updateStageMetrics updates metrics for a specific stage
func (lp *LinearPipeline) updateStageMetrics(stageName string, duration time.Duration, success bool) {
	lp.metrics.mu.Lock()
	defer lp.metrics.mu.Unlock()
	
	stageMetrics, exists := lp.metrics.StageMetrics[stageName]
	if !exists {
		stageMetrics = &stages.StageMetrics{}
		lp.metrics.StageMetrics[stageName] = stageMetrics
	}
	
	stageMetrics.ProcessedCount++
	if !success {
		stageMetrics.ErrorCount++
	}
	
	// Update average latency
	if stageMetrics.AvgLatency == 0 {
		stageMetrics.AvgLatency = duration
	} else {
		stageMetrics.AvgLatency = (stageMetrics.AvgLatency + duration) / 2
	}
	
	// Update max latency
	if duration > stageMetrics.MaxLatency {
		stageMetrics.MaxLatency = duration
	}
	
	// Update min latency
	if duration < stageMetrics.MinLatency || stageMetrics.MinLatency == 0 {
		stageMetrics.MinLatency = duration
	}
	
	// Update pipeline-level metrics
	lp.metrics.ProcessedCount++
	if !success {
		lp.metrics.ErrorCount++
	}
	lp.metrics.LastProcessTime = time.Now()
	
	// Calculate throughput (items per second)
	elapsed := time.Since(lp.metrics.StartTime).Seconds()
	if elapsed > 0 {
		lp.metrics.Throughput = float64(lp.metrics.ProcessedCount) / elapsed
	}
}

// GetMetrics returns current pipeline metrics
func (lp *LinearPipeline) GetMetrics() *PipelineMetrics {
	lp.metrics.mu.RLock()
	defer lp.metrics.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	metricsCopy := &PipelineMetrics{
		ProcessedCount:  lp.metrics.ProcessedCount,
		ErrorCount:      lp.metrics.ErrorCount,
		Throughput:      lp.metrics.Throughput,
		Latency:         lp.metrics.Latency,
		StartTime:       lp.metrics.StartTime,
		LastProcessTime: lp.metrics.LastProcessTime,
		StageMetrics:    make(map[string]*stages.StageMetrics),
	}
	
	// Copy stage metrics
	for name, metrics := range lp.metrics.StageMetrics {
		metricsCopy.StageMetrics[name] = &stages.StageMetrics{
			ProcessedCount: metrics.ProcessedCount,
			ErrorCount:     metrics.ErrorCount,
			AvgLatency:     metrics.AvgLatency,
			MaxLatency:     metrics.MaxLatency,
			MinLatency:     metrics.MinLatency,
			Throughput:     metrics.Throughput,
		}
	}
	
	return metricsCopy
}

// GetName returns the pipeline name
func (lp *LinearPipeline) GetName() string {
	return lp.name
}

// IsRunning returns whether the pipeline is currently running
func (lp *LinearPipeline) IsRunning() bool {
	lp.mu.RLock()
	defer lp.mu.RUnlock()
	return lp.running
}

// GetStageCount returns the number of stages in the pipeline
func (lp *LinearPipeline) GetStageCount() int {
	lp.mu.RLock()
	defer lp.mu.RUnlock()
	return len(lp.stages)
}
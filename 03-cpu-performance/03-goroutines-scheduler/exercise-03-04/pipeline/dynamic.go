package pipeline

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"exercise-03-04/stages"
	"exercise-03-04/buffer"
	"exercise-03-04/backpressure"
)

// DynamicPipeline allows runtime modification of pipeline structure
type DynamicPipeline struct {
	mu               sync.RWMutex
	name             string
	stages           map[string]*DynamicStage
	stageOrder       []string
	connections      map[string][]string // stage -> list of next stages
	buffers          map[string]*buffer.AdaptiveBuffer
	backpressureCtrl *backpressure.BackpressureController
	metrics          *PipelineMetrics
	ctx              context.Context
	cancel           context.CancelFunc
	running          int32
	inputChan        chan interface{}
	outputChan       chan interface{}
	errorChan        chan error
	bufferSize       int
	version          int64 // For tracking configuration changes
}

// DynamicStage wraps a stage with dynamic capabilities
type DynamicStage struct {
	stage     stages.Stage
	inputChan chan interface{}
	outputChans map[string]chan interface{}
	running   int32
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewDynamicPipeline creates a new dynamic pipeline
func NewDynamicPipeline(name string, bufferSize int) *DynamicPipeline {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &DynamicPipeline{
		name:        name,
		stages:      make(map[string]*DynamicStage),
		stageOrder:  make([]string, 0),
		connections: make(map[string][]string),
		buffers:     make(map[string]*buffer.AdaptiveBuffer),
		metrics:     &PipelineMetrics{StageMetrics: make(map[string]*stages.StageMetrics)},
		ctx:         ctx,
		cancel:      cancel,
		inputChan:   make(chan interface{}, bufferSize),
		outputChan:  make(chan interface{}, bufferSize),
		errorChan:   make(chan error, bufferSize),
		bufferSize:  bufferSize,
	}
}

// AddStage adds a new stage to the pipeline
func (dp *DynamicPipeline) AddStage(stage stages.Stage) error {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	
	stageName := stage.GetName()
	if _, exists := dp.stages[stageName]; exists {
		return fmt.Errorf("stage %s already exists", stageName)
	}
	
	// Create dynamic stage wrapper
	stageCtx, stageCancel := context.WithCancel(dp.ctx)
	dynamicStage := &DynamicStage{
		stage:       stage,
		inputChan:   make(chan interface{}, dp.bufferSize),
		outputChans: make(map[string]chan interface{}),
		ctx:         stageCtx,
		cancel:      stageCancel,
	}
	
	dp.stages[stageName] = dynamicStage
	dp.stageOrder = append(dp.stageOrder, stageName)
	dp.connections[stageName] = make([]string, 0)
	dp.metrics.StageMetrics[stageName] = &stages.StageMetrics{}
	
	// If pipeline is running, start the stage
	if atomic.LoadInt32(&dp.running) == 1 {
		if err := dp.startStage(stageName); err != nil {
			return fmt.Errorf("failed to start stage %s: %w", stageName, err)
		}
	}
	
	atomic.AddInt64(&dp.version, 1)
	return nil
}

// RemoveStage removes a stage from the pipeline
func (dp *DynamicPipeline) RemoveStage(stageName string) error {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	
	dynamicStage, exists := dp.stages[stageName]
	if !exists {
		return fmt.Errorf("stage %s does not exist", stageName)
	}
	
	// Stop the stage if running
	if atomic.LoadInt32(&dynamicStage.running) == 1 {
		if err := dp.stopStage(stageName); err != nil {
			return fmt.Errorf("failed to stop stage %s: %w", stageName, err)
		}
	}
	
	// Remove connections to this stage
	for fromStage, connections := range dp.connections {
		newConnections := make([]string, 0)
		for _, toStage := range connections {
			if toStage != stageName {
				newConnections = append(newConnections, toStage)
			}
		}
		dp.connections[fromStage] = newConnections
	}
	
	// Remove from stage order
	newOrder := make([]string, 0)
	for _, name := range dp.stageOrder {
		if name != stageName {
			newOrder = append(newOrder, name)
		}
	}
	dp.stageOrder = newOrder
	
	// Clean up
	delete(dp.stages, stageName)
	delete(dp.connections, stageName)
	delete(dp.metrics.StageMetrics, stageName)
	
	// Remove associated buffer
	if buf, exists := dp.buffers[stageName]; exists {
		buf.Close()
		delete(dp.buffers, stageName)
	}
	
	atomic.AddInt64(&dp.version, 1)
	return nil
}

// ConnectStages creates a connection between two stages
func (dp *DynamicPipeline) ConnectStages(fromStage, toStage string) error {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	
	// Verify both stages exist
	if _, exists := dp.stages[fromStage]; !exists {
		return fmt.Errorf("source stage %s does not exist", fromStage)
	}
	if _, exists := dp.stages[toStage]; !exists {
		return fmt.Errorf("destination stage %s does not exist", toStage)
	}
	
	// Check if connection already exists
	for _, existing := range dp.connections[fromStage] {
		if existing == toStage {
			return fmt.Errorf("connection from %s to %s already exists", fromStage, toStage)
		}
	}
	
	// Add connection
	dp.connections[fromStage] = append(dp.connections[fromStage], toStage)
	
	// Create output channel for the connection
	outputChan := make(chan interface{}, dp.bufferSize)
	dp.stages[fromStage].outputChans[toStage] = outputChan
	
	// Create buffer for the connection
	bufferKey := fmt.Sprintf("%s->%s", fromStage, toStage)
	dp.buffers[bufferKey] = buffer.NewAdaptiveBuffer(dp.bufferSize, dp.bufferSize/4, dp.bufferSize*2)
	
	atomic.AddInt64(&dp.version, 1)
	return nil
}

// DisconnectStages removes a connection between two stages
func (dp *DynamicPipeline) DisconnectStages(fromStage, toStage string) error {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	
	// Remove from connections
	newConnections := make([]string, 0)
	for _, existing := range dp.connections[fromStage] {
		if existing != toStage {
			newConnections = append(newConnections, existing)
		}
	}
	dp.connections[fromStage] = newConnections
	
	// Close and remove output channel
	if outputChan, exists := dp.stages[fromStage].outputChans[toStage]; exists {
		close(outputChan)
		delete(dp.stages[fromStage].outputChans, toStage)
	}
	
	// Remove buffer
	bufferKey := fmt.Sprintf("%s->%s", fromStage, toStage)
	if buf, exists := dp.buffers[bufferKey]; exists {
		buf.Close()
		delete(dp.buffers, bufferKey)
	}
	
	atomic.AddInt64(&dp.version, 1)
	return nil
}

// SetBackpressureController sets the backpressure controller
func (dp *DynamicPipeline) SetBackpressureController(ctrl *backpressure.BackpressureController) {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	dp.backpressureCtrl = ctrl
}

// Start starts the dynamic pipeline
func (dp *DynamicPipeline) Start() error {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	
	if atomic.LoadInt32(&dp.running) == 1 {
		return fmt.Errorf("pipeline %s is already running", dp.name)
	}
	
	if len(dp.stages) == 0 {
		return fmt.Errorf("pipeline %s has no stages", dp.name)
	}
	
	// Start all stages
	for stageName := range dp.stages {
		if err := dp.startStage(stageName); err != nil {
			return fmt.Errorf("failed to start stage %s: %w", stageName, err)
		}
	}
	
	// Start input router to forward data from inputChan to first stage
	go dp.runInputRouter()
	
	atomic.StoreInt32(&dp.running, 1)
	dp.metrics.StartTime = time.Now()
	
	return nil
}

// Stop stops the dynamic pipeline
func (dp *DynamicPipeline) Stop() error {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	
	if atomic.LoadInt32(&dp.running) == 0 {
		return nil
	}
	
	// Cancel context to signal shutdown
	dp.cancel()
	
	// Close input channel
	close(dp.inputChan)
	
	// Stop all stages
	for stageName := range dp.stages {
		if err := dp.stopStage(stageName); err != nil {
			continue
		}
	}
	
	// Close buffers
	for _, buf := range dp.buffers {
		buf.Close()
	}
	
	// Close output and error channels
	close(dp.outputChan)
	close(dp.errorChan)
	
	atomic.StoreInt32(&dp.running, 0)
	return nil
}

// startStage starts a specific stage
func (dp *DynamicPipeline) startStage(stageName string) error {
	dynamicStage, exists := dp.stages[stageName]
	if !exists {
		return fmt.Errorf("stage %s does not exist", stageName)
	}
	
	if atomic.LoadInt32(&dynamicStage.running) == 1 {
		return nil // Already running
	}
	
	// Start the underlying stage
	if err := dynamicStage.stage.Start(dynamicStage.ctx); err != nil {
		return err
	}
	
	// Start the stage worker
	dynamicStage.wg.Add(1)
	go dp.runDynamicStage(stageName)
	
	atomic.StoreInt32(&dynamicStage.running, 1)
	return nil
}

// stopStage stops a specific stage
func (dp *DynamicPipeline) stopStage(stageName string) error {
	dynamicStage, exists := dp.stages[stageName]
	if !exists {
		return fmt.Errorf("stage %s does not exist", stageName)
	}
	
	if atomic.LoadInt32(&dynamicStage.running) == 0 {
		return nil // Already stopped
	}
	
	// Cancel stage context
	dynamicStage.cancel()
	
	// Close input channel
	close(dynamicStage.inputChan)
	
	// Wait for worker to finish
	dynamicStage.wg.Wait()
	
	// Stop the underlying stage
	if err := dynamicStage.stage.Stop(); err != nil {
		return err
	}
	
	// Close output channels
	for _, outputChan := range dynamicStage.outputChans {
		close(outputChan)
	}
	
	atomic.StoreInt32(&dynamicStage.running, 0)
	return nil
}

// Process sends data into the pipeline
func (dp *DynamicPipeline) Process(data interface{}) error {
	if atomic.LoadInt32(&dp.running) == 0 {
		return fmt.Errorf("pipeline %s is not running", dp.name)
	}
	
	// Check backpressure
	if dp.backpressureCtrl != nil {
		load := dp.calculateCurrentLoad()
		action := dp.backpressureCtrl.CheckAndApply(load)
		
		switch action {
		case backpressure.ActionReject:
			return fmt.Errorf("request rejected due to high load")
		case backpressure.ActionDrop:
			return nil
		}
	}
	
	select {
	case dp.inputChan <- data:
		return nil
	case <-dp.ctx.Done():
		return dp.ctx.Err()
	default:
		return fmt.Errorf("pipeline input buffer is full")
	}
}

// GetOutput returns the output channel
func (dp *DynamicPipeline) GetOutput() <-chan interface{} {
	return dp.outputChan
}

// GetErrors returns the error channel
func (dp *DynamicPipeline) GetErrors() <-chan error {
	return dp.errorChan
}

// runInputRouter routes data from pipeline input to the first stage
func (dp *DynamicPipeline) runInputRouter() {
	for {
		select {
		case data, ok := <-dp.inputChan:
			if !ok {
				return
			}
			
			// Find the first stage (one with no incoming connections)
			dp.mu.RLock()
			firstStage := dp.findFirstStage()
			dp.mu.RUnlock()
			
			if firstStage != "" {
				if stageObj, exists := dp.stages[firstStage]; exists {
					select {
					case stageObj.inputChan <- data:
					case <-dp.ctx.Done():
						return
					default:
						// Buffer full, send error
						select {
						case dp.errorChan <- fmt.Errorf("input buffer overflow for stage %s", firstStage):
						case <-dp.ctx.Done():
							return
						}
					}
				}
			}
			
		case <-dp.ctx.Done():
			return
		}
	}
}

// findFirstStage finds the stage with no incoming connections
func (dp *DynamicPipeline) findFirstStage() string {
	// Create a set of stages that have incoming connections
	hasIncoming := make(map[string]bool)
	for _, connections := range dp.connections {
		for _, target := range connections {
			hasIncoming[target] = true
		}
	}
	
	// Find the first stage without incoming connections
	for stageName := range dp.stages {
		if !hasIncoming[stageName] {
			return stageName
		}
	}
	
	// If no stage found, return the first in order
	if len(dp.stageOrder) > 0 {
		return dp.stageOrder[0]
	}
	
	return ""
}

// runDynamicStage runs a dynamic stage worker
func (dp *DynamicPipeline) runDynamicStage(stageName string) {
	dynamicStage := dp.stages[stageName]
	defer dynamicStage.wg.Done()
	
	for {
		select {
		case data, ok := <-dynamicStage.inputChan:
			if !ok {
				return
			}
			
			start := time.Now()
			result, err := dynamicStage.stage.Process(data)
			processTime := time.Since(start)
			
			// Update metrics
			dp.updateStageMetrics(stageName, processTime, err == nil)
			
			if err != nil {
				select {
				case dp.errorChan <- fmt.Errorf("stage %s error: %w", stageName, err):
				case <-dynamicStage.ctx.Done():
					return
				}
				continue
			}
			
			// Skip nil results (filtered data)
			if result == nil {
				continue
			}
			
			// Send result to connected stages
			dp.mu.RLock()
			connections := dp.connections[stageName]
			dp.mu.RUnlock()
			
			if len(connections) == 0 {
				// No connections, send to output
				select {
				case dp.outputChan <- result:
				case <-dynamicStage.ctx.Done():
					return
				}
			} else {
				// Send to connected stages
				for _, nextStage := range connections {
					if nextStageObj, exists := dp.stages[nextStage]; exists {
						select {
						case nextStageObj.inputChan <- result:
						case <-dynamicStage.ctx.Done():
							return
						default:
							// Buffer full, could apply backpressure here
							select {
							case dp.errorChan <- fmt.Errorf("buffer overflow from %s to %s", stageName, nextStage):
							case <-dynamicStage.ctx.Done():
								return
							}
						}
					}
				}
			}
			
		case <-dynamicStage.ctx.Done():
			return
		}
	}
}

// calculateCurrentLoad calculates the current pipeline load
func (dp *DynamicPipeline) calculateCurrentLoad() float64 {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	
	if len(dp.stages) == 0 {
		return 0.0
	}
	
	totalLoad := 0.0
	stageCount := 0
	
	// Calculate average load across all stage input channels
	for _, dynamicStage := range dp.stages {
		if atomic.LoadInt32(&dynamicStage.running) == 1 {
			load := float64(len(dynamicStage.inputChan)) / float64(cap(dynamicStage.inputChan))
			totalLoad += load
			stageCount++
		}
	}
	
	if stageCount == 0 {
		return 0.0
	}
	
	return totalLoad / float64(stageCount)
}

// updateStageMetrics updates metrics for a specific stage
func (dp *DynamicPipeline) updateStageMetrics(stageName string, duration time.Duration, success bool) {
	dp.metrics.mu.Lock()
	defer dp.metrics.mu.Unlock()
	
	stageMetrics, exists := dp.metrics.StageMetrics[stageName]
	if !exists {
		stageMetrics = &stages.StageMetrics{}
		dp.metrics.StageMetrics[stageName] = stageMetrics
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
	dp.metrics.ProcessedCount++
	if !success {
		dp.metrics.ErrorCount++
	}
	dp.metrics.LastProcessTime = time.Now()
	
	// Calculate throughput
	elapsed := time.Since(dp.metrics.StartTime).Seconds()
	if elapsed > 0 {
		dp.metrics.Throughput = float64(dp.metrics.ProcessedCount) / elapsed
	}
}

// GetMetrics returns current pipeline metrics
func (dp *DynamicPipeline) GetMetrics() *PipelineMetrics {
	dp.metrics.mu.RLock()
	defer dp.metrics.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	metricsCopy := &PipelineMetrics{
		ProcessedCount:  dp.metrics.ProcessedCount,
		ErrorCount:      dp.metrics.ErrorCount,
		Throughput:      dp.metrics.Throughput,
		Latency:         dp.metrics.Latency,
		StartTime:       dp.metrics.StartTime,
		LastProcessTime: dp.metrics.LastProcessTime,
		StageMetrics:    make(map[string]*stages.StageMetrics),
	}
	
	// Copy stage metrics
	for name, metrics := range dp.metrics.StageMetrics {
		metricsCopy.StageMetrics[name] = &stages.StageMetrics{
			ProcessedCount:     metrics.ProcessedCount,
			ErrorCount:         metrics.ErrorCount,
			AvgLatency:        metrics.AvgLatency,
			MaxLatency:        metrics.MaxLatency,
		}
	}
	
	return metricsCopy
}

// GetName returns the pipeline name
func (dp *DynamicPipeline) GetName() string {
	return dp.name
}

// IsRunning returns whether the pipeline is currently running
func (dp *DynamicPipeline) IsRunning() bool {
	return atomic.LoadInt32(&dp.running) == 1
}

// GetStageNames returns the names of all stages
func (dp *DynamicPipeline) GetStageNames() []string {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	
	names := make([]string, len(dp.stageOrder))
	copy(names, dp.stageOrder)
	return names
}

// GetConnections returns the current stage connections
func (dp *DynamicPipeline) GetConnections() map[string][]string {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	
	connections := make(map[string][]string)
	for from, toList := range dp.connections {
		connections[from] = make([]string, len(toList))
		copy(connections[from], toList)
	}
	return connections
}

// GetVersion returns the current configuration version
func (dp *DynamicPipeline) GetVersion() int64 {
	return atomic.LoadInt64(&dp.version)
}
package pipeline

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"exercise-03-04/stages"
	"exercise-03-04/buffer"
	"exercise-03-04/backpressure"
)

// FanOutFanInPipeline implements fan-out/fan-in pattern
type FanOutFanInPipeline struct {
	mu               sync.RWMutex
	name             string
	preStages        []stages.Stage
	parallelStages   []stages.Stage
	postStages       []stages.Stage
	parallelism      int
	buffers          []*buffer.AdaptiveBuffer
	backpressureCtrl *backpressure.BackpressureController
	metrics          *PipelineMetrics
	ctx              context.Context
	cancel           context.CancelFunc
	running          bool
	inputChan        chan interface{}
	outputChan       chan interface{}
	errorChan        chan error
	fanOutChan       chan interface{}
	fanInChan        chan interface{}
	wg               sync.WaitGroup
	bufferSize       int
	workerChans      []chan interface{}
	resultChans      []chan interface{}
}

// NewFanOutFanInPipeline creates a new fan-out/fan-in pipeline
func NewFanOutFanInPipeline(name string, parallelism, bufferSize int) *FanOutFanInPipeline {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &FanOutFanInPipeline{
		name:         name,
		parallelism:  parallelism,
		preStages:    make([]stages.Stage, 0),
		parallelStages: make([]stages.Stage, 0),
		postStages:   make([]stages.Stage, 0),
		buffers:      make([]*buffer.AdaptiveBuffer, 0),
		metrics:      &PipelineMetrics{StageMetrics: make(map[string]*stages.StageMetrics)},
		ctx:          ctx,
		cancel:       cancel,
		inputChan:    make(chan interface{}, bufferSize),
		outputChan:   make(chan interface{}, bufferSize),
		errorChan:    make(chan error, bufferSize),
		fanOutChan:   make(chan interface{}, bufferSize),
		fanInChan:    make(chan interface{}, bufferSize*parallelism),
		bufferSize:   bufferSize,
		workerChans:  make([]chan interface{}, parallelism),
		resultChans:  make([]chan interface{}, parallelism),
	}
}

// AddPreStage adds a stage to run before fan-out
func (fofp *FanOutFanInPipeline) AddPreStage(stage stages.Stage) {
	fofp.mu.Lock()
	defer fofp.mu.Unlock()
	fofp.preStages = append(fofp.preStages, stage)
	fofp.metrics.StageMetrics[stage.GetName()] = &stages.StageMetrics{}
}

// AddParallelStage adds a stage to run in parallel
func (fofp *FanOutFanInPipeline) AddParallelStage(stage stages.Stage) {
	fofp.mu.Lock()
	defer fofp.mu.Unlock()
	fofp.parallelStages = append(fofp.parallelStages, stage)
	fofp.metrics.StageMetrics[stage.GetName()] = &stages.StageMetrics{}
}

// AddPostStage adds a stage to run after fan-in
func (fofp *FanOutFanInPipeline) AddPostStage(stage stages.Stage) {
	fofp.mu.Lock()
	defer fofp.mu.Unlock()
	fofp.postStages = append(fofp.postStages, stage)
	fofp.metrics.StageMetrics[stage.GetName()] = &stages.StageMetrics{}
}

// SetBackpressureController sets the backpressure controller
func (fofp *FanOutFanInPipeline) SetBackpressureController(ctrl *backpressure.BackpressureController) {
	fofp.mu.Lock()
	defer fofp.mu.Unlock()
	fofp.backpressureCtrl = ctrl
}

// Start starts the fan-out/fan-in pipeline
func (fofp *FanOutFanInPipeline) Start() error {
	fofp.mu.Lock()
	defer fofp.mu.Unlock()
	
	if fofp.running {
		return fmt.Errorf("pipeline %s is already running", fofp.name)
	}
	
	if len(fofp.parallelStages) == 0 {
		return fmt.Errorf("pipeline %s has no parallel stages", fofp.name)
	}
	
	// Initialize worker and result channels
	for i := 0; i < fofp.parallelism; i++ {
		fofp.workerChans[i] = make(chan interface{}, fofp.bufferSize)
		fofp.resultChans[i] = make(chan interface{}, fofp.bufferSize)
	}
	
	// Start all stages
	allStages := append(append(fofp.preStages, fofp.parallelStages...), fofp.postStages...)
	for _, stage := range allStages {
		if err := stage.Start(fofp.ctx); err != nil {
			return fmt.Errorf("failed to start stage %s: %w", stage.GetName(), err)
		}
	}
	
	// Start pre-processing stages
	if len(fofp.preStages) > 0 {
		fofp.wg.Add(1)
		go fofp.runPreStages()
	} else {
		// If no pre-stages, connect input directly to fan-out
		fofp.wg.Add(1)
		go fofp.forwardInputToFanOut()
	}
	
	// Start fan-out
	fofp.wg.Add(1)
	go fofp.runFanOut()
	
	// Start parallel workers
	for i := 0; i < fofp.parallelism; i++ {
		fofp.wg.Add(1)
		go fofp.runParallelWorker(i)
	}
	
	// Start fan-in
	fofp.wg.Add(1)
	go fofp.runFanIn()
	
	// Start post-processing stages
	if len(fofp.postStages) > 0 {
		fofp.wg.Add(1)
		go fofp.runPostStages()
	} else {
		// If no post-stages, connect fan-in directly to output
		fofp.wg.Add(1)
		go fofp.forwardFanInToOutput()
	}
	
	fofp.running = true
	fofp.metrics.StartTime = time.Now()
	
	return nil
}

// Stop stops the pipeline
func (fofp *FanOutFanInPipeline) Stop() error {
	fofp.mu.Lock()
	defer fofp.mu.Unlock()
	
	if !fofp.running {
		return nil
	}
	
	// Cancel context to signal shutdown
	fofp.cancel()
	
	// Close input channel to signal shutdown
	select {
	case <-fofp.inputChan:
	default:
		close(fofp.inputChan)
	}
	
	// Wait for all workers to finish
	fofp.wg.Wait()
	
	// Stop all stages
	allStages := append(append(fofp.preStages, fofp.parallelStages...), fofp.postStages...)
	for _, stage := range allStages {
		if err := stage.Stop(); err != nil {
			continue
		}
	}
	
	// Close buffers
	for _, buf := range fofp.buffers {
		buf.Close()
	}
	
	fofp.running = false
	return nil
}

// Process sends data into the pipeline
func (fofp *FanOutFanInPipeline) Process(data interface{}) error {
	fofp.mu.RLock()
	running := fofp.running
	fofp.mu.RUnlock()
	
	if !running {
		return fmt.Errorf("pipeline %s is not running", fofp.name)
	}
	
	// Check backpressure
	if fofp.backpressureCtrl != nil {
		load := fofp.calculateCurrentLoad()
		action := fofp.backpressureCtrl.CheckAndApply(load)
		
		switch action {
		case backpressure.ActionReject:
			return fmt.Errorf("request rejected due to high load")
		case backpressure.ActionDrop:
			return nil
		}
	}
	
	select {
	case fofp.inputChan <- data:
		return nil
	case <-fofp.ctx.Done():
		return fofp.ctx.Err()
	default:
		return fmt.Errorf("pipeline input buffer is full")
	}
}

// GetOutput returns the output channel
func (fofp *FanOutFanInPipeline) GetOutput() <-chan interface{} {
	return fofp.outputChan
}

// GetErrors returns the error channel
func (fofp *FanOutFanInPipeline) GetErrors() <-chan error {
	return fofp.errorChan
}

// forwardInputToFanOut forwards input directly to fan-out when no pre-stages
func (fofp *FanOutFanInPipeline) forwardInputToFanOut() {
	defer fofp.wg.Done()
	defer func() {
		select {
		case <-fofp.fanOutChan:
		default:
			close(fofp.fanOutChan)
		}
	}()
	
	for {
		select {
		case data, ok := <-fofp.inputChan:
			if !ok {
				return
			}
			select {
			case fofp.fanOutChan <- data:
			case <-fofp.ctx.Done():
				return
			}
		case <-fofp.ctx.Done():
			return
		}
	}
}

// runPreStages runs the pre-processing stages
func (fofp *FanOutFanInPipeline) runPreStages() {
	defer fofp.wg.Done()
	defer func() {
		select {
		case <-fofp.fanOutChan:
		default:
			close(fofp.fanOutChan)
		}
	}()
	
	for {
		select {
		case data, ok := <-fofp.inputChan:
			if !ok {
				return
			}
			
			// Process through all pre-stages
			current := data
			filtered := false
			for _, stage := range fofp.preStages {
				start := time.Now()
				result, err := stage.Process(current)
				processTime := time.Since(start)
				
				fofp.updateStageMetrics(stage.GetName(), processTime, err == nil)
				
				if err != nil {
					select {
					case fofp.errorChan <- fmt.Errorf("pre-stage %s error: %w", stage.GetName(), err):
					case <-fofp.ctx.Done():
						return
					}
					filtered = true
					break
				}
				
				// Check if result is nil (filtered)
				if result == nil {
					filtered = true
					break
				}
				
				current = result
			}
			
			// Skip sending to fan-out if filtered
			if !filtered {
				// Send to fan-out
				select {
				case fofp.fanOutChan <- current:
				case <-fofp.ctx.Done():
					return
				}
			}
			
		case <-fofp.ctx.Done():
			return
		}
	}
}

// runFanOut distributes work to parallel workers
func (fofp *FanOutFanInPipeline) runFanOut() {
	defer fofp.wg.Done()
	defer func() {
		for i := 0; i < fofp.parallelism; i++ {
			select {
			case <-fofp.workerChans[i]:
			default:
				close(fofp.workerChans[i])
			}
		}
	}()
	
	workerIndex := 0
	for {
		select {
		case data, ok := <-fofp.fanOutChan:
			if !ok {
				return
			}
			
			// Round-robin distribution to workers
			select {
			case fofp.workerChans[workerIndex] <- data:
				workerIndex = (workerIndex + 1) % fofp.parallelism
			case <-fofp.ctx.Done():
				return
			}
			
		case <-fofp.ctx.Done():
			return
		}
	}
}

// runParallelWorker runs a parallel worker
func (fofp *FanOutFanInPipeline) runParallelWorker(workerID int) {
	defer fofp.wg.Done()
	defer func() {
		select {
		case <-fofp.resultChans[workerID]:
		default:
			close(fofp.resultChans[workerID])
		}
	}()
	
	for {
		select {
		case data, ok := <-fofp.workerChans[workerID]:
			if !ok {
				return
			}
			
			// Process through all parallel stages
			current := data
			filtered := false
			for _, stage := range fofp.parallelStages {
				start := time.Now()
				result, err := stage.Process(current)
				processTime := time.Since(start)
				
				fofp.updateStageMetrics(stage.GetName(), processTime, err == nil)
				
				if err != nil {
					select {
					case fofp.errorChan <- fmt.Errorf("parallel-stage %s error: %w", stage.GetName(), err):
					case <-fofp.ctx.Done():
						return
					}
					filtered = true
					break
				}
				
				// Check if result is nil (filtered)
				if result == nil {
					filtered = true
					break
				}
				
				current = result
			}
			
			// Skip sending to fan-in if filtered
			if !filtered {
				// Send result to fan-in
				select {
				case fofp.resultChans[workerID] <- current:
				case <-fofp.ctx.Done():
					return
				}
			}
			
		case <-fofp.ctx.Done():
			return
		}
	}
}

// runFanIn collects results from parallel workers
func (fofp *FanOutFanInPipeline) runFanIn() {
	defer fofp.wg.Done()
	defer func() {
		select {
		case <-fofp.fanInChan:
		default:
			close(fofp.fanInChan)
		}
	}()
	
	// Use select to receive from any worker result channel
	for {
		// Create cases for select statement
		cases := make([]reflect.SelectCase, len(fofp.resultChans)+1)
		
		// Add result channels
		for i, ch := range fofp.resultChans {
			cases[i] = reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(ch),
			}
		}
		
		// Add context done case
		cases[len(fofp.resultChans)] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(fofp.ctx.Done()),
		}
		
		chosen, value, ok := reflect.Select(cases)
		
		if chosen == len(fofp.resultChans) {
			// Context done
			return
		}
		
		if !ok {
			// Channel closed, remove it from cases
			continue
		}
		
		// Send result to fan-in channel
		select {
		case fofp.fanInChan <- value.Interface():
		case <-fofp.ctx.Done():
			return
		}
	}
}

// forwardFanInToOutput forwards fan-in results directly to output when no post-stages
func (fofp *FanOutFanInPipeline) forwardFanInToOutput() {
	defer fofp.wg.Done()
	defer func() {
		select {
		case <-fofp.outputChan:
		default:
			close(fofp.outputChan)
		}
		select {
		case <-fofp.errorChan:
		default:
			close(fofp.errorChan)
		}
	}()
	
	for {
		select {
		case data, ok := <-fofp.fanInChan:
			if !ok {
				return
			}
			select {
			case fofp.outputChan <- data:
			case <-fofp.ctx.Done():
				return
			}
		case <-fofp.ctx.Done():
			return
		}
	}
}

// runPostStages runs the post-processing stages
func (fofp *FanOutFanInPipeline) runPostStages() {
	defer fofp.wg.Done()
	defer func() {
		select {
		case <-fofp.outputChan:
		default:
			close(fofp.outputChan)
		}
		select {
		case <-fofp.errorChan:
		default:
			close(fofp.errorChan)
		}
	}()
	
	for {
		select {
		case data, ok := <-fofp.fanInChan:
			if !ok {
				return
			}
			
			// Process through all post-stages
			current := data
			filtered := false
			for _, stage := range fofp.postStages {
				start := time.Now()
				result, err := stage.Process(current)
				processTime := time.Since(start)
				
				fofp.updateStageMetrics(stage.GetName(), processTime, err == nil)
				
				if err != nil {
					select {
					case fofp.errorChan <- fmt.Errorf("post-stage %s error: %w", stage.GetName(), err):
					case <-fofp.ctx.Done():
						return
					}
					filtered = true
					break
				}
				
				// Check if result is nil (filtered)
				if result == nil {
					filtered = true
					break
				}
				
				current = result
			}
			
			// Skip sending to output if filtered
			if !filtered {
				// Send to output
				select {
				case fofp.outputChan <- current:
				case <-fofp.ctx.Done():
					return
				}
			}
			
		case <-fofp.ctx.Done():
			return
		}
	}
}

// calculateCurrentLoad calculates the current pipeline load
func (fofp *FanOutFanInPipeline) calculateCurrentLoad() float64 {
	// Calculate load based on channel utilization
	inputLoad := float64(len(fofp.inputChan)) / float64(cap(fofp.inputChan))
	fanOutLoad := float64(len(fofp.fanOutChan)) / float64(cap(fofp.fanOutChan))
	fanInLoad := float64(len(fofp.fanInChan)) / float64(cap(fofp.fanInChan))
	
	// Average worker channel load
	workerLoad := 0.0
	for _, ch := range fofp.workerChans {
		workerLoad += float64(len(ch)) / float64(cap(ch))
	}
	workerLoad /= float64(len(fofp.workerChans))
	
	return (inputLoad + fanOutLoad + fanInLoad + workerLoad) / 4.0
}

// updateStageMetrics updates metrics for a specific stage
func (fofp *FanOutFanInPipeline) updateStageMetrics(stageName string, duration time.Duration, success bool) {
	fofp.metrics.mu.Lock()
	defer fofp.metrics.mu.Unlock()
	
	stageMetrics, exists := fofp.metrics.StageMetrics[stageName]
	if !exists {
		stageMetrics = &stages.StageMetrics{}
		fofp.metrics.StageMetrics[stageName] = stageMetrics
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
	fofp.metrics.ProcessedCount++
	if !success {
		fofp.metrics.ErrorCount++
	}
	fofp.metrics.LastProcessTime = time.Now()
	
	// Calculate throughput
	elapsed := time.Since(fofp.metrics.StartTime).Seconds()
	if elapsed > 0 {
		fofp.metrics.Throughput = float64(fofp.metrics.ProcessedCount) / elapsed
	}
}

// GetMetrics returns current pipeline metrics
func (fofp *FanOutFanInPipeline) GetMetrics() *PipelineMetrics {
	fofp.metrics.mu.RLock()
	defer fofp.metrics.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	metricsCopy := &PipelineMetrics{
		ProcessedCount:  fofp.metrics.ProcessedCount,
		ErrorCount:      fofp.metrics.ErrorCount,
		Throughput:      fofp.metrics.Throughput,
		Latency:         fofp.metrics.Latency,
		StartTime:       fofp.metrics.StartTime,
		LastProcessTime: fofp.metrics.LastProcessTime,
		StageMetrics:    make(map[string]*stages.StageMetrics),
	}
	
	// Copy stage metrics
	for name, metrics := range fofp.metrics.StageMetrics {
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
func (fofp *FanOutFanInPipeline) GetName() string {
	return fofp.name
}

// IsRunning returns whether the pipeline is currently running
func (fofp *FanOutFanInPipeline) IsRunning() bool {
	fofp.mu.RLock()
	defer fofp.mu.RUnlock()
	return fofp.running
}
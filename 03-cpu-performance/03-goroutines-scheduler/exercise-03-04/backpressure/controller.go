package backpressure

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Action represents a backpressure action
type Action int

const (
	ActionNone Action = iota
	ActionThrottle
	ActionDrop
	ActionBlock
	ActionReject
)

// BackpressureController manages backpressure in pipelines
type BackpressureController struct {
	mu                sync.RWMutex
	thresholds        *Thresholds
	currentLoad       float64
	lastCheck         time.Time
	checkInterval     time.Duration
	actions           []ActionHandler
	metrics           *ControllerMetrics
	ctx               context.Context
	cancel            context.CancelFunc
	loadHistory       []float64
	historySize       int
	adaptiveThreshold bool
}

// Thresholds defines backpressure thresholds
type Thresholds struct {
	LowWaterMark    float64 // Below this: no action
	MediumWaterMark float64 // Above this: throttle
	HighWaterMark   float64 // Above this: drop
	CriticalMark    float64 // Above this: block/reject
}

// ActionHandler defines how to handle backpressure actions
type ActionHandler interface {
	Handle(action Action, load float64) error
}

// ControllerMetrics tracks backpressure controller performance
type ControllerMetrics struct {
	ThrottleCount int64
	DropCount     int64
	BlockCount    int64
	RejectCount   int64
	AvgLoad       float64
	PeakLoad      float64
	mu            sync.RWMutex
}

// NewBackpressureController creates a new backpressure controller
func NewBackpressureController(thresholds *Thresholds) *BackpressureController {
	ctx, cancel := context.WithCancel(context.Background())
	
	bc := &BackpressureController{
		thresholds:        thresholds,
		checkInterval:     time.Millisecond * 100,
		metrics:           &ControllerMetrics{},
		ctx:               ctx,
		cancel:            cancel,
		loadHistory:       make([]float64, 0, 100),
		historySize:       100,
		adaptiveThreshold: true,
		lastCheck:         time.Now(),
	}
	
	// Start monitoring
	go bc.monitor()
	
	return bc
}

// CheckAndApply checks current load and applies appropriate backpressure
func (bc *BackpressureController) CheckAndApply(currentLoad float64) Action {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	bc.currentLoad = currentLoad
	bc.updateLoadHistory(currentLoad)
	bc.updateMetrics(currentLoad)
	
	// Determine action based on load
	action := bc.determineAction(currentLoad)
	
	// Apply action through handlers
	for _, handler := range bc.actions {
		if err := handler.Handle(action, currentLoad); err != nil {
			// Log error but continue
			continue
		}
	}
	
	// Update action metrics
	bc.updateActionMetrics(action)
	
	return action
}

// determineAction decides what action to take based on current load
func (bc *BackpressureController) determineAction(load float64) Action {
	thresholds := bc.getEffectiveThresholds()
	
	switch {
	case load >= thresholds.CriticalMark:
		return ActionReject
	case load >= thresholds.HighWaterMark:
		return ActionDrop
	case load >= thresholds.MediumWaterMark:
		return ActionThrottle
	default:
		return ActionNone
	}
}

// getEffectiveThresholds returns thresholds, potentially adjusted for adaptive behavior
func (bc *BackpressureController) getEffectiveThresholds() *Thresholds {
	if !bc.adaptiveThreshold || len(bc.loadHistory) < 10 {
		return bc.thresholds
	}
	
	// Calculate adaptive thresholds based on load history
	avgLoad := bc.calculateAverageLoad()
	stdDev := bc.calculateStandardDeviation(avgLoad)
	
	// Adjust thresholds based on recent patterns
	adaptive := &Thresholds{
		LowWaterMark:    bc.thresholds.LowWaterMark,
		MediumWaterMark: avgLoad + stdDev*0.5,
		HighWaterMark:   avgLoad + stdDev*1.0,
		CriticalMark:    avgLoad + stdDev*1.5,
	}
	
	// Ensure adaptive thresholds don't exceed original limits
	if adaptive.MediumWaterMark > bc.thresholds.MediumWaterMark {
		adaptive.MediumWaterMark = bc.thresholds.MediumWaterMark
	}
	if adaptive.HighWaterMark > bc.thresholds.HighWaterMark {
		adaptive.HighWaterMark = bc.thresholds.HighWaterMark
	}
	if adaptive.CriticalMark > bc.thresholds.CriticalMark {
		adaptive.CriticalMark = bc.thresholds.CriticalMark
	}
	
	return adaptive
}

// AddActionHandler adds a handler for backpressure actions
func (bc *BackpressureController) AddActionHandler(handler ActionHandler) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.actions = append(bc.actions, handler)
}

// updateLoadHistory maintains a rolling window of load values
func (bc *BackpressureController) updateLoadHistory(load float64) {
	bc.loadHistory = append(bc.loadHistory, load)
	if len(bc.loadHistory) > bc.historySize {
		bc.loadHistory = bc.loadHistory[1:]
	}
}

// calculateAverageLoad computes average load from history
func (bc *BackpressureController) calculateAverageLoad() float64 {
	if len(bc.loadHistory) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, load := range bc.loadHistory {
		sum += load
	}
	return sum / float64(len(bc.loadHistory))
}

// calculateStandardDeviation computes standard deviation of load history
func (bc *BackpressureController) calculateStandardDeviation(avg float64) float64 {
	if len(bc.loadHistory) < 2 {
		return 0
	}
	
	sum := 0.0
	for _, load := range bc.loadHistory {
		diff := load - avg
		sum += diff * diff
	}
	
	variance := sum / float64(len(bc.loadHistory)-1)
	return variance // Simplified: not taking square root for performance
}

// updateMetrics updates controller metrics
func (bc *BackpressureController) updateMetrics(load float64) {
	bc.metrics.mu.Lock()
	defer bc.metrics.mu.Unlock()
	
	// Update average load (simple moving average)
	bc.metrics.AvgLoad = (bc.metrics.AvgLoad + load) / 2
	
	// Update peak load
	if load > bc.metrics.PeakLoad {
		bc.metrics.PeakLoad = load
	}
}

// updateActionMetrics updates action-specific metrics
func (bc *BackpressureController) updateActionMetrics(action Action) {
	switch action {
	case ActionThrottle:
		atomic.AddInt64(&bc.metrics.ThrottleCount, 1)
	case ActionDrop:
		atomic.AddInt64(&bc.metrics.DropCount, 1)
	case ActionBlock:
		atomic.AddInt64(&bc.metrics.BlockCount, 1)
	case ActionReject:
		atomic.AddInt64(&bc.metrics.RejectCount, 1)
	}
}

// monitor runs background monitoring and adjustment
func (bc *BackpressureController) monitor() {
	ticker := time.NewTicker(bc.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			bc.performPeriodicCheck()
		case <-bc.ctx.Done():
			return
		}
	}
}

// performPeriodicCheck performs periodic health checks and adjustments
func (bc *BackpressureController) performPeriodicCheck() {
	bc.mu.RLock()
	lastCheck := bc.lastCheck
	bc.mu.RUnlock()
	
	// Check if we haven't received updates recently
	if time.Since(lastCheck) > bc.checkInterval*5 {
		// Assume load is decreasing if no recent updates
		bc.mu.Lock()
		bc.currentLoad *= 0.9 // Decay factor
		bc.lastCheck = time.Now()
		bc.mu.Unlock()
	}
	
	// Perform adaptive threshold adjustment if enabled
	if bc.adaptiveThreshold {
		bc.adjustThresholds()
	}
}

// adjustThresholds dynamically adjusts thresholds based on patterns
func (bc *BackpressureController) adjustThresholds() {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	if len(bc.loadHistory) < 20 {
		return
	}
	
	// Analyze recent trends
	recentAvg := bc.calculateRecentAverage(10)
	overallAvg := bc.calculateAverageLoad()
	
	// If recent load is consistently higher, slightly increase thresholds
	if recentAvg > overallAvg*1.2 {
		bc.thresholds.MediumWaterMark *= 1.05
		bc.thresholds.HighWaterMark *= 1.05
		bc.thresholds.CriticalMark *= 1.05
	}
	
	// If recent load is consistently lower, slightly decrease thresholds
	if recentAvg < overallAvg*0.8 {
		bc.thresholds.MediumWaterMark *= 0.95
		bc.thresholds.HighWaterMark *= 0.95
		bc.thresholds.CriticalMark *= 0.95
	}
}

// calculateRecentAverage calculates average of recent N values
func (bc *BackpressureController) calculateRecentAverage(n int) float64 {
	historyLen := len(bc.loadHistory)
	if historyLen == 0 {
		return 0
	}
	
	start := historyLen - n
	if start < 0 {
		start = 0
	}
	
	sum := 0.0
	count := 0
	for i := start; i < historyLen; i++ {
		sum += bc.loadHistory[i]
		count++
	}
	
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// GetMetrics returns current controller metrics
func (bc *BackpressureController) GetMetrics() ControllerMetrics {
	bc.metrics.mu.RLock()
	defer bc.metrics.mu.RUnlock()
	
	return ControllerMetrics{
		ThrottleCount: atomic.LoadInt64(&bc.metrics.ThrottleCount),
		DropCount:     atomic.LoadInt64(&bc.metrics.DropCount),
		BlockCount:    atomic.LoadInt64(&bc.metrics.BlockCount),
		RejectCount:   atomic.LoadInt64(&bc.metrics.RejectCount),
		AvgLoad:       bc.metrics.AvgLoad,
		PeakLoad:      bc.metrics.PeakLoad,
	}
}

// GetCurrentLoad returns the current load value
func (bc *BackpressureController) GetCurrentLoad() float64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.currentLoad
}

// SetThresholds updates the backpressure thresholds
func (bc *BackpressureController) SetThresholds(thresholds *Thresholds) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.thresholds = thresholds
}

// SetAdaptiveThreshold enables or disables adaptive threshold adjustment
func (bc *BackpressureController) SetAdaptiveThreshold(enabled bool) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.adaptiveThreshold = enabled
}

// Close shuts down the backpressure controller
func (bc *BackpressureController) Close() {
	bc.cancel()
}
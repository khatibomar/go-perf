package backpressure

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// ThrottleAction implements throttling backpressure
type ThrottleAction struct {
	mu           sync.RWMutex
	throttleRate float64 // requests per second
	lastRequest  time.Time
	interval     time.Duration
	enabled      int32
}

// NewThrottleAction creates a new throttle action
func NewThrottleAction(requestsPerSecond float64) *ThrottleAction {
	interval := time.Duration(float64(time.Second) / requestsPerSecond)
	return &ThrottleAction{
		throttleRate: requestsPerSecond,
		interval:     interval,
		lastRequest:  time.Now(),
	}
}

// Handle implements ActionHandler interface for throttling
func (ta *ThrottleAction) Handle(action Action, load float64) error {
	if action != ActionThrottle {
		atomic.StoreInt32(&ta.enabled, 0)
		return nil
	}
	
	atomic.StoreInt32(&ta.enabled, 1)
	
	// Adjust throttle rate based on load
	ta.mu.Lock()
	defer ta.mu.Unlock()
	
	// Reduce rate as load increases
	adjustedRate := ta.throttleRate * (1.0 - (load-0.5)*0.5)
	if adjustedRate < ta.throttleRate*0.1 {
		adjustedRate = ta.throttleRate * 0.1
	}
	
	ta.interval = time.Duration(float64(time.Second) / adjustedRate)
	return nil
}

// ShouldThrottle checks if a request should be throttled
func (ta *ThrottleAction) ShouldThrottle() bool {
	if atomic.LoadInt32(&ta.enabled) == 0 {
		return false
	}
	
	ta.mu.Lock()
	defer ta.mu.Unlock()
	
	now := time.Now()
	if now.Sub(ta.lastRequest) < ta.interval {
		return true
	}
	
	ta.lastRequest = now
	return false
}

// Wait blocks until throttling period is over
func (ta *ThrottleAction) Wait(ctx context.Context) error {
	if atomic.LoadInt32(&ta.enabled) == 0 {
		return nil
	}
	
	ta.mu.RLock()
	interval := ta.interval
	lastRequest := ta.lastRequest
	ta.mu.RUnlock()
	
	waitTime := interval - time.Since(lastRequest)
	if waitTime <= 0 {
		return nil
	}
	
	timer := time.NewTimer(waitTime)
	defer timer.Stop()
	
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// DropAction implements dropping backpressure
type DropAction struct {
	mu           sync.RWMutex
	dropRate     float64 // probability of dropping (0.0 to 1.0)
	enabled      int32
	dropCount    int64
	totalCount   int64
	lastAdjust   time.Time
	adjustPeriod time.Duration
}

// NewDropAction creates a new drop action
func NewDropAction(initialDropRate float64) *DropAction {
	return &DropAction{
		dropRate:     initialDropRate,
		lastAdjust:   time.Now(),
		adjustPeriod: time.Second,
	}
}

// Handle implements ActionHandler interface for dropping
func (da *DropAction) Handle(action Action, load float64) error {
	if action != ActionDrop {
		atomic.StoreInt32(&da.enabled, 0)
		return nil
	}
	
	atomic.StoreInt32(&da.enabled, 1)
	
	// Adjust drop rate based on load
	da.mu.Lock()
	defer da.mu.Unlock()
	
	// Increase drop rate as load increases
	if load > 0.8 {
		da.dropRate = (load - 0.5) * 0.8 // Scale from 0.3*0.8=0.24 to 0.5*0.8=0.4
	} else {
		da.dropRate = (load - 0.5) * 0.4 // Lower drop rate for moderate load
	}
	
	// Ensure drop rate is within bounds
	if da.dropRate < 0 {
		da.dropRate = 0
	}
	if da.dropRate > 0.9 {
		da.dropRate = 0.9
	}
	
	return nil
}

// ShouldDrop determines if a request should be dropped
func (da *DropAction) ShouldDrop() bool {
	if atomic.LoadInt32(&da.enabled) == 0 {
		return false
	}
	
	atomic.AddInt64(&da.totalCount, 1)
	
	da.mu.RLock()
	dropRate := da.dropRate
	da.mu.RUnlock()
	
	// Simple probabilistic dropping based on current time
	// Using nano time for pseudo-randomness
	random := float64(time.Now().UnixNano()%1000) / 1000.0
	
	if random < dropRate {
		atomic.AddInt64(&da.dropCount, 1)
		return true
	}
	
	return false
}

// GetDropRate returns current drop rate
func (da *DropAction) GetDropRate() float64 {
	da.mu.RLock()
	defer da.mu.RUnlock()
	return da.dropRate
}

// GetDropStats returns drop statistics
func (da *DropAction) GetDropStats() (dropped, total int64, rate float64) {
	dropped = atomic.LoadInt64(&da.dropCount)
	total = atomic.LoadInt64(&da.totalCount)
	if total > 0 {
		rate = float64(dropped) / float64(total)
	}
	return
}

// BlockAction implements blocking backpressure
type BlockAction struct {
	mu          sync.RWMutex
	blockTime   time.Duration
	enabled     int32
	blockCount  int64
	lastBlock   time.Time
	maxBlockers int
	currentBlockers int32
}

// NewBlockAction creates a new block action
func NewBlockAction(blockTime time.Duration, maxBlockers int) *BlockAction {
	return &BlockAction{
		blockTime:   blockTime,
		maxBlockers: maxBlockers,
	}
}

// Handle implements ActionHandler interface for blocking
func (ba *BlockAction) Handle(action Action, load float64) error {
	if action != ActionBlock {
		atomic.StoreInt32(&ba.enabled, 0)
		return nil
	}
	
	atomic.StoreInt32(&ba.enabled, 1)
	
	// Adjust block time based on load
	ba.mu.Lock()
	defer ba.mu.Unlock()
	
	// Increase block time as load increases
	baseTime := ba.blockTime
	loadFactor := (load - 0.7) * 2 // Scale factor for load above 0.7
	if loadFactor < 0 {
		loadFactor = 0
	}
	
	ba.blockTime = time.Duration(float64(baseTime) * (1 + loadFactor))
	ba.lastBlock = time.Now()
	
	return nil
}

// ShouldBlock determines if a request should be blocked
func (ba *BlockAction) ShouldBlock() bool {
	if atomic.LoadInt32(&ba.enabled) == 0 {
		return false
	}
	
	// Check if we've reached max concurrent blockers
	current := atomic.LoadInt32(&ba.currentBlockers)
	if int(current) >= ba.maxBlockers {
		return false // Don't block if too many are already blocked
	}
	
	return true
}

// Block blocks the current goroutine
func (ba *BlockAction) Block(ctx context.Context) error {
	if !ba.ShouldBlock() {
		return nil
	}
	
	atomic.AddInt32(&ba.currentBlockers, 1)
	atomic.AddInt64(&ba.blockCount, 1)
	defer atomic.AddInt32(&ba.currentBlockers, -1)
	
	ba.mu.RLock()
	blockTime := ba.blockTime
	ba.mu.RUnlock()
	
	timer := time.NewTimer(blockTime)
	defer timer.Stop()
	
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetBlockStats returns blocking statistics
func (ba *BlockAction) GetBlockStats() (blockCount int64, currentBlockers int32) {
	return atomic.LoadInt64(&ba.blockCount), atomic.LoadInt32(&ba.currentBlockers)
}

// RejectAction implements rejection backpressure
type RejectAction struct {
	enabled     int32
	rejectCount int64
	totalCount  int64
}

// NewRejectAction creates a new reject action
func NewRejectAction() *RejectAction {
	return &RejectAction{}
}

// Handle implements ActionHandler interface for rejection
func (ra *RejectAction) Handle(action Action, load float64) error {
	if action != ActionReject {
		atomic.StoreInt32(&ra.enabled, 0)
		return nil
	}
	
	atomic.StoreInt32(&ra.enabled, 1)
	return nil
}

// ShouldReject determines if a request should be rejected
func (ra *RejectAction) ShouldReject() bool {
	atomic.AddInt64(&ra.totalCount, 1)
	
	if atomic.LoadInt32(&ra.enabled) == 1 {
		atomic.AddInt64(&ra.rejectCount, 1)
		return true
	}
	
	return false
}

// GetRejectStats returns rejection statistics
func (ra *RejectAction) GetRejectStats() (rejected, total int64, rate float64) {
	rejected = atomic.LoadInt64(&ra.rejectCount)
	total = atomic.LoadInt64(&ra.totalCount)
	if total > 0 {
		rate = float64(rejected) / float64(total)
	}
	return
}

// CompositeActionHandler combines multiple action handlers
type CompositeActionHandler struct {
	handlers []ActionHandler
	mu       sync.RWMutex
}

// NewCompositeActionHandler creates a new composite action handler
func NewCompositeActionHandler(handlers ...ActionHandler) *CompositeActionHandler {
	return &CompositeActionHandler{
		handlers: handlers,
	}
}

// Handle implements ActionHandler interface for composite actions
func (cah *CompositeActionHandler) Handle(action Action, load float64) error {
	cah.mu.RLock()
	defer cah.mu.RUnlock()
	
	for _, handler := range cah.handlers {
		if err := handler.Handle(action, load); err != nil {
			// Continue with other handlers even if one fails
			continue
		}
	}
	
	return nil
}

// AddHandler adds a new action handler
func (cah *CompositeActionHandler) AddHandler(handler ActionHandler) {
	cah.mu.Lock()
	defer cah.mu.Unlock()
	cah.handlers = append(cah.handlers, handler)
}

// RemoveHandler removes an action handler
func (cah *CompositeActionHandler) RemoveHandler(handler ActionHandler) {
	cah.mu.Lock()
	defer cah.mu.Unlock()
	
	for i, h := range cah.handlers {
		if h == handler {
			cah.handlers = append(cah.handlers[:i], cah.handlers[i+1:]...)
			break
		}
	}
}
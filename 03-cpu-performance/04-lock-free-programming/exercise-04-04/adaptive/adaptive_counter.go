package adaptive

import (
	"runtime"
	"sync/atomic"
	"time"
	"exercise-04-04/counters"
)

// CounterStrategy represents different counter strategies
type CounterStrategy int

const (
	AtomicStrategy CounterStrategy = iota
	StripedStrategy
	ShardedStrategy
	NUMAStrategy
)

// AdaptiveCounter switches between different counter strategies based on contention
type AdaptiveCounter struct {
	currentStrategy CounterStrategy
	atomicCounter   *counters.AtomicCounter
	stripedCounter  *counters.StripedCounter
	shardedCounter  *counters.ShardedCounter
	numaCounter     *counters.NUMACounter
	
	// Metrics for adaptation
	contentionLevel int64
	lastSwitch      time.Time
	switchCooldown  time.Duration
	operationCount  int64
	measurementWindow time.Duration
	lastMeasurement   time.Time
	_               [32]byte // Padding
}

func NewAdaptiveCounter() *AdaptiveCounter {
	return &AdaptiveCounter{
		currentStrategy:   AtomicStrategy,
		atomicCounter:     counters.NewAtomicCounter(),
		stripedCounter:    counters.NewStripedCounter(runtime.NumCPU()),
		shardedCounter:    counters.NewShardedCounter(),
		numaCounter:       counters.NewNUMACounter(),
		switchCooldown:    100 * time.Millisecond,
		measurementWindow: 50 * time.Millisecond,
		lastMeasurement:   time.Now(),
	}
}

func (ac *AdaptiveCounter) Add(delta int64) {
	ac.recordOperation()
	ac.maybeAdapt()
	
	switch ac.currentStrategy {
	case AtomicStrategy:
		ac.atomicCounter.Add(delta)
	case StripedStrategy:
		ac.stripedCounter.Add(delta)
	case ShardedStrategy:
		ac.shardedCounter.Add(delta)
	case NUMAStrategy:
		ac.numaCounter.Add(delta)
	}
}

func (ac *AdaptiveCounter) Inc() {
	ac.Add(1)
}

func (ac *AdaptiveCounter) Dec() {
	ac.Add(-1)
}

func (ac *AdaptiveCounter) Get() int64 {
	switch ac.currentStrategy {
	case AtomicStrategy:
		return ac.atomicCounter.Get()
	case StripedStrategy:
		return ac.stripedCounter.Get()
	case ShardedStrategy:
		return ac.shardedCounter.Get()
	case NUMAStrategy:
		return ac.numaCounter.Get()
	default:
		return 0
	}
}

func (ac *AdaptiveCounter) Reset() {
	ac.atomicCounter.Reset()
	ac.stripedCounter.Reset()
	ac.shardedCounter.Reset()
	ac.numaCounter.Reset()
	atomic.StoreInt64(&ac.contentionLevel, 0)
	atomic.StoreInt64(&ac.operationCount, 0)
}

func (ac *AdaptiveCounter) recordOperation() {
	atomic.AddInt64(&ac.operationCount, 1)
	
	// Simple contention detection based on operation frequency
	now := time.Now()
	if now.Sub(ac.lastMeasurement) > ac.measurementWindow {
		operations := atomic.SwapInt64(&ac.operationCount, 0)
		contention := operations * int64(ac.measurementWindow) / int64(time.Second)
		atomic.StoreInt64(&ac.contentionLevel, contention)
		ac.lastMeasurement = now
	}
}

func (ac *AdaptiveCounter) maybeAdapt() {
	now := time.Now()
	if now.Sub(ac.lastSwitch) < ac.switchCooldown {
		return
	}
	
	contention := atomic.LoadInt64(&ac.contentionLevel)
	newStrategy := ac.selectOptimalStrategy(contention)
	
	if newStrategy != ac.currentStrategy {
		ac.switchStrategy(newStrategy)
		ac.lastSwitch = now
	}
}

func (ac *AdaptiveCounter) selectOptimalStrategy(contention int64) CounterStrategy {
	numCPU := int64(runtime.NumCPU())
	
	// Strategy selection based on contention level
	switch {
	case contention < numCPU*10:
		return AtomicStrategy
	case contention < numCPU*50:
		return StripedStrategy
	case contention < numCPU*200:
		return ShardedStrategy
	default:
		return NUMAStrategy
	}
}

func (ac *AdaptiveCounter) switchStrategy(newStrategy CounterStrategy) {
	// Synchronize values when switching
	currentValue := ac.Get()
	
	ac.currentStrategy = newStrategy
	
	// Reset all counters and set the new active one
	ac.atomicCounter.Reset()
	ac.stripedCounter.Reset()
	ac.shardedCounter.Reset()
	ac.numaCounter.Reset()
	
	// Set the current value in the new strategy
	switch newStrategy {
	case AtomicStrategy:
		ac.atomicCounter.Add(currentValue)
	case StripedStrategy:
		ac.stripedCounter.Add(currentValue)
	case ShardedStrategy:
		ac.shardedCounter.Add(currentValue)
	case NUMAStrategy:
		ac.numaCounter.Add(currentValue)
	}
}

func (ac *AdaptiveCounter) GetCurrentStrategy() CounterStrategy {
	return ac.currentStrategy
}

func (ac *AdaptiveCounter) GetContentionLevel() int64 {
	return atomic.LoadInt64(&ac.contentionLevel)
}

func (ac *AdaptiveCounter) GetStrategyName() string {
	switch ac.currentStrategy {
	case AtomicStrategy:
		return "Atomic"
	case StripedStrategy:
		return "Striped"
	case ShardedStrategy:
		return "Sharded"
	case NUMAStrategy:
		return "NUMA"
	default:
		return "Unknown"
	}
}
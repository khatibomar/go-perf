package approximate

import (
	"math"
	"runtime"
	"sync/atomic"
)

// HierarchicalCounter implements hierarchical counter design
type HierarchicalCounter struct {
	levels        []counterLevel
	numLevels     int
	baseThreshold int64
	fanout        int
}

type counterLevel struct {
	counters  []paddedCounter
	threshold int64
	_         [40]byte // Padding
}

type paddedCounter struct {
	value int64
	_     [56]byte // Padding
}

func NewHierarchicalCounter(baseThreshold int64, numLevels, fanout int) *HierarchicalCounter {
	if numLevels <= 0 {
		numLevels = 3
	}
	if fanout <= 0 {
		fanout = 4
	}
	
	hc := &HierarchicalCounter{
		levels:        make([]counterLevel, numLevels),
		numLevels:     numLevels,
		baseThreshold: baseThreshold,
		fanout:        fanout,
	}
	
	// Initialize levels with exponentially increasing thresholds
	for level := 0; level < numLevels; level++ {
		numCounters := int(math.Pow(float64(fanout), float64(level)))
		if numCounters == 0 {
			numCounters = 1
		}
		threshold := baseThreshold * int64(math.Pow(2, float64(level)))
		
		hc.levels[level] = counterLevel{
			counters:  make([]paddedCounter, numCounters),
			threshold: threshold,
		}
	}
	
	return hc
}

func (hc *HierarchicalCounter) Add(delta int64) {
	hc.addToLevel(0, delta)
}

func (hc *HierarchicalCounter) addToLevel(level int, delta int64) {
	if level >= hc.numLevels {
		return
	}
	
	counterIdx := hc.getCounterIndex(level)
	atomic.AddInt64(&hc.levels[level].counters[counterIdx].value, delta)
}

func (hc *HierarchicalCounter) Inc() {
	hc.Add(1)
}

func (hc *HierarchicalCounter) Dec() {
	hc.Add(-1)
}

func (hc *HierarchicalCounter) Get() int64 {
	var total int64
	for level := range hc.levels {
		for counterIdx := range hc.levels[level].counters {
			total += atomic.LoadInt64(&hc.levels[level].counters[counterIdx].value)
		}
	}
	return total
}

func (hc *HierarchicalCounter) GetByLevel() []int64 {
	levelTotals := make([]int64, hc.numLevels)
	for level := range hc.levels {
		var levelTotal int64
		for counterIdx := range hc.levels[level].counters {
			levelTotal += atomic.LoadInt64(&hc.levels[level].counters[counterIdx].value)
		}
		levelTotals[level] = levelTotal
	}
	return levelTotals
}

func (hc *HierarchicalCounter) Reset() {
	for level := range hc.levels {
		for counterIdx := range hc.levels[level].counters {
			atomic.StoreInt64(&hc.levels[level].counters[counterIdx].value, 0)
		}
	}
}

func (hc *HierarchicalCounter) getCounterIndex(level int) int {
	gid := runtime.NumGoroutine()
	numCounters := len(hc.levels[level].counters)
	return gid % numCounters
}

func (hc *HierarchicalCounter) GetNumLevels() int {
	return hc.numLevels
}

func (hc *HierarchicalCounter) GetBaseThreshold() int64 {
	return hc.baseThreshold
}

func (hc *HierarchicalCounter) GetFanout() int {
	return hc.fanout
}
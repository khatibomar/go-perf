package adaptive

import (
	"runtime"
	"sync/atomic"
	"time"
	"exercise-04-04/counters"
)

// LoadBalancer distributes operations across multiple counter instances
type LoadBalancer struct {
	counters    []Counter
	loadMetrics []loadMetric
	numCounters int
	balanceMode BalanceMode
	roundRobin  int64
	_           [32]byte // Padding
}

type Counter interface {
	Add(delta int64)
	Inc()
	Dec()
	Get() int64
	Reset()
}

type loadMetric struct {
	operations int64
	lastUpdate time.Time
	latency    int64 // in nanoseconds
	_          [40]byte // Padding
}

type BalanceMode int

const (
	RoundRobinMode BalanceMode = iota
	LeastLoadedMode
	LatencyBasedMode
	AffinityMode
)

func NewLoadBalancer(numCounters int, mode BalanceMode) *LoadBalancer {
	if numCounters <= 0 {
		numCounters = runtime.NumCPU()
	}
	
	lb := &LoadBalancer{
		counters:    make([]Counter, numCounters),
		loadMetrics: make([]loadMetric, numCounters),
		numCounters: numCounters,
		balanceMode: mode,
	}
	
	// Initialize counters with different strategies for diversity
	for i := 0; i < numCounters; i++ {
		switch i % 4 {
		case 0:
			lb.counters[i] = counters.NewAtomicCounter()
		case 1:
			lb.counters[i] = counters.NewStripedCounter(4)
		case 2:
			lb.counters[i] = counters.NewShardedCounter()
		case 3:
			lb.counters[i] = counters.NewNUMACounter()
		}
		lb.loadMetrics[i].lastUpdate = time.Now()
	}
	
	return lb
}

func (lb *LoadBalancer) Add(delta int64) {
	counterIdx := lb.selectCounter()
	start := time.Now()
	
	lb.counters[counterIdx].Add(delta)
	
	// Update metrics
	latency := time.Since(start).Nanoseconds()
	atomic.AddInt64(&lb.loadMetrics[counterIdx].operations, 1)
	atomic.StoreInt64(&lb.loadMetrics[counterIdx].latency, latency)
	lb.loadMetrics[counterIdx].lastUpdate = time.Now()
}

func (lb *LoadBalancer) Inc() {
	lb.Add(1)
}

func (lb *LoadBalancer) Dec() {
	lb.Add(-1)
}

func (lb *LoadBalancer) Get() int64 {
	var total int64
	for _, counter := range lb.counters {
		total += counter.Get()
	}
	return total
}

func (lb *LoadBalancer) Reset() {
	for i, counter := range lb.counters {
		counter.Reset()
		atomic.StoreInt64(&lb.loadMetrics[i].operations, 0)
		atomic.StoreInt64(&lb.loadMetrics[i].latency, 0)
		lb.loadMetrics[i].lastUpdate = time.Now()
	}
	atomic.StoreInt64(&lb.roundRobin, 0)
}

func (lb *LoadBalancer) selectCounter() int {
	switch lb.balanceMode {
	case RoundRobinMode:
		return lb.selectRoundRobin()
	case LeastLoadedMode:
		return lb.selectLeastLoaded()
	case LatencyBasedMode:
		return lb.selectLowestLatency()
	case AffinityMode:
		return lb.selectByAffinity()
	default:
		return lb.selectRoundRobin()
	}
}

func (lb *LoadBalancer) selectRoundRobin() int {
	idx := atomic.AddInt64(&lb.roundRobin, 1) - 1
	return int(idx) % lb.numCounters
}

func (lb *LoadBalancer) selectLeastLoaded() int {
	minLoad := int64(^uint64(0) >> 1) // Max int64
	selectedIdx := 0
	
	for i := 0; i < lb.numCounters; i++ {
		load := atomic.LoadInt64(&lb.loadMetrics[i].operations)
		if load < minLoad {
			minLoad = load
			selectedIdx = i
		}
	}
	
	return selectedIdx
}

func (lb *LoadBalancer) selectLowestLatency() int {
	minLatency := int64(^uint64(0) >> 1) // Max int64
	selectedIdx := 0
	
	for i := 0; i < lb.numCounters; i++ {
		latency := atomic.LoadInt64(&lb.loadMetrics[i].latency)
		if latency == 0 {
			// No latency data, prefer this counter
			return i
		}
		if latency < minLatency {
			minLatency = latency
			selectedIdx = i
		}
	}
	
	return selectedIdx
}

func (lb *LoadBalancer) selectByAffinity() int {
	// Use goroutine ID for CPU affinity simulation
	gid := runtime.NumGoroutine()
	return gid % lb.numCounters
}

func (lb *LoadBalancer) GetLoadMetrics() []LoadMetric {
	metrics := make([]LoadMetric, lb.numCounters)
	for i := 0; i < lb.numCounters; i++ {
		metrics[i] = LoadMetric{
			Operations: atomic.LoadInt64(&lb.loadMetrics[i].operations),
			Latency:    atomic.LoadInt64(&lb.loadMetrics[i].latency),
			LastUpdate: lb.loadMetrics[i].lastUpdate,
		}
	}
	return metrics
}

func (lb *LoadBalancer) GetCounterValues() []int64 {
	values := make([]int64, lb.numCounters)
	for i, counter := range lb.counters {
		values[i] = counter.Get()
	}
	return values
}

func (lb *LoadBalancer) GetBalanceMode() BalanceMode {
	return lb.balanceMode
}

func (lb *LoadBalancer) SetBalanceMode(mode BalanceMode) {
	lb.balanceMode = mode
}

func (lb *LoadBalancer) GetNumCounters() int {
	return lb.numCounters
}

type LoadMetric struct {
	Operations int64
	Latency    int64
	LastUpdate time.Time
}

func (bm BalanceMode) String() string {
	switch bm {
	case RoundRobinMode:
		return "RoundRobin"
	case LeastLoadedMode:
		return "LeastLoaded"
	case LatencyBasedMode:
		return "LatencyBased"
	case AffinityMode:
		return "Affinity"
	default:
		return "Unknown"
	}
}
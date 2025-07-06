package workloads

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

// Workload represents a memory allocation workload
type Workload interface {
	Run(ctx context.Context) error
	Stop()
	GetStats() WorkloadStats
}

// WorkloadStats contains statistics about workload execution
type WorkloadStats struct {
	TotalAllocations uint64        `json:"total_allocations"`
	TotalBytes       uint64        `json:"total_bytes"`
	Duration         time.Duration `json:"duration"`
	AverageRate      float64       `json:"average_rate"`
	PeakRate         float64       `json:"peak_rate"`
	MinRate          float64       `json:"min_rate"`
	ErrorCount       uint64        `json:"error_count"`
}

// BurstConfig configures burst workload
type BurstConfig struct {
	BurstSize     int           `json:"burst_size"`
	BurstInterval time.Duration `json:"burst_interval"`
	ObjectSize    int           `json:"object_size"`
	BurstCount    int           `json:"burst_count"`
	Randomness    float64       `json:"randomness"`
}

// BurstWorkload generates allocation bursts
type BurstWorkload struct {
	config   *BurstConfig
	stats    WorkloadStats
	running  bool
	stopChan chan struct{}
	mutex    sync.RWMutex
	data     [][]byte
}

// NewBurstWorkload creates a new burst workload
func NewBurstWorkload(config *BurstConfig) (*BurstWorkload, error) {
	if config.BurstSize <= 0 {
		return nil, fmt.Errorf("burst size must be positive")
	}
	if config.ObjectSize <= 0 {
		return nil, fmt.Errorf("object size must be positive")
	}

	return &BurstWorkload{
		config:   config,
		stopChan: make(chan struct{}),
		data:     make([][]byte, 0),
	}, nil
}

// Run executes the burst workload
func (bw *BurstWorkload) Run(ctx context.Context) error {
	bw.mutex.Lock()
	if bw.running {
		bw.mutex.Unlock()
		return fmt.Errorf("workload is already running")
	}
	bw.running = true
	bw.stats = WorkloadStats{}
	bw.mutex.Unlock()

	startTime := time.Now()
	ticker := time.NewTicker(bw.config.BurstInterval)
	defer ticker.Stop()

	burstCount := 0
	for {
		select {
		case <-ctx.Done():
			bw.finalize(startTime)
			return nil
		case <-bw.stopChan:
			bw.finalize(startTime)
			return nil
		case <-ticker.C:
			if bw.config.BurstCount > 0 && burstCount >= bw.config.BurstCount {
				bw.finalize(startTime)
				return nil
			}

			bw.executeBurst()
			burstCount++

			// Force GC occasionally to see allocation patterns
			if burstCount%10 == 0 {
				runtime.GC()
			}
		}
	}
}

// Stop stops the workload
func (bw *BurstWorkload) Stop() {
	bw.mutex.Lock()
	defer bw.mutex.Unlock()

	if bw.running {
		bw.running = false
		close(bw.stopChan)
	}
}

// GetStats returns workload statistics
func (bw *BurstWorkload) GetStats() WorkloadStats {
	bw.mutex.RLock()
	defer bw.mutex.RUnlock()
	return bw.stats
}

// executeBurst executes a single burst of allocations
func (bw *BurstWorkload) executeBurst() {
	burstStart := time.Now()
	burstSize := bw.config.BurstSize

	// Add randomness if configured
	if bw.config.Randomness > 0 {
		variation := int(float64(burstSize) * bw.config.Randomness)
		burstSize += rand.Intn(2*variation) - variation
		if burstSize < 1 {
			burstSize = 1
		}
	}

	// Allocate objects in burst
	for i := 0; i < burstSize; i++ {
		size := bw.config.ObjectSize
		if bw.config.Randomness > 0 {
			variation := int(float64(size) * bw.config.Randomness)
			size += rand.Intn(2*variation) - variation
			if size < 1 {
				size = 1
			}
		}

		data := make([]byte, size)
		// Write some data to ensure allocation
		for j := 0; j < len(data); j += 64 {
			data[j] = byte(i % 256)
		}

		bw.data = append(bw.data, data)

		bw.mutex.Lock()
		bw.stats.TotalAllocations++
		bw.stats.TotalBytes += uint64(size)
		bw.mutex.Unlock()
	}

	// Calculate burst rate
	burstDuration := time.Since(burstStart)
	burstRate := float64(bw.stats.TotalBytes) / burstDuration.Seconds()

	bw.mutex.Lock()
	if burstRate > bw.stats.PeakRate {
		bw.stats.PeakRate = burstRate
	}
	if bw.stats.MinRate == 0 || burstRate < bw.stats.MinRate {
		bw.stats.MinRate = burstRate
	}
	bw.mutex.Unlock()

	// Occasionally clear old data to prevent unlimited growth
	if len(bw.data) > 10000 {
		bw.data = bw.data[len(bw.data)/2:]
		runtime.GC()
	}
}

// finalize calculates final statistics
func (bw *BurstWorkload) finalize(startTime time.Time) {
	bw.mutex.Lock()
	defer bw.mutex.Unlock()

	bw.running = false
	bw.stats.Duration = time.Since(startTime)
	if bw.stats.Duration.Seconds() > 0 {
		bw.stats.AverageRate = float64(bw.stats.TotalBytes) / bw.stats.Duration.Seconds()
	}
}

// SteadyConfig configures steady growth workload
type SteadyConfig struct {
	InitialRate float64 `json:"initial_rate"`
	GrowthRate  float64 `json:"growth_rate"`
	ObjectSize  int     `json:"object_size"`
	MaxRate     float64 `json:"max_rate"`
}

// SteadyWorkload generates steady allocation growth
type SteadyWorkload struct {
	config      *SteadyConfig
	stats       WorkloadStats
	running     bool
	stopChan    chan struct{}
	mutex       sync.RWMutex
	currentRate float64
	data        [][]byte
}

// NewSteadyWorkload creates a new steady growth workload
func NewSteadyWorkload(config *SteadyConfig) (*SteadyWorkload, error) {
	if config.InitialRate <= 0 {
		return nil, fmt.Errorf("initial rate must be positive")
	}
	if config.GrowthRate <= 0 {
		return nil, fmt.Errorf("growth rate must be positive")
	}

	return &SteadyWorkload{
		config:      config,
		stopChan:    make(chan struct{}),
		currentRate: config.InitialRate,
		data:        make([][]byte, 0),
	}, nil
}

// Run executes the steady growth workload
func (sw *SteadyWorkload) Run(ctx context.Context) error {
	sw.mutex.Lock()
	if sw.running {
		sw.mutex.Unlock()
		return fmt.Errorf("workload is already running")
	}
	sw.running = true
	sw.stats = WorkloadStats{}
	sw.mutex.Unlock()

	startTime := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			sw.finalize(startTime)
			return nil
		case <-sw.stopChan:
			sw.finalize(startTime)
			return nil
		case <-ticker.C:
			sw.allocateAtCurrentRate()
			sw.updateRate()
		}
	}
}

// Stop stops the workload
func (sw *SteadyWorkload) Stop() {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()

	if sw.running {
		sw.running = false
		close(sw.stopChan)
	}
}

// GetStats returns workload statistics
func (sw *SteadyWorkload) GetStats() WorkloadStats {
	sw.mutex.RLock()
	defer sw.mutex.RUnlock()
	return sw.stats
}

// allocateAtCurrentRate allocates memory at the current rate
func (sw *SteadyWorkload) allocateAtCurrentRate() {
	// Calculate how many objects to allocate in this interval
	interval := 100 * time.Millisecond
	objectsToAllocate := int(sw.currentRate * interval.Seconds())

	if objectsToAllocate < 1 {
		// Use probabilistic allocation for low rates
		if rand.Float64() < sw.currentRate*interval.Seconds() {
			objectsToAllocate = 1
		}
	}

	for i := 0; i < objectsToAllocate; i++ {
		data := make([]byte, sw.config.ObjectSize)
		// Write some data to ensure allocation
		for j := 0; j < len(data); j += 64 {
			data[j] = byte(i % 256)
		}

		sw.data = append(sw.data, data)

		sw.mutex.Lock()
		sw.stats.TotalAllocations++
		sw.stats.TotalBytes += uint64(sw.config.ObjectSize)
		sw.mutex.Unlock()
	}

	// Update rate statistics
	sw.mutex.Lock()
	if sw.currentRate > sw.stats.PeakRate {
		sw.stats.PeakRate = sw.currentRate
	}
	if sw.stats.MinRate == 0 || sw.currentRate < sw.stats.MinRate {
		sw.stats.MinRate = sw.currentRate
	}
	sw.mutex.Unlock()

	// Occasionally clear old data
	if len(sw.data) > 5000 {
		sw.data = sw.data[len(sw.data)/2:]
		runtime.GC()
	}
}

// updateRate updates the current allocation rate
func (sw *SteadyWorkload) updateRate() {
	sw.currentRate *= sw.config.GrowthRate
	if sw.config.MaxRate > 0 && sw.currentRate > sw.config.MaxRate {
		sw.currentRate = sw.config.MaxRate
	}
}

// finalize calculates final statistics
func (sw *SteadyWorkload) finalize(startTime time.Time) {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()

	sw.running = false
	sw.stats.Duration = time.Since(startTime)
	if sw.stats.Duration.Seconds() > 0 {
		sw.stats.AverageRate = float64(sw.stats.TotalBytes) / sw.stats.Duration.Seconds()
	}
}

// CyclicConfig configures cyclic workload
type CyclicConfig struct {
	Period     time.Duration `json:"period"`
	Amplitude  int           `json:"amplitude"`
	BaseRate   int           `json:"base_rate"`
	ObjectSize int           `json:"object_size"`
	Phase      float64       `json:"phase"`
}

// CyclicWorkload generates cyclic allocation patterns
type CyclicWorkload struct {
	config    *CyclicConfig
	stats     WorkloadStats
	running   bool
	stopChan  chan struct{}
	mutex     sync.RWMutex
	startTime time.Time
	data      [][]byte
}

// NewCyclicWorkload creates a new cyclic workload
func NewCyclicWorkload(config *CyclicConfig) (*CyclicWorkload, error) {
	if config.Period <= 0 {
		return nil, fmt.Errorf("period must be positive")
	}
	if config.Amplitude < 0 {
		return nil, fmt.Errorf("amplitude must be non-negative")
	}

	return &CyclicWorkload{
		config:   config,
		stopChan: make(chan struct{}),
		data:     make([][]byte, 0),
	}, nil
}

// Run executes the cyclic workload
func (cw *CyclicWorkload) Run(ctx context.Context) error {
	cw.mutex.Lock()
	if cw.running {
		cw.mutex.Unlock()
		return fmt.Errorf("workload is already running")
	}
	cw.running = true
	cw.stats = WorkloadStats{}
	cw.startTime = time.Now()
	cw.mutex.Unlock()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			cw.finalize()
			return nil
		case <-cw.stopChan:
			cw.finalize()
			return nil
		case <-ticker.C:
			cw.allocateAtCyclicRate()
		}
	}
}

// Stop stops the workload
func (cw *CyclicWorkload) Stop() {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	if cw.running {
		cw.running = false
		close(cw.stopChan)
	}
}

// GetStats returns workload statistics
func (cw *CyclicWorkload) GetStats() WorkloadStats {
	cw.mutex.RLock()
	defer cw.mutex.RUnlock()
	return cw.stats
}

// allocateAtCyclicRate allocates memory following a cyclic pattern
func (cw *CyclicWorkload) allocateAtCyclicRate() {
	elapsed := time.Since(cw.startTime)
	
	// Calculate current position in cycle
	cyclePosition := float64(elapsed) / float64(cw.config.Period)
	cyclePhase := 2 * math.Pi * (cyclePosition + cw.config.Phase)
	
	// Calculate current rate using sine wave
	cyclicComponent := math.Sin(cyclePhase)
	currentRate := float64(cw.config.BaseRate) + float64(cw.config.Amplitude)*cyclicComponent
	if currentRate < 0 {
		currentRate = 0
	}

	// Calculate objects to allocate in this interval
	interval := 50 * time.Millisecond
	objectsToAllocate := int(currentRate * interval.Seconds())

	if objectsToAllocate < 1 && currentRate > 0 {
		// Use probabilistic allocation for low rates
		if rand.Float64() < currentRate*interval.Seconds() {
			objectsToAllocate = 1
		}
	}

	for i := 0; i < objectsToAllocate; i++ {
		data := make([]byte, cw.config.ObjectSize)
		// Write some data to ensure allocation
		for j := 0; j < len(data); j += 64 {
			data[j] = byte(i % 256)
		}

		cw.data = append(cw.data, data)

		cw.mutex.Lock()
		cw.stats.TotalAllocations++
		cw.stats.TotalBytes += uint64(cw.config.ObjectSize)
		cw.mutex.Unlock()
	}

	// Update rate statistics
	cw.mutex.Lock()
	if currentRate > cw.stats.PeakRate {
		cw.stats.PeakRate = currentRate
	}
	if cw.stats.MinRate == 0 || currentRate < cw.stats.MinRate {
		cw.stats.MinRate = currentRate
	}
	cw.mutex.Unlock()

	// Occasionally clear old data
	if len(cw.data) > 3000 {
		cw.data = cw.data[len(cw.data)/2:]
		runtime.GC()
	}
}

// finalize calculates final statistics
func (cw *CyclicWorkload) finalize() {
	cw.mutex.Lock()
	defer cw.mutex.Unlock()

	cw.running = false
	cw.stats.Duration = time.Since(cw.startTime)
	if cw.stats.Duration.Seconds() > 0 {
		cw.stats.AverageRate = float64(cw.stats.TotalBytes) / cw.stats.Duration.Seconds()
	}
}

// MixedConfig configures mixed workload
type MixedConfig struct {
	Patterns []string  `json:"patterns"`
	Weights  []float64 `json:"weights"`
}

// MixedWorkload combines multiple allocation patterns
type MixedWorkload struct {
	config    *MixedConfig
	stats     WorkloadStats
	running   bool
	stopChan  chan struct{}
	mutex     sync.RWMutex
	workloads []Workload
}

// NewMixedWorkload creates a new mixed workload
func NewMixedWorkload(config *MixedConfig) (*MixedWorkload, error) {
	if len(config.Patterns) != len(config.Weights) {
		return nil, fmt.Errorf("patterns and weights must have same length")
	}

	mw := &MixedWorkload{
		config:   config,
		stopChan: make(chan struct{}),
	}

	// Create constituent workloads
	for i, pattern := range config.Patterns {
		weight := config.Weights[i]
		var workload Workload
		var err error

		switch pattern {
		case "burst":
			workload, err = NewBurstWorkload(&BurstConfig{
				BurstSize:     int(100 * weight),
				BurstInterval: time.Duration(float64(time.Second) / weight),
				ObjectSize:    512,
			})
		case "growth":
			workload, err = NewSteadyWorkload(&SteadyConfig{
				InitialRate: 10 * weight,
				GrowthRate:  1.01,
				ObjectSize:  256,
				MaxRate:     1000 * weight,
			})
		case "cyclic":
			workload, err = NewCyclicWorkload(&CyclicConfig{
				Period:     30 * time.Second,
				Amplitude:  int(50 * weight),
				BaseRate:   int(20 * weight),
				ObjectSize: 128,
			})
		default:
			return nil, fmt.Errorf("unknown pattern: %s", pattern)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create %s workload: %v", pattern, err)
		}

		mw.workloads = append(mw.workloads, workload)
	}

	return mw, nil
}

// Run executes the mixed workload
func (mw *MixedWorkload) Run(ctx context.Context) error {
	mw.mutex.Lock()
	if mw.running {
		mw.mutex.Unlock()
		return fmt.Errorf("workload is already running")
	}
	mw.running = true
	mw.stats = WorkloadStats{}
	mw.mutex.Unlock()

	// Start all constituent workloads
	var wg sync.WaitGroup
	for _, workload := range mw.workloads {
		wg.Add(1)
		go func(w Workload) {
			defer wg.Done()
			w.Run(ctx)
		}(workload)
	}

	// Wait for completion or cancellation
	select {
	case <-ctx.Done():
	case <-mw.stopChan:
	}

	// Stop all workloads
	for _, workload := range mw.workloads {
		workload.Stop()
	}

	wg.Wait()
	mw.aggregateStats()
	return nil
}

// Stop stops the workload
func (mw *MixedWorkload) Stop() {
	mw.mutex.Lock()
	defer mw.mutex.Unlock()

	if mw.running {
		mw.running = false
		for _, workload := range mw.workloads {
			workload.Stop()
		}
		close(mw.stopChan)
	}
}

// GetStats returns workload statistics
func (mw *MixedWorkload) GetStats() WorkloadStats {
	mw.mutex.RLock()
	defer mw.mutex.RUnlock()
	return mw.stats
}

// aggregateStats aggregates statistics from all constituent workloads
func (mw *MixedWorkload) aggregateStats() {
	mw.mutex.Lock()
	defer mw.mutex.Unlock()

	var totalAllocations, totalBytes uint64
	var maxDuration time.Duration
	var peakRate, minRate float64
	minRate = math.MaxFloat64

	for _, workload := range mw.workloads {
		stats := workload.GetStats()
		totalAllocations += stats.TotalAllocations
		totalBytes += stats.TotalBytes

		if stats.Duration > maxDuration {
			maxDuration = stats.Duration
		}
		if stats.PeakRate > peakRate {
			peakRate = stats.PeakRate
		}
		if stats.MinRate < minRate {
			minRate = stats.MinRate
		}
	}

	mw.stats.TotalAllocations = totalAllocations
	mw.stats.TotalBytes = totalBytes
	mw.stats.Duration = maxDuration
	mw.stats.PeakRate = peakRate
	mw.stats.MinRate = minRate

	if maxDuration.Seconds() > 0 {
		mw.stats.AverageRate = float64(totalBytes) / maxDuration.Seconds()
	}
}
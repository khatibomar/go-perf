package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
	"time"
)

// BenchmarkGCTuningStrategies benchmarks different GC tuning strategies
func BenchmarkGCTuningStrategies(b *testing.B) {
	tuner := NewGCTuner()
	workloadProfile := &WorkloadProfile{
		Name:               "Benchmark Workload",
		AllocationPattern:  "mixed",
		ObjectLifetime:     "short",
		ConcurrencyLevel:   "medium",
		MemoryPressure:     "medium",
		LatencySensitivity: "medium",
		ThroughputPriority: "high",
	}
	
	strategies := []struct {
		name     string
		strategy TuningStrategy
	}{
		{"Latency", StrategyLatency},
		{"Throughput", StrategyThroughput},
		{"Memory", StrategyMemory},
		{"Balanced", StrategyBalanced},
	}
	
	for _, strategy := range strategies {
		b.Run(strategy.name, func(b *testing.B) {
			// Get optimized configuration
			config, err := tuner.TuneForStrategy(strategy.strategy, workloadProfile)
			if err != nil {
				b.Fatalf("Failed to tune for strategy: %v", err)
			}
			
			// Store original configuration
			originalGOGC := debug.SetGCPercent(-1)
			originalGOMAXPROCS := runtime.GOMAXPROCS(0)
			defer func() {
				debug.SetGCPercent(originalGOGC)
				runtime.GOMAXPROCS(originalGOMAXPROCS)
			}()
			
			// Apply configuration
			err = tuner.ApplyConfig(config)
			if err != nil {
				b.Fatalf("Failed to apply config: %v", err)
			}
			
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Mixed allocation workload
				for j := 0; j < 100; j++ {
					// Small objects
					_ = make([]byte, 64+j%64)
					
					// Medium objects
					if j%5 == 0 {
						_ = make([]int, 256)
					}
					
					// Large objects
					if j%20 == 0 {
						_ = make([]byte, 8192)
					}
					
					// String allocations
					_ = fmt.Sprintf("test_%d_%d", i, j)
					
					// Map allocations
					if j%10 == 0 {
						m := make(map[int]string)
						m[j] = fmt.Sprintf("value_%d", j)
						_ = m
					}
				}
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportGCTuningMetrics(b, startStats, endStats, config)
		})
	}
}

// BenchmarkGOGCValues benchmarks different GOGC values
func BenchmarkGOGCValues(b *testing.B) {
	gogcValues := []int{25, 50, 75, 100, 150, 200, 300, 500}
	
	for _, gogc := range gogcValues {
		b.Run(fmt.Sprintf("GOGC_%d", gogc), func(b *testing.B) {
			// Store original GOGC
			originalGOGC := debug.SetGCPercent(-1)
			defer debug.SetGCPercent(originalGOGC)
			
			// Set new GOGC value
			debug.SetGCPercent(gogc)
			
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Allocation-heavy workload
				data := make([][]byte, 100)
				for j := range data {
					data[j] = make([]byte, 1024+j%1024)
				}
				
				// Process data to prevent optimization
				sum := 0
				for _, slice := range data {
					for _, b := range slice {
						sum += int(b)
					}
				}
				_ = sum
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportBasicGCMetrics(b, startStats, endStats, gogc)
		})
	}
}

// BenchmarkGOMAXPROCSValues benchmarks different GOMAXPROCS values
func BenchmarkGOMAXPROCSValues(b *testing.B) {
	maxProcs := []int{1, 2, 4, 8, runtime.NumCPU()}
	
	for _, procs := range maxProcs {
		if procs > runtime.NumCPU() {
			continue
		}
		
		b.Run(fmt.Sprintf("GOMAXPROCS_%d", procs), func(b *testing.B) {
			// Store original GOMAXPROCS
			originalGOMAXPROCS := runtime.GOMAXPROCS(0)
			defer runtime.GOMAXPROCS(originalGOMAXPROCS)
			
			// Set new GOMAXPROCS value
			runtime.GOMAXPROCS(procs)
			
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Concurrent allocation workload
				var wg sync.WaitGroup
				for w := 0; w < procs; w++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for j := 0; j < 50; j++ {
							_ = make([]byte, 1024)
							_ = make(map[int]string)
							_ = fmt.Sprintf("worker_%d_%d", w, j)
						}
					}()
				}
				wg.Wait()
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportBasicGCMetrics(b, startStats, endStats, procs)
		})
	}
}

// BenchmarkMemoryLimits benchmarks different memory limit settings
func BenchmarkMemoryLimits(b *testing.B) {
	memoryLimits := []int64{
		-1,                    // No limit
		64 * 1024 * 1024,     // 64MB
		128 * 1024 * 1024,    // 128MB
		256 * 1024 * 1024,    // 256MB
		512 * 1024 * 1024,    // 512MB
	}
	
	for _, limit := range memoryLimits {
		limitName := "NoLimit"
		if limit > 0 {
			limitName = fmt.Sprintf("%dMB", limit/(1024*1024))
		}
		
		b.Run(fmt.Sprintf("MemLimit_%s", limitName), func(b *testing.B) {
			// Store original memory limit
			originalLimit := debug.SetMemoryLimit(-1)
			defer debug.SetMemoryLimit(originalLimit)
			
			// Set new memory limit
			debug.SetMemoryLimit(limit)
			
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Memory-intensive workload
				data := make([][]byte, 50)
				for j := range data {
					size := 1024 * (j + 1)
					if limit > 0 && int64(size) > limit/100 {
						size = int(limit / 100) // Avoid exceeding limit
					}
					data[j] = make([]byte, size)
				}
				
				// Use data to prevent optimization
				for _, slice := range data {
					if len(slice) > 0 {
						slice[0] = byte(i % 256)
					}
				}
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportMemoryLimitMetrics(b, startStats, endStats, limit)
		})
	}
}

// BenchmarkEnvironmentConfigurations benchmarks different environment configurations
func BenchmarkEnvironmentConfigurations(b *testing.B) {
	manager := NewGCConfigManager()
	environments := manager.ListEnvironments()
	
	for _, envName := range environments {
		b.Run(fmt.Sprintf("Env_%s", envName), func(b *testing.B) {
			// Store original configuration
			originalGOGC := debug.SetGCPercent(-1)
			originalGOMAXPROCS := runtime.GOMAXPROCS(0)
			originalLimit := debug.SetMemoryLimit(-1)
			defer func() {
				debug.SetGCPercent(originalGOGC)
				runtime.GOMAXPROCS(originalGOMAXPROCS)
				debug.SetMemoryLimit(originalLimit)
			}()
			
			// Apply environment configuration
			err := manager.ApplyEnvironmentConfiguration(envName)
			if err != nil {
				b.Fatalf("Failed to apply environment config: %v", err)
			}
			
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Environment-specific workload
				switch envName {
				case "development":
					// Development workload with debugging
					for j := 0; j < 50; j++ {
						_ = make([]byte, 512)
						_ = fmt.Sprintf("debug_%d_%d", i, j)
					}
				case "production":
					// Production workload optimized for performance
					data := make([][]byte, 100)
					for j := range data {
						data[j] = make([]byte, 1024)
					}
					// Process data
					for _, slice := range data {
						for k := range slice {
							slice[k] = byte(k % 256)
						}
					}
				case "testing":
					// Testing workload with consistent behavior
					for j := 0; j < 25; j++ {
						_ = make([]int, 128)
						_ = make(map[int]string)
					}
				default:
					// Default workload
					_ = make([]byte, 1024)
				}
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportEnvironmentMetrics(b, startStats, endStats, envName)
		})
	}
}

// BenchmarkConcurrentGCTuning benchmarks GC tuning under concurrent load
func BenchmarkConcurrentGCTuning(b *testing.B) {
	tuner := NewGCTuner()
	workerCounts := []int{1, 2, 4, 8}
	strategies := []TuningStrategy{StrategyLatency, StrategyThroughput, StrategyBalanced}
	
	for _, workers := range workerCounts {
		for _, strategy := range strategies {
			strategyName := map[TuningStrategy]string{
				StrategyLatency:    "Latency",
				StrategyThroughput: "Throughput",
				StrategyBalanced:   "Balanced",
			}[strategy]
			
			b.Run(fmt.Sprintf("Workers_%d_%s", workers, strategyName), func(b *testing.B) {
				// Get and apply configuration
				config, err := tuner.TuneForStrategy(strategy, nil)
				if err != nil {
					b.Fatalf("Failed to tune for strategy: %v", err)
				}
				
				// Store original configuration
				originalGOGC := debug.SetGCPercent(-1)
				originalGOMAXPROCS := runtime.GOMAXPROCS(0)
				defer func() {
					debug.SetGCPercent(originalGOGC)
					runtime.GOMAXPROCS(originalGOMAXPROCS)
				}()
				
				err = tuner.ApplyConfig(config)
				if err != nil {
					b.Fatalf("Failed to apply config: %v", err)
				}
				
				var startStats, endStats runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&startStats)
				
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var wg sync.WaitGroup
					for w := 0; w < workers; w++ {
						wg.Add(1)
						go func(workerID int) {
							defer wg.Done()
							for j := 0; j < 100/workers; j++ {
								// Worker-specific allocations
								_ = make([]byte, 512+workerID*64)
								_ = fmt.Sprintf("worker_%d_%d_%d", workerID, i, j)
								
								if j%5 == 0 {
									m := make(map[int]string)
									m[j] = fmt.Sprintf("data_%d", j)
									_ = m
								}
							}
						}(w)
					}
					wg.Wait()
				}
				b.StopTimer()
				
				runtime.ReadMemStats(&endStats)
				reportConcurrentGCMetrics(b, startStats, endStats, workers, config)
			})
		}
	}
}

// BenchmarkRealWorldGCScenarios benchmarks realistic GC scenarios
func BenchmarkRealWorldGCScenarios(b *testing.B) {
	scenarios := []struct {
		name     string
		workload func(int)
		config   *GCConfig
	}{
		{
			name: "WebServer",
			workload: func(iteration int) {
				// Simulate HTTP request processing
				for i := 0; i < 10; i++ {
					// Request data
					request := make([]byte, 2048)
					headers := make(map[string]string, 10)
					for j := 0; j < 10; j++ {
						headers[fmt.Sprintf("header-%d", j)] = fmt.Sprintf("value-%d", j)
					}
					
					// Response generation
					response := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\n\r\n", len(request))
					_ = response
				}
			},
			config: &GCConfig{
				GOGC:          100,
				GOMAXPROCS:    runtime.NumCPU(),
				GCPercent:     100,
				GCCPUFraction: 0.25,
			},
		},
		{
			name: "DatabaseProcessing",
			workload: func(iteration int) {
				// Simulate database query processing
				rows := make([]map[string]interface{}, 100)
				for i := range rows {
					rows[i] = make(map[string]interface{})
					rows[i]["id"] = i
					rows[i]["name"] = fmt.Sprintf("record_%d", i)
					rows[i]["data"] = make([]byte, 256)
					rows[i]["timestamp"] = time.Now()
				}
				
				// Process results
				for _, row := range rows {
					_ = fmt.Sprintf("%v", row)
				}
			},
			config: &GCConfig{
				GOGC:          150,
				GOMAXPROCS:    runtime.NumCPU(),
				GCPercent:     150,
				GCCPUFraction: 0.3,
			},
		},
		{
			name: "JSONProcessing",
			workload: func(iteration int) {
				// Simulate JSON processing
				for i := 0; i < 20; i++ {
					data := make(map[string]interface{})
					data["id"] = i
					data["values"] = make([]int, 50)
					for j := range data["values"].([]int) {
						data["values"].([]int)[j] = j * i
					}
					data["metadata"] = map[string]string{
						"type":      "test",
						"version":   "1.0",
						"timestamp": time.Now().String(),
					}
					
					// Serialize (simulate)
					_ = fmt.Sprintf("%v", data)
				}
			},
			config: &GCConfig{
				GOGC:          75,
				GOMAXPROCS:    runtime.NumCPU(),
				GCPercent:     75,
				GCCPUFraction: 0.2,
			},
		},
		{
			name: "StreamProcessing",
			workload: func(iteration int) {
				// Simulate stream processing
				buffer := make([][]byte, 0, 100)
				for i := 0; i < 100; i++ {
					event := make([]byte, 128+i%128)
					for j := range event {
						event[j] = byte((i + j) % 256)
					}
					buffer = append(buffer, event)
					
					// Process in batches
					if len(buffer) >= 10 {
						// Process batch
						sum := 0
						for _, event := range buffer {
							for _, b := range event {
								sum += int(b)
							}
						}
						_ = sum
						buffer = buffer[:0] // Reset buffer
					}
				}
			},
			config: &GCConfig{
				GOGC:          200,
				GOMAXPROCS:    runtime.NumCPU(),
				GCPercent:     200,
				GCCPUFraction: 0.4,
			},
		},
	}
	
	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Store original configuration
			originalGOGC := debug.SetGCPercent(-1)
			originalGOMAXPROCS := runtime.GOMAXPROCS(0)
			defer func() {
				debug.SetGCPercent(originalGOGC)
				runtime.GOMAXPROCS(originalGOMAXPROCS)
			}()
			
			// Apply scenario configuration
			debug.SetGCPercent(scenario.config.GOGC)
			runtime.GOMAXPROCS(scenario.config.GOMAXPROCS)
			
			var startStats, endStats runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&startStats)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				scenario.workload(i)
			}
			b.StopTimer()
			
			runtime.ReadMemStats(&endStats)
			reportScenarioMetrics(b, startStats, endStats, scenario.name, scenario.config)
		})
	}
}

// Helper functions for reporting metrics

func reportGCTuningMetrics(b *testing.B, start, end runtime.MemStats, config *GCConfig) {
	gcCycles := end.NumGC - start.NumGC
	totalPause := end.PauseTotalNs - start.PauseTotalNs
	allocated := end.TotalAlloc - start.TotalAlloc
	heapSize := end.HeapSys
	heapInUse := end.HeapInuse
	
	b.ReportMetric(float64(gcCycles), "gc-cycles")
	b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
	b.ReportMetric(float64(allocated)/(1024*1024), "allocated-mb")
	b.ReportMetric(float64(heapSize)/(1024*1024), "heap-size-mb")
	b.ReportMetric(float64(heapInUse)/(1024*1024), "heap-inuse-mb")
	b.ReportMetric(float64(config.GOGC), "gogc")
	b.ReportMetric(float64(config.GOMAXPROCS), "gomaxprocs")
	b.ReportMetric(config.GCCPUFraction*100, "gc-cpu-pct")
	
	if gcCycles > 0 {
		b.ReportMetric(float64(totalPause)/float64(gcCycles)/1e6, "avg-pause-ms")
	}
}

func reportBasicGCMetrics(b *testing.B, start, end runtime.MemStats, value int) {
	gcCycles := end.NumGC - start.NumGC
	totalPause := end.PauseTotalNs - start.PauseTotalNs
	allocated := end.TotalAlloc - start.TotalAlloc
	
	b.ReportMetric(float64(gcCycles), "gc-cycles")
	b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
	b.ReportMetric(float64(allocated)/(1024*1024), "allocated-mb")
	b.ReportMetric(float64(value), "config-value")
	
	if gcCycles > 0 {
		b.ReportMetric(float64(totalPause)/float64(gcCycles)/1e6, "avg-pause-ms")
	}
}

func reportMemoryLimitMetrics(b *testing.B, start, end runtime.MemStats, limit int64) {
	gcCycles := end.NumGC - start.NumGC
	totalPause := end.PauseTotalNs - start.PauseTotalNs
	allocated := end.TotalAlloc - start.TotalAlloc
	heapSize := end.HeapSys
	
	b.ReportMetric(float64(gcCycles), "gc-cycles")
	b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
	b.ReportMetric(float64(allocated)/(1024*1024), "allocated-mb")
	b.ReportMetric(float64(heapSize)/(1024*1024), "heap-size-mb")
	
	if limit > 0 {
		b.ReportMetric(float64(limit)/(1024*1024), "limit-mb")
		b.ReportMetric(float64(heapSize)/float64(limit)*100, "heap-limit-pct")
	}
}

func reportEnvironmentMetrics(b *testing.B, start, end runtime.MemStats, envName string) {
	gcCycles := end.NumGC - start.NumGC
	totalPause := end.PauseTotalNs - start.PauseTotalNs
	allocated := end.TotalAlloc - start.TotalAlloc
	heapInUse := end.HeapInuse
	
	b.ReportMetric(float64(gcCycles), "gc-cycles")
	b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
	b.ReportMetric(float64(allocated)/(1024*1024), "allocated-mb")
	b.ReportMetric(float64(heapInUse)/(1024*1024), "heap-inuse-mb")
	
	if gcCycles > 0 {
		b.ReportMetric(float64(totalPause)/float64(gcCycles)/1e6, "avg-pause-ms")
	}
}

func reportConcurrentGCMetrics(b *testing.B, start, end runtime.MemStats, workers int, config *GCConfig) {
	gcCycles := end.NumGC - start.NumGC
	totalPause := end.PauseTotalNs - start.PauseTotalNs
	allocated := end.TotalAlloc - start.TotalAlloc
	
	b.ReportMetric(float64(gcCycles), "gc-cycles")
	b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
	b.ReportMetric(float64(allocated)/(1024*1024), "allocated-mb")
	b.ReportMetric(float64(workers), "workers")
	b.ReportMetric(float64(config.GOGC), "gogc")
	
	if gcCycles > 0 {
		b.ReportMetric(float64(totalPause)/float64(gcCycles)/1e6, "avg-pause-ms")
	}
}

func reportScenarioMetrics(b *testing.B, start, end runtime.MemStats, scenarioName string, config *GCConfig) {
	gcCycles := end.NumGC - start.NumGC
	totalPause := end.PauseTotalNs - start.PauseTotalNs
	allocated := end.TotalAlloc - start.TotalAlloc
	heapInUse := end.HeapInuse
	
	b.ReportMetric(float64(gcCycles), "gc-cycles")
	b.ReportMetric(float64(totalPause)/1e6, "gc-pause-ms")
	b.ReportMetric(float64(allocated)/(1024*1024), "allocated-mb")
	b.ReportMetric(float64(heapInUse)/(1024*1024), "heap-inuse-mb")
	b.ReportMetric(float64(config.GOGC), "gogc")
	b.ReportMetric(config.GCCPUFraction*100, "gc-cpu-pct")
	
	if gcCycles > 0 {
		b.ReportMetric(float64(totalPause)/float64(gcCycles)/1e6, "avg-pause-ms")
	}
}

// TestGCTuner tests the GC tuner functionality
func TestGCTuner(t *testing.T) {
	tuner := NewGCTuner()
	
	// Test configuration collection
	metrics := tuner.CollectMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be collected")
	}
	
	if metrics.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
	
	// Test strategy tuning
	workloadProfile := &WorkloadProfile{
		Name:               "Test Workload",
		AllocationPattern:  "mixed",
		ObjectLifetime:     "short",
		ConcurrencyLevel:   "medium",
		MemoryPressure:     "medium",
		LatencySensitivity: "high",
		ThroughputPriority: "medium",
	}
	
	config, err := tuner.TuneForStrategy(StrategyLatency, workloadProfile)
	if err != nil {
		t.Errorf("Failed to tune for latency strategy: %v", err)
	}
	
	if config.GOGC >= 100 {
		t.Error("Expected GOGC to be lowered for latency strategy")
	}
	
	// Test configuration application
	originalGOGC := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(originalGOGC)
	
	err = tuner.ApplyConfig(config)
	if err != nil {
		t.Errorf("Failed to apply config: %v", err)
	}
	
	// Test monitoring
	tuner.StartMonitoring(time.Millisecond * 100)
	time.Sleep(time.Millisecond * 250)
	tuner.StopMonitoring()
	
	// Test report generation
	report := tuner.GenerateReport()
	if len(report) == 0 {
		t.Error("Expected non-empty report")
	}
}

// TestGCConfigManager tests the configuration manager functionality
func TestGCConfigManager(t *testing.T) {
	manager := NewGCConfigManager()
	
	// Test environment listing
	envs := manager.ListEnvironments()
	if len(envs) == 0 {
		t.Error("Expected environments to be available")
	}
	
	// Test environment configuration
	err := manager.ApplyEnvironmentConfiguration("development")
	if err != nil {
		t.Errorf("Failed to apply development environment: %v", err)
	}
	
	// Test configuration validation
	validConfig := &GCConfig{
		GOGC:          100,
		GOMAXPROCS:    runtime.NumCPU(),
		GCPercent:     100,
		GCCPUFraction: 0.25,
	}
	
	valid, message := manager.ValidateConfiguration(validConfig)
	if !valid {
		t.Errorf("Valid configuration failed validation: %s", message)
	}
	
	invalidConfig := &GCConfig{
		GOGC:          5, // Too low
		GOMAXPROCS:    1000, // Too high
		GCCPUFraction: 0.9, // Too high
	}
	
	valid, _ = manager.ValidateConfiguration(invalidConfig)
	if valid {
		t.Error("Invalid configuration passed validation")
	}
	
	// Test scenario triggers
	mockMetrics := &GCMetrics{
		AveragePause:     15.0, // High pause time
		GCFrequency:      5.0,
		HeapUtilization:  90.0, // High heap utilization
		GCOverhead:       8.0,
		Throughput:       300.0, // Low throughput
	}
	
	triggeredScenarios := manager.CheckScenarioTriggers(mockMetrics)
	if len(triggeredScenarios) == 0 {
		t.Error("Expected scenarios to be triggered with high metrics")
	}
	
	// Test report generation
	report := manager.GenerateConfigurationReport()
	if len(report) == 0 {
		t.Error("Expected non-empty configuration report")
	}
}

// TestConcurrentGCTuning tests concurrent access to GC tuning
func TestConcurrentGCTuning(t *testing.T) {
	tuner := NewGCTuner()
	var wg sync.WaitGroup
	
	// Start monitoring
	tuner.StartMonitoring(time.Millisecond * 50)
	defer tuner.StopMonitoring()
	
	// Concurrent metric collection
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				metrics := tuner.CollectMetrics()
				if metrics == nil {
					t.Error("Failed to collect metrics concurrently")
				}
				time.Sleep(time.Millisecond * 10)
			}
		}()
	}
	
	wg.Wait()
}

// TestMemoryLeakDetection tests for memory leaks in GC tuning
func TestMemoryLeakDetection(t *testing.T) {
	var startStats, endStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startStats)
	
	// Create and use tuner extensively
	for i := 0; i < 1000; i++ {
		tuner := NewGCTuner()
		metrics := tuner.CollectMetrics()
		_ = metrics
		
		config, _ := tuner.TuneForStrategy(StrategyBalanced, nil)
		_ = config
	}
	
	// Force GC and check for leaks
	runtime.GC()
	runtime.GC() // Double GC to ensure cleanup
	runtime.ReadMemStats(&endStats)
	
	heapGrowth := endStats.HeapInuse - startStats.HeapInuse
	
	// Allow some growth but not excessive
	if heapGrowth > 1024*1024 { // More than 1MB growth is suspicious
		t.Errorf("Potential memory leak detected: heap grew by %d bytes", heapGrowth)
	}
}
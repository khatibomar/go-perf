package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// AlertLevel represents the severity of an alert
type AlertLevel int

const (
	AlertLevelInfo AlertLevel = iota
	AlertLevelWarning
	AlertLevelCritical
)

func (al AlertLevel) String() string {
	switch al {
	case AlertLevelInfo:
		return "INFO"
	case AlertLevelWarning:
		return "WARNING"
	case AlertLevelCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Alert represents a monitoring alert
type Alert struct {
	ID          string                 `json:"id"`
	Level       AlertLevel             `json:"level"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// SimpleAlertManager provides basic alert management
type SimpleAlertManager struct {
	alerts map[string]*Alert
	mu     sync.RWMutex
}

// NewSimpleAlertManager creates a new simple alert manager
func NewSimpleAlertManager() *SimpleAlertManager {
	return &SimpleAlertManager{
		alerts: make(map[string]*Alert),
	}
}

// TriggerAlert adds a new alert
func (sam *SimpleAlertManager) TriggerAlert(alert *Alert) {
	sam.mu.Lock()
	defer sam.mu.Unlock()
	sam.alerts[alert.ID] = alert
}

// GetAlerts returns all alerts
func (sam *SimpleAlertManager) GetAlerts() []*Alert {
	sam.mu.RLock()
	defer sam.mu.RUnlock()
	
	alerts := make([]*Alert, 0, len(sam.alerts))
	for _, alert := range sam.alerts {
		alerts = append(alerts, alert)
	}
	return alerts
}

// GetActiveAlerts returns unresolved alerts
func (sam *SimpleAlertManager) GetActiveAlerts() []*Alert {
	sam.mu.RLock()
	defer sam.mu.RUnlock()
	
	var activeAlerts []*Alert
	for _, alert := range sam.alerts {
		if !alert.Resolved {
			activeAlerts = append(activeAlerts, alert)
		}
	}
	return activeAlerts
}

// ResolveAlert marks an alert as resolved
func (sam *SimpleAlertManager) ResolveAlert(alertID string) {
	sam.mu.Lock()
	defer sam.mu.Unlock()
	
	if alert, exists := sam.alerts[alertID]; exists {
		alert.Resolved = true
		now := time.Now()
		alert.ResolvedAt = &now
	}
}

// Start starts the alert manager (placeholder)
func (sam *SimpleAlertManager) Start() {
	// Placeholder for alert manager startup logic
}

// Stop stops the alert manager (placeholder)
func (sam *SimpleAlertManager) Stop() {
	// Placeholder for alert manager shutdown logic
}

// ProductionMemoryMonitor provides comprehensive memory monitoring for production environments
type ProductionMemoryMonitor struct {
	detector     *LeakDetector
	config       *ProductionMonitorConfig
	metrics      *ProductionMetrics
	alertManager *SimpleAlertManager
	healthCheck  *HealthChecker
	server       *http.Server
	mu           sync.RWMutex
	isRunning    bool
	stopChan     chan struct{}
}

// ProductionMonitorConfig contains configuration for production monitoring
type ProductionMonitorConfig struct {
	// Monitoring intervals
	MemoryCheckInterval    time.Duration
	HealthCheckInterval    time.Duration
	MetricsRetentionPeriod time.Duration
	
	// Thresholds
	MemoryUsageThreshold     float64 // Percentage
	GoroutineCountThreshold  int
	GCFrequencyThreshold     float64 // GCs per second
	HeapSizeThreshold        uint64  // Bytes
	
	// Alerting
	EnableAlerting     bool
	AlertCooldown      time.Duration
	CriticalThreshold  float64
	
	// HTTP Server
	HTTPPort           int
	EnableHTTPServer   bool
	HTTPReadTimeout    time.Duration
	HTTPWriteTimeout   time.Duration
	
	// Logging
	LogLevel           string
	LogToFile          bool
	LogFilePath        string
	
	// Performance
	MaxConcurrentChecks int
	BufferSize         int
}

// ProductionMetrics holds production monitoring metrics
type ProductionMetrics struct {
	StartTime           time.Time
	TotalChecks         int64
	LeaksDetected       int64
	AlertsTriggered     int64
	LastCheckTime       time.Time
	LastLeakDetection   time.Time
	SystemMetrics       *SystemMetrics
	PerformanceMetrics  *PerformanceMetrics
	mu                  sync.RWMutex
}

// SystemMetrics contains system-level metrics
type SystemMetrics struct {
	MemoryUsage      MemoryUsageMetrics
	GoroutineCount   int
	GCStats          GCMetrics
	HeapStats        HeapMetrics
	SystemLoad       SystemLoadMetrics
	Timestamp        time.Time
}

// MemoryUsageMetrics contains detailed memory usage information
type MemoryUsageMetrics struct {
	Allocated     uint64  // Currently allocated bytes
	TotalAlloc    uint64  // Total allocated bytes (cumulative)
	Sys           uint64  // System memory obtained from OS
	Lookups       uint64  // Number of pointer lookups
	Mallocs       uint64  // Number of mallocs
	Frees         uint64  // Number of frees
	HeapAlloc     uint64  // Heap allocated bytes
	HeapSys       uint64  // Heap system bytes
	HeapIdle      uint64  // Heap idle bytes
	HeapInuse     uint64  // Heap in-use bytes
	HeapReleased  uint64  // Heap released bytes
	HeapObjects   uint64  // Number of heap objects
	StackInuse    uint64  // Stack in-use bytes
	StackSys      uint64  // Stack system bytes
	MSpanInuse    uint64  // MSpan in-use bytes
	MSpanSys      uint64  // MSpan system bytes
	MCacheInuse   uint64  // MCache in-use bytes
	MCacheSys     uint64  // MCache system bytes
	BuckHashSys   uint64  // Profiling bucket hash table bytes
	GCSys         uint64  // GC metadata bytes
	OtherSys      uint64  // Other system bytes
	UsagePercent  float64 // Memory usage percentage
}

// GCMetrics contains garbage collection metrics
type GCMetrics struct {
	NumGC        uint32        // Number of GC cycles
	NumForcedGC  uint32        // Number of forced GC cycles
	GCCPUFraction float64      // Fraction of CPU time used by GC
	PauseTotal   time.Duration // Total GC pause time
	PauseNs      []uint64      // Recent GC pause times
	PauseEnd     []uint64      // Recent GC pause end times
	LastGC       time.Time     // Time of last GC
	NextGC       uint64        // Target heap size for next GC
	BySize       []GCSizeClass // Per-size class allocation stats
}

// GCSizeClass contains per-size class allocation statistics
type GCSizeClass struct {
	Size    uint32
	Mallocs uint64
	Frees   uint64
}

// HeapMetrics contains heap-specific metrics
type HeapMetrics struct {
	Size        uint64
	Used        uint64
	Idle        uint64
	Released    uint64
	Objects     uint64
	Fragmentation float64
}

// SystemLoadMetrics contains system load information
type SystemLoadMetrics struct {
	CPUUsage    float64
	LoadAverage float64
	Processes   int
	Threads     int
}

// PerformanceMetrics contains monitoring performance metrics
type PerformanceMetrics struct {
	AverageCheckDuration time.Duration
	MaxCheckDuration     time.Duration
	MinCheckDuration     time.Duration
	TotalCheckTime       time.Duration
	ChecksPerSecond      float64
	MemoryOverhead       uint64
}

// HealthChecker monitors the health of the monitoring system itself
type HealthChecker struct {
	lastCheck     time.Time
	isHealthy     bool
	errorCount    int
	maxErrors     int
	checkInterval time.Duration
	mu            sync.RWMutex
}

// NewProductionMemoryMonitor creates a new production memory monitor
func NewProductionMemoryMonitor(config *ProductionMonitorConfig) *ProductionMemoryMonitor {
	if config == nil {
		config = DefaultProductionMonitorConfig()
	}
	
	detectorConfig := DefaultLeakDetectorConfig()
	detectorConfig.SampleInterval = config.MemoryCheckInterval
	
	return &ProductionMemoryMonitor{
		detector:     NewLeakDetector(detectorConfig),
		config:       config,
		metrics:      NewProductionMetrics(),
		alertManager: NewSimpleAlertManager(),
		healthCheck:  NewHealthChecker(config.HealthCheckInterval, 10),
		stopChan:     make(chan struct{}),
	}
}

// DefaultProductionMonitorConfig returns default production monitoring configuration
func DefaultProductionMonitorConfig() *ProductionMonitorConfig {
	return &ProductionMonitorConfig{
		MemoryCheckInterval:      30 * time.Second,
		HealthCheckInterval:      60 * time.Second,
		MetricsRetentionPeriod:   24 * time.Hour,
		MemoryUsageThreshold:     80.0,
		GoroutineCountThreshold:  10000,
		GCFrequencyThreshold:     10.0,
		HeapSizeThreshold:        1024 * 1024 * 1024, // 1GB
		EnableAlerting:           true,
		AlertCooldown:            5 * time.Minute,
		CriticalThreshold:        95.0,
		HTTPPort:                 8080,
		EnableHTTPServer:         true,
		HTTPReadTimeout:          10 * time.Second,
		HTTPWriteTimeout:         10 * time.Second,
		LogLevel:                 "INFO",
		LogToFile:                false,
		LogFilePath:              "/var/log/memory-monitor.log",
		MaxConcurrentChecks:      5,
		BufferSize:               1000,
	}
}

// NewProductionMetrics creates new production metrics
func NewProductionMetrics() *ProductionMetrics {
	return &ProductionMetrics{
		StartTime:          time.Now(),
		SystemMetrics:      &SystemMetrics{},
		PerformanceMetrics: &PerformanceMetrics{},
	}
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(interval time.Duration, maxErrors int) *HealthChecker {
	return &HealthChecker{
		isHealthy:     true,
		maxErrors:     maxErrors,
		checkInterval: interval,
	}
}

// Start begins production monitoring
func (pmm *ProductionMemoryMonitor) Start() error {
	pmm.mu.Lock()
	defer pmm.mu.Unlock()
	
	if pmm.isRunning {
		return fmt.Errorf("production monitor is already running")
	}
	
	// Start leak detector
	if err := pmm.detector.Start(); err != nil {
		return fmt.Errorf("failed to start leak detector: %v", err)
	}
	
	// Start alert manager
	if pmm.config.EnableAlerting {
		pmm.alertManager.Start()
	}
	
	// Start HTTP server
	if pmm.config.EnableHTTPServer {
		if err := pmm.startHTTPServer(); err != nil {
			return fmt.Errorf("failed to start HTTP server: %v", err)
		}
	}
	
	// Start monitoring routines
	go pmm.monitoringLoop()
	go pmm.healthCheckLoop()
	go pmm.metricsCleanupLoop()
	
	pmm.isRunning = true
	log.Printf("Production memory monitor started on port %d", pmm.config.HTTPPort)
	
	return nil
}

// Stop stops production monitoring
func (pmm *ProductionMemoryMonitor) Stop() error {
	pmm.mu.Lock()
	defer pmm.mu.Unlock()
	
	if !pmm.isRunning {
		return fmt.Errorf("production monitor is not running")
	}
	
	// Signal stop
	close(pmm.stopChan)
	
	// Stop components
	pmm.detector.Stop()
	pmm.alertManager.Stop()
	
	// Stop HTTP server
	if pmm.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		pmm.server.Shutdown(ctx)
	}
	
	pmm.isRunning = false
	log.Println("Production memory monitor stopped")
	
	return nil
}

// monitoringLoop runs the main monitoring loop
func (pmm *ProductionMemoryMonitor) monitoringLoop() {
	ticker := time.NewTicker(pmm.config.MemoryCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-pmm.stopChan:
			return
		case <-ticker.C:
			pmm.performMemoryCheck()
		}
	}
}

// performMemoryCheck performs a comprehensive memory check
func (pmm *ProductionMemoryMonitor) performMemoryCheck() {
	checkStart := time.Now()
	
	// Collect system metrics
	systemMetrics := pmm.collectSystemMetrics()
	
	// Update metrics
	pmm.updateMetrics(systemMetrics, time.Since(checkStart))
	
	// Check for leaks
	report := pmm.detector.GenerateReport()
	if len(report.Detections) > 0 {
		// Convert []LeakDetection to []*LeakDetection
		detectionPtrs := make([]*LeakDetection, len(report.Detections))
		for i := range report.Detections {
			detectionPtrs[i] = &report.Detections[i]
		}
		pmm.handleLeakDetections(detectionPtrs)
	}
	
	// Check thresholds and trigger alerts
	pmm.checkThresholds(systemMetrics)
}

// collectSystemMetrics collects comprehensive system metrics
func (pmm *ProductionMemoryMonitor) collectSystemMetrics() *SystemMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return &SystemMetrics{
		MemoryUsage: MemoryUsageMetrics{
			Allocated:     memStats.Alloc,
			TotalAlloc:    memStats.TotalAlloc,
			Sys:           memStats.Sys,
			Lookups:       memStats.Lookups,
			Mallocs:       memStats.Mallocs,
			Frees:         memStats.Frees,
			HeapAlloc:     memStats.HeapAlloc,
			HeapSys:       memStats.HeapSys,
			HeapIdle:      memStats.HeapIdle,
			HeapInuse:     memStats.HeapInuse,
			HeapReleased:  memStats.HeapReleased,
			HeapObjects:   memStats.HeapObjects,
			StackInuse:    memStats.StackInuse,
			StackSys:      memStats.StackSys,
			MSpanInuse:    memStats.MSpanInuse,
			MSpanSys:      memStats.MSpanSys,
			MCacheInuse:   memStats.MCacheInuse,
			MCacheSys:     memStats.MCacheSys,
			BuckHashSys:   memStats.BuckHashSys,
			GCSys:         memStats.GCSys,
			OtherSys:      memStats.OtherSys,
			UsagePercent:  float64(memStats.Alloc) / float64(memStats.Sys) * 100,
		},
		GoroutineCount: runtime.NumGoroutine(),
		GCStats: GCMetrics{
			NumGC:         memStats.NumGC,
			NumForcedGC:   memStats.NumForcedGC,
			GCCPUFraction: memStats.GCCPUFraction,
			PauseTotal:    time.Duration(memStats.PauseTotalNs),
			PauseNs:       memStats.PauseNs[:],
			PauseEnd:      memStats.PauseEnd[:],
			LastGC:        time.Unix(0, int64(memStats.LastGC)),
			NextGC:        memStats.NextGC,
		},
		HeapStats: HeapMetrics{
			Size:          memStats.HeapSys,
			Used:          memStats.HeapInuse,
			Idle:          memStats.HeapIdle,
			Released:      memStats.HeapReleased,
			Objects:       memStats.HeapObjects,
			Fragmentation: float64(memStats.HeapIdle) / float64(memStats.HeapSys) * 100,
		},
		Timestamp: time.Now(),
	}
}

// updateMetrics updates production metrics
func (pmm *ProductionMemoryMonitor) updateMetrics(systemMetrics *SystemMetrics, checkDuration time.Duration) {
	pmm.metrics.mu.Lock()
	defer pmm.metrics.mu.Unlock()
	
	pmm.metrics.TotalChecks++
	pmm.metrics.LastCheckTime = time.Now()
	pmm.metrics.SystemMetrics = systemMetrics
	
	// Update performance metrics
	perf := pmm.metrics.PerformanceMetrics
	perf.TotalCheckTime += checkDuration
	
	if perf.MaxCheckDuration == 0 || checkDuration > perf.MaxCheckDuration {
		perf.MaxCheckDuration = checkDuration
	}
	
	if perf.MinCheckDuration == 0 || checkDuration < perf.MinCheckDuration {
		perf.MinCheckDuration = checkDuration
	}
	
	perf.AverageCheckDuration = perf.TotalCheckTime / time.Duration(pmm.metrics.TotalChecks)
	
	if pmm.metrics.TotalChecks > 0 {
		elapsedTime := time.Since(pmm.metrics.StartTime)
		perf.ChecksPerSecond = float64(pmm.metrics.TotalChecks) / elapsedTime.Seconds()
	}
}

// handleLeakDetections handles detected memory leaks
func (pmm *ProductionMemoryMonitor) handleLeakDetections(detections []*LeakDetection) {
	pmm.metrics.mu.Lock()
	pmm.metrics.LeaksDetected += int64(len(detections))
	pmm.metrics.LastLeakDetection = time.Now()
	pmm.metrics.mu.Unlock()
	
	for _, detection := range detections {
		log.Printf("Memory leak detected: Type=%s, Severity=%s, Description=%s",
			detection.Type, detection.Severity, detection.Description)
		
		// Trigger alert if enabled
		if pmm.config.EnableAlerting {
			alert := &Alert{
				ID:          fmt.Sprintf("leak_%d", time.Now().UnixNano()),
				Level:       AlertLevelCritical,
				Title:       fmt.Sprintf("Memory Leak Detected: %s", detection.Type),
				Description: detection.Description,
				Timestamp:   time.Now(),
				Metadata: map[string]interface{}{
					"leak_type": detection.Type,
					"severity":  detection.Severity,
					"metrics":   detection.Metrics,
				},
			}
			pmm.alertManager.TriggerAlert(alert)
		}
	}
}

// checkThresholds checks system metrics against configured thresholds
func (pmm *ProductionMemoryMonitor) checkThresholds(metrics *SystemMetrics) {
	// Check memory usage threshold
	if metrics.MemoryUsage.UsagePercent > pmm.config.MemoryUsageThreshold {
		pmm.triggerThresholdAlert("Memory Usage", 
			fmt.Sprintf("Memory usage %.2f%% exceeds threshold %.2f%%",
				metrics.MemoryUsage.UsagePercent, pmm.config.MemoryUsageThreshold))
	}
	
	// Check goroutine count threshold
	if metrics.GoroutineCount > pmm.config.GoroutineCountThreshold {
		pmm.triggerThresholdAlert("Goroutine Count",
			fmt.Sprintf("Goroutine count %d exceeds threshold %d",
				metrics.GoroutineCount, pmm.config.GoroutineCountThreshold))
	}
	
	// Check heap size threshold
	if metrics.MemoryUsage.HeapSys > pmm.config.HeapSizeThreshold {
		pmm.triggerThresholdAlert("Heap Size",
			fmt.Sprintf("Heap size %d bytes exceeds threshold %d bytes",
				metrics.MemoryUsage.HeapSys, pmm.config.HeapSizeThreshold))
	}
	
	// Check GC frequency
	if metrics.GCStats.GCCPUFraction > pmm.config.GCFrequencyThreshold/100 {
		pmm.triggerThresholdAlert("GC Pressure",
			fmt.Sprintf("GC CPU fraction %.4f exceeds threshold %.4f",
				metrics.GCStats.GCCPUFraction, pmm.config.GCFrequencyThreshold/100))
	}
}

// triggerThresholdAlert triggers an alert for threshold violations
func (pmm *ProductionMemoryMonitor) triggerThresholdAlert(alertType, message string) {
	if !pmm.config.EnableAlerting {
		return
	}
	
	alert := &Alert{
		ID:          fmt.Sprintf("threshold_%s_%d", alertType, time.Now().UnixNano()),
		Level:       AlertLevelWarning,
		Title:       fmt.Sprintf("Threshold Violation: %s", alertType),
		Description: message,
		Timestamp:   time.Now(),
		Metadata: map[string]interface{}{
			"alert_type": alertType,
			"threshold":  true,
		},
	}
	
	pmm.alertManager.TriggerAlert(alert)
	pmm.metrics.mu.Lock()
	pmm.metrics.AlertsTriggered++
	pmm.metrics.mu.Unlock()
}

// healthCheckLoop runs health checks for the monitoring system
func (pmm *ProductionMemoryMonitor) healthCheckLoop() {
	ticker := time.NewTicker(pmm.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-pmm.stopChan:
			return
		case <-ticker.C:
			pmm.performHealthCheck()
		}
	}
}

// performHealthCheck performs a health check of the monitoring system
func (pmm *ProductionMemoryMonitor) performHealthCheck() {
	pmm.healthCheck.mu.Lock()
	defer pmm.healthCheck.mu.Unlock()
	
	pmm.healthCheck.lastCheck = time.Now()
	
	// Check if detector is running
	if !pmm.detector.IsRunning() {
		pmm.healthCheck.errorCount++
		log.Printf("Health check failed: leak detector is not running")
	} else {
		pmm.healthCheck.errorCount = 0
	}
	
	// Update health status
	pmm.healthCheck.isHealthy = pmm.healthCheck.errorCount < pmm.healthCheck.maxErrors
	
	if !pmm.healthCheck.isHealthy {
		log.Printf("Production monitor is unhealthy: %d consecutive errors", pmm.healthCheck.errorCount)
	}
}

// metricsCleanupLoop periodically cleans up old metrics
func (pmm *ProductionMemoryMonitor) metricsCleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-pmm.stopChan:
			return
		case <-ticker.C:
			// Cleanup old metrics based on retention period
			// This is a placeholder for actual cleanup logic
			log.Println("Performing metrics cleanup")
		}
	}
}

// startHTTPServer starts the HTTP monitoring server
func (pmm *ProductionMemoryMonitor) startHTTPServer() error {
	mux := http.NewServeMux()
	
	// Register endpoints
	mux.HandleFunc("/health", pmm.handleHealth)
	mux.HandleFunc("/metrics", pmm.handleMetrics)
	mux.HandleFunc("/leaks", pmm.handleLeaks)
	mux.HandleFunc("/alerts", pmm.handleAlerts)
	mux.HandleFunc("/system", pmm.handleSystemMetrics)
	mux.HandleFunc("/", pmm.handleDashboard)
	
	pmm.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", pmm.config.HTTPPort),
		Handler:      mux,
		ReadTimeout:  pmm.config.HTTPReadTimeout,
		WriteTimeout: pmm.config.HTTPWriteTimeout,
	}
	
	go func() {
		if err := pmm.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()
	
	return nil
}

// HTTP handlers

// handleHealth handles health check requests
func (pmm *ProductionMemoryMonitor) handleHealth(w http.ResponseWriter, r *http.Request) {
	pmm.healthCheck.mu.RLock()
	isHealthy := pmm.healthCheck.isHealthy
	lastCheck := pmm.healthCheck.lastCheck
	errorCount := pmm.healthCheck.errorCount
	pmm.healthCheck.mu.RUnlock()
	
	status := "healthy"
	statusCode := http.StatusOK
	
	if !isHealthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}
	
	response := map[string]interface{}{
		"status":      status,
		"last_check":  lastCheck,
		"error_count": errorCount,
		"uptime":      time.Since(pmm.metrics.StartTime).String(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// handleMetrics handles metrics requests
func (pmm *ProductionMemoryMonitor) handleMetrics(w http.ResponseWriter, r *http.Request) {
	pmm.metrics.mu.RLock()
	metrics := *pmm.metrics
	pmm.metrics.mu.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// handleLeaks handles leak detection requests
func (pmm *ProductionMemoryMonitor) handleLeaks(w http.ResponseWriter, r *http.Request) {
	report := pmm.detector.GenerateReport()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// handleAlerts handles alert requests
func (pmm *ProductionMemoryMonitor) handleAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := pmm.alertManager.GetActiveAlerts()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// handleSystemMetrics handles system metrics requests
func (pmm *ProductionMemoryMonitor) handleSystemMetrics(w http.ResponseWriter, r *http.Request) {
	systemMetrics := pmm.collectSystemMetrics()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(systemMetrics)
}

// handleDashboard handles dashboard requests
func (pmm *ProductionMemoryMonitor) handleDashboard(w http.ResponseWriter, r *http.Request) {
	dashboardHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Production Memory Monitor</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .metric { margin: 10px 0; padding: 10px; border: 1px solid #ddd; }
        .healthy { background-color: #d4edda; }
        .warning { background-color: #fff3cd; }
        .critical { background-color: #f8d7da; }
    </style>
</head>
<body>
    <h1>Production Memory Monitor Dashboard</h1>
    <div id="status">Loading...</div>
    <div id="metrics">Loading...</div>
    <script>
        function updateDashboard() {
            fetch('/health')
                .then(response => response.json())
                .then(data => {
                    document.getElementById('status').innerHTML = 
                        '<h2>Health Status: ' + data.status + '</h2>' +
                        '<p>Uptime: ' + data.uptime + '</p>' +
                        '<p>Last Check: ' + data.last_check + '</p>';
                });
            
            fetch('/metrics')
                .then(response => response.json())
                .then(data => {
                    document.getElementById('metrics').innerHTML = 
                        '<h2>Metrics</h2>' +
                        '<p>Total Checks: ' + data.TotalChecks + '</p>' +
                        '<p>Leaks Detected: ' + data.LeaksDetected + '</p>' +
                        '<p>Alerts Triggered: ' + data.AlertsTriggered + '</p>';
                });
        }
        
        updateDashboard();
        setInterval(updateDashboard, 30000); // Update every 30 seconds
    </script>
</body>
</html>`
	
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(dashboardHTML))
}

// GetMetrics returns current production metrics
func (pmm *ProductionMemoryMonitor) GetMetrics() *ProductionMetrics {
	pmm.metrics.mu.RLock()
	defer pmm.metrics.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	metricsCopy := *pmm.metrics
	return &metricsCopy
}

// IsHealthy returns the health status of the monitor
func (pmm *ProductionMemoryMonitor) IsHealthy() bool {
	pmm.healthCheck.mu.RLock()
	defer pmm.healthCheck.mu.RUnlock()
	
	return pmm.healthCheck.isHealthy
}

// ProductionMonitorExample demonstrates how to use the production memory monitor
func ProductionMonitorExample() {
	// This is an example function, not meant to be called from main
	config := DefaultProductionMonitorConfig()
	config.HTTPPort = 8081
	config.MemoryCheckInterval = 10 * time.Second
	
	monitor := NewProductionMemoryMonitor(config)
	
	// Start monitoring
	if err := monitor.Start(); err != nil {
		log.Fatalf("Failed to start production monitor: %v", err)
	}
	
	// Let it run for a while
	time.Sleep(2 * time.Minute)
	
	// Check metrics
	metrics := monitor.GetMetrics()
	log.Printf("Total checks: %d", metrics.TotalChecks)
	log.Printf("Leaks detected: %d", metrics.LeaksDetected)
	log.Printf("Is healthy: %v", monitor.IsHealthy())
	
	// Stop monitoring
	if err := monitor.Stop(); err != nil {
		log.Printf("Error stopping monitor: %v", err)
	}
}
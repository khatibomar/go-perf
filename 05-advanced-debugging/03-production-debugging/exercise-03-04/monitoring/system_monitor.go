// monitoring/system_monitor.go
package monitoring

import (
	"context"
	"runtime"
	"sync"
	"time"

	"exercise-03-04/incident"
)

type SystemMonitor struct {
	config        Config
	metrics       *SystemMetrics
	mu            sync.RWMutex
	alertCallback func(incident.Alert)
	lastAlert     time.Time
}

type Config struct {
	SampleInterval  time.Duration
	MetricsRetention time.Duration
	AlertThresholds AlertThresholds
}

type AlertThresholds struct {
	CPUUsage       float64
	MemoryUsage    float64
	DiskUsage      float64
	ResponseTime   time.Duration
	ErrorRate      float64
	GoroutineCount int
}

type SystemMetrics struct {
	Timestamp      time.Time              `json:"timestamp"`
	CPUUsage       float64                `json:"cpu_usage"`
	MemoryUsage    float64                `json:"memory_usage"`
	DiskUsage      float64                `json:"disk_usage"`
	GoroutineCount int                    `json:"goroutine_count"`
	HeapAlloc      uint64                 `json:"heap_alloc"`
	HeapSys        uint64                 `json:"heap_sys"`
	NumGC          uint32                 `json:"num_gc"`
	ResponseTime   time.Duration          `json:"response_time"`
	ErrorRate      float64                `json:"error_rate"`
	CustomMetrics  map[string]interface{} `json:"custom_metrics"`
}

type EmergencyProfile struct {
	Timestamp    time.Time              `json:"timestamp"`
	CPUProfile   string                 `json:"cpu_profile"`
	MemProfile   string                 `json:"mem_profile"`
	Goroutines   int                    `json:"goroutines"`
	StackTraces  []string               `json:"stack_traces"`
	Metrics      map[string]interface{} `json:"metrics"`
}

type EmergencyAnalysis struct {
	Timestamp     time.Time              `json:"timestamp"`
	Bottlenecks   []string               `json:"bottlenecks"`
	Recommendations []string             `json:"recommendations"`
	Severity      string                 `json:"severity"`
	Metrics       map[string]interface{} `json:"metrics"`
	RootCause     string                 `json:"root_cause"`
}

func NewSystemMonitor(config Config) *SystemMonitor {
	sm := &SystemMonitor{
		config: config,
		metrics: &SystemMetrics{
			CustomMetrics: make(map[string]interface{}),
		},
	}
	// Collect initial metrics
	sm.collectMetrics()
	return sm
}

func (sm *SystemMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(sm.config.SampleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.collectMetrics()
			sm.checkAlerts()
		}
	}
}

func (sm *SystemMonitor) collectMetrics() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	sm.metrics = &SystemMetrics{
		Timestamp:      time.Now(),
		CPUUsage:       sm.getCPUUsage(),
		MemoryUsage:    float64(m.Alloc) / float64(m.Sys) * 100,
		DiskUsage:      sm.getDiskUsage(),
		GoroutineCount: runtime.NumGoroutine(),
		HeapAlloc:      m.Alloc,
		HeapSys:        m.HeapSys,
		NumGC:          m.NumGC,
		ResponseTime:   sm.getAverageResponseTime(),
		ErrorRate:      sm.getErrorRate(),
		CustomMetrics:  make(map[string]interface{}),
	}
}

func (sm *SystemMonitor) getCPUUsage() float64 {
	// Simplified CPU usage calculation
	// In a real implementation, this would use system calls or libraries
	return float64(runtime.NumGoroutine()) / 100.0 * 10.0
}

func (sm *SystemMonitor) getDiskUsage() float64 {
	// Simplified disk usage calculation
	// In a real implementation, this would check actual disk usage
	return 45.0
}

func (sm *SystemMonitor) getAverageResponseTime() time.Duration {
	// Simplified response time calculation
	return time.Millisecond * 150
}

func (sm *SystemMonitor) getErrorRate() float64 {
	// Simplified error rate calculation
	return 0.5
}

func (sm *SystemMonitor) checkAlerts() {
	if sm.alertCallback == nil {
		return
	}

	sm.mu.RLock()
	metrics := sm.metrics
	sm.mu.RUnlock()

	// Prevent alert spam
	if time.Since(sm.lastAlert) < time.Minute {
		return
	}

	// Check CPU usage
	if metrics.CPUUsage > sm.config.AlertThresholds.CPUUsage {
		sm.sendAlert("cpu_high", "High CPU usage detected", incident.SeverityHigh, map[string]interface{}{
			"cpu_usage": metrics.CPUUsage,
			"threshold": sm.config.AlertThresholds.CPUUsage,
		})
	}

	// Check memory usage
	if metrics.MemoryUsage > sm.config.AlertThresholds.MemoryUsage {
		sm.sendAlert("memory_high", "High memory usage detected", incident.SeverityHigh, map[string]interface{}{
			"memory_usage": metrics.MemoryUsage,
			"threshold":    sm.config.AlertThresholds.MemoryUsage,
		})
	}

	// Check goroutine count
	if metrics.GoroutineCount > sm.config.AlertThresholds.GoroutineCount {
		sm.sendAlert("goroutine_leak", "High goroutine count detected", incident.SeverityMedium, map[string]interface{}{
			"goroutine_count": metrics.GoroutineCount,
			"threshold":       sm.config.AlertThresholds.GoroutineCount,
		})
	}

	// Check response time
	if metrics.ResponseTime > sm.config.AlertThresholds.ResponseTime {
		sm.sendAlert("response_slow", "Slow response time detected", incident.SeverityMedium, map[string]interface{}{
			"response_time": metrics.ResponseTime,
			"threshold":     sm.config.AlertThresholds.ResponseTime,
		})
	}

	// Check error rate
	if metrics.ErrorRate > sm.config.AlertThresholds.ErrorRate {
		sm.sendAlert("error_rate_high", "High error rate detected", incident.SeverityHigh, map[string]interface{}{
			"error_rate": metrics.ErrorRate,
			"threshold":  sm.config.AlertThresholds.ErrorRate,
		})
	}
}

func (sm *SystemMonitor) sendAlert(alertType, message string, severity incident.Severity, metrics map[string]interface{}) {
	alert := incident.Alert{
		Type:      alertType,
		Message:   message,
		Severity:  severity,
		Timestamp: time.Now(),
		Metrics:   metrics,
		Source:    "system_monitor",
	}

	sm.alertCallback(alert)
	sm.lastAlert = time.Now()
}

func (sm *SystemMonitor) RegisterAlertCallback(callback func(incident.Alert)) {
	sm.alertCallback = callback
}

func (sm *SystemMonitor) GetCurrentMetrics() *SystemMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.metrics == nil {
		return &SystemMetrics{
			Timestamp:     time.Now(),
			CustomMetrics: make(map[string]interface{}),
		}
	}

	// Return a copy to prevent race conditions
	metricsCopy := *sm.metrics
	metricsCopy.CustomMetrics = make(map[string]interface{})
	for k, v := range sm.metrics.CustomMetrics {
		metricsCopy.CustomMetrics[k] = v
	}

	return &metricsCopy
}

func (sm *SystemMonitor) EmergencyProfile() (*EmergencyProfile, error) {
	// Collect emergency profiling data
	profile := &EmergencyProfile{
		Timestamp:   time.Now(),
		CPUProfile:  "cpu_profile_data_placeholder",
		MemProfile:  "mem_profile_data_placeholder",
		Goroutines:  runtime.NumGoroutine(),
		StackTraces: sm.getStackTraces(),
		Metrics:     make(map[string]interface{}),
	}

	// Add current metrics
	currentMetrics := sm.GetCurrentMetrics()
	profile.Metrics["cpu_usage"] = currentMetrics.CPUUsage
	profile.Metrics["memory_usage"] = currentMetrics.MemoryUsage
	profile.Metrics["goroutine_count"] = currentMetrics.GoroutineCount
	profile.Metrics["heap_alloc"] = currentMetrics.HeapAlloc

	return profile, nil
}

func (sm *SystemMonitor) EmergencyAnalyze() (*EmergencyAnalysis, error) {
	currentMetrics := sm.GetCurrentMetrics()

	analysis := &EmergencyAnalysis{
		Timestamp:       time.Now(),
		Bottlenecks:     []string{},
		Recommendations: []string{},
		Severity:        "low",
		Metrics:         make(map[string]interface{}),
		RootCause:       "No critical issues detected",
	}

	// Analyze current metrics
	if currentMetrics.CPUUsage > sm.config.AlertThresholds.CPUUsage {
		analysis.Bottlenecks = append(analysis.Bottlenecks, "High CPU usage")
		analysis.Recommendations = append(analysis.Recommendations, "Investigate CPU-intensive operations")
		analysis.Severity = "high"
		analysis.RootCause = "CPU bottleneck detected"
	}

	if currentMetrics.MemoryUsage > sm.config.AlertThresholds.MemoryUsage {
		analysis.Bottlenecks = append(analysis.Bottlenecks, "High memory usage")
		analysis.Recommendations = append(analysis.Recommendations, "Check for memory leaks")
		if analysis.Severity == "low" {
			analysis.Severity = "medium"
			analysis.RootCause = "Memory pressure detected"
		}
	}

	if currentMetrics.GoroutineCount > sm.config.AlertThresholds.GoroutineCount {
		analysis.Bottlenecks = append(analysis.Bottlenecks, "High goroutine count")
		analysis.Recommendations = append(analysis.Recommendations, "Check for goroutine leaks")
		if analysis.Severity == "low" {
			analysis.Severity = "medium"
			analysis.RootCause = "Potential goroutine leak"
		}
	}

	if currentMetrics.ResponseTime > sm.config.AlertThresholds.ResponseTime {
		analysis.Bottlenecks = append(analysis.Bottlenecks, "Slow response times")
		analysis.Recommendations = append(analysis.Recommendations, "Optimize request handling")
		if analysis.Severity == "low" {
			analysis.Severity = "medium"
			analysis.RootCause = "Performance degradation detected"
		}
	}

	// Add metrics to analysis
	analysis.Metrics["cpu_usage"] = currentMetrics.CPUUsage
	analysis.Metrics["memory_usage"] = currentMetrics.MemoryUsage
	analysis.Metrics["goroutine_count"] = currentMetrics.GoroutineCount
	analysis.Metrics["response_time"] = currentMetrics.ResponseTime
	analysis.Metrics["error_rate"] = currentMetrics.ErrorRate

	return analysis, nil
}

func (sm *SystemMonitor) getStackTraces() []string {
	// Get stack traces of all goroutines
	buf := make([]byte, 1024*1024)
	n := runtime.Stack(buf, true)
	stackTrace := string(buf[:n])

	// Split into individual goroutine traces
	// This is a simplified implementation
	return []string{stackTrace}
}

func (sm *SystemMonitor) Shutdown() error {
	// Cleanup resources if needed
	return nil
}
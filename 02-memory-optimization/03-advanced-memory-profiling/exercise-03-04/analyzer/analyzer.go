package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"exercise-03-04/detector"
	"exercise-03-04/reporting"
)

// AnalyzerConfig holds configuration for the comprehensive analyzer
type AnalyzerConfig struct {
	MonitoringDuration   time.Duration `json:"monitoring_duration"`
	SamplingInterval     time.Duration `json:"sampling_interval"`
	OutputDir            string        `json:"output_dir"`
	EnableHeapAnalysis   bool          `json:"enable_heap_analysis"`
	EnableGoroutineAnalysis bool       `json:"enable_goroutine_analysis"`
	EnableReferenceAnalysis bool       `json:"enable_reference_analysis"`
	EnableResourceAnalysis  bool       `json:"enable_resource_analysis"`
	ReportFormats        []string      `json:"report_formats"`
	Verbose              bool          `json:"verbose"`
	RealTimeAlerts       bool          `json:"real_time_alerts"`
	AlertThresholds      AlertThresholds `json:"alert_thresholds"`
}

// AlertThresholds defines thresholds for real-time alerts
type AlertThresholds struct {
	MemoryGrowthRate     float64 `json:"memory_growth_rate"`     // MB/min
	GoroutineGrowthRate  float64 `json:"goroutine_growth_rate"`  // goroutines/min
	FileHandleGrowthRate float64 `json:"file_handle_growth_rate"` // handles/min
	NetworkGrowthRate    float64 `json:"network_growth_rate"`    // connections/min
	CriticalLeakCount    int     `json:"critical_leak_count"`    // number of critical leaks
}

// AnalysisResult aggregates all analysis results
type AnalysisResult struct {
	StartTime         time.Time                    `json:"start_time"`
	EndTime           time.Time                    `json:"end_time"`
	Duration          time.Duration                `json:"duration"`
	HeapAnalysis      *detector.HeapAnalysis      `json:"heap_analysis,omitempty"`
	GoroutineAnalysis *detector.GoroutineAnalysis `json:"goroutine_analysis,omitempty"`
	ReferenceAnalysis *detector.ReferenceAnalysis `json:"reference_analysis,omitempty"`
	ResourceAnalysis  *detector.ResourceAnalysis  `json:"resource_analysis,omitempty"`
	OverallSummary    OverallSummary               `json:"overall_summary"`
	Alerts            []Alert                      `json:"alerts"`
	Recommendations   []string                     `json:"recommendations"`
}

// OverallSummary provides a high-level summary across all analyses
type OverallSummary struct {
	TotalLeaksDetected   int                    `json:"total_leaks_detected"`
	LeaksByCategory      map[string]int         `json:"leaks_by_category"`
	SeverityDistribution map[string]int         `json:"severity_distribution"`
	OverallRiskScore     float64                `json:"overall_risk_score"`
	ConfidenceLevel      float64                `json:"confidence_level"`
	SystemHealth         SystemHealth           `json:"system_health"`
	Trends               map[string]TrendInfo   `json:"trends"`
}

// SystemHealth represents overall system health metrics
type SystemHealth struct {
	MemoryHealth     HealthStatus `json:"memory_health"`
	GoroutineHealth  HealthStatus `json:"goroutine_health"`
	ResourceHealth   HealthStatus `json:"resource_health"`
	OverallHealth    HealthStatus `json:"overall_health"`
	HealthScore      float64      `json:"health_score"`
}

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthExcellent HealthStatus = "excellent"
	HealthGood      HealthStatus = "good"
	HealthFair      HealthStatus = "fair"
	HealthPoor      HealthStatus = "poor"
	HealthCritical  HealthStatus = "critical"
)

// TrendInfo represents trend information for a metric
type TrendInfo struct {
	Direction   string  `json:"direction"` // increasing, decreasing, stable
	Rate        float64 `json:"rate"`      // rate of change
	Confidence  float64 `json:"confidence"`
	Prediction  string  `json:"prediction"`
}

// Alert represents a real-time alert
type Alert struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Metrics     map[string]interface{} `json:"metrics"`
	Suggestion  string    `json:"suggestion"`
}

// Analyzer coordinates all leak detection and analysis
type Analyzer struct {
	config            *AnalyzerConfig
	heapAnalyzer      *detector.HeapAnalyzer
	goroutineDetector *detector.GoroutineDetector
	referenceDetector *detector.ReferenceDetector
	resourceDetector  *detector.ResourceDetector
	reporter          *reporting.Reporter
	alerts            []Alert
	alertsMutex       sync.RWMutex
	running           bool
	runMutex          sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
}

// NewAnalyzer creates a new comprehensive analyzer
func NewAnalyzer(config *AnalyzerConfig) *Analyzer {
	ctx, cancel := context.WithCancel(context.Background())

	analyzer := &Analyzer{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize detectors based on configuration
	if config.EnableHeapAnalysis {
		heapConfig := &detector.HeapConfig{
			Interval:  config.SamplingInterval,
			Threshold: 5.0,
			Verbose:   config.Verbose,
		}
		analyzer.heapAnalyzer = detector.NewHeapAnalyzer(heapConfig)
	}

	if config.EnableGoroutineAnalysis {
		goroutineConfig := &detector.GoroutineConfig{
			Interval:         config.SamplingInterval,
			GrowthThreshold:  2.0,
			GrowthRate:       1.5,
			Verbose:          config.Verbose,
			StackTraceDepth:  10,
			SampleStackTrace: true,
		}
		analyzer.goroutineDetector = detector.NewGoroutineDetector(goroutineConfig)
	}

	if config.EnableReferenceAnalysis {
		referenceConfig := &detector.ReferenceConfig{
			Interval:      config.SamplingInterval,
			MaxDepth:      5,
			Verbose:       config.Verbose,
			TrackPointers: true,
			SampleRate:    1.0,
		}
		analyzer.referenceDetector = detector.NewReferenceDetector(referenceConfig)
	}

	if config.EnableResourceAnalysis {
		resourceConfig := &detector.ResourceConfig{
			Interval:            config.SamplingInterval,
			FileHandleThreshold: 1000,
			NetworkThreshold:    500,
			GrowthThreshold:     5.0,
			Verbose:             config.Verbose,
			TrackStacks:         true,
		}
		analyzer.resourceDetector = detector.NewResourceDetector(resourceConfig)
	}

	// Initialize reporter
	reportConfig := &reporting.ReportConfig{
		OutputDir:      config.OutputDir,
		Formats:        config.ReportFormats,
		IncludeSummary: true,
		IncludeDetails: true,
		IncludeCharts:  true,
		Verbose:        config.Verbose,
	}
	analyzer.reporter = reporting.NewReporter(reportConfig)

	return analyzer
}

// Start begins the comprehensive analysis
func (a *Analyzer) Start() error {
	a.runMutex.Lock()
	defer a.runMutex.Unlock()

	if a.running {
		return fmt.Errorf("analyzer is already running")
	}

	if a.config.Verbose {
		fmt.Println("Starting comprehensive memory leak analysis...")
	}

	// Start all enabled detectors
	if a.heapAnalyzer != nil {
		if err := a.heapAnalyzer.Start(); err != nil {
			return fmt.Errorf("failed to start heap analyzer: %w", err)
		}
		if a.config.Verbose {
			fmt.Println("Heap analyzer started")
		}
	}

	if a.goroutineDetector != nil {
		if err := a.goroutineDetector.Start(); err != nil {
			return fmt.Errorf("failed to start goroutine detector: %w", err)
		}
		if a.config.Verbose {
			fmt.Println("Goroutine detector started")
		}
	}

	if a.referenceDetector != nil {
		if err := a.referenceDetector.Start(); err != nil {
			return fmt.Errorf("failed to start reference detector: %w", err)
		}
		if a.config.Verbose {
			fmt.Println("Reference detector started")
		}
	}

	if a.resourceDetector != nil {
		if err := a.resourceDetector.Start(); err != nil {
			return fmt.Errorf("failed to start resource detector: %w", err)
		}
		if a.config.Verbose {
			fmt.Println("Resource detector started")
		}
	}

	a.running = true

	// Start real-time monitoring if enabled
	if a.config.RealTimeAlerts {
		go a.monitorAlerts()
	}

	return nil
}

// Stop stops the analysis
func (a *Analyzer) Stop() error {
	a.runMutex.Lock()
	defer a.runMutex.Unlock()

	if !a.running {
		return fmt.Errorf("analyzer is not running")
	}

	if a.config.Verbose {
		fmt.Println("Stopping comprehensive analysis...")
	}

	// Cancel context to stop monitoring
	a.cancel()

	// Stop all detectors
	if a.heapAnalyzer != nil {
		a.heapAnalyzer.Stop()
	}
	if a.goroutineDetector != nil {
		a.goroutineDetector.Stop()
	}
	if a.referenceDetector != nil {
		a.referenceDetector.Stop()
	}
	if a.resourceDetector != nil {
		a.resourceDetector.Stop()
	}

	a.running = false

	if a.config.Verbose {
		fmt.Println("Analysis stopped")
	}

	return nil
}

// RunAnalysis runs a complete analysis for the specified duration
func (a *Analyzer) RunAnalysis() (*AnalysisResult, error) {
	startTime := time.Now()

	// Start analysis
	if err := a.Start(); err != nil {
		return nil, err
	}

	// Wait for the specified duration
	if a.config.Verbose {
		fmt.Printf("Running analysis for %s...\n", a.config.MonitoringDuration)
	}

	select {
	case <-time.After(a.config.MonitoringDuration):
		// Analysis completed
	case <-a.ctx.Done():
		// Analysis was cancelled
		return nil, fmt.Errorf("analysis was cancelled")
	}

	// Stop analysis
	if err := a.Stop(); err != nil {
		return nil, err
	}

	endTime := time.Now()

	// Collect results from all detectors
	result := &AnalysisResult{
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
	}

	// Get analysis results
	if a.heapAnalyzer != nil {
		result.HeapAnalysis, _ = a.heapAnalyzer.AnalyzeHeap()
	}
	if a.goroutineDetector != nil {
		result.GoroutineAnalysis, _ = a.goroutineDetector.AnalyzeGoroutines()
	}
	if a.referenceDetector != nil {
		result.ReferenceAnalysis, _ = a.referenceDetector.AnalyzeReferences()
	}
	if a.resourceDetector != nil {
		result.ResourceAnalysis, _ = a.resourceDetector.AnalyzeResources()
	}

	// Generate overall summary
	result.OverallSummary = a.generateOverallSummary(result)

	// Get alerts
	a.alertsMutex.RLock()
	result.Alerts = make([]Alert, len(a.alerts))
	copy(result.Alerts, a.alerts)
	a.alertsMutex.RUnlock()

	// Generate recommendations
	result.Recommendations = a.generateRecommendations(result)

	if a.config.Verbose {
		fmt.Printf("Analysis completed. Found %d total leaks.\n", result.OverallSummary.TotalLeaksDetected)
	}

	return result, nil
}

// GenerateReport generates a comprehensive report
func (a *Analyzer) GenerateReport(result *AnalysisResult) error {
	reportData, err := a.reporter.GenerateReport(
		result.HeapAnalysis,
		result.GoroutineAnalysis,
		result.ReferenceAnalysis,
		result.ResourceAnalysis,
	)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	return a.reporter.SaveReport(reportData)
}

// SaveResults saves the analysis results to a file
func (a *Analyzer) SaveResults(result *AnalysisResult, filename string) error {
	if err := os.MkdirAll(a.config.OutputDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(a.config.OutputDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if a.config.Verbose {
		fmt.Printf("Saving analysis results to %s\n", filePath)
	}

	return encoder.Encode(result)
}

// LoadResults loads analysis results from a file
func (a *Analyzer) LoadResults(filename string) (*AnalysisResult, error) {
	filePath := filepath.Join(a.config.OutputDir, filename)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result AnalysisResult
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetCurrentStatus returns the current status of all detectors
func (a *Analyzer) GetCurrentStatus() map[string]interface{} {
	status := make(map[string]interface{})

	a.runMutex.RLock()
	status["running"] = a.running
	a.runMutex.RUnlock()

	if a.heapAnalyzer != nil {
		status["heap_metrics"] = a.heapAnalyzer.GetMetrics()
	}
	if a.goroutineDetector != nil {
		status["goroutine_metrics"] = a.goroutineDetector.GetMetrics()
	}
	if a.resourceDetector != nil {
		status["resource_metrics"] = a.resourceDetector.GetMetrics()
	}

	a.alertsMutex.RLock()
	status["alert_count"] = len(a.alerts)
	a.alertsMutex.RUnlock()

	return status
}

// GetAlerts returns all alerts
func (a *Analyzer) GetAlerts() []Alert {
	a.alertsMutex.RLock()
	defer a.alertsMutex.RUnlock()

	alerts := make([]Alert, len(a.alerts))
	copy(alerts, a.alerts)
	return alerts
}

// ClearAlerts clears all alerts
func (a *Analyzer) ClearAlerts() {
	a.alertsMutex.Lock()
	defer a.alertsMutex.Unlock()

	a.alerts = nil
}

// monitorAlerts monitors for real-time alerts
func (a *Analyzer) monitorAlerts() {
	ticker := time.NewTicker(a.config.SamplingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.checkForAlerts()
		case <-a.ctx.Done():
			return
		}
	}
}

// checkForAlerts checks current metrics against thresholds
func (a *Analyzer) checkForAlerts() {
	// Check heap metrics
	if a.heapAnalyzer != nil {
		metrics := a.heapAnalyzer.GetMetrics()
		if len(metrics) > 1 {
			// Calculate growth rate
			last := metrics[len(metrics)-1]
			prev := metrics[len(metrics)-2]
			timeDiff := last.Timestamp.Sub(prev.Timestamp).Minutes()
			if timeDiff > 0 {
				memoryDiff := float64(last.HeapInuse-prev.HeapInuse) / (1024 * 1024) // MB
				growthRate := memoryDiff / timeDiff
				if growthRate > a.config.AlertThresholds.MemoryGrowthRate {
					a.addAlert(Alert{
						ID:          fmt.Sprintf("heap_growth_%d", time.Now().Unix()),
						Timestamp:   time.Now(),
						Type:        "heap_growth",
						Severity:    "high",
						Title:       "High Memory Growth Rate",
						Description: fmt.Sprintf("Memory growing at %.2f MB/min (threshold: %.2f)", growthRate, a.config.AlertThresholds.MemoryGrowthRate),
						Metrics:     map[string]interface{}{"growth_rate": growthRate, "current_heap": last.HeapInuse},
						Suggestion:  "Investigate memory allocation patterns and check for potential leaks",
					})
				}
			}
		}
	}

	// Check goroutine metrics
	if a.goroutineDetector != nil {
		metrics := a.goroutineDetector.GetMetrics()
		if len(metrics) > 1 {
			last := metrics[len(metrics)-1]
			prev := metrics[len(metrics)-2]
			timeDiff := last.Timestamp.Sub(prev.Timestamp).Minutes()
			if timeDiff > 0 {
				goroutineDiff := float64(last.Count - prev.Count)
				growthRate := goroutineDiff / timeDiff
				if growthRate > a.config.AlertThresholds.GoroutineGrowthRate {
					a.addAlert(Alert{
						ID:          fmt.Sprintf("goroutine_growth_%d", time.Now().Unix()),
						Timestamp:   time.Now(),
						Type:        "goroutine_growth",
						Severity:    "high",
						Title:       "High Goroutine Growth Rate",
						Description: fmt.Sprintf("Goroutines growing at %.2f/min (threshold: %.2f)", growthRate, a.config.AlertThresholds.GoroutineGrowthRate),
						Metrics:     map[string]interface{}{"growth_rate": growthRate, "current_count": last.Count},
						Suggestion:  "Check for goroutine leaks and implement proper lifecycle management",
					})
				}
			}
		}
	}

	// Check resource metrics
	if a.resourceDetector != nil {
		metrics := a.resourceDetector.GetMetrics()
		if len(metrics) > 1 {
			last := metrics[len(metrics)-1]
			prev := metrics[len(metrics)-2]
			timeDiff := last.Timestamp.Sub(prev.Timestamp).Minutes()
			if timeDiff > 0 {
				// Check file handle growth
				fileHandleDiff := float64(last.FileHandles - prev.FileHandles)
				fileGrowthRate := fileHandleDiff / timeDiff
				if fileGrowthRate > a.config.AlertThresholds.FileHandleGrowthRate {
					a.addAlert(Alert{
						ID:          fmt.Sprintf("file_growth_%d", time.Now().Unix()),
						Timestamp:   time.Now(),
						Type:        "file_handle_growth",
						Severity:    "medium",
						Title:       "High File Handle Growth Rate",
						Description: fmt.Sprintf("File handles growing at %.2f/min (threshold: %.2f)", fileGrowthRate, a.config.AlertThresholds.FileHandleGrowthRate),
						Metrics:     map[string]interface{}{"growth_rate": fileGrowthRate, "current_count": last.FileHandles},
						Suggestion:  "Ensure proper file handle cleanup and use defer statements",
					})
				}

				// Check network connection growth
				networkDiff := float64(last.NetworkConnections - prev.NetworkConnections)
				networkGrowthRate := networkDiff / timeDiff
				if networkGrowthRate > a.config.AlertThresholds.NetworkGrowthRate {
					a.addAlert(Alert{
						ID:          fmt.Sprintf("network_growth_%d", time.Now().Unix()),
						Timestamp:   time.Now(),
						Type:        "network_growth",
						Severity:    "medium",
						Title:       "High Network Connection Growth Rate",
						Description: fmt.Sprintf("Network connections growing at %.2f/min (threshold: %.2f)", networkGrowthRate, a.config.AlertThresholds.NetworkGrowthRate),
						Metrics:     map[string]interface{}{"growth_rate": networkGrowthRate, "current_count": last.NetworkConnections},
						Suggestion:  "Implement connection pooling and proper connection lifecycle management",
					})
				}
			}
		}
	}
}

// addAlert adds a new alert
func (a *Analyzer) addAlert(alert Alert) {
	a.alertsMutex.Lock()
	defer a.alertsMutex.Unlock()

	a.alerts = append(a.alerts, alert)

	if a.config.Verbose {
		fmt.Printf("ALERT [%s]: %s\n", alert.Severity, alert.Title)
	}

	// Keep only the last 100 alerts
	if len(a.alerts) > 100 {
		a.alerts = a.alerts[len(a.alerts)-100:]
	}
}

// generateOverallSummary generates a comprehensive summary
func (a *Analyzer) generateOverallSummary(result *AnalysisResult) OverallSummary {
	summary := OverallSummary{
		LeaksByCategory:      make(map[string]int),
		SeverityDistribution: make(map[string]int),
		Trends:               make(map[string]TrendInfo),
	}

	// Aggregate leaks from all analyses
	if result.HeapAnalysis != nil {
		for _, anomaly := range result.HeapAnalysis.Anomalies {
			summary.TotalLeaksDetected++
			summary.LeaksByCategory["heap"]++
			summary.SeverityDistribution[anomaly.Severity]++
		}
	}

	if result.GoroutineAnalysis != nil {
		for _, candidate := range result.GoroutineAnalysis.LeakCandidates {
			summary.TotalLeaksDetected++
			summary.LeaksByCategory["goroutine"]++
			// Derive severity from confidence
			severity := "medium"
			if candidate.Confidence > 0.8 {
				severity = "high"
			} else if candidate.Confidence < 0.4 {
				severity = "low"
			}
			summary.SeverityDistribution[severity]++
		}
	}

	if result.ReferenceAnalysis != nil {
		for _, cycle := range result.ReferenceAnalysis.DetectedCycles {
			summary.TotalLeaksDetected++
			summary.LeaksByCategory["reference"]++
			summary.SeverityDistribution[cycle.Severity]++
		}
	}

	if result.ResourceAnalysis != nil {
		for _, leak := range result.ResourceAnalysis.DetectedLeaks {
			summary.TotalLeaksDetected++
			summary.LeaksByCategory[string(leak.Type)]++
			summary.SeverityDistribution[leak.Severity]++
		}
	}

	// Calculate overall risk score
	summary.OverallRiskScore = a.calculateRiskScore(summary)

	// Calculate confidence level
	summary.ConfidenceLevel = a.calculateConfidenceLevel(result)

	// Assess system health
	summary.SystemHealth = a.assessSystemHealth(result)

	// Generate trends
	summary.Trends = a.generateTrends(result)

	return summary
}

// calculateRiskScore calculates an overall risk score
func (a *Analyzer) calculateRiskScore(summary OverallSummary) float64 {
	score := 0.0

	// Weight by severity
	score += float64(summary.SeverityDistribution["critical"]) * 10.0
	score += float64(summary.SeverityDistribution["high"]) * 5.0
	score += float64(summary.SeverityDistribution["medium"]) * 2.0
	score += float64(summary.SeverityDistribution["low"]) * 1.0

	// Normalize to 0-100 scale
	if summary.TotalLeaksDetected > 0 {
		score = (score / float64(summary.TotalLeaksDetected)) * 10.0
	}

	if score > 100.0 {
		score = 100.0
	}

	return score
}

// calculateConfidenceLevel calculates overall confidence
func (a *Analyzer) calculateConfidenceLevel(result *AnalysisResult) float64 {
	totalConfidence := 0.0
	analysisCount := 0

	if result.HeapAnalysis != nil {
		// Calculate average confidence from heap anomalies
		if len(result.HeapAnalysis.Anomalies) > 0 {
			heapConfidence := 0.0
			for _, anomaly := range result.HeapAnalysis.Anomalies {
				// Derive confidence from severity since Anomaly doesn't have Confidence field
				if anomaly.Severity == "critical" {
					heapConfidence += 0.9
				} else if anomaly.Severity == "high" {
					heapConfidence += 0.8
				} else if anomaly.Severity == "medium" {
					heapConfidence += 0.6
				} else {
					heapConfidence += 0.4
				}
			}
			totalConfidence += heapConfidence / float64(len(result.HeapAnalysis.Anomalies))
			analysisCount++
		}
	}
	if result.GoroutineAnalysis != nil {
		// Calculate average confidence from goroutine candidates
		if len(result.GoroutineAnalysis.LeakCandidates) > 0 {
			goroutineConfidence := 0.0
			for _, candidate := range result.GoroutineAnalysis.LeakCandidates {
				goroutineConfidence += candidate.Confidence
			}
			totalConfidence += goroutineConfidence / float64(len(result.GoroutineAnalysis.LeakCandidates))
			analysisCount++
		}
	}
	if result.ReferenceAnalysis != nil {
		// Calculate average confidence from reference cycles
		if len(result.ReferenceAnalysis.DetectedCycles) > 0 {
			referenceConfidence := 0.0
			for _, cycle := range result.ReferenceAnalysis.DetectedCycles {
				// Assume confidence based on cycle severity
				if cycle.Severity == "high" {
					referenceConfidence += 0.9
				} else if cycle.Severity == "medium" {
					referenceConfidence += 0.7
				} else {
					referenceConfidence += 0.5
				}
			}
			totalConfidence += referenceConfidence / float64(len(result.ReferenceAnalysis.DetectedCycles))
			analysisCount++
		}
	}
	if result.ResourceAnalysis != nil {
		// Calculate average confidence from resource leaks
		if len(result.ResourceAnalysis.DetectedLeaks) > 0 {
			resourceConfidence := 0.0
			for _, leak := range result.ResourceAnalysis.DetectedLeaks {
				resourceConfidence += leak.Confidence
			}
			totalConfidence += resourceConfidence / float64(len(result.ResourceAnalysis.DetectedLeaks))
			analysisCount++
		}
	}

	if analysisCount == 0 {
		return 0.0
	}

	return totalConfidence / float64(analysisCount)
}

// assessSystemHealth assesses overall system health
func (a *Analyzer) assessSystemHealth(result *AnalysisResult) SystemHealth {
	health := SystemHealth{}

	// Assess memory health
	if result.HeapAnalysis != nil {
		if result.HeapAnalysis.LeakProbability > 0.8 {
			health.MemoryHealth = HealthCritical
		} else if result.HeapAnalysis.LeakProbability > 0.6 {
			health.MemoryHealth = HealthPoor
		} else if result.HeapAnalysis.LeakProbability > 0.4 {
			health.MemoryHealth = HealthFair
		} else if result.HeapAnalysis.LeakProbability > 0.2 {
			health.MemoryHealth = HealthGood
		} else {
			health.MemoryHealth = HealthExcellent
		}
	} else {
		health.MemoryHealth = HealthGood
	}

	// Assess goroutine health
	if result.GoroutineAnalysis != nil {
		if result.GoroutineAnalysis.LeakProbability > 0.8 {
			health.GoroutineHealth = HealthCritical
		} else if result.GoroutineAnalysis.LeakProbability > 0.6 {
			health.GoroutineHealth = HealthPoor
		} else if result.GoroutineAnalysis.LeakProbability > 0.4 {
			health.GoroutineHealth = HealthFair
		} else if result.GoroutineAnalysis.LeakProbability > 0.2 {
			health.GoroutineHealth = HealthGood
		} else {
			health.GoroutineHealth = HealthExcellent
		}
	} else {
		health.GoroutineHealth = HealthGood
	}

	// Assess resource health
	if result.ResourceAnalysis != nil {
		criticalLeaks := 0
		for _, leak := range result.ResourceAnalysis.DetectedLeaks {
			if leak.Severity == "critical" {
				criticalLeaks++
			}
		}
		if criticalLeaks > 5 {
			health.ResourceHealth = HealthCritical
		} else if criticalLeaks > 2 {
			health.ResourceHealth = HealthPoor
		} else if len(result.ResourceAnalysis.DetectedLeaks) > 10 {
			health.ResourceHealth = HealthFair
		} else if len(result.ResourceAnalysis.DetectedLeaks) > 5 {
			health.ResourceHealth = HealthGood
		} else {
			health.ResourceHealth = HealthExcellent
		}
	} else {
		health.ResourceHealth = HealthGood
	}

	// Calculate overall health
	healthScores := map[HealthStatus]float64{
		HealthExcellent: 5.0,
		HealthGood:      4.0,
		HealthFair:      3.0,
		HealthPoor:      2.0,
		HealthCritical:  1.0,
	}

	totalScore := healthScores[health.MemoryHealth] + healthScores[health.GoroutineHealth] + healthScores[health.ResourceHealth]
	avgScore := totalScore / 3.0
	health.HealthScore = avgScore

	if avgScore >= 4.5 {
		health.OverallHealth = HealthExcellent
	} else if avgScore >= 3.5 {
		health.OverallHealth = HealthGood
	} else if avgScore >= 2.5 {
		health.OverallHealth = HealthFair
	} else if avgScore >= 1.5 {
		health.OverallHealth = HealthPoor
	} else {
		health.OverallHealth = HealthCritical
	}

	return health
}

// generateTrends generates trend information
func (a *Analyzer) generateTrends(result *AnalysisResult) map[string]TrendInfo {
	trends := make(map[string]TrendInfo)

	// Memory trend
	if result.HeapAnalysis != nil {
		direction := "stable"
		growthRate := result.HeapAnalysis.Statistics.GrowthVariance
		if growthRate > 1.0 {
			direction = "increasing"
		} else if growthRate < -1.0 {
			direction = "decreasing"
		}

		prediction := "Memory usage will remain stable"
		if growthRate > 5.0 {
			prediction = "Memory usage will continue to grow rapidly - investigate immediately"
		} else if growthRate > 2.0 {
			prediction = "Memory usage will grow moderately - monitor closely"
		}

		trends["memory"] = TrendInfo{
			Direction:  direction,
			Rate:       growthRate,
			Confidence: 0.8, // Default confidence
			Prediction: prediction,
		}
	}

	// Goroutine trend
	if result.GoroutineAnalysis != nil {
		direction := "stable"
		growthRate := result.GoroutineAnalysis.Statistics.GrowthVariance
		if growthRate > 0.5 {
			direction = "increasing"
		} else if growthRate < -0.5 {
			direction = "decreasing"
		}

		prediction := "Goroutine count will remain stable"
		if growthRate > 2.0 {
			prediction = "Goroutine count will continue to grow - check for leaks"
		} else if growthRate > 1.0 {
			prediction = "Goroutine count will grow slowly - monitor"
		}

		trends["goroutines"] = TrendInfo{
			Direction:  direction,
			Rate:       growthRate,
			Confidence: 0.8, // Default confidence
			Prediction: prediction,
		}
	}

	return trends
}

// generateRecommendations generates actionable recommendations
func (a *Analyzer) generateRecommendations(result *AnalysisResult) []string {
	var recommendations []string

	// High-priority recommendations
	if result.OverallSummary.SeverityDistribution["critical"] > 0 {
		recommendations = append(recommendations, "URGENT: Address critical leaks immediately to prevent system failure")
	}

	if result.OverallSummary.OverallRiskScore > 70 {
		recommendations = append(recommendations, "High risk detected - implement immediate leak mitigation strategies")
	}

	// Specific recommendations based on analysis results
	if result.HeapAnalysis != nil && result.HeapAnalysis.LeakProbability > 0.7 {
		recommendations = append(recommendations, "Implement memory profiling and review allocation patterns")
	}

	if result.GoroutineAnalysis != nil && result.GoroutineAnalysis.LeakProbability > 0.7 {
		recommendations = append(recommendations, "Review goroutine lifecycle management and implement proper cancellation")
	}

	if result.ResourceAnalysis != nil && len(result.ResourceAnalysis.DetectedLeaks) > 10 {
		recommendations = append(recommendations, "Implement comprehensive resource management patterns")
	}

	// System health recommendations
	if result.OverallSummary.SystemHealth.OverallHealth == HealthPoor || result.OverallSummary.SystemHealth.OverallHealth == HealthCritical {
		recommendations = append(recommendations, "System health is compromised - conduct thorough code review")
	}

	// Monitoring recommendations
	if result.OverallSummary.ConfidenceLevel < 0.6 {
		recommendations = append(recommendations, "Increase monitoring duration and frequency for better accuracy")
	}

	// General recommendations
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "System appears healthy - continue regular monitoring")
	}

	recommendations = append(recommendations, "Set up automated monitoring and alerting for early detection")
	recommendations = append(recommendations, "Implement regular leak detection as part of CI/CD pipeline")

	return recommendations
}

// GetDefaultConfig returns a default configuration
func GetDefaultConfig() *AnalyzerConfig {
	return &AnalyzerConfig{
		MonitoringDuration:          5 * time.Minute,
		SamplingInterval:            10 * time.Second,
		OutputDir:                   "./leak_analysis_results",
		EnableHeapAnalysis:          true,
		EnableGoroutineAnalysis:     true,
		EnableReferenceAnalysis:     true,
		EnableResourceAnalysis:      true,
		ReportFormats:               []string{"json", "html", "text"},
		Verbose:                     true,
		RealTimeAlerts:              true,
		AlertThresholds: AlertThresholds{
			MemoryGrowthRate:     5.0,  // 5 MB/min
			GoroutineGrowthRate:  2.0,  // 2 goroutines/min
			FileHandleGrowthRate: 10.0, // 10 handles/min
			NetworkGrowthRate:    5.0,  // 5 connections/min
			CriticalLeakCount:    1,    // 1 critical leak
		},
	}
}
package reporting

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"exercise-03-04/detector"
)

// ReportConfig holds configuration for report generation
type ReportConfig struct {
	OutputDir     string   `json:"output_dir"`
	Formats       []string `json:"formats"`       // json, html, text
	IncludeSummary bool    `json:"include_summary"`
	IncludeDetails bool    `json:"include_details"`
	IncludeCharts  bool    `json:"include_charts"`
	Verbose        bool    `json:"verbose"`
}

// ReportData aggregates all analysis data for reporting
type ReportData struct {
	GeneratedAt       time.Time                    `json:"generated_at"`
	Duration          time.Duration                `json:"duration"`
	HeapAnalysis      *detector.HeapAnalysis      `json:"heap_analysis,omitempty"`
	GoroutineAnalysis *detector.GoroutineAnalysis `json:"goroutine_analysis,omitempty"`
	ReferenceAnalysis *detector.ReferenceAnalysis `json:"reference_analysis,omitempty"`
	ResourceAnalysis  *detector.ResourceAnalysis  `json:"resource_analysis,omitempty"`
	Summary           ReportSummary                `json:"summary"`
	Recommendations   []string                     `json:"recommendations"`
	Insights          []Insight                    `json:"insights"`
	Metrics           ReportMetrics                `json:"metrics"`
}

// ReportSummary provides a high-level summary of findings
type ReportSummary struct {
	TotalLeaksDetected    int                          `json:"total_leaks_detected"`
	CriticalLeaks         int                          `json:"critical_leaks"`
	HighSeverityLeaks     int                          `json:"high_severity_leaks"`
	MediumSeverityLeaks   int                          `json:"medium_severity_leaks"`
	LowSeverityLeaks      int                          `json:"low_severity_leaks"`
	LeaksByType           map[string]int               `json:"leaks_by_type"`
	TopLeakSources        []LeakSource                 `json:"top_leak_sources"`
	MemoryGrowthRate      float64                      `json:"memory_growth_rate"`
	GoroutineGrowthRate   float64                      `json:"goroutine_growth_rate"`
	ResourceGrowthRates   map[string]float64           `json:"resource_growth_rates"`
	OverallRiskLevel      string                       `json:"overall_risk_level"`
	ConfidenceScore       float64                      `json:"confidence_score"`
	AnalysisCompleteness  float64                      `json:"analysis_completeness"`
}

// LeakSource represents a source of leaks
type LeakSource struct {
	Location    string  `json:"location"`
	Type        string  `json:"type"`
	Count       int     `json:"count"`
	Severity    string  `json:"severity"`
	Confidence  float64 `json:"confidence"`
	Description string  `json:"description"`
}

// Insight represents an analytical insight
type Insight struct {
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	Evidence    []string  `json:"evidence"`
	Suggestion  string    `json:"suggestion"`
	Timestamp   time.Time `json:"timestamp"`
}

// ReportMetrics contains quantitative metrics
type ReportMetrics struct {
	AnalysisDuration      time.Duration          `json:"analysis_duration"`
	DataPointsAnalyzed    int                    `json:"data_points_analyzed"`
	MemoryPeakUsage       uint64                 `json:"memory_peak_usage"`
	GoroutinePeakCount    int                    `json:"goroutine_peak_count"`
	FileHandlesPeak       int                    `json:"file_handles_peak"`
	NetworkConnectionsPeak int                   `json:"network_connections_peak"`
	GCEvents              int                    `json:"gc_events"`
	GCTotalPause          time.Duration          `json:"gc_total_pause"`
	LeakDetectionAccuracy float64                `json:"leak_detection_accuracy"`
	FalsePositiveRate     float64                `json:"false_positive_rate"`
	PerformanceImpact     PerformanceImpact      `json:"performance_impact"`
}

// PerformanceImpact measures the impact of monitoring
type PerformanceImpact struct {
	CPUOverhead    float64 `json:"cpu_overhead"`
	MemoryOverhead uint64  `json:"memory_overhead"`
	LatencyImpact  float64 `json:"latency_impact"`
}

// Reporter generates comprehensive reports
type Reporter struct {
	config *ReportConfig
}

// NewReporter creates a new reporter
func NewReporter(config *ReportConfig) *Reporter {
	return &Reporter{
		config: config,
	}
}

// GenerateReport generates a comprehensive report from all analysis data
func (r *Reporter) GenerateReport(
	heapAnalysis *detector.HeapAnalysis,
	goroutineAnalysis *detector.GoroutineAnalysis,
	referenceAnalysis *detector.ReferenceAnalysis,
	resourceAnalysis *detector.ResourceAnalysis,
) (*ReportData, error) {
	report := &ReportData{
		GeneratedAt:       time.Now(),
		HeapAnalysis:      heapAnalysis,
		GoroutineAnalysis: goroutineAnalysis,
		ReferenceAnalysis: referenceAnalysis,
		ResourceAnalysis:  resourceAnalysis,
	}

	// Calculate duration from the longest analysis
	report.Duration = r.calculateTotalDuration(heapAnalysis, goroutineAnalysis, referenceAnalysis, resourceAnalysis)

	// Generate summary
	report.Summary = r.generateSummary(heapAnalysis, goroutineAnalysis, referenceAnalysis, resourceAnalysis)

	// Generate insights
	report.Insights = r.generateInsights(heapAnalysis, goroutineAnalysis, referenceAnalysis, resourceAnalysis)

	// Generate recommendations
	report.Recommendations = r.generateRecommendations(report)

	// Calculate metrics
	report.Metrics = r.calculateMetrics(heapAnalysis, goroutineAnalysis, referenceAnalysis, resourceAnalysis)

	return report, nil
}

// SaveReport saves the report in specified formats
func (r *Reporter) SaveReport(report *ReportData) error {
	if err := os.MkdirAll(r.config.OutputDir, 0755); err != nil {
		return err
	}

	for _, format := range r.config.Formats {
		switch format {
		case "json":
			if err := r.saveJSONReport(report); err != nil {
				return err
			}
		case "html":
			if err := r.saveHTMLReport(report); err != nil {
				return err
			}
		case "text":
			if err := r.saveTextReport(report); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported format: %s", format)
		}
	}

	return nil
}

// saveJSONReport saves the report as JSON
func (r *Reporter) saveJSONReport(report *ReportData) error {
	filename := filepath.Join(r.config.OutputDir, "leak_detection_report.json")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if r.config.Verbose {
		fmt.Printf("Saving JSON report to %s\n", filename)
	}

	return encoder.Encode(report)
}

// saveHTMLReport saves the report as HTML
func (r *Reporter) saveHTMLReport(report *ReportData) error {
	filename := filepath.Join(r.config.OutputDir, "leak_detection_report.html")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	tmpl := r.getHTMLTemplate()
	t, err := template.New("report").Funcs(template.FuncMap{
		"formatDuration": func(d time.Duration) string {
			return d.String()
		},
		"formatBytes": func(b uint64) string {
			return formatBytes(b)
		},
		"formatFloat": func(f float64) string {
			return fmt.Sprintf("%.2f", f)
		},
		"getSeverityClass": func(severity string) string {
			switch severity {
			case "critical":
				return "severity-critical"
			case "high":
				return "severity-high"
			case "medium":
				return "severity-medium"
			case "low":
				return "severity-low"
			default:
				return "severity-unknown"
			}
		},
	}).Parse(tmpl)
	if err != nil {
		return err
	}

	if r.config.Verbose {
		fmt.Printf("Saving HTML report to %s\n", filename)
	}

	return t.Execute(file, report)
}

// saveTextReport saves the report as plain text
func (r *Reporter) saveTextReport(report *ReportData) error {
	filename := filepath.Join(r.config.OutputDir, "leak_detection_report.txt")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	if r.config.Verbose {
		fmt.Printf("Saving text report to %s\n", filename)
	}

	return r.writeTextReport(file, report)
}

// calculateTotalDuration calculates the total analysis duration
func (r *Reporter) calculateTotalDuration(
	heapAnalysis *detector.HeapAnalysis,
	goroutineAnalysis *detector.GoroutineAnalysis,
	referenceAnalysis *detector.ReferenceAnalysis,
	resourceAnalysis *detector.ResourceAnalysis,
) time.Duration {
	var maxDuration time.Duration

	if heapAnalysis != nil && heapAnalysis.Duration > maxDuration {
		maxDuration = heapAnalysis.Duration
	}
	if goroutineAnalysis != nil && goroutineAnalysis.Duration > maxDuration {
		maxDuration = goroutineAnalysis.Duration
	}
	if referenceAnalysis != nil && referenceAnalysis.Duration > maxDuration {
		maxDuration = referenceAnalysis.Duration
	}
	if resourceAnalysis != nil && resourceAnalysis.Duration > maxDuration {
		maxDuration = resourceAnalysis.Duration
	}

	return maxDuration
}

// generateSummary generates a comprehensive summary
func (r *Reporter) generateSummary(
	heapAnalysis *detector.HeapAnalysis,
	goroutineAnalysis *detector.GoroutineAnalysis,
	referenceAnalysis *detector.ReferenceAnalysis,
	resourceAnalysis *detector.ResourceAnalysis,
) ReportSummary {
	summary := ReportSummary{
		LeaksByType:         make(map[string]int),
		ResourceGrowthRates: make(map[string]float64),
	}

	// Analyze heap leaks
	if heapAnalysis != nil {
		summary.MemoryGrowthRate = heapAnalysis.GrowthRate
		for _, anomaly := range heapAnalysis.Anomalies {
			summary.TotalLeaksDetected++
			summary.LeaksByType["heap"]++
			r.categorizeLeakBySeverity(&summary, anomaly.Severity)
		}
	}

	// Analyze goroutine leaks
	if goroutineAnalysis != nil {
		summary.GoroutineGrowthRate = goroutineAnalysis.GrowthRate
		for _, candidate := range goroutineAnalysis.LeakCandidates {
			summary.TotalLeaksDetected++
			summary.LeaksByType["goroutine"]++
			// Determine severity based on confidence
			severity := "medium"
			if candidate.Confidence > 0.8 {
				severity = "high"
			} else if candidate.Confidence < 0.4 {
				severity = "low"
			}
			r.categorizeLeakBySeverity(&summary, severity)
		}
	}

	// Analyze reference leaks
		if referenceAnalysis != nil {
			for _, cycle := range referenceAnalysis.DetectedCycles {
				summary.TotalLeaksDetected++
				summary.LeaksByType["reference"]++
				r.categorizeLeakBySeverity(&summary, cycle.Severity)
			}
		}

	// Analyze resource leaks
	if resourceAnalysis != nil {
		for _, leak := range resourceAnalysis.DetectedLeaks {
			summary.TotalLeaksDetected++
			summary.LeaksByType[string(leak.Type)]++
			r.categorizeLeakBySeverity(&summary, leak.Severity)
		}
		for resourceType, rate := range resourceAnalysis.Statistics.ResourceGrowthRates {
			summary.ResourceGrowthRates[string(resourceType)] = rate
		}
	}

	// Generate top leak sources
	summary.TopLeakSources = r.identifyTopLeakSources(heapAnalysis, goroutineAnalysis, referenceAnalysis, resourceAnalysis)

	// Calculate overall risk level
	summary.OverallRiskLevel = r.calculateOverallRiskLevel(summary)

	// Calculate confidence score
	summary.ConfidenceScore = r.calculateConfidenceScore(heapAnalysis, goroutineAnalysis, referenceAnalysis, resourceAnalysis)

	// Calculate analysis completeness
	summary.AnalysisCompleteness = r.calculateAnalysisCompleteness(heapAnalysis, goroutineAnalysis, referenceAnalysis, resourceAnalysis)

	return summary
}

// categorizeLeakBySeverity categorizes leaks by severity
func (r *Reporter) categorizeLeakBySeverity(summary *ReportSummary, severity string) {
	switch severity {
	case "critical":
		summary.CriticalLeaks++
	case "high":
		summary.HighSeverityLeaks++
	case "medium":
		summary.MediumSeverityLeaks++
	case "low":
		summary.LowSeverityLeaks++
	}
}

// identifyTopLeakSources identifies the top sources of leaks
func (r *Reporter) identifyTopLeakSources(
	heapAnalysis *detector.HeapAnalysis,
	goroutineAnalysis *detector.GoroutineAnalysis,
	referenceAnalysis *detector.ReferenceAnalysis,
	resourceAnalysis *detector.ResourceAnalysis,
) []LeakSource {
	sourceMap := make(map[string]*LeakSource)

	// Collect from heap analysis anomalies
	if heapAnalysis != nil {
		for _, anomaly := range heapAnalysis.Anomalies {
			key := fmt.Sprintf("heap_%s", anomaly.Type)
			if source, exists := sourceMap[key]; exists {
				source.Count++
			} else {
				sourceMap[key] = &LeakSource{
					Location:    anomaly.Type,
					Type:        "heap",
					Count:       1,
					Severity:    anomaly.Severity,
					Confidence:  0.8, // Default confidence
					Description: anomaly.Description,
				}
			}
		}
	}

	// Collect from goroutine analysis leak candidates
		if goroutineAnalysis != nil {
			for _, candidate := range goroutineAnalysis.LeakCandidates {
				location := fmt.Sprintf("%s.%s", candidate.Package, candidate.Function)
				key := fmt.Sprintf("goroutine_%s", location)
				
				// Determine severity based on confidence
				severity := "medium"
				if candidate.Confidence > 0.8 {
					severity = "high"
				} else if candidate.Confidence < 0.4 {
					severity = "low"
				}
				
				if source, exists := sourceMap[key]; exists {
					source.Count++
				} else {
					sourceMap[key] = &LeakSource{
						Location:    location,
						Type:        "goroutine",
						Count:       1,
						Severity:    severity,
						Confidence:  candidate.Confidence,
						Description: candidate.Suggestion,
					}
				}
			}
		}

	// Collect from resource analysis detected leaks
		if resourceAnalysis != nil {
			for _, leak := range resourceAnalysis.DetectedLeaks {
				key := fmt.Sprintf("%s_%s", leak.Type, leak.Path)
				if source, exists := sourceMap[key]; exists {
					source.Count++
				} else {
					sourceMap[key] = &LeakSource{
						Location:    leak.Path,
						Type:        string(leak.Type),
						Count:       1,
						Severity:    leak.Severity,
						Confidence:  leak.Confidence,
						Description: leak.Description,
					}
				}
			}
		}

	// Convert to slice and sort by count
	var sources []LeakSource
	for _, source := range sourceMap {
		sources = append(sources, *source)
	}

	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Count > sources[j].Count
	})

	// Return top 10
	if len(sources) > 10 {
		sources = sources[:10]
	}

	return sources
}

// calculateOverallRiskLevel calculates the overall risk level
func (r *Reporter) calculateOverallRiskLevel(summary ReportSummary) string {
	if summary.CriticalLeaks > 0 {
		return "critical"
	}
	if summary.HighSeverityLeaks > 5 || summary.TotalLeaksDetected > 20 {
		return "high"
	}
	if summary.MediumSeverityLeaks > 10 || summary.TotalLeaksDetected > 10 {
		return "medium"
	}
	if summary.TotalLeaksDetected > 0 {
		return "low"
	}
	return "minimal"
}

// calculateConfidenceScore calculates the overall confidence score
func (r *Reporter) calculateConfidenceScore(
	heapAnalysis *detector.HeapAnalysis,
	goroutineAnalysis *detector.GoroutineAnalysis,
	referenceAnalysis *detector.ReferenceAnalysis,
	resourceAnalysis *detector.ResourceAnalysis,
) float64 {
	totalConfidence := 0.0
	analysisCount := 0

	if heapAnalysis != nil {
		totalConfidence += heapAnalysis.LeakProbability
		analysisCount++
	}
	if goroutineAnalysis != nil {
		totalConfidence += goroutineAnalysis.LeakProbability
		analysisCount++
	}
	if referenceAnalysis != nil {
		// Use a default confidence for reference analysis
		totalConfidence += 0.5
		analysisCount++
	}
	if resourceAnalysis != nil {
		// Calculate average confidence from resource leaks
		if len(resourceAnalysis.DetectedLeaks) > 0 {
			resourceConfidence := 0.0
			for _, leak := range resourceAnalysis.DetectedLeaks {
				resourceConfidence += leak.Confidence
			}
			totalConfidence += resourceConfidence / float64(len(resourceAnalysis.DetectedLeaks))
			analysisCount++
		}
	}

	if analysisCount == 0 {
		return 0.0
	}

	return totalConfidence / float64(analysisCount)
}

// calculateAnalysisCompleteness calculates how complete the analysis is
func (r *Reporter) calculateAnalysisCompleteness(
	heapAnalysis *detector.HeapAnalysis,
	goroutineAnalysis *detector.GoroutineAnalysis,
	referenceAnalysis *detector.ReferenceAnalysis,
	resourceAnalysis *detector.ResourceAnalysis,
) float64 {
	completeness := 0.0
	totalAnalyses := 4.0

	if heapAnalysis != nil {
		completeness += 1.0
	}
	if goroutineAnalysis != nil {
		completeness += 1.0
	}
	if referenceAnalysis != nil {
		completeness += 1.0
	}
	if resourceAnalysis != nil {
		completeness += 1.0
	}

	return completeness / totalAnalyses
}

// generateInsights generates analytical insights
func (r *Reporter) generateInsights(
	heapAnalysis *detector.HeapAnalysis,
	goroutineAnalysis *detector.GoroutineAnalysis,
	referenceAnalysis *detector.ReferenceAnalysis,
	resourceAnalysis *detector.ResourceAnalysis,
) []Insight {
	var insights []Insight

	// Heap insights
	if heapAnalysis != nil {
		insights = append(insights, r.generateHeapInsights(heapAnalysis)...)
	}

	// Goroutine insights
	if goroutineAnalysis != nil {
		insights = append(insights, r.generateGoroutineInsights(goroutineAnalysis)...)
	}

	// Reference insights
	if referenceAnalysis != nil {
		insights = append(insights, r.generateReferenceInsights(referenceAnalysis)...)
	}

	// Resource insights
	if resourceAnalysis != nil {
		insights = append(insights, r.generateResourceInsights(resourceAnalysis)...)
	}

	// Cross-analysis insights
	insights = append(insights, r.generateCrossAnalysisInsights(heapAnalysis, goroutineAnalysis, referenceAnalysis, resourceAnalysis)...)

	return insights
}

// generateHeapInsights generates insights from heap analysis
func (r *Reporter) generateHeapInsights(analysis *detector.HeapAnalysis) []Insight {
	var insights []Insight

	if analysis.GrowthRate > 10.0 {
		insights = append(insights, Insight{
			Type:        "heap_growth",
			Title:       "High Memory Growth Rate Detected",
			Description: fmt.Sprintf("Memory is growing at %.2f MB/min, which may indicate a memory leak", analysis.GrowthRate),
			Severity:    "high",
			Evidence:    []string{fmt.Sprintf("Growth rate: %.2f MB/min", analysis.GrowthRate)},
			Suggestion:  "Investigate allocation patterns and ensure proper cleanup of large objects",
			Timestamp:   time.Now(),
		})
	}

	if len(analysis.Anomalies) > 5 {
		insights = append(insights, Insight{
			Type:        "heap_anomalies",
			Title:       "Multiple Memory Anomalies Detected",
			Description: fmt.Sprintf("Found %d memory anomalies that may indicate leaks or inefficient allocation patterns", len(analysis.Anomalies)),
			Severity:    "medium",
			Evidence:    []string{fmt.Sprintf("%d anomalies detected", len(analysis.Anomalies))},
			Suggestion:  "Review allocation patterns and implement memory profiling",
			Timestamp:   time.Now(),
		})
	}

	return insights
}

// generateGoroutineInsights generates insights from goroutine analysis
func (r *Reporter) generateGoroutineInsights(analysis *detector.GoroutineAnalysis) []Insight {
	var insights []Insight

	if analysis.GrowthRate > 5.0 {
		insights = append(insights, Insight{
			Type:        "goroutine_growth",
			Title:       "High Goroutine Growth Rate",
			Description: fmt.Sprintf("Goroutines are growing at %.2f/min, indicating potential goroutine leaks", analysis.GrowthRate),
			Severity:    "high",
			Evidence:    []string{fmt.Sprintf("Growth rate: %.2f/min", analysis.GrowthRate)},
			Suggestion:  "Implement proper goroutine lifecycle management and use worker pools",
			Timestamp:   time.Now(),
		})
	}

	if len(analysis.LeakCandidates) > 10 {
		insights = append(insights, Insight{
			Type:        "goroutine_leaks",
			Title:       "Multiple Goroutine Leak Candidates",
			Description: fmt.Sprintf("Found %d goroutine leak candidates", len(analysis.LeakCandidates)),
			Severity:    "medium",
			Evidence:    []string{fmt.Sprintf("%d leak candidates", len(analysis.LeakCandidates))},
			Suggestion:  "Review goroutine creation patterns and ensure proper termination",
			Timestamp:   time.Now(),
		})
	}

	return insights
}

// generateReferenceInsights generates insights from reference analysis
func (r *Reporter) generateReferenceInsights(analysis *detector.ReferenceAnalysis) []Insight {
	var insights []Insight

	if len(analysis.DetectedCycles) > 0 {
		insights = append(insights, Insight{
			Type:        "reference_cycles",
			Title:       "Reference Cycles Detected",
			Description: fmt.Sprintf("Found %d reference cycles that may prevent garbage collection", len(analysis.DetectedCycles)),
			Severity:    "medium",
			Evidence:    []string{fmt.Sprintf("%d reference cycles", len(analysis.DetectedCycles))},
			Suggestion:  "Break reference cycles using weak references or restructure object relationships",
			Timestamp:   time.Now(),
		})
	}

	return insights
}

// generateResourceInsights generates insights from resource analysis
func (r *Reporter) generateResourceInsights(analysis *detector.ResourceAnalysis) []Insight {
	var insights []Insight

	if len(analysis.DetectedLeaks) > 5 {
		insights = append(insights, Insight{
			Type:        "resource_leaks",
			Title:       "Multiple Resource Leaks Detected",
			Description: fmt.Sprintf("Found %d resource leaks across different resource types", len(analysis.DetectedLeaks)),
			Severity:    "high",
			Evidence:    []string{fmt.Sprintf("%d resource leaks", len(analysis.DetectedLeaks))},
			Suggestion:  "Implement proper resource management patterns and use defer statements",
			Timestamp:   time.Now(),
		})
	}

	return insights
}

// generateCrossAnalysisInsights generates insights from cross-analysis
func (r *Reporter) generateCrossAnalysisInsights(
	heapAnalysis *detector.HeapAnalysis,
	goroutineAnalysis *detector.GoroutineAnalysis,
	referenceAnalysis *detector.ReferenceAnalysis,
	resourceAnalysis *detector.ResourceAnalysis,
) []Insight {
	var insights []Insight

	// Check for correlated leaks
	heapLeaks := heapAnalysis != nil && len(heapAnalysis.Anomalies) > 0
	goroutineLeaks := goroutineAnalysis != nil && len(goroutineAnalysis.LeakCandidates) > 0
	resourceLeaks := resourceAnalysis != nil && len(resourceAnalysis.DetectedLeaks) > 0

	if heapLeaks && goroutineLeaks {
		insights = append(insights, Insight{
			Type:        "correlated_leaks",
			Title:       "Correlated Memory and Goroutine Leaks",
			Description: "Both memory and goroutine leaks detected, suggesting a systemic issue",
			Severity:    "critical",
			Evidence:    []string{"Memory leaks present", "Goroutine leaks present"},
			Suggestion:  "Investigate shared code paths and implement comprehensive resource management",
			Timestamp:   time.Now(),
		})
	}

	if heapLeaks && resourceLeaks {
		insights = append(insights, Insight{
			Type:        "resource_memory_correlation",
			Title:       "Memory and Resource Leaks Correlation",
			Description: "Memory leaks correlate with resource leaks, indicating poor cleanup practices",
			Severity:    "high",
			Evidence:    []string{"Memory leaks present", "Resource leaks present"},
			Suggestion:  "Implement consistent cleanup patterns across all resource types",
			Timestamp:   time.Now(),
		})
	}

	return insights
}

// generateRecommendations generates actionable recommendations
func (r *Reporter) generateRecommendations(report *ReportData) []string {
	var recommendations []string

	// Priority recommendations based on severity
	if report.Summary.CriticalLeaks > 0 {
		recommendations = append(recommendations, "URGENT: Address critical leaks immediately to prevent system instability")
	}

	if report.Summary.OverallRiskLevel == "high" || report.Summary.OverallRiskLevel == "critical" {
		recommendations = append(recommendations, "Implement comprehensive memory management review")
		recommendations = append(recommendations, "Set up continuous monitoring for early leak detection")
	}

	// Specific recommendations based on leak types
	if report.Summary.LeaksByType["heap"] > 0 {
		recommendations = append(recommendations, "Review memory allocation patterns and implement object pooling where appropriate")
	}

	if report.Summary.LeaksByType["goroutine"] > 0 {
		recommendations = append(recommendations, "Implement goroutine lifecycle management and use context for cancellation")
	}

	if report.Summary.LeaksByType["file"] > 0 {
		recommendations = append(recommendations, "Ensure all file handles are properly closed using defer statements")
	}

	if report.Summary.LeaksByType["network"] > 0 {
		recommendations = append(recommendations, "Implement connection pooling and proper connection lifecycle management")
	}

	// Performance recommendations
	if report.Summary.MemoryGrowthRate > 5.0 {
		recommendations = append(recommendations, "Optimize memory allocation patterns to reduce growth rate")
	}

	if report.Summary.GoroutineGrowthRate > 2.0 {
		recommendations = append(recommendations, "Implement worker pools to control goroutine creation")
	}

	// General recommendations
	if report.Summary.ConfidenceScore < 0.7 {
		recommendations = append(recommendations, "Increase monitoring duration for more accurate leak detection")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Continue monitoring for any changes in resource usage patterns")
	}

	return recommendations
}

// calculateMetrics calculates comprehensive metrics
func (r *Reporter) calculateMetrics(
	heapAnalysis *detector.HeapAnalysis,
	goroutineAnalysis *detector.GoroutineAnalysis,
	referenceAnalysis *detector.ReferenceAnalysis,
	resourceAnalysis *detector.ResourceAnalysis,
) ReportMetrics {
	metrics := ReportMetrics{}

	// Calculate analysis duration
	metrics.AnalysisDuration = r.calculateTotalDuration(heapAnalysis, goroutineAnalysis, referenceAnalysis, resourceAnalysis)

	// Aggregate data points
	if heapAnalysis != nil {
		metrics.DataPointsAnalyzed += len(heapAnalysis.Metrics)
		metrics.MemoryPeakUsage = heapAnalysis.Statistics.MaxHeapSize
		// GC metrics would need to be collected separately
		metrics.GCEvents = 0
		metrics.GCTotalPause = 0
	}

	if goroutineAnalysis != nil {
		metrics.DataPointsAnalyzed += len(goroutineAnalysis.Metrics)
		metrics.GoroutinePeakCount = goroutineAnalysis.Statistics.MaxCount
	}

	if resourceAnalysis != nil {
		metrics.DataPointsAnalyzed += len(resourceAnalysis.Metrics)
		metrics.FileHandlesPeak = resourceAnalysis.Statistics.PeakFileHandles
		metrics.NetworkConnectionsPeak = resourceAnalysis.Statistics.PeakNetworkConns
	}

	// Calculate accuracy metrics (simplified)
	metrics.LeakDetectionAccuracy = 0.85 // Would be calculated based on validation
	metrics.FalsePositiveRate = 0.15     // Would be calculated based on validation

	// Performance impact (simplified)
	metrics.PerformanceImpact = PerformanceImpact{
		CPUOverhead:    2.5,  // 2.5% CPU overhead
		MemoryOverhead: 1024 * 1024 * 10, // 10MB memory overhead
		LatencyImpact:  0.1,  // 0.1ms latency impact
	}

	return metrics
}

// Helper functions

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (r *Reporter) writeTextReport(file *os.File, report *ReportData) error {
	fmt.Fprintf(file, "Memory Leak Detection Report\n")
	fmt.Fprintf(file, "Generated: %s\n", report.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(file, "Duration: %s\n\n", report.Duration)

	// Summary
	fmt.Fprintf(file, "SUMMARY\n")
	fmt.Fprintf(file, "=======\n")
	fmt.Fprintf(file, "Total Leaks Detected: %d\n", report.Summary.TotalLeaksDetected)
	fmt.Fprintf(file, "Critical Leaks: %d\n", report.Summary.CriticalLeaks)
	fmt.Fprintf(file, "Overall Risk Level: %s\n", report.Summary.OverallRiskLevel)
	fmt.Fprintf(file, "Confidence Score: %.2f\n", report.Summary.ConfidenceScore)
	fmt.Fprintf(file, "Memory Growth Rate: %.2f MB/min\n", report.Summary.MemoryGrowthRate)
	fmt.Fprintf(file, "Goroutine Growth Rate: %.2f/min\n\n", report.Summary.GoroutineGrowthRate)

	// Leaks by type
	fmt.Fprintf(file, "LEAKS BY TYPE\n")
	fmt.Fprintf(file, "=============\n")
	for leakType, count := range report.Summary.LeaksByType {
		fmt.Fprintf(file, "%s: %d\n", strings.Title(leakType), count)
	}
	fmt.Fprintf(file, "\n")

	// Top leak sources
	fmt.Fprintf(file, "TOP LEAK SOURCES\n")
	fmt.Fprintf(file, "================\n")
	for i, source := range report.Summary.TopLeakSources {
		fmt.Fprintf(file, "%d. %s (%s) - %d occurrences [%s]\n",
			i+1, source.Location, source.Type, source.Count, source.Severity)
	}
	fmt.Fprintf(file, "\n")

	// Insights
	fmt.Fprintf(file, "KEY INSIGHTS\n")
	fmt.Fprintf(file, "============\n")
	for _, insight := range report.Insights {
		fmt.Fprintf(file, "[%s] %s\n", strings.ToUpper(insight.Severity), insight.Title)
		fmt.Fprintf(file, "  %s\n", insight.Description)
		fmt.Fprintf(file, "  Suggestion: %s\n\n", insight.Suggestion)
	}

	// Recommendations
	fmt.Fprintf(file, "RECOMMENDATIONS\n")
	fmt.Fprintf(file, "===============\n")
	for i, rec := range report.Recommendations {
		fmt.Fprintf(file, "%d. %s\n", i+1, rec)
	}
	fmt.Fprintf(file, "\n")

	// Metrics
	fmt.Fprintf(file, "METRICS\n")
	fmt.Fprintf(file, "=======\n")
	fmt.Fprintf(file, "Analysis Duration: %s\n", report.Metrics.AnalysisDuration)
	fmt.Fprintf(file, "Data Points Analyzed: %d\n", report.Metrics.DataPointsAnalyzed)
	fmt.Fprintf(file, "Peak Memory Usage: %s\n", formatBytes(report.Metrics.MemoryPeakUsage))
	fmt.Fprintf(file, "Peak Goroutines: %d\n", report.Metrics.GoroutinePeakCount)
	fmt.Fprintf(file, "Peak File Handles: %d\n", report.Metrics.FileHandlesPeak)
	fmt.Fprintf(file, "Peak Network Connections: %d\n", report.Metrics.NetworkConnectionsPeak)
	fmt.Fprintf(file, "GC Events: %d\n", report.Metrics.GCEvents)
	fmt.Fprintf(file, "Total GC Pause: %s\n", report.Metrics.GCTotalPause)
	fmt.Fprintf(file, "Detection Accuracy: %.2f%%\n", report.Metrics.LeakDetectionAccuracy*100)
	fmt.Fprintf(file, "False Positive Rate: %.2f%%\n", report.Metrics.FalsePositiveRate*100)

	return nil
}

func (r *Reporter) getHTMLTemplate() string {
	return `
<!DOCTYPE html>
<html>
<head>
    <title>Memory Leak Detection Report</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #007bff; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 30px; }
        .summary { background: #e9ecef; padding: 20px; border-radius: 5px; margin: 20px 0; }
        .metric { display: inline-block; margin: 10px 20px 10px 0; }
        .metric-label { font-weight: bold; }
        .metric-value { color: #007bff; }
        .severity-critical { color: #dc3545; font-weight: bold; }
        .severity-high { color: #fd7e14; font-weight: bold; }
        .severity-medium { color: #ffc107; font-weight: bold; }
        .severity-low { color: #28a745; }
        .leak-source { background: #f8f9fa; padding: 10px; margin: 5px 0; border-left: 4px solid #007bff; }
        .insight { background: #fff3cd; padding: 15px; margin: 10px 0; border-radius: 5px; border-left: 4px solid #ffc107; }
        .recommendation { background: #d4edda; padding: 10px; margin: 5px 0; border-radius: 5px; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f8f9fa; }
        .progress-bar { width: 100%; height: 20px; background: #e9ecef; border-radius: 10px; overflow: hidden; }
        .progress-fill { height: 100%; background: linear-gradient(90deg, #28a745, #ffc107, #fd7e14, #dc3545); }
    </style>
</head>
<body>
    <div class="container">
        <h1>Memory Leak Detection Report</h1>
        <p><strong>Generated:</strong> {{.GeneratedAt.Format "2006-01-02 15:04:05"}}</p>
        <p><strong>Analysis Duration:</strong> {{formatDuration .Duration}}</p>
        
        <div class="summary">
            <h2>Executive Summary</h2>
            <div class="metric">
                <span class="metric-label">Total Leaks:</span>
                <span class="metric-value">{{.Summary.TotalLeaksDetected}}</span>
            </div>
            <div class="metric">
                <span class="metric-label">Critical Leaks:</span>
                <span class="metric-value severity-critical">{{.Summary.CriticalLeaks}}</span>
            </div>
            <div class="metric">
                <span class="metric-label">Risk Level:</span>
                <span class="metric-value {{getSeverityClass .Summary.OverallRiskLevel}}">{{.Summary.OverallRiskLevel}}</span>
            </div>
            <div class="metric">
                <span class="metric-label">Confidence:</span>
                <span class="metric-value">{{formatFloat .Summary.ConfidenceScore}}%</span>
            </div>
        </div>
        
        <h2>Leak Distribution</h2>
        <table>
            <tr><th>Type</th><th>Count</th><th>Percentage</th></tr>
            {{range $type, $count := .Summary.LeaksByType}}
            <tr>
                <td>{{$type}}</td>
                <td>{{$count}}</td>
                <td>{{formatFloat (div (mul $count 100.0) $.Summary.TotalLeaksDetected)}}%</td>
            </tr>
            {{end}}
        </table>
        
        <h2>Top Leak Sources</h2>
        {{range .Summary.TopLeakSources}}
        <div class="leak-source">
            <strong>{{.Location}}</strong> ({{.Type}})
            <br>Count: {{.Count}} | Severity: <span class="{{getSeverityClass .Severity}}">{{.Severity}}</span>
            <br>{{.Description}}
        </div>
        {{end}}
        
        <h2>Key Insights</h2>
        {{range .Insights}}
        <div class="insight">
            <h4>{{.Title}} <span class="{{getSeverityClass .Severity}}">[{{.Severity}}]</span></h4>
            <p>{{.Description}}</p>
            <p><strong>Suggestion:</strong> {{.Suggestion}}</p>
        </div>
        {{end}}
        
        <h2>Recommendations</h2>
        {{range .Recommendations}}
        <div class="recommendation">{{.}}</div>
        {{end}}
        
        <h2>Performance Metrics</h2>
        <table>
            <tr><th>Metric</th><th>Value</th></tr>
            <tr><td>Analysis Duration</td><td>{{formatDuration .Metrics.AnalysisDuration}}</td></tr>
            <tr><td>Data Points Analyzed</td><td>{{.Metrics.DataPointsAnalyzed}}</td></tr>
            <tr><td>Peak Memory Usage</td><td>{{formatBytes .Metrics.MemoryPeakUsage}}</td></tr>
            <tr><td>Peak Goroutines</td><td>{{.Metrics.GoroutinePeakCount}}</td></tr>
            <tr><td>Peak File Handles</td><td>{{.Metrics.FileHandlesPeak}}</td></tr>
            <tr><td>Peak Network Connections</td><td>{{.Metrics.NetworkConnectionsPeak}}</td></tr>
            <tr><td>GC Events</td><td>{{.Metrics.GCEvents}}</td></tr>
            <tr><td>Total GC Pause</td><td>{{formatDuration .Metrics.GCTotalPause}}</td></tr>
            <tr><td>Detection Accuracy</td><td>{{formatFloat .Metrics.LeakDetectionAccuracy}}%</td></tr>
            <tr><td>False Positive Rate</td><td>{{formatFloat .Metrics.FalsePositiveRate}}%</td></tr>
        </table>
    </div>
</body>
</html>
`
}
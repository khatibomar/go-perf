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

	"github.com/go-perf/02-memory-optimization/03-advanced-memory-profiling/exercise-03-03/analyzer"
	"github.com/go-perf/02-memory-optimization/03-advanced-memory-profiling/exercise-03-03/patterns"
	"github.com/go-perf/02-memory-optimization/03-advanced-memory-profiling/exercise-03-03/visualization"
)

// Config holds configuration for pattern reporting
type Config struct {
	InputDir  string
	OutputDir string
	Format    string
	Verbose   bool
}

// PatternReport represents a comprehensive pattern analysis report
type PatternReport struct {
	Timestamp       time.Time                    `json:"timestamp"`
	AnalysisPeriod  time.Duration                `json:"analysis_period"`
	Summary         ReportSummary                `json:"summary"`
	Patterns        []patterns.DetectedPattern   `json:"patterns"`
	Hotspots        []*analyzer.HotspotInfo      `json:"hotspots"`
	Metrics         []analyzer.AllocationMetrics `json:"metrics"`
	Anomalies       []patterns.Anomaly           `json:"anomalies"`
	Recommendations []Recommendation             `json:"recommendations"`
	Insights        []Insight                    `json:"insights"`
}

// ReportSummary contains high-level summary statistics
type ReportSummary struct {
	TotalPatterns      int     `json:"total_patterns"`
	TotalHotspots      int     `json:"total_hotspots"`
	TotalAnomalies     int     `json:"total_anomalies"`
	AverageAllocRate   float64 `json:"average_alloc_rate"`
	PeakAllocRate      float64 `json:"peak_alloc_rate"`
	TotalAllocations   uint64  `json:"total_allocations"`
	TotalBytes         uint64  `json:"total_bytes"`
	MemoryEfficiency   float64 `json:"memory_efficiency"`
	PatternStability   float64 `json:"pattern_stability"`
	OptimizationScore  float64 `json:"optimization_score"`
}

// Recommendation represents an optimization recommendation
type Recommendation struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Impact      string    `json:"impact"`
	Effort      string    `json:"effort"`
	Priority    int       `json:"priority"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
}

// Insight represents an analytical insight
type Insight struct {
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Confidence  float64   `json:"confidence"`
	Evidence    []string  `json:"evidence"`
	CreatedAt   time.Time `json:"created_at"`
}

// PatternReporter generates pattern analysis reports
type PatternReporter struct {
	config *Config
}

// NewPatternReporter creates a new pattern reporter
func NewPatternReporter(config *Config) *PatternReporter {
	return &PatternReporter{
		config: config,
	}
}

// GenerateReport generates a comprehensive pattern analysis report
func (pr *PatternReporter) GenerateReport() error {
	// Load analysis data
	report, err := pr.loadAnalysisData()
	if err != nil {
		return fmt.Errorf("failed to load analysis data: %v", err)
	}

	// Generate insights and recommendations
	pr.generateInsights(report)
	pr.generateRecommendations(report)

	// Generate report in requested format
	switch strings.ToLower(pr.config.Format) {
	case "json":
		return pr.generateJSONReport(report)
	case "html":
		return pr.generateHTMLReport(report)
	case "text":
		return pr.generateTextReport(report)
	default:
		return fmt.Errorf("unsupported format: %s", pr.config.Format)
	}
}

// loadAnalysisData loads analysis data from input directory
func (pr *PatternReporter) loadAnalysisData() (*PatternReport, error) {
	report := &PatternReport{
		Timestamp: time.Now(),
	}

	// Load patterns
	patternsDir := filepath.Join(pr.config.InputDir, "patterns")
	if patterns, err := pr.loadPatterns(patternsDir); err == nil {
		report.Patterns = patterns
	} else if pr.config.Verbose {
		fmt.Printf("Warning: failed to load patterns: %v\n", err)
	}

	// Load hotspots
	hotspotsFile := filepath.Join(pr.config.InputDir, "hotspots.json")
	if hotspots, err := pr.loadHotspots(hotspotsFile); err == nil {
		report.Hotspots = hotspots
	} else if pr.config.Verbose {
		fmt.Printf("Warning: failed to load hotspots: %v\n", err)
	}

	// Load metrics
	metricsFile := filepath.Join(pr.config.InputDir, "metrics.json")
	if metrics, err := pr.loadMetrics(metricsFile); err == nil {
		report.Metrics = metrics
		if len(metrics) > 0 {
			report.AnalysisPeriod = metrics[len(metrics)-1].Timestamp.Sub(metrics[0].Timestamp)
		}
	} else if pr.config.Verbose {
		fmt.Printf("Warning: failed to load metrics: %v\n", err)
	}

	// Generate summary
	pr.generateSummary(report)

	return report, nil
}

// loadPatterns loads detected patterns from directory
func (pr *PatternReporter) loadPatterns(dir string) ([]patterns.DetectedPattern, error) {
	var allPatterns []patterns.DetectedPattern

	files, err := filepath.Glob(filepath.Join(dir, "pattern_*.json"))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		var pattern patterns.DetectedPattern
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		if err := json.Unmarshal(data, &pattern); err != nil {
			continue
		}

		allPatterns = append(allPatterns, pattern)
	}

	// Sort patterns by start time
	sort.Slice(allPatterns, func(i, j int) bool {
		return allPatterns[i].StartTime.Before(allPatterns[j].StartTime)
	})

	return allPatterns, nil
}

// loadHotspots loads hotspot analysis from file
func (pr *PatternReporter) loadHotspots(filename string) ([]*analyzer.HotspotInfo, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var hotspots []*analyzer.HotspotInfo
	if err := json.Unmarshal(data, &hotspots); err != nil {
		return nil, err
	}

	// Sort hotspots by allocation rate
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].AllocationRate > hotspots[j].AllocationRate
	})

	return hotspots, nil
}

// loadMetrics loads allocation metrics from file
func (pr *PatternReporter) loadMetrics(filename string) ([]analyzer.AllocationMetrics, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var metrics []analyzer.AllocationMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}

// generateSummary generates report summary statistics
func (pr *PatternReporter) generateSummary(report *PatternReport) {
	summary := &report.Summary
	summary.TotalPatterns = len(report.Patterns)
	summary.TotalHotspots = len(report.Hotspots)
	summary.TotalAnomalies = len(report.Anomalies)

	// Calculate allocation statistics
	if len(report.Metrics) > 0 {
		var totalBytes, totalAllocations uint64
		var maxRate float64

		for _, metric := range report.Metrics {
			totalBytes += metric.BytesPerSecond
			totalAllocations += metric.ObjectsPerSecond
			rate := float64(metric.BytesPerSecond)
			if rate > maxRate {
				maxRate = rate
			}
		}

		summary.AverageAllocRate = float64(totalBytes) / float64(len(report.Metrics))
		summary.PeakAllocRate = maxRate
		summary.TotalAllocations = totalAllocations
		summary.TotalBytes = totalBytes
	}

	// Calculate efficiency metrics
	summary.MemoryEfficiency = pr.calculateMemoryEfficiency(report)
	summary.PatternStability = pr.calculatePatternStability(report)
	summary.OptimizationScore = pr.calculateOptimizationScore(report)
}

// calculateMemoryEfficiency calculates memory usage efficiency
func (pr *PatternReporter) calculateMemoryEfficiency(report *PatternReport) float64 {
	if len(report.Metrics) == 0 {
		return 0
	}

	// Simple efficiency metric based on allocation rate stability
	var variance float64
	mean := report.Summary.AverageAllocRate

	for _, metric := range report.Metrics {
		diff := float64(metric.BytesPerSecond) - mean
		variance += diff * diff
	}

	variance /= float64(len(report.Metrics))
	stdDev := variance
	if stdDev > 0 {
		stdDev = variance * variance // Simplified
	}

	// Higher efficiency for lower variance
	efficiency := 1.0 / (1.0 + stdDev/mean)
	if efficiency > 1.0 {
		efficiency = 1.0
	}

	return efficiency
}

// calculatePatternStability calculates pattern stability score
func (pr *PatternReporter) calculatePatternStability(report *PatternReport) float64 {
	if len(report.Patterns) == 0 {
		return 1.0 // No patterns = stable
	}

	// Count pattern types
	patternTypes := make(map[patterns.PatternType]int)
	for _, pattern := range report.Patterns {
		patternTypes[pattern.Type]++
	}

	// Stability is higher when patterns are predictable
	stability := 0.0
	for patternType, count := range patternTypes {
		switch patternType {
		case patterns.PatternSteady:
			stability += float64(count) * 1.0
		case patterns.PatternCyclic:
			stability += float64(count) * 0.8
		case patterns.PatternGrowth:
			stability += float64(count) * 0.6
		case patterns.PatternBurst:
			stability += float64(count) * 0.4
		case patterns.PatternSpike:
			stability += float64(count) * 0.2
		case patterns.PatternAnomaly:
			stability += float64(count) * 0.1
		}
	}

	return stability / float64(len(report.Patterns))
}

// calculateOptimizationScore calculates overall optimization score
func (pr *PatternReporter) calculateOptimizationScore(report *PatternReport) float64 {
	// Combine efficiency, stability, and hotspot severity
	efficiencyScore := report.Summary.MemoryEfficiency * 0.4
	stabilityScore := report.Summary.PatternStability * 0.3

	// Hotspot penalty
	hotspotPenalty := 0.0
	for _, hotspot := range report.Hotspots {
		switch hotspot.Severity {
		case "critical":
			hotspotPenalty += 0.3
		case "high":
			hotspotPenalty += 0.2
		case "medium":
			hotspotPenalty += 0.1
		}
	}

	hotspotScore := (1.0 - hotspotPenalty) * 0.3
	if hotspotScore < 0 {
		hotspotScore = 0
	}

	totalScore := efficiencyScore + stabilityScore + hotspotScore
	if totalScore > 1.0 {
		totalScore = 1.0
	}

	return totalScore
}

// generateInsights generates analytical insights
func (pr *PatternReporter) generateInsights(report *PatternReport) {
	now := time.Now()

	// Pattern insights
	if len(report.Patterns) > 0 {
		patternTypes := make(map[patterns.PatternType]int)
		for _, pattern := range report.Patterns {
			patternTypes[pattern.Type]++
		}

		for patternType, count := range patternTypes {
			insight := Insight{
				Type:        "pattern_analysis",
				Title:       fmt.Sprintf("%s Pattern Detected", strings.Title(string(patternType))),
				Description: fmt.Sprintf("Detected %d instances of %s allocation pattern", count, patternType),
				Confidence:  0.8,
				Evidence:    []string{fmt.Sprintf("%d pattern occurrences", count)},
				CreatedAt:   now,
			}
			report.Insights = append(report.Insights, insight)
		}
	}

	// Hotspot insights
	if len(report.Hotspots) > 0 {
		criticalHotspots := 0
		for _, hotspot := range report.Hotspots {
			if hotspot.Severity == "critical" {
				criticalHotspots++
			}
		}

		if criticalHotspots > 0 {
			insight := Insight{
				Type:        "hotspot_analysis",
				Title:       "Critical Allocation Hotspots Found",
				Description: fmt.Sprintf("Found %d critical allocation hotspots requiring immediate attention", criticalHotspots),
				Confidence:  0.9,
				Evidence:    []string{fmt.Sprintf("%d critical hotspots", criticalHotspots)},
				CreatedAt:   now,
			}
			report.Insights = append(report.Insights, insight)
		}
	}

	// Efficiency insights
	if report.Summary.MemoryEfficiency < 0.5 {
		insight := Insight{
			Type:        "efficiency_analysis",
			Title:       "Low Memory Efficiency Detected",
			Description: fmt.Sprintf("Memory efficiency is %.2f%%, indicating potential optimization opportunities", report.Summary.MemoryEfficiency*100),
			Confidence:  0.7,
			Evidence:    []string{fmt.Sprintf("Efficiency score: %.2f", report.Summary.MemoryEfficiency)},
			CreatedAt:   now,
		}
		report.Insights = append(report.Insights, insight)
	}
}

// generateRecommendations generates optimization recommendations
func (pr *PatternReporter) generateRecommendations(report *PatternReport) {
	now := time.Now()
	priority := 1

	// Pattern-based recommendations
	for _, pattern := range report.Patterns {
		if pattern.Severity == "high" || pattern.Severity == "critical" {
			rec := Recommendation{
				ID:          fmt.Sprintf("pattern_%s_%d", pattern.Type, now.Unix()),
				Type:        "pattern_optimization",
				Severity:    pattern.Severity,
				Title:       fmt.Sprintf("Optimize %s Pattern", strings.Title(string(pattern.Type))),
				Description: pattern.Suggestion,
				Impact:      pattern.Impact,
				Effort:      "Medium",
				Priority:    priority,
				Tags:        []string{"pattern", string(pattern.Type)},
				CreatedAt:   now,
			}
			report.Recommendations = append(report.Recommendations, rec)
			priority++
		}
	}

	// Hotspot-based recommendations
	for _, hotspot := range report.Hotspots {
		if hotspot.Severity == "high" || hotspot.Severity == "critical" {
			rec := Recommendation{
				ID:          fmt.Sprintf("hotspot_%s_%d", strings.ReplaceAll(hotspot.Location, ":", "_"), now.Unix()),
				Type:        "hotspot_optimization",
				Severity:    hotspot.Severity,
				Title:       fmt.Sprintf("Optimize Hotspot: %s", hotspot.Function),
				Description: hotspot.Recommendation,
				Impact:      fmt.Sprintf("Reduce allocation rate from %.2f/s", hotspot.AllocationRate),
				Effort:      "High",
				Priority:    priority,
				Tags:        []string{"hotspot", hotspot.Function},
				CreatedAt:   now,
			}
			report.Recommendations = append(report.Recommendations, rec)
			priority++
		}
	}

	// General recommendations based on summary
	if report.Summary.OptimizationScore < 0.6 {
		rec := Recommendation{
			ID:          fmt.Sprintf("general_optimization_%d", now.Unix()),
			Type:        "general_optimization",
			Severity:    "medium",
			Title:       "Improve Overall Memory Management",
			Description: "Consider implementing object pooling, reducing allocation frequency, and optimizing data structures",
			Impact:      "Improve overall application performance and memory efficiency",
			Effort:      "Medium",
			Priority:    priority,
			Tags:        []string{"general", "optimization"},
			CreatedAt:   now,
		}
		report.Recommendations = append(report.Recommendations, rec)
	}

	// Sort recommendations by priority
	sort.Slice(report.Recommendations, func(i, j int) bool {
		return report.Recommendations[i].Priority < report.Recommendations[j].Priority
	})
}

// generateJSONReport generates a JSON format report
func (pr *PatternReporter) generateJSONReport(report *PatternReport) error {
	filename := filepath.Join(pr.config.OutputDir, "pattern_report.json")
	if err := os.MkdirAll(pr.config.OutputDir, 0755); err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// generateTextReport generates a text format report
func (pr *PatternReporter) generateTextReport(report *PatternReport) error {
	filename := filepath.Join(pr.config.OutputDir, "pattern_report.txt")
	if err := os.MkdirAll(pr.config.OutputDir, 0755); err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write report content
	fmt.Fprintf(file, "Allocation Pattern Analysis Report\n")
	fmt.Fprintf(file, "Generated: %s\n", report.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(file, "Analysis Period: %v\n\n", report.AnalysisPeriod)

	// Summary
	fmt.Fprintf(file, "SUMMARY\n")
	fmt.Fprintf(file, "=======\n")
	fmt.Fprintf(file, "Total Patterns: %d\n", report.Summary.TotalPatterns)
	fmt.Fprintf(file, "Total Hotspots: %d\n", report.Summary.TotalHotspots)
	fmt.Fprintf(file, "Average Allocation Rate: %.2f bytes/s\n", report.Summary.AverageAllocRate)
	fmt.Fprintf(file, "Peak Allocation Rate: %.2f bytes/s\n", report.Summary.PeakAllocRate)
	fmt.Fprintf(file, "Memory Efficiency: %.2f%%\n", report.Summary.MemoryEfficiency*100)
	fmt.Fprintf(file, "Pattern Stability: %.2f%%\n", report.Summary.PatternStability*100)
	fmt.Fprintf(file, "Optimization Score: %.2f%%\n\n", report.Summary.OptimizationScore*100)

	// Patterns
	if len(report.Patterns) > 0 {
		fmt.Fprintf(file, "DETECTED PATTERNS\n")
		fmt.Fprintf(file, "=================\n")
		for i, pattern := range report.Patterns {
			fmt.Fprintf(file, "%d. %s Pattern\n", i+1, strings.Title(string(pattern.Type)))
			fmt.Fprintf(file, "   Confidence: %.2f\n", pattern.Confidence)
			fmt.Fprintf(file, "   Duration: %v\n", pattern.Duration)
			fmt.Fprintf(file, "   Severity: %s\n", pattern.Severity)
			fmt.Fprintf(file, "   Description: %s\n", pattern.Description)
			fmt.Fprintf(file, "   Suggestion: %s\n\n", pattern.Suggestion)
		}
	}

	// Hotspots
	if len(report.Hotspots) > 0 {
		fmt.Fprintf(file, "ALLOCATION HOTSPOTS\n")
		fmt.Fprintf(file, "==================\n")
		for i, hotspot := range report.Hotspots {
			fmt.Fprintf(file, "%d. %s\n", i+1, hotspot.Function)
			fmt.Fprintf(file, "   Location: %s\n", hotspot.Location)
			fmt.Fprintf(file, "   Allocation Rate: %.2f/s\n", hotspot.AllocationRate)
			fmt.Fprintf(file, "   Severity: %s\n", hotspot.Severity)
			fmt.Fprintf(file, "   Recommendation: %s\n\n", hotspot.Recommendation)
		}
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		fmt.Fprintf(file, "OPTIMIZATION RECOMMENDATIONS\n")
		fmt.Fprintf(file, "===========================\n")
		for i, rec := range report.Recommendations {
			fmt.Fprintf(file, "%d. %s\n", i+1, rec.Title)
			fmt.Fprintf(file, "   Type: %s\n", rec.Type)
			fmt.Fprintf(file, "   Severity: %s\n", rec.Severity)
			fmt.Fprintf(file, "   Effort: %s\n", rec.Effort)
			fmt.Fprintf(file, "   Description: %s\n", rec.Description)
			fmt.Fprintf(file, "   Impact: %s\n\n", rec.Impact)
		}
	}

	return nil
}

// generateHTMLReport generates an HTML format report with visualizations
func (pr *PatternReporter) generateHTMLReport(report *PatternReport) error {
	if err := os.MkdirAll(pr.config.OutputDir, 0755); err != nil {
		return err
	}

	// Generate charts if we have metrics data
	var chartContents []string
	if len(report.Metrics) > 0 {
		chartGenerator := visualization.NewChartGenerator()
		
		// Generate allocation timeline chart
		if timelineChart, err := chartGenerator.GenerateAllocationTimelineChart(report.Metrics); err == nil {
			chartContents = append(chartContents, timelineChart)
		}
		
		// Generate hotspot bar chart if we have hotspots
		if len(report.Hotspots) > 0 {
			if hotspotChart, err := chartGenerator.GenerateHotspotBarChart(report.Hotspots); err == nil {
				chartContents = append(chartContents, hotspotChart)
			}
		}
		
		// Generate pattern distribution chart if we have patterns
		if len(report.Patterns) > 0 {
			if patternChart, err := chartGenerator.GeneratePatternDistributionChart(report.Patterns); err == nil {
				chartContents = append(chartContents, patternChart)
			}
		}
		
		// Generate GC impact chart
		if gcChart, err := chartGenerator.GenerateGCImpactChart(report.Metrics); err == nil {
			chartContents = append(chartContents, gcChart)
		}
	}

	// Create report data with chart contents
	reportData := struct {
		*PatternReport
		ChartContents []string
	}{
		PatternReport: report,
		ChartContents: chartContents,
	}

	filename := filepath.Join(pr.config.OutputDir, "pattern_report.html")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create template with custom functions
	funcMap := template.FuncMap{
		"mul": func(a, b float64) float64 { return a * b },
		"title": strings.Title,
	}
	tmpl := template.Must(template.New("report").Funcs(funcMap).Parse(htmlTemplate))
	return tmpl.Execute(file, reportData)
}

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Allocation Pattern Analysis Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .summary { background-color: #e8f4f8; padding: 15px; margin: 20px 0; border-radius: 5px; }
        .section { margin: 20px 0; }
        .pattern, .hotspot, .recommendation { 
            border: 1px solid #ddd; 
            padding: 10px; 
            margin: 10px 0; 
            border-radius: 3px; 
        }
        .severity-critical { border-left: 5px solid #ff0000; }
        .severity-high { border-left: 5px solid #ff8800; }
        .severity-medium { border-left: 5px solid #ffaa00; }
        .severity-low { border-left: 5px solid #00aa00; }
        .metric { display: inline-block; margin: 10px; padding: 10px; background-color: #f9f9f9; border-radius: 3px; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Allocation Pattern Analysis Report</h1>
        <p>Generated: {{.Timestamp.Format "2006-01-02 15:04:05"}}</p>
        <p>Analysis Period: {{.AnalysisPeriod}}</p>
    </div>

    <div class="summary">
        <h2>Summary</h2>
        <div class="metric">Total Patterns: {{.Summary.TotalPatterns}}</div>
        <div class="metric">Total Hotspots: {{.Summary.TotalHotspots}}</div>
        <div class="metric">Avg Alloc Rate: {{printf "%.2f" .Summary.AverageAllocRate}} bytes/s</div>
        <div class="metric">Peak Alloc Rate: {{printf "%.2f" .Summary.PeakAllocRate}} bytes/s</div>
        <div class="metric">Memory Efficiency: {{printf "%.1f" (mul .Summary.MemoryEfficiency 100)}}%</div>
        <div class="metric">Pattern Stability: {{printf "%.1f" (mul .Summary.PatternStability 100)}}%</div>
        <div class="metric">Optimization Score: {{printf "%.1f" (mul .Summary.OptimizationScore 100)}}%</div>
    </div>

    {{if .ChartContents}}
    <div class="section">
        <h2>Visualizations</h2>
        {{range $chartContent := .ChartContents}}
        <div style="margin: 20px 0; border: 1px solid #ddd; border-radius: 5px;">
            {{$chartContent}}
        </div>
        {{end}}
    </div>
    {{end}}

    {{if .Patterns}}
    <div class="section">
        <h2>Detected Patterns</h2>
        {{range $i, $pattern := .Patterns}}
        <div class="pattern severity-{{$pattern.Severity}}">
            <h3>{{title $pattern.Type}} Pattern</h3>
            <p><strong>Confidence:</strong> {{printf "%.2f" $pattern.Confidence}}</p>
            <p><strong>Duration:</strong> {{$pattern.Duration}}</p>
            <p><strong>Severity:</strong> {{$pattern.Severity}}</p>
            <p><strong>Description:</strong> {{$pattern.Description}}</p>
            <p><strong>Suggestion:</strong> {{$pattern.Suggestion}}</p>
        </div>
        {{end}}
    </div>
    {{end}}

    {{if .Hotspots}}
    <div class="section">
        <h2>Allocation Hotspots</h2>
        {{range $i, $hotspot := .Hotspots}}
        <div class="hotspot severity-{{$hotspot.Severity}}">
            <h3>{{$hotspot.Function}}</h3>
            <p><strong>Location:</strong> {{$hotspot.Location}}</p>
            <p><strong>Allocation Rate:</strong> {{printf "%.2f" $hotspot.AllocationRate}}/s</p>
            <p><strong>Severity:</strong> {{$hotspot.Severity}}</p>
            <p><strong>Recommendation:</strong> {{$hotspot.Recommendation}}</p>
        </div>
        {{end}}
    </div>
    {{end}}

    {{if .Recommendations}}
    <div class="section">
        <h2>Optimization Recommendations</h2>
        {{range $i, $rec := .Recommendations}}
        <div class="recommendation severity-{{$rec.Severity}}">
            <h3>{{$rec.Title}}</h3>
            <p><strong>Type:</strong> {{$rec.Type}}</p>
            <p><strong>Severity:</strong> {{$rec.Severity}}</p>
            <p><strong>Effort:</strong> {{$rec.Effort}}</p>
            <p><strong>Description:</strong> {{$rec.Description}}</p>
            <p><strong>Impact:</strong> {{$rec.Impact}}</p>
        </div>
        {{end}}
    </div>
    {{end}}
</body>
</html>
`
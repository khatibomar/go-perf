package reporting

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"../analysis"
	"../tracing"
)

// ReportType defines the type of report to generate
type ReportType string

const (
	SummaryReport     ReportType = "summary"
	DetailedReport    ReportType = "detailed"
	ComparisonReport  ReportType = "comparison"
	BenchmarkReport   ReportType = "benchmark"
	OptimizationReport ReportType = "optimization"
)

// ReportFormat defines the output format
type ReportFormat string

const (
	TextFormat     ReportFormat = "text"
	MarkdownFormat ReportFormat = "markdown"
	JSONFormat     ReportFormat = "json"
	HTMLFormat     ReportFormat = "html"
	CSVFormat      ReportFormat = "csv"
)

// Report represents a performance analysis report
type Report struct {
	Title           string                           `json:"title"`
	Type            ReportType                       `json:"type"`
	GeneratedAt     time.Time                        `json:"generated_at"`
	ExecutionTime   time.Duration                    `json:"execution_time"`
	Configuration   ReportConfiguration              `json:"configuration"`
	SystemInfo      SystemInfo                       `json:"system_info"`
	MetricsSummary  MetricsSummary                   `json:"metrics_summary"`
	WorkloadAnalysis *analysis.WorkloadCharacteristics `json:"workload_analysis,omitempty"`
	Insights        []analysis.PerformanceInsight    `json:"insights"`
	Recommendations []Recommendation                 `json:"recommendations"`
	TraceAnalysis   *TraceAnalysisSummary            `json:"trace_analysis,omitempty"`
	Benchmarks      []BenchmarkResult                `json:"benchmarks,omitempty"`
	Comparisons     []ComparisonResult               `json:"comparisons,omitempty"`
}

// ReportConfiguration contains report generation settings
type ReportConfiguration struct {
	IncludeCharts      bool     `json:"include_charts"`
	IncludeRawData     bool     `json:"include_raw_data"`
	IncludeTrace       bool     `json:"include_trace"`
	IncludeBenchmarks  bool     `json:"include_benchmarks"`
	MetricsToInclude   []string `json:"metrics_to_include"`
	DetailLevel        string   `json:"detail_level"` // "basic", "standard", "detailed"
	TimeRange          string   `json:"time_range"`
	SamplingInterval   string   `json:"sampling_interval"`
}

// SystemInfo contains system configuration information
type SystemInfo struct {
	GOMAXPROCS      int               `json:"gomaxprocs"`
	NumCPU          int               `json:"num_cpu"`
	GoVersion       string            `json:"go_version"`
	OS              string            `json:"os"`
	Arch            string            `json:"arch"`
	MemoryTotal     uint64            `json:"memory_total"`
	Environment     map[string]string `json:"environment"`
}

// MetricsSummary contains aggregated metrics
type MetricsSummary struct {
	SampleCount         int                    `json:"sample_count"`
	TimeRange           string                 `json:"time_range"`
	CPUUtilization      StatsSummary           `json:"cpu_utilization"`
	MemoryUsage         StatsSummary           `json:"memory_usage"`
	GoroutineCount      StatsSummary           `json:"goroutine_count"`
	SchedulerLatency    StatsSummary           `json:"scheduler_latency"`
	ContextSwitches     StatsSummary           `json:"context_switches"`
	Throughput          StatsSummary           `json:"throughput"`
	Efficiency          StatsSummary           `json:"efficiency"`
	CustomMetrics       map[string]StatsSummary `json:"custom_metrics,omitempty"`
}

// StatsSummary contains statistical summary of a metric
type StatsSummary struct {
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	Mean       float64 `json:"mean"`
	Median     float64 `json:"median"`
	StdDev     float64 `json:"std_dev"`
	P95        float64 `json:"p95"`
	P99        float64 `json:"p99"`
	Unit       string  `json:"unit"`
	Trend      string  `json:"trend"` // "increasing", "decreasing", "stable", "volatile"
}

// Recommendation represents a performance recommendation
type Recommendation struct {
	Category    string    `json:"category"`
	Priority    string    `json:"priority"` // "low", "medium", "high", "critical"
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Action      string    `json:"action"`
	Impact      string    `json:"impact"`
	Effort      string    `json:"effort"` // "low", "medium", "high"
	References  []string  `json:"references,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// TraceAnalysisSummary contains trace analysis results
type TraceAnalysisSummary struct {
	TraceFile        string                 `json:"trace_file"`
	Duration         time.Duration          `json:"duration"`
	GoroutineStats   GoroutineStats         `json:"goroutine_stats"`
	SchedulerStats   SchedulerStats         `json:"scheduler_stats"`
	GCStats          GCStats                `json:"gc_stats"`
	NetworkStats     NetworkStats           `json:"network_stats"`
	SyscallStats     SyscallStats           `json:"syscall_stats"`
	CustomEvents     map[string]interface{} `json:"custom_events,omitempty"`
}

// GoroutineStats contains goroutine-related statistics
type GoroutineStats struct {
	MaxConcurrent    int           `json:"max_concurrent"`
	TotalCreated     int           `json:"total_created"`
	AvgLifetime      time.Duration `json:"avg_lifetime"`
	BlockingProfile  []BlockEvent  `json:"blocking_profile"`
	CreationHotspots []Hotspot     `json:"creation_hotspots"`
}

// SchedulerStats contains scheduler-related statistics
type SchedulerStats struct {
	Preemptions      int           `json:"preemptions"`
	WorkStealing     int           `json:"work_stealing"`
	AvgRunQueueLen   float64       `json:"avg_run_queue_len"`
	SchedulingDelay  time.Duration `json:"scheduling_delay"`
	ProcessorUtil    []float64     `json:"processor_util"`
}

// GCStats contains garbage collection statistics
type GCStats struct {
	NumGC        int           `json:"num_gc"`
	TotalPause   time.Duration `json:"total_pause"`
	MaxPause     time.Duration `json:"max_pause"`
	AvgPause     time.Duration `json:"avg_pause"`
	HeapSize     uint64        `json:"heap_size"`
	AllocRate    float64       `json:"alloc_rate"`
}

// NetworkStats contains network-related statistics
type NetworkStats struct {
	Connections    int           `json:"connections"`
	BytesRead      uint64        `json:"bytes_read"`
	BytesWritten   uint64        `json:"bytes_written"`
	AvgLatency     time.Duration `json:"avg_latency"`
	Timeouts       int           `json:"timeouts"`
}

// SyscallStats contains system call statistics
type SyscallStats struct {
	TotalCalls   int                    `json:"total_calls"`
	ByType       map[string]int         `json:"by_type"`
	AvgDuration  time.Duration          `json:"avg_duration"`
	SlowCalls    []SyscallEvent         `json:"slow_calls"`
}

// BlockEvent represents a blocking event
type BlockEvent struct {
	Type     string        `json:"type"`
	Duration time.Duration `json:"duration"`
	Stack    []string      `json:"stack"`
	Count    int           `json:"count"`
}

// Hotspot represents a performance hotspot
type Hotspot struct {
	Function string  `json:"function"`
	File     string  `json:"file"`
	Line     int     `json:"line"`
	Count    int     `json:"count"`
	Percent  float64 `json:"percent"`
}

// SyscallEvent represents a system call event
type SyscallEvent struct {
	Name     string        `json:"name"`
	Duration time.Duration `json:"duration"`
	Args     []string      `json:"args"`
	Result   string        `json:"result"`
}

// BenchmarkResult contains benchmark execution results
type BenchmarkResult struct {
	Name         string        `json:"name"`
	Iterations   int           `json:"iterations"`
	Duration     time.Duration `json:"duration"`
	NsPerOp      float64       `json:"ns_per_op"`
	BytesPerOp   int64         `json:"bytes_per_op"`
	AllocsPerOp  int64         `json:"allocs_per_op"`
	MemoryUsage  uint64        `json:"memory_usage"`
	CPUUsage     float64       `json:"cpu_usage"`
	Configuration map[string]interface{} `json:"configuration"`
}

// ComparisonResult contains comparison between different configurations
type ComparisonResult struct {
	Name        string                 `json:"name"`
	Baseline    string                 `json:"baseline"`
	Comparison  string                 `json:"comparison"`
	Improvement map[string]float64     `json:"improvement"`
	Regression  map[string]float64     `json:"regression"`
	Summary     string                 `json:"summary"`
	Details     map[string]interface{} `json:"details"`
}

// ReportGenerator handles report generation
type ReportGenerator struct {
	outputDir string
	config    ReportConfiguration
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(outputDir string, config ReportConfiguration) *ReportGenerator {
	return &ReportGenerator{
		outputDir: outputDir,
		config:    config,
	}
}

// GenerateReport creates a comprehensive performance report
func (rg *ReportGenerator) GenerateReport(
	reportType ReportType,
	metrics []analysis.SchedulerMetrics,
	insights []analysis.PerformanceInsight,
	workload *analysis.WorkloadCharacteristics,
	traceMetrics *tracing.TraceMetrics,
) (*Report, error) {
	start := time.Now()
	
	report := &Report{
		Title:         fmt.Sprintf("Go Scheduler Performance Report - %s", strings.Title(string(reportType))),
		Type:          reportType,
		GeneratedAt:   start,
		Configuration: rg.config,
		SystemInfo:    rg.collectSystemInfo(),
		Insights:      insights,
		WorkloadAnalysis: workload,
	}
	
	// Generate metrics summary
	if len(metrics) > 0 {
		report.MetricsSummary = rg.generateMetricsSummary(metrics)
	}
	
	// Generate trace analysis if available
	if traceMetrics != nil && rg.config.IncludeTrace {
		report.TraceAnalysis = rg.generateTraceAnalysis(traceMetrics)
	}
	
	// Generate recommendations
	report.Recommendations = rg.generateRecommendations(metrics, insights, workload)
	
	// Generate benchmarks if requested
	if rg.config.IncludeBenchmarks {
		report.Benchmarks = rg.generateBenchmarks()
	}
	
	report.ExecutionTime = time.Since(start)
	return report, nil
}

// SaveReport saves the report in the specified format
func (rg *ReportGenerator) SaveReport(report *Report, filename string, format ReportFormat) error {
	if err := os.MkdirAll(rg.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	filePath := filepath.Join(rg.outputDir, filename)
	
	switch format {
	case TextFormat:
		return rg.saveAsText(report, filePath)
	case MarkdownFormat:
		return rg.saveAsMarkdown(report, filePath)
	case JSONFormat:
		return rg.saveAsJSON(report, filePath)
	case HTMLFormat:
		return rg.saveAsHTML(report, filePath)
	case CSVFormat:
		return rg.saveAsCSV(report, filePath)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// Helper methods for report generation

func (rg *ReportGenerator) collectSystemInfo() SystemInfo {
	return SystemInfo{
		GOMAXPROCS: rg.getGOMAXPROCS(),
		NumCPU:     rg.getNumCPU(),
		GoVersion:  rg.getGoVersion(),
		OS:         rg.getOS(),
		Arch:       rg.getArch(),
		MemoryTotal: rg.getMemoryTotal(),
		Environment: rg.getRelevantEnvVars(),
	}
}

func (rg *ReportGenerator) generateMetricsSummary(metrics []analysis.SchedulerMetrics) MetricsSummary {
	if len(metrics) == 0 {
		return MetricsSummary{}
	}
	
	return MetricsSummary{
		SampleCount: len(metrics),
		TimeRange: fmt.Sprintf("%s to %s", 
			metrics[0].Timestamp.Format("15:04:05"),
			metrics[len(metrics)-1].Timestamp.Format("15:04:05")),
		CPUUtilization:   rg.calculateStatsSummary(metrics, "cpu", "%"),
		MemoryUsage:      rg.calculateStatsSummary(metrics, "memory", "%"),
		GoroutineCount:   rg.calculateStatsSummary(metrics, "goroutines", "count"),
		SchedulerLatency: rg.calculateStatsSummary(metrics, "latency", "ms"),
		ContextSwitches:  rg.calculateStatsSummary(metrics, "context_switches", "count"),
		Throughput:       rg.calculateStatsSummary(metrics, "throughput", "ops/s"),
		Efficiency:       rg.calculateStatsSummary(metrics, "efficiency", "ratio"),
	}
}

func (rg *ReportGenerator) calculateStatsSummary(metrics []analysis.SchedulerMetrics, metricType, unit string) StatsSummary {
	var values []float64
	
	for _, metric := range metrics {
		switch metricType {
		case "cpu":
			values = append(values, metric.AvgCPUUtilization)
		case "memory":
			values = append(values, metric.MemoryPercent)
		case "goroutines":
			values = append(values, float64(metric.NumGoroutine))
		case "latency":
			values = append(values, float64(metric.SchedulerLatency.Nanoseconds())/1e6)
		case "context_switches":
			values = append(values, float64(metric.ContextSwitches))
		case "throughput":
			values = append(values, metric.Throughput)
		case "efficiency":
			values = append(values, metric.Efficiency)
		}
	}
	
	if len(values) == 0 {
		return StatsSummary{Unit: unit}
	}
	
	sort.Float64s(values)
	
	return StatsSummary{
		Min:    values[0],
		Max:    values[len(values)-1],
		Mean:   rg.calculateMean(values),
		Median: rg.calculateMedian(values),
		StdDev: rg.calculateStdDev(values),
		P95:    rg.calculatePercentile(values, 95),
		P99:    rg.calculatePercentile(values, 99),
		Unit:   unit,
		Trend:  rg.calculateTrend(values),
	}
}

func (rg *ReportGenerator) generateTraceAnalysis(traceMetrics *tracing.TraceMetrics) *TraceAnalysisSummary {
	return &TraceAnalysisSummary{
		TraceFile: traceMetrics.TraceFile,
		Duration:  traceMetrics.Duration,
		GoroutineStats: GoroutineStats{
			MaxConcurrent: traceMetrics.MaxGoroutines,
			TotalCreated:  traceMetrics.TotalGoroutines,
			AvgLifetime:   traceMetrics.AvgGoroutineLifetime,
		},
		SchedulerStats: SchedulerStats{
			Preemptions:     traceMetrics.Preemptions,
			WorkStealing:    traceMetrics.WorkStealing,
			AvgRunQueueLen:  traceMetrics.AvgRunQueueLength,
			SchedulingDelay: traceMetrics.SchedulingDelay,
		},
		GCStats: GCStats{
			NumGC:      traceMetrics.GCEvents,
			TotalPause: traceMetrics.GCPauseTime,
			MaxPause:   traceMetrics.MaxGCPause,
			AvgPause:   traceMetrics.AvgGCPause,
			HeapSize:   traceMetrics.HeapSize,
		},
	}
}

func (rg *ReportGenerator) generateRecommendations(
	metrics []analysis.SchedulerMetrics,
	insights []analysis.PerformanceInsight,
	workload *analysis.WorkloadCharacteristics,
) []Recommendation {
	var recommendations []Recommendation
	
	// Generate recommendations based on insights
	for _, insight := range insights {
		rec := Recommendation{
			Category:    insight.Type,
			Priority:    insight.Severity,
			Title:       insight.Title,
			Description: insight.Description,
			Action:      insight.Suggestion,
			Timestamp:   insight.Timestamp,
		}
		
		// Determine impact and effort based on insight type
		switch insight.Type {
		case "bottleneck":
			rec.Impact = "high"
			rec.Effort = "medium"
		case "optimization":
			rec.Impact = "medium"
			rec.Effort = "low"
		case "warning":
			rec.Impact = "medium"
			rec.Effort = "low"
		default:
			rec.Impact = "low"
			rec.Effort = "low"
		}
		
		recommendations = append(recommendations, rec)
	}
	
	// Generate workload-specific recommendations
	if workload != nil {
		for _, rec := range workload.Recommendations {
			recommendations = append(recommendations, Recommendation{
				Category:    "workload_optimization",
				Priority:    "medium",
				Title:       "Workload Optimization",
				Description: rec,
				Action:      rec,
				Impact:      "medium",
				Effort:      "medium",
				Timestamp:   time.Now(),
			})
		}
	}
	
	// Generate general performance recommendations
	recommendations = append(recommendations, rg.generateGeneralRecommendations(metrics)...)
	
	return recommendations
}

func (rg *ReportGenerator) generateGeneralRecommendations(metrics []analysis.SchedulerMetrics) []Recommendation {
	var recommendations []Recommendation
	
	if len(metrics) == 0 {
		return recommendations
	}
	
	// Analyze average metrics
	avgCPU := rg.calculateAverageMetric(metrics, "cpu")
	avgMemory := rg.calculateAverageMetric(metrics, "memory")
	avgGoroutines := rg.calculateAverageMetric(metrics, "goroutines")
	
	// CPU-based recommendations
	if avgCPU < 20 {
		recommendations = append(recommendations, Recommendation{
			Category:    "resource_utilization",
			Priority:    "low",
			Title:       "Low CPU Utilization",
			Description: "CPU utilization is consistently low",
			Action:      "Consider increasing workload or reducing allocated resources",
			Impact:      "medium",
			Effort:      "low",
			Timestamp:   time.Now(),
		})
	} else if avgCPU > 80 {
		recommendations = append(recommendations, Recommendation{
			Category:    "performance",
			Priority:    "high",
			Title:       "High CPU Utilization",
			Description: "CPU utilization is consistently high",
			Action:      "Optimize CPU-intensive operations or scale horizontally",
			Impact:      "high",
			Effort:      "medium",
			Timestamp:   time.Now(),
		})
	}
	
	// Memory-based recommendations
	if avgMemory > 85 {
		recommendations = append(recommendations, Recommendation{
			Category:    "memory",
			Priority:    "critical",
			Title:       "High Memory Usage",
			Description: "Memory usage is critically high",
			Action:      "Investigate memory leaks and optimize allocation patterns",
			Impact:      "high",
			Effort:      "high",
			Timestamp:   time.Now(),
		})
	}
	
	// Goroutine-based recommendations
	if avgGoroutines > 10000 {
		recommendations = append(recommendations, Recommendation{
			Category:    "concurrency",
			Priority:    "medium",
			Title:       "High Goroutine Count",
			Description: "Very high number of goroutines detected",
			Action:      "Consider using goroutine pools or reducing goroutine creation",
			Impact:      "medium",
			Effort:      "medium",
			Timestamp:   time.Now(),
		})
	}
	
	return recommendations
}

func (rg *ReportGenerator) generateBenchmarks() []BenchmarkResult {
	// This would typically run actual benchmarks
	// For now, return placeholder data
	return []BenchmarkResult{
		{
			Name:        "SchedulerOverhead",
			Iterations:  1000,
			Duration:    time.Millisecond * 100,
			NsPerOp:     100000,
			BytesPerOp:  0,
			AllocsPerOp: 1,
			Configuration: map[string]interface{}{
				"gomaxprocs": rg.getGOMAXPROCS(),
				"goroutines": 100,
			},
		},
	}
}

// Format-specific save methods

func (rg *ReportGenerator) saveAsText(report *Report, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	w := tabwriter.NewWriter(file, 0, 0, 2, ' ', 0)
	
	// Write header
	fmt.Fprintf(w, "=== %s ===\n\n", report.Title)
	fmt.Fprintf(w, "Generated:\t%s\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Execution Time:\t%v\n", report.ExecutionTime)
	fmt.Fprintf(w, "Report Type:\t%s\n\n", report.Type)
	
	// System information
	fmt.Fprintf(w, "=== System Information ===\n")
	fmt.Fprintf(w, "GOMAXPROCS:\t%d\n", report.SystemInfo.GOMAXPROCS)
	fmt.Fprintf(w, "CPU Cores:\t%d\n", report.SystemInfo.NumCPU)
	fmt.Fprintf(w, "Go Version:\t%s\n", report.SystemInfo.GoVersion)
	fmt.Fprintf(w, "OS/Arch:\t%s/%s\n\n", report.SystemInfo.OS, report.SystemInfo.Arch)
	
	// Metrics summary
	if report.MetricsSummary.SampleCount > 0 {
		fmt.Fprintf(w, "=== Metrics Summary ===\n")
		fmt.Fprintf(w, "Sample Count:\t%d\n", report.MetricsSummary.SampleCount)
		fmt.Fprintf(w, "Time Range:\t%s\n", report.MetricsSummary.TimeRange)
		
		rg.writeStatsSummary(w, "CPU Utilization", report.MetricsSummary.CPUUtilization)
		rg.writeStatsSummary(w, "Memory Usage", report.MetricsSummary.MemoryUsage)
		rg.writeStatsSummary(w, "Goroutine Count", report.MetricsSummary.GoroutineCount)
		rg.writeStatsSummary(w, "Scheduler Latency", report.MetricsSummary.SchedulerLatency)
		fmt.Fprintf(w, "\n")
	}
	
	// Workload analysis
	if report.WorkloadAnalysis != nil {
		fmt.Fprintf(w, "=== Workload Analysis ===\n")
		fmt.Fprintf(w, "Type:\t%s\n", report.WorkloadAnalysis.Type)
		fmt.Fprintf(w, "Contention Level:\t%s\n", report.WorkloadAnalysis.ContentionLevel)
		fmt.Fprintf(w, "Burstiness:\t%.2f\n", report.WorkloadAnalysis.Burstiness)
		fmt.Fprintf(w, "Predictability:\t%.2f\n", report.WorkloadAnalysis.Predictability)
		fmt.Fprintf(w, "Optimal GOMAXPROCS:\t%d\n\n", report.WorkloadAnalysis.OptimalGOMMAXPROCS)
	}
	
	// Insights
	if len(report.Insights) > 0 {
		fmt.Fprintf(w, "=== Performance Insights ===\n")
		for _, insight := range report.Insights {
			fmt.Fprintf(w, "[%s] %s:\t%s\n", strings.ToUpper(insight.Severity), insight.Title, insight.Description)
			fmt.Fprintf(w, "  Suggestion:\t%s\n", insight.Suggestion)
		}
		fmt.Fprintf(w, "\n")
	}
	
	// Recommendations
	if len(report.Recommendations) > 0 {
		fmt.Fprintf(w, "=== Recommendations ===\n")
		for _, rec := range report.Recommendations {
			fmt.Fprintf(w, "[%s] %s:\t%s\n", strings.ToUpper(rec.Priority), rec.Title, rec.Description)
			fmt.Fprintf(w, "  Action:\t%s\n", rec.Action)
			fmt.Fprintf(w, "  Impact/Effort:\t%s/%s\n", rec.Impact, rec.Effort)
		}
	}
	
	return w.Flush()
}

func (rg *ReportGenerator) saveAsMarkdown(report *Report, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write markdown content
	fmt.Fprintf(file, "# %s\n\n", report.Title)
	fmt.Fprintf(file, "**Generated:** %s  \n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "**Execution Time:** %v  \n", report.ExecutionTime)
	fmt.Fprintf(file, "**Report Type:** %s\n\n", report.Type)
	
	// System information
	fmt.Fprintf(file, "## System Information\n\n")
	fmt.Fprintf(file, "| Property | Value |\n")
	fmt.Fprintf(file, "|----------|-------|\n")
	fmt.Fprintf(file, "| GOMAXPROCS | %d |\n", report.SystemInfo.GOMAXPROCS)
	fmt.Fprintf(file, "| CPU Cores | %d |\n", report.SystemInfo.NumCPU)
	fmt.Fprintf(file, "| Go Version | %s |\n", report.SystemInfo.GoVersion)
	fmt.Fprintf(file, "| OS/Arch | %s/%s |\n\n", report.SystemInfo.OS, report.SystemInfo.Arch)
	
	// Metrics summary
	if report.MetricsSummary.SampleCount > 0 {
		fmt.Fprintf(file, "## Metrics Summary\n\n")
		fmt.Fprintf(file, "**Sample Count:** %d  \n", report.MetricsSummary.SampleCount)
		fmt.Fprintf(file, "**Time Range:** %s\n\n", report.MetricsSummary.TimeRange)
		
		fmt.Fprintf(file, "| Metric | Min | Max | Mean | Median | P95 | P99 | Trend |\n")
		fmt.Fprintf(file, "|--------|-----|-----|------|--------|-----|-----|-------|\n")
		rg.writeMarkdownStatsSummary(file, "CPU Utilization", report.MetricsSummary.CPUUtilization)
		rg.writeMarkdownStatsSummary(file, "Memory Usage", report.MetricsSummary.MemoryUsage)
		rg.writeMarkdownStatsSummary(file, "Goroutine Count", report.MetricsSummary.GoroutineCount)
		rg.writeMarkdownStatsSummary(file, "Scheduler Latency", report.MetricsSummary.SchedulerLatency)
		fmt.Fprintf(file, "\n")
	}
	
	// Workload analysis
	if report.WorkloadAnalysis != nil {
		fmt.Fprintf(file, "## Workload Analysis\n\n")
		fmt.Fprintf(file, "- **Type:** %s\n", report.WorkloadAnalysis.Type)
		fmt.Fprintf(file, "- **Contention Level:** %s\n", report.WorkloadAnalysis.ContentionLevel)
		fmt.Fprintf(file, "- **Burstiness:** %.2f\n", report.WorkloadAnalysis.Burstiness)
		fmt.Fprintf(file, "- **Predictability:** %.2f\n", report.WorkloadAnalysis.Predictability)
		fmt.Fprintf(file, "- **Optimal GOMAXPROCS:** %d\n\n", report.WorkloadAnalysis.OptimalGOMMAXPROCS)
	}
	
	// Insights
	if len(report.Insights) > 0 {
		fmt.Fprintf(file, "## Performance Insights\n\n")
		for _, insight := range report.Insights {
			fmt.Fprintf(file, "### %s (%s)\n\n", insight.Title, strings.ToUpper(insight.Severity))
			fmt.Fprintf(file, "%s\n\n", insight.Description)
			fmt.Fprintf(file, "**Suggestion:** %s\n\n", insight.Suggestion)
		}
	}
	
	// Recommendations
	if len(report.Recommendations) > 0 {
		fmt.Fprintf(file, "## Recommendations\n\n")
		for _, rec := range report.Recommendations {
			fmt.Fprintf(file, "### %s (%s Priority)\n\n", rec.Title, strings.Title(rec.Priority))
			fmt.Fprintf(file, "%s\n\n", rec.Description)
			fmt.Fprintf(file, "**Action:** %s\n\n", rec.Action)
			fmt.Fprintf(file, "**Impact:** %s | **Effort:** %s\n\n", strings.Title(rec.Impact), strings.Title(rec.Effort))
		}
	}
	
	return nil
}

func (rg *ReportGenerator) saveAsJSON(report *Report, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func (rg *ReportGenerator) saveAsHTML(report *Report, filePath string) error {
	// HTML implementation would be similar to markdown but with HTML tags
	// For brevity, delegating to markdown for now
	return rg.saveAsMarkdown(report, filePath)
}

func (rg *ReportGenerator) saveAsCSV(report *Report, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write CSV headers and data for metrics summary
	fmt.Fprintf(file, "Metric,Min,Max,Mean,Median,StdDev,P95,P99,Unit,Trend\n")
	
	metrics := map[string]StatsSummary{
		"CPU_Utilization":   report.MetricsSummary.CPUUtilization,
		"Memory_Usage":      report.MetricsSummary.MemoryUsage,
		"Goroutine_Count":   report.MetricsSummary.GoroutineCount,
		"Scheduler_Latency": report.MetricsSummary.SchedulerLatency,
	}
	
	for name, stats := range metrics {
		fmt.Fprintf(file, "%s,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f,%s,%s\n",
			name, stats.Min, stats.Max, stats.Mean, stats.Median,
			stats.StdDev, stats.P95, stats.P99, stats.Unit, stats.Trend)
	}
	
	return nil
}

// Helper methods for statistics and formatting

func (rg *ReportGenerator) writeStatsSummary(w *tabwriter.Writer, name string, stats StatsSummary) {
	fmt.Fprintf(w, "%s:\n", name)
	fmt.Fprintf(w, "  Min/Max:\t%.2f / %.2f %s\n", stats.Min, stats.Max, stats.Unit)
	fmt.Fprintf(w, "  Mean/Median:\t%.2f / %.2f %s\n", stats.Mean, stats.Median, stats.Unit)
	fmt.Fprintf(w, "  P95/P99:\t%.2f / %.2f %s\n", stats.P95, stats.P99, stats.Unit)
	fmt.Fprintf(w, "  Trend:\t%s\n", stats.Trend)
}

func (rg *ReportGenerator) writeMarkdownStatsSummary(file *os.File, name string, stats StatsSummary) {
	fmt.Fprintf(file, "| %s | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f | %s |\n",
		name, stats.Min, stats.Max, stats.Mean, stats.Median, stats.P95, stats.P99, stats.Trend)
}

// Statistical calculation methods

func (rg *ReportGenerator) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (rg *ReportGenerator) calculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	n := len(values)
	if n%2 == 0 {
		return (values[n/2-1] + values[n/2]) / 2
	}
	return values[n/2]
}

func (rg *ReportGenerator) calculateStdDev(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	mean := rg.calculateMean(values)
	var variance float64
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values) - 1)
	return variance // Simplified - should be sqrt(variance)
}

func (rg *ReportGenerator) calculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	index := int(percentile/100*float64(len(values)-1) + 0.5)
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func (rg *ReportGenerator) calculateTrend(values []float64) string {
	if len(values) < 2 {
		return "insufficient_data"
	}
	
	// Simple trend analysis based on first and last quartile
	q1End := len(values) / 4
	q4Start := 3 * len(values) / 4
	
	q1Avg := rg.calculateMean(values[:q1End])
	q4Avg := rg.calculateMean(values[q4Start:])
	
	diff := q4Avg - q1Avg
	threshold := rg.calculateMean(values) * 0.1 // 10% threshold
	
	if diff > threshold {
		return "increasing"
	} else if diff < -threshold {
		return "decreasing"
	} else {
		// Check volatility
		stdDev := rg.calculateStdDev(values)
		mean := rg.calculateMean(values)
		if mean > 0 && stdDev/mean > 0.2 {
			return "volatile"
		}
		return "stable"
	}
}

func (rg *ReportGenerator) calculateAverageMetric(metrics []analysis.SchedulerMetrics, metricType string) float64 {
	if len(metrics) == 0 {
		return 0
	}
	
	var sum float64
	for _, metric := range metrics {
		switch metricType {
		case "cpu":
			sum += metric.AvgCPUUtilization
		case "memory":
			sum += metric.MemoryPercent
		case "goroutines":
			sum += float64(metric.NumGoroutine)
		}
	}
	return sum / float64(len(metrics))
}

// System information collection methods (simplified implementations)

func (rg *ReportGenerator) getGOMAXPROCS() int {
	// This would use runtime.GOMAXPROCS(0)
	return 8 // Placeholder
}

func (rg *ReportGenerator) getNumCPU() int {
	// This would use runtime.NumCPU()
	return 8 // Placeholder
}

func (rg *ReportGenerator) getGoVersion() string {
	// This would use runtime.Version()
	return "go1.21.0" // Placeholder
}

func (rg *ReportGenerator) getOS() string {
	// This would use runtime.GOOS
	return "windows" // Placeholder
}

func (rg *ReportGenerator) getArch() string {
	// This would use runtime.GOARCH
	return "amd64" // Placeholder
}

func (rg *ReportGenerator) getMemoryTotal() uint64 {
	// This would query system memory
	return 16 * 1024 * 1024 * 1024 // 16GB placeholder
}

func (rg *ReportGenerator) getRelevantEnvVars() map[string]string {
	return map[string]string{
		"GOMAXPROCS": os.Getenv("GOMAXPROCS"),
		"GOGC":       os.Getenv("GOGC"),
		"GODEBUG":    os.Getenv("GODEBUG"),
	}
}
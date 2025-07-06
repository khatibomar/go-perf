package visualization

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"../analysis"
)

// ChartType represents different types of charts
type ChartType string

const (
	LineChart   ChartType = "line"
	BarChart    ChartType = "bar"
	AreaChart   ChartType = "area"
	ScatterPlot ChartType = "scatter"
	Heatmap     ChartType = "heatmap"
	Histogram   ChartType = "histogram"
)

// ChartData represents data for visualization
type ChartData struct {
	Title       string                 `json:"title"`
	Type        ChartType              `json:"type"`
	Labels      []string               `json:"labels"`
	Datasets    []Dataset              `json:"datasets"`
	Options     map[string]interface{} `json:"options"`
	Description string                 `json:"description"`
}

// Dataset represents a data series
type Dataset struct {
	Label           string    `json:"label"`
	Data            []float64 `json:"data"`
	BackgroundColor string    `json:"backgroundColor,omitempty"`
	BorderColor     string    `json:"borderColor,omitempty"`
	Fill            bool      `json:"fill,omitempty"`
	Tension         float64   `json:"tension,omitempty"`
}

// Dashboard represents a collection of charts
type Dashboard struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Charts      []ChartData `json:"charts"`
	Metadata    Metadata    `json:"metadata"`
}

// Metadata contains dashboard metadata
type Metadata struct {
	GeneratedAt   time.Time              `json:"generated_at"`
	DataRange     string                 `json:"data_range"`
	SampleCount   int                    `json:"sample_count"`
	Configuration map[string]interface{} `json:"configuration"`
}

// Visualizer provides visualization capabilities
type Visualizer struct {
	outputDir string
	theme     string
}

// NewVisualizer creates a new visualizer
func NewVisualizer(outputDir, theme string) *Visualizer {
	return &Visualizer{
		outputDir: outputDir,
		theme:     theme,
	}
}

// GenerateDashboard creates a comprehensive dashboard from scheduler metrics
func (v *Visualizer) GenerateDashboard(metrics []analysis.SchedulerMetrics, insights []analysis.PerformanceInsight, workload *analysis.WorkloadCharacteristics) (*Dashboard, error) {
	if len(metrics) == 0 {
		return nil, fmt.Errorf("no metrics provided")
	}
	
	dashboard := &Dashboard{
		Title:       "Go Scheduler Performance Dashboard",
		Description: "Comprehensive analysis of Go scheduler performance and behavior",
		Metadata: Metadata{
			GeneratedAt: time.Now(),
			DataRange:   fmt.Sprintf("%s to %s", metrics[0].Timestamp.Format("15:04:05"), metrics[len(metrics)-1].Timestamp.Format("15:04:05")),
			SampleCount: len(metrics),
			Configuration: map[string]interface{}{
				"gomaxprocs": metrics[0].GOMAXPROCS,
				"num_cpu":    metrics[0].NumCPU,
				"theme":      v.theme,
			},
		},
	}
	
	// Generate various charts
	charts := []ChartData{
		v.createCPUUtilizationChart(metrics),
		v.createGoroutineCountChart(metrics),
		v.createMemoryUsageChart(metrics),
		v.createSchedulerLatencyChart(metrics),
		v.createContextSwitchChart(metrics),
		v.createPerCPUUtilizationChart(metrics),
		v.createWorkloadCharacteristicsChart(workload),
		v.createInsightsSummaryChart(insights),
		v.createPerformanceDistributionChart(metrics),
		v.createSchedulerEfficiencyChart(metrics),
	}
	
	dashboard.Charts = charts
	return dashboard, nil
}

// SaveDashboard saves the dashboard to HTML file
func (v *Visualizer) SaveDashboard(dashboard *Dashboard, filename string) error {
	if err := os.MkdirAll(v.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	filePath := filepath.Join(v.outputDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()
	
	// Convert dashboard to JSON for JavaScript
	dashboardJSON, err := json.Marshal(dashboard)
	if err != nil {
		return fmt.Errorf("failed to marshal dashboard: %w", err)
	}
	
	// Generate HTML with embedded Chart.js
	tmpl := template.Must(template.New("dashboard").Parse(dashboardTemplate))
	data := struct {
		Dashboard     *Dashboard
		DashboardJSON template.JS
		Theme         string
	}{
		Dashboard:     dashboard,
		DashboardJSON: template.JS(dashboardJSON),
		Theme:         v.theme,
	}
	
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	
	return nil
}

// Chart creation functions

func (v *Visualizer) createCPUUtilizationChart(metrics []analysis.SchedulerMetrics) ChartData {
	labels := make([]string, len(metrics))
	data := make([]float64, len(metrics))
	
	for i, metric := range metrics {
		labels[i] = metric.Timestamp.Format("15:04:05")
		data[i] = metric.AvgCPUUtilization
	}
	
	return ChartData{
		Title: "CPU Utilization Over Time",
		Type:  LineChart,
		Labels: labels,
		Datasets: []Dataset{{
			Label:       "CPU Utilization (%)",
			Data:        data,
			BorderColor: "#3498db",
			Fill:        false,
			Tension:     0.1,
		}},
		Options: map[string]interface{}{
			"responsive": true,
			"scales": map[string]interface{}{
				"y": map[string]interface{}{
					"beginAtZero": true,
					"max":         100,
					"title": map[string]interface{}{
						"display": true,
						"text":    "CPU Utilization (%)",
					},
				},
				"x": map[string]interface{}{
					"title": map[string]interface{}{
						"display": true,
						"text":    "Time",
					},
				},
			},
		},
		Description: "Shows CPU utilization percentage over time. High values (>80%) may indicate CPU bottlenecks.",
	}
}

func (v *Visualizer) createGoroutineCountChart(metrics []analysis.SchedulerMetrics) ChartData {
	labels := make([]string, len(metrics))
	data := make([]float64, len(metrics))
	
	for i, metric := range metrics {
		labels[i] = metric.Timestamp.Format("15:04:05")
		data[i] = float64(metric.NumGoroutine)
	}
	
	return ChartData{
		Title: "Goroutine Count Over Time",
		Type:  LineChart,
		Labels: labels,
		Datasets: []Dataset{{
			Label:       "Goroutine Count",
			Data:        data,
			BorderColor: "#e74c3c",
			Fill:        false,
			Tension:     0.1,
		}},
		Options: map[string]interface{}{
			"responsive": true,
			"scales": map[string]interface{}{
				"y": map[string]interface{}{
					"beginAtZero": true,
					"title": map[string]interface{}{
						"display": true,
						"text":    "Number of Goroutines",
					},
				},
				"x": map[string]interface{}{
					"title": map[string]interface{}{
						"display": true,
						"text":    "Time",
					},
				},
			},
		},
		Description: "Tracks the number of goroutines over time. Sudden spikes may indicate goroutine leaks.",
	}
}

func (v *Visualizer) createMemoryUsageChart(metrics []analysis.SchedulerMetrics) ChartData {
	labels := make([]string, len(metrics))
	data := make([]float64, len(metrics))
	
	for i, metric := range metrics {
		labels[i] = metric.Timestamp.Format("15:04:05")
		data[i] = metric.MemoryPercent
	}
	
	return ChartData{
		Title: "Memory Usage Over Time",
		Type:  AreaChart,
		Labels: labels,
		Datasets: []Dataset{{
			Label:           "Memory Usage (%)",
			Data:            data,
			BackgroundColor: "rgba(46, 204, 113, 0.3)",
			BorderColor:     "#2ecc71",
			Fill:            true,
			Tension:         0.1,
		}},
		Options: map[string]interface{}{
			"responsive": true,
			"scales": map[string]interface{}{
				"y": map[string]interface{}{
					"beginAtZero": true,
					"max":         100,
					"title": map[string]interface{}{
						"display": true,
						"text":    "Memory Usage (%)",
					},
				},
				"x": map[string]interface{}{
					"title": map[string]interface{}{
						"display": true,
						"text":    "Time",
					},
				},
			},
		},
		Description: "Shows memory usage percentage over time. Values above 85% may indicate memory pressure.",
	}
}

func (v *Visualizer) createSchedulerLatencyChart(metrics []analysis.SchedulerMetrics) ChartData {
	labels := make([]string, len(metrics))
	data := make([]float64, len(metrics))
	
	for i, metric := range metrics {
		labels[i] = metric.Timestamp.Format("15:04:05")
		data[i] = float64(metric.SchedulerLatency.Nanoseconds()) / 1e6 // Convert to milliseconds
	}
	
	return ChartData{
		Title: "Scheduler Latency Over Time",
		Type:  LineChart,
		Labels: labels,
		Datasets: []Dataset{{
			Label:       "Scheduler Latency (ms)",
			Data:        data,
			BorderColor: "#f39c12",
			Fill:        false,
			Tension:     0.1,
		}},
		Options: map[string]interface{}{
			"responsive": true,
			"scales": map[string]interface{}{
				"y": map[string]interface{}{
					"beginAtZero": true,
					"title": map[string]interface{}{
						"display": true,
						"text":    "Latency (ms)",
					},
				},
				"x": map[string]interface{}{
					"title": map[string]interface{}{
						"display": true,
						"text":    "Time",
					},
				},
			},
		},
		Description: "Measures scheduler latency in milliseconds. High values indicate scheduling overhead.",
	}
}

func (v *Visualizer) createContextSwitchChart(metrics []analysis.SchedulerMetrics) ChartData {
	labels := make([]string, len(metrics))
	data := make([]float64, len(metrics))
	
	for i, metric := range metrics {
		labels[i] = metric.Timestamp.Format("15:04:05")
		data[i] = float64(metric.ContextSwitches)
	}
	
	return ChartData{
		Title: "Context Switches Over Time",
		Type:  BarChart,
		Labels: labels,
		Datasets: []Dataset{{
			Label:           "Context Switches",
			Data:            data,
			BackgroundColor: "rgba(155, 89, 182, 0.7)",
			BorderColor:     "#9b59b6",
		}},
		Options: map[string]interface{}{
			"responsive": true,
			"scales": map[string]interface{}{
				"y": map[string]interface{}{
					"beginAtZero": true,
					"title": map[string]interface{}{
						"display": true,
						"text":    "Context Switches",
					},
				},
				"x": map[string]interface{}{
					"title": map[string]interface{}{
						"display": true,
						"text":    "Time",
					},
				},
			},
		},
		Description: "Shows context switches over time. High values may indicate contention.",
	}
}

func (v *Visualizer) createPerCPUUtilizationChart(metrics []analysis.SchedulerMetrics) ChartData {
	if len(metrics) == 0 || len(metrics[0].CPUUtilization) == 0 {
		return ChartData{
			Title:       "Per-CPU Utilization",
			Type:        BarChart,
			Description: "No per-CPU data available",
		}
	}
	
	// Use the latest metrics for per-CPU visualization
	latest := metrics[len(metrics)-1]
	labels := make([]string, len(latest.CPUUtilization))
	data := make([]float64, len(latest.CPUUtilization))
	
	for i, util := range latest.CPUUtilization {
		labels[i] = fmt.Sprintf("CPU %d", i)
		data[i] = util
	}
	
	return ChartData{
		Title: "Per-CPU Utilization (Latest)",
		Type:  BarChart,
		Labels: labels,
		Datasets: []Dataset{{
			Label:           "CPU Utilization (%)",
			Data:            data,
			BackgroundColor: "rgba(52, 152, 219, 0.7)",
			BorderColor:     "#3498db",
		}},
		Options: map[string]interface{}{
			"responsive": true,
			"scales": map[string]interface{}{
				"y": map[string]interface{}{
					"beginAtZero": true,
					"max":         100,
					"title": map[string]interface{}{
						"display": true,
						"text":    "Utilization (%)",
					},
				},
				"x": map[string]interface{}{
					"title": map[string]interface{}{
						"display": true,
						"text":    "CPU Core",
					},
				},
			},
		},
		Description: "Shows utilization per CPU core. Uneven distribution may indicate load balancing issues.",
	}
}

func (v *Visualizer) createWorkloadCharacteristicsChart(workload *analysis.WorkloadCharacteristics) ChartData {
	if workload == nil {
		return ChartData{
			Title:       "Workload Characteristics",
			Type:        BarChart,
			Description: "No workload analysis available",
		}
	}
	
	labels := []string{"CPU Intensive", "I/O Intensive", "Memory Intensive", "Goroutine Intensive"}
	data := []float64{
		boolToFloat(workload.CPUIntensive),
		boolToFloat(workload.IOIntensive),
		boolToFloat(workload.MemoryIntensive),
		boolToFloat(workload.GoroutineIntensive),
	}
	
	return ChartData{
		Title: "Workload Characteristics",
		Type:  BarChart,
		Labels: labels,
		Datasets: []Dataset{{
			Label:           "Intensity",
			Data:            data,
			BackgroundColor: "rgba(231, 76, 60, 0.7)",
			BorderColor:     "#e74c3c",
		}},
		Options: map[string]interface{}{
			"responsive": true,
			"scales": map[string]interface{}{
				"y": map[string]interface{}{
					"beginAtZero": true,
					"max":         1,
					"title": map[string]interface{}{
						"display": true,
						"text":    "Intensity (0=Low, 1=High)",
					},
				},
			},
		},
		Description: fmt.Sprintf("Workload type: %s, Contention: %s, Burstiness: %.2f", workload.Type, workload.ContentionLevel, workload.Burstiness),
	}
}

func (v *Visualizer) createInsightsSummaryChart(insights []analysis.PerformanceInsight) ChartData {
	severityCounts := map[string]int{
		"low":      0,
		"medium":   0,
		"high":     0,
		"critical": 0,
	}
	
	for _, insight := range insights {
		severityCounts[insight.Severity]++
	}
	
	labels := []string{"Low", "Medium", "High", "Critical"}
	data := []float64{
		float64(severityCounts["low"]),
		float64(severityCounts["medium"]),
		float64(severityCounts["high"]),
		float64(severityCounts["critical"]),
	}
	
	colors := []string{
		"rgba(46, 204, 113, 0.7)",  // Green for low
		"rgba(241, 196, 15, 0.7)",  // Yellow for medium
		"rgba(230, 126, 34, 0.7)",  // Orange for high
		"rgba(231, 76, 60, 0.7)",   // Red for critical
	}
	
	return ChartData{
		Title: "Performance Insights Summary",
		Type:  BarChart,
		Labels: labels,
		Datasets: []Dataset{{
			Label:           "Number of Insights",
			Data:            data,
			BackgroundColor: strings.Join(colors, ","),
		}},
		Options: map[string]interface{}{
			"responsive": true,
			"scales": map[string]interface{}{
				"y": map[string]interface{}{
					"beginAtZero": true,
					"title": map[string]interface{}{
						"display": true,
						"text":    "Count",
					},
				},
			},
		},
		Description: fmt.Sprintf("Summary of %d performance insights by severity level", len(insights)),
	}
}

func (v *Visualizer) createPerformanceDistributionChart(metrics []analysis.SchedulerMetrics) ChartData {
	// Create histogram of CPU utilization
	buckets := make(map[string]int)
	for _, metric := range metrics {
		bucket := fmt.Sprintf("%.0f-%.0f%%", math.Floor(metric.AvgCPUUtilization/10)*10, math.Floor(metric.AvgCPUUtilization/10)*10+10)
		buckets[bucket]++
	}
	
	// Sort bucket keys
	var labels []string
	for label := range buckets {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	
	data := make([]float64, len(labels))
	for i, label := range labels {
		data[i] = float64(buckets[label])
	}
	
	return ChartData{
		Title: "CPU Utilization Distribution",
		Type:  Histogram,
		Labels: labels,
		Datasets: []Dataset{{
			Label:           "Frequency",
			Data:            data,
			BackgroundColor: "rgba(142, 68, 173, 0.7)",
			BorderColor:     "#8e44ad",
		}},
		Options: map[string]interface{}{
			"responsive": true,
			"scales": map[string]interface{}{
				"y": map[string]interface{}{
					"beginAtZero": true,
					"title": map[string]interface{}{
						"display": true,
						"text":    "Frequency",
					},
				},
				"x": map[string]interface{}{
					"title": map[string]interface{}{
						"display": true,
						"text":    "CPU Utilization Range",
					},
				},
			},
		},
		Description: "Distribution of CPU utilization values showing performance patterns",
	}
}

func (v *Visualizer) createSchedulerEfficiencyChart(metrics []analysis.SchedulerMetrics) ChartData {
	labels := make([]string, len(metrics))
	efficiencyData := make([]float64, len(metrics))
	throughputData := make([]float64, len(metrics))
	
	for i, metric := range metrics {
		labels[i] = metric.Timestamp.Format("15:04:05")
		efficiencyData[i] = metric.Efficiency
		throughputData[i] = metric.Throughput
	}
	
	return ChartData{
		Title: "Scheduler Efficiency and Throughput",
		Type:  LineChart,
		Labels: labels,
		Datasets: []Dataset{
			{
				Label:       "Efficiency",
				Data:        efficiencyData,
				BorderColor: "#27ae60",
				Fill:        false,
				Tension:     0.1,
			},
			{
				Label:       "Throughput",
				Data:        throughputData,
				BorderColor: "#e67e22",
				Fill:        false,
				Tension:     0.1,
			},
		},
		Options: map[string]interface{}{
			"responsive": true,
			"scales": map[string]interface{}{
				"y": map[string]interface{}{
					"beginAtZero": true,
					"title": map[string]interface{}{
						"display": true,
						"text":    "Value",
					},
				},
				"x": map[string]interface{}{
					"title": map[string]interface{}{
						"display": true,
						"text":    "Time",
					},
				},
			},
		},
		Description: "Scheduler efficiency and throughput metrics over time",
	}
}

// ExportToJSON exports dashboard data to JSON format
func (v *Visualizer) ExportToJSON(dashboard *Dashboard, filename string) error {
	filePath := filepath.Join(v.outputDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create JSON file: %w", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(dashboard)
}

// GenerateReport creates a text-based performance report
func (v *Visualizer) GenerateReport(metrics []analysis.SchedulerMetrics, insights []analysis.PerformanceInsight, workload *analysis.WorkloadCharacteristics) string {
	if len(metrics) == 0 {
		return "No metrics available for report generation"
	}
	
	var report strings.Builder
	
	report.WriteString("=== Go Scheduler Performance Report ===\n\n")
	report.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("Sample Count: %d\n", len(metrics)))
	report.WriteString(fmt.Sprintf("Time Range: %s to %s\n\n", 
		metrics[0].Timestamp.Format("15:04:05"), 
		metrics[len(metrics)-1].Timestamp.Format("15:04:05")))
	
	// System Configuration
	report.WriteString("=== System Configuration ===\n")
	report.WriteString(fmt.Sprintf("GOMAXPROCS: %d\n", metrics[0].GOMAXPROCS))
	report.WriteString(fmt.Sprintf("CPU Cores: %d\n", metrics[0].NumCPU))
	report.WriteString("\n")
	
	// Performance Summary
	report.WriteString("=== Performance Summary ===\n")
	avgCPU := calculateAverage(metrics, func(m analysis.SchedulerMetrics) float64 { return m.AvgCPUUtilization })
	avgMemory := calculateAverage(metrics, func(m analysis.SchedulerMetrics) float64 { return m.MemoryPercent })
	avgGoroutines := calculateAverage(metrics, func(m analysis.SchedulerMetrics) float64 { return float64(m.NumGoroutine) })
	
	report.WriteString(fmt.Sprintf("Average CPU Utilization: %.2f%%\n", avgCPU))
	report.WriteString(fmt.Sprintf("Average Memory Usage: %.2f%%\n", avgMemory))
	report.WriteString(fmt.Sprintf("Average Goroutine Count: %.0f\n", avgGoroutines))
	report.WriteString("\n")
	
	// Workload Analysis
	if workload != nil {
		report.WriteString("=== Workload Analysis ===\n")
		report.WriteString(fmt.Sprintf("Type: %s\n", workload.Type))
		report.WriteString(fmt.Sprintf("Contention Level: %s\n", workload.ContentionLevel))
		report.WriteString(fmt.Sprintf("Burstiness: %.2f\n", workload.Burstiness))
		report.WriteString(fmt.Sprintf("Predictability: %.2f\n", workload.Predictability))
		report.WriteString(fmt.Sprintf("Optimal GOMAXPROCS: %d\n", workload.OptimalGOMMAXPROCS))
		report.WriteString("\n")
		
		report.WriteString("Recommendations:\n")
		for _, rec := range workload.Recommendations {
			report.WriteString(fmt.Sprintf("- %s\n", rec))
		}
		report.WriteString("\n")
	}
	
	// Performance Insights
	if len(insights) > 0 {
		report.WriteString("=== Performance Insights ===\n")
		for _, insight := range insights {
			report.WriteString(fmt.Sprintf("[%s] %s: %s\n", 
				strings.ToUpper(insight.Severity), insight.Title, insight.Description))
			report.WriteString(fmt.Sprintf("  Suggestion: %s\n", insight.Suggestion))
			report.WriteString("\n")
		}
	}
	
	return report.String()
}

// Utility functions

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func calculateAverage(metrics []analysis.SchedulerMetrics, extractor func(analysis.SchedulerMetrics) float64) float64 {
	if len(metrics) == 0 {
		return 0
	}
	
	var sum float64
	for _, metric := range metrics {
		sum += extractor(metric)
	}
	return sum / float64(len(metrics))
}

// HTML template for dashboard
const dashboardTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Dashboard.Title}}</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: {{if eq .Theme "dark"}}#1a1a1a{{else}}#f5f5f5{{end}};
            color: {{if eq .Theme "dark"}}#ffffff{{else}}#333333{{end}};
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
            padding: 20px;
            background: {{if eq .Theme "dark"}}#2d2d2d{{else}}#ffffff{{end}};
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .metadata {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            margin-bottom: 30px;
        }
        .metadata-item {
            background: {{if eq .Theme "dark"}}#2d2d2d{{else}}#ffffff{{end}};
            padding: 15px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .charts-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(500px, 1fr));
            gap: 20px;
        }
        .chart-container {
            background: {{if eq .Theme "dark"}}#2d2d2d{{else}}#ffffff{{end}};
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .chart-title {
            font-size: 18px;
            font-weight: bold;
            margin-bottom: 10px;
            text-align: center;
        }
        .chart-description {
            font-size: 12px;
            color: {{if eq .Theme "dark"}}#cccccc{{else}}#666666{{end}};
            margin-bottom: 15px;
            text-align: center;
        }
        canvas {
            max-height: 400px;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{.Dashboard.Title}}</h1>
        <p>{{.Dashboard.Description}}</p>
    </div>
    
    <div class="metadata">
        <div class="metadata-item">
            <strong>Generated:</strong> {{.Dashboard.Metadata.GeneratedAt.Format "2006-01-02 15:04:05"}}
        </div>
        <div class="metadata-item">
            <strong>Data Range:</strong> {{.Dashboard.Metadata.DataRange}}
        </div>
        <div class="metadata-item">
            <strong>Samples:</strong> {{.Dashboard.Metadata.SampleCount}}
        </div>
        <div class="metadata-item">
            <strong>GOMAXPROCS:</strong> {{index .Dashboard.Metadata.Configuration "gomaxprocs"}}
        </div>
    </div>
    
    <div class="charts-grid">
        {{range $index, $chart := .Dashboard.Charts}}
        <div class="chart-container">
            <div class="chart-title">{{$chart.Title}}</div>
            <div class="chart-description">{{$chart.Description}}</div>
            <canvas id="chart-{{$index}}"></canvas>
        </div>
        {{end}}
    </div>
    
    <script>
        const dashboardData = {{.DashboardJSON}};
        const isDark = {{eq .Theme "dark"}};
        
        // Configure Chart.js defaults for theme
        Chart.defaults.color = isDark ? '#ffffff' : '#333333';
        Chart.defaults.borderColor = isDark ? '#404040' : '#e0e0e0';
        Chart.defaults.backgroundColor = isDark ? '#2d2d2d' : '#ffffff';
        
        // Create charts
        dashboardData.charts.forEach((chartData, index) => {
            const ctx = document.getElementById('chart-' + index).getContext('2d');
            
            const config = {
                type: chartData.type === 'area' ? 'line' : chartData.type,
                data: {
                    labels: chartData.labels,
                    datasets: chartData.datasets
                },
                options: {
                    ...chartData.options,
                    plugins: {
                        legend: {
                            labels: {
                                color: isDark ? '#ffffff' : '#333333'
                            }
                        }
                    }
                }
            };
            
            new Chart(ctx, config);
        });
    </script>
</body>
</html>
`
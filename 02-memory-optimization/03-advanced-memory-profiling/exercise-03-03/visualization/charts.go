package visualization

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"

	"github.com/go-perf/02-memory-optimization/03-advanced-memory-profiling/exercise-03-03/analyzer"
	"github.com/go-perf/02-memory-optimization/03-advanced-memory-profiling/exercise-03-03/patterns"
)

// ChartConfig holds configuration for chart generation
type ChartConfig struct {
	OutputDir string
	Width     string
	Height    string
	Theme     string
}

// ChartGenerator generates various charts for allocation analysis
type ChartGenerator struct {
	config *ChartConfig
}

// NewChartGenerator creates a new chart generator with default config
func NewChartGenerator() *ChartGenerator {
	return NewChartGeneratorWithConfig(&ChartConfig{
		Width:  "1200px",
		Height: "600px",
		Theme:  types.ThemeWesteros,
	})
}

// NewChartGeneratorWithConfig creates a new chart generator with custom config
func NewChartGeneratorWithConfig(config *ChartConfig) *ChartGenerator {
	if config.Width == "" {
		config.Width = "1200px"
	}
	if config.Height == "" {
		config.Height = "600px"
	}
	if config.Theme == "" {
		config.Theme = types.ThemeWesteros
	}

	return &ChartGenerator{
		config: config,
	}
}

// GenerateAllocationTimelineChart generates a timeline chart of allocation metrics
func (cg *ChartGenerator) GenerateAllocationTimelineChart(metrics []analyzer.AllocationMetrics) (string, error) {
	if len(metrics) == 0 {
		return "", fmt.Errorf("no metrics data available")
	}

	// Sort metrics by timestamp
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Timestamp.Before(metrics[j].Timestamp)
	})

	// Create line chart
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  cg.config.Theme,
			Width:  cg.config.Width,
			Height: cg.config.Height,
		}),
		charts.WithTitleOpts(opts.Title{
			Title:    "Allocation Timeline",
			Subtitle: "Memory allocation metrics over time",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Time",
			Type: "time",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Rate/Size",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: opts.Bool(true),
		}),
	)

	// Prepare data
	timeLabels := make([]string, len(metrics))
	allocationRates := make([]opts.LineData, len(metrics))
	heapSizes := make([]opts.LineData, len(metrics))
	objectCounts := make([]opts.LineData, len(metrics))

	for i, metric := range metrics {
		timeLabels[i] = metric.Timestamp.Format("15:04:05")
		allocationRates[i] = opts.LineData{Value: float64(metric.BytesPerSecond)}
		heapSizes[i] = opts.LineData{Value: float64(metric.HeapSize) / 1024 / 1024} // Convert to MB
		objectCounts[i] = opts.LineData{Value: float64(metric.ObjectsPerSecond)}
	}

	// Add series
	line.SetXAxis(timeLabels).
		AddSeries("Allocation Rate (allocs/s)", allocationRates).
		AddSeries("Heap Size (MB)", heapSizes).
		AddSeries("Object Count", objectCounts)

	// Return chart as HTML string
	return cg.renderChart(line)
}

// GeneratePatternDistributionChart generates a pie chart of pattern distribution
func (cg *ChartGenerator) GeneratePatternDistributionChart(patterns []patterns.DetectedPattern) (string, error) {
	if len(patterns) == 0 {
		return "", fmt.Errorf("no patterns data available")
	}

	// Count patterns by type
	patternCounts := make(map[string]int)
	for _, pattern := range patterns {
		patternCounts[string(pattern.Type)]++
	}

	// Create pie chart
	pie := charts.NewPie()
	pie.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  cg.config.Theme,
			Width:  cg.config.Width,
			Height: cg.config.Height,
		}),
		charts.WithTitleOpts(opts.Title{
			Title:    "Pattern Distribution",
			Subtitle: "Distribution of detected allocation patterns",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: opts.Bool(true),
		}),
	)

	// Prepare data
	pieData := make([]opts.PieData, 0, len(patternCounts))
	for patternType, count := range patternCounts {
		pieData = append(pieData, opts.PieData{
			Name:  patternType,
			Value: count,
		})
	}

	// Add series
	pie.AddSeries("Patterns", pieData)

	// Return chart as HTML string
	return cg.renderChart(pie)
}

// GenerateHotspotBarChart generates a bar chart of allocation hotspots
func (cg *ChartGenerator) GenerateHotspotBarChart(hotspots []*analyzer.HotspotInfo) (string, error) {
	if len(hotspots) == 0 {
		return "", fmt.Errorf("no hotspots data available")
	}

	// Sort hotspots by allocation rate (descending)
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].AllocationRate > hotspots[j].AllocationRate
	})

	// Take top 20 hotspots
	maxHotspots := 20
	if len(hotspots) < maxHotspots {
		maxHotspots = len(hotspots)
	}
	topHotspots := hotspots[:maxHotspots]

	// Create bar chart
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  cg.config.Theme,
			Width:  cg.config.Width,
			Height: cg.config.Height,
		}),
		charts.WithTitleOpts(opts.Title{
			Title:    "Top Allocation Hotspots",
			Subtitle: "Functions with highest allocation rates",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Function",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Allocation Rate (allocs/s)",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: opts.Bool(true),
		}),
	)

	// Prepare data
	functionNames := make([]string, len(topHotspots))
	allocationRates := make([]opts.BarData, len(topHotspots))

	for i, hotspot := range topHotspots {
		// Truncate long function names
		funcName := hotspot.Function
		if len(funcName) > 30 {
			funcName = funcName[:27] + "..."
		}
		functionNames[i] = funcName
		allocationRates[i] = opts.BarData{
			Value: hotspot.AllocationRate,
			ItemStyle: &opts.ItemStyle{
				Color: cg.getSeverityColor(hotspot.Severity),
			},
		}
	}

	// Add series
	bar.SetXAxis(functionNames).
		AddSeries("Allocation Rate", allocationRates)

	// Return chart as HTML string
	return cg.renderChart(bar)
}

// GenerateGCImpactChart generates a chart showing GC impact over time
func (cg *ChartGenerator) GenerateGCImpactChart(metrics []analyzer.AllocationMetrics) (string, error) {
	if len(metrics) == 0 {
		return "", fmt.Errorf("no metrics data available")
	}

	// Sort metrics by timestamp
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Timestamp.Before(metrics[j].Timestamp)
	})

	// Create combination chart (line + bar)
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  cg.config.Theme,
			Width:  cg.config.Width,
			Height: cg.config.Height,
		}),
		charts.WithTitleOpts(opts.Title{
			Title:    "GC Impact Analysis",
			Subtitle: "Garbage collection frequency and pause times",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Time",
			Type: "time",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "GC Metrics",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: opts.Bool(true),
		}),
	)

	// Prepare data
	timeLabels := make([]string, len(metrics))
	gcCounts := make([]opts.LineData, len(metrics))
	gcPauseTimes := make([]opts.LineData, len(metrics))
	heapSizes := make([]opts.LineData, len(metrics))

	for i, metric := range metrics {
		timeLabels[i] = metric.Timestamp.Format("15:04:05")
		gcCounts[i] = opts.LineData{Value: metric.NumGC}
		// GCPauses is already a uint64 representing total pause time
		gcPauseTimes[i] = opts.LineData{Value: float64(metric.GCPauses) / 1000000}  // Convert nanoseconds to ms
		heapSizes[i] = opts.LineData{Value: float64(metric.HeapSize) / 1024 / 1024} // Convert to MB
	}

	// Add series
	line.SetXAxis(timeLabels).
		AddSeries("GC Count", gcCounts).
		AddSeries("Total Pause Time (ms)", gcPauseTimes).
		AddSeries("Heap Size (MB)", heapSizes)

	// Return chart as HTML string
	return cg.renderChart(line)
}

// GeneratePatternSeverityChart generates a stacked bar chart of pattern severity
func (cg *ChartGenerator) GeneratePatternSeverityChart(patterns []patterns.DetectedPattern) (string, error) {
	if len(patterns) == 0 {
		return "", fmt.Errorf("no patterns data available")
	}

	// Group patterns by type and severity
	patternSeverity := make(map[string]map[string]int)
	for _, pattern := range patterns {
		patternTypeStr := string(pattern.Type)
		if patternSeverity[patternTypeStr] == nil {
			patternSeverity[patternTypeStr] = make(map[string]int)
		}
		patternSeverity[patternTypeStr][pattern.Severity]++
	}

	// Create stacked bar chart
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  cg.config.Theme,
			Width:  cg.config.Width,
			Height: cg.config.Height,
		}),
		charts.WithTitleOpts(opts.Title{
			Title:    "Pattern Severity Distribution",
			Subtitle: "Severity levels by pattern type",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Pattern Type",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Count",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: opts.Bool(true),
		}),
	)

	// Prepare data
	patternTypes := make([]string, 0, len(patternSeverity))
	for patternType := range patternSeverity {
		patternTypes = append(patternTypes, patternType)
	}
	sort.Strings(patternTypes)

	severityLevels := []string{"low", "medium", "high", "critical"}
	severityData := make(map[string][]opts.BarData)

	for _, severity := range severityLevels {
		data := make([]opts.BarData, len(patternTypes))
		for i, patternType := range patternTypes {
			count := patternSeverity[patternType][severity]
			data[i] = opts.BarData{
				Value: count,
				ItemStyle: &opts.ItemStyle{
					Color: cg.getSeverityColor(severity),
				},
			}
		}
		severityData[severity] = data
	}

	// Add series
	bar.SetXAxis(patternTypes)
	for _, severity := range severityLevels {
		bar.AddSeries(severity, severityData[severity], charts.WithBarChartOpts(opts.BarChart{
			Stack: "severity",
		}))
	}

	// Return chart as HTML string
	return cg.renderChart(bar)
}

// GenerateAllocationHeatmap generates a heatmap of allocation activity
func (cg *ChartGenerator) GenerateAllocationHeatmap(metrics []analyzer.AllocationMetrics) (string, error) {
	if len(metrics) == 0 {
		return "", fmt.Errorf("no metrics data available")
	}

	// Sort metrics by timestamp
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Timestamp.Before(metrics[j].Timestamp)
	})

	// Group metrics by hour and minute for heatmap
	heatmapData := make([]opts.HeatMapData, 0)
	for _, metric := range metrics {
		hour := metric.Timestamp.Hour()
		minute := metric.Timestamp.Minute()
		intensity := float64(metric.BytesPerSecond)

		heatmapData = append(heatmapData, opts.HeatMapData{
			Value: [3]interface{}{minute, hour, intensity},
		})
	}

	// Create heatmap
	heatmap := charts.NewHeatMap()
	heatmap.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  cg.config.Theme,
			Width:  cg.config.Width,
			Height: cg.config.Height,
		}),
		charts.WithTitleOpts(opts.Title{
			Title:    "Allocation Activity Heatmap",
			Subtitle: "Allocation intensity by time of day",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Minute",
			Type: "value",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Hour",
			Type: "value",
		}),
		charts.WithVisualMapOpts(opts.VisualMap{
			Show:       opts.Bool(true),
			Calculable: opts.Bool(true),
			Min:        0,
			Max:        float32(cg.getMaxBytesPerSecond(metrics)),
			InRange: &opts.VisualMapInRange{
				Color: []string{"#50a3ba", "#eac736", "#d94e5d"},
			},
		}),
	)

	// Add series
	heatmap.AddSeries("Allocation Rate", heatmapData)

	// Return chart as HTML string
	return cg.renderChart(heatmap)
}

// GenerateComparisonChart generates a comparison chart between two datasets
func (cg *ChartGenerator) GenerateComparisonChart(baseline, current []analyzer.AllocationMetrics) (string, error) {
	if len(baseline) == 0 || len(current) == 0 {
		return "", fmt.Errorf("insufficient data for comparison")
	}

	// Create line chart for comparison
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Theme:  cg.config.Theme,
			Width:  cg.config.Width,
			Height: cg.config.Height,
		}),
		charts.WithTitleOpts(opts.Title{
			Title:    "Baseline vs Current Comparison",
			Subtitle: "Allocation metrics comparison",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Time Index",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Allocation Rate",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: opts.Bool(true),
		}),
	)

	// Normalize data length
	maxLen := len(baseline)
	if len(current) > maxLen {
		maxLen = len(current)
	}

	// Prepare data
	timeLabels := make([]string, maxLen)
	baselineRates := make([]opts.LineData, maxLen)
	currentRates := make([]opts.LineData, maxLen)

	for i := 0; i < maxLen; i++ {
		timeLabels[i] = fmt.Sprintf("%d", i)

		if i < len(baseline) {
			baselineRates[i] = opts.LineData{Value: float64(baseline[i].BytesPerSecond)}
		} else {
			baselineRates[i] = opts.LineData{Value: 0}
		}

		if i < len(current) {
			currentRates[i] = opts.LineData{Value: float64(current[i].BytesPerSecond)}
		} else {
			currentRates[i] = opts.LineData{Value: 0}
		}
	}

	// Add series
	line.SetXAxis(timeLabels).
		AddSeries("Baseline", baselineRates).
		AddSeries("Current", currentRates)

	// Return chart as HTML string
	return cg.renderChart(line)
}

// GenerateDashboard generates an HTML dashboard with all charts
func (cg *ChartGenerator) GenerateDashboard() error {
	dashboardHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Allocation Pattern Analysis Dashboard</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .chart-container {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(600px, 1fr));
            gap: 20px;
            margin-bottom: 20px;
        }
        .chart-frame {
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .chart-title {
            background: #333;
            color: white;
            padding: 10px;
            margin: 0;
            font-size: 16px;
        }
        iframe {
            width: 100%;
            height: 600px;
            border: none;
        }
        .footer {
            text-align: center;
            margin-top: 30px;
            color: #666;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>Allocation Pattern Analysis Dashboard</h1>
        <p>Generated on ` + time.Now().Format("2006-01-02 15:04:05") + `</p>
    </div>
    
    <div class="chart-container">
        <div class="chart-frame">
            <h3 class="chart-title">Allocation Timeline</h3>
            <iframe src="allocation_timeline.html"></iframe>
        </div>
        
        <div class="chart-frame">
            <h3 class="chart-title">Pattern Distribution</h3>
            <iframe src="pattern_distribution.html"></iframe>
        </div>
        
        <div class="chart-frame">
            <h3 class="chart-title">Top Hotspots</h3>
            <iframe src="hotspots_bar.html"></iframe>
        </div>
        
        <div class="chart-frame">
            <h3 class="chart-title">GC Impact</h3>
            <iframe src="gc_impact.html"></iframe>
        </div>
        
        <div class="chart-frame">
            <h3 class="chart-title">Pattern Severity</h3>
            <iframe src="pattern_severity.html"></iframe>
        </div>
        
        <div class="chart-frame">
            <h3 class="chart-title">Allocation Heatmap</h3>
            <iframe src="allocation_heatmap.html"></iframe>
        </div>
    </div>
    
    <div class="footer">
        <p>Go Performance Analysis Tool - Allocation Pattern Analysis</p>
    </div>
</body>
</html>
`

	filename := filepath.Join(cg.config.OutputDir, "dashboard.html")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(dashboardHTML)
	return err
}

// Helper functions

// renderChart renders a chart to HTML string
func (cg *ChartGenerator) renderChart(chart interface{}) (string, error) {
	var buf bytes.Buffer

	// Type assertion to get the Render method
	switch c := chart.(type) {
	case *charts.Line:
		if err := c.Render(&buf); err != nil {
			return "", err
		}
	case *charts.Bar:
		if err := c.Render(&buf); err != nil {
			return "", err
		}
	case *charts.Pie:
		if err := c.Render(&buf); err != nil {
			return "", err
		}
	case *charts.HeatMap:
		if err := c.Render(&buf); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported chart type")
	}

	return buf.String(), nil
}

// getSeverityColor returns color based on severity level
func (cg *ChartGenerator) getSeverityColor(severity string) string {
	switch severity {
	case "low":
		return "#52c41a" // Green
	case "medium":
		return "#faad14" // Orange
	case "high":
		return "#fa541c" // Red-orange
	case "critical":
		return "#f5222d" // Red
	default:
		return "#1890ff" // Blue
	}
}

// getMaxBytesPerSecond finds the maximum bytes per second in metrics
func (cg *ChartGenerator) getMaxBytesPerSecond(metrics []analyzer.AllocationMetrics) float64 {
	maxRate := 0.0
	for _, metric := range metrics {
		if float64(metric.BytesPerSecond) > maxRate {
			maxRate = float64(metric.BytesPerSecond)
		}
	}
	return maxRate
}

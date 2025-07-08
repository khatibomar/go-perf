// reporter/reporter.go
package reporter

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"exercise-04-01/loadtest"
)

type Reporter interface {
	GenerateReport(result *loadtest.LoadTestResult, config ReportConfig) error
}

type ReportConfig struct {
	OutputPath   string
	Format       string // "json", "html", "csv"
	Title        string
	Description  string
	IncludeCharts bool
}

type JSONReporter struct{}

type HTMLReporter struct{}

type CSVReporter struct{}

type ReportData struct {
	Title       string
	Description string
	Timestamp   time.Time
	Result      *loadtest.LoadTestResult
	Summary     TestSummary
}

type TestSummary struct {
	Duration        string
	TotalRequests   int64
	SuccessRate     float64
	ErrorRate       float64
	Throughput      float64
	AvgLatency      string
	P95Latency      string
	P99Latency      string
	TopErrors       []ErrorSummary
	ScenarioSummary []ScenarioSummary
}

type ErrorSummary struct {
	Error string
	Count int64
	Rate  float64
}

type ScenarioSummary struct {
	Name        string
	Users       int
	Requests    int64
	SuccessRate float64
	Throughput  float64
	AvgLatency  string
}

func NewReporter(format string) Reporter {
	switch strings.ToLower(format) {
	case "json":
		return &JSONReporter{}
	case "html":
		return &HTMLReporter{}
	case "csv":
		return &CSVReporter{}
	default:
		return &JSONReporter{}
	}
}

func (r *JSONReporter) GenerateReport(result *loadtest.LoadTestResult, config ReportConfig) error {
	reportData := r.prepareReportData(result, config)

	data, err := json.MarshalIndent(reportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report data: %w", err)
	}

	outputFile := filepath.Join(config.OutputPath, "load_test_report.json")
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	fmt.Printf("JSON report generated: %s\n", outputFile)
	return nil
}

func (r *HTMLReporter) GenerateReport(result *loadtest.LoadTestResult, config ReportConfig) error {
	reportData := r.prepareReportData(result, config)

	tmpl := template.Must(template.New("report").Parse(htmlTemplate))

	outputFile := filepath.Join(config.OutputPath, "load_test_report.html")
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, reportData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("HTML report generated: %s\n", outputFile)
	return nil
}

func (r *CSVReporter) GenerateReport(result *loadtest.LoadTestResult, config ReportConfig) error {
	outputFile := filepath.Join(config.OutputPath, "load_test_report.csv")
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	// Write CSV header
	header := "Metric,Value\n"
	if _, err := file.WriteString(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write metrics
	metrics := []string{
		fmt.Sprintf("Duration,%v", result.EndTime.Sub(result.StartTime)),
		fmt.Sprintf("Total Requests,%d", result.TotalRequests),
		fmt.Sprintf("Success Requests,%d", result.SuccessRequests),
		fmt.Sprintf("Failed Requests,%d", result.FailedRequests),
		fmt.Sprintf("Throughput,%.2f", result.Throughput),
		fmt.Sprintf("Min Latency,%v", result.Latencies.Min),
		fmt.Sprintf("Max Latency,%v", result.Latencies.Max),
		fmt.Sprintf("Mean Latency,%v", result.Latencies.Mean),
		fmt.Sprintf("P50 Latency,%v", result.Latencies.P50),
		fmt.Sprintf("P90 Latency,%v", result.Latencies.P90),
		fmt.Sprintf("P95 Latency,%v", result.Latencies.P95),
		fmt.Sprintf("P99 Latency,%v", result.Latencies.P99),
		fmt.Sprintf("P999 Latency,%v", result.Latencies.P999),
	}

	for _, metric := range metrics {
		if _, err := file.WriteString(metric + "\n"); err != nil {
			return fmt.Errorf("failed to write CSV data: %w", err)
		}
	}

	// Write scenario data if available
	if len(result.Scenarios) > 0 {
		if _, err := file.WriteString("\nScenario,Users,Total Requests,Success Requests,Failed Requests,Success Rate,Avg Latency,Throughput\n"); err != nil {
			return fmt.Errorf("failed to write scenario header: %w", err)
		}

		for name, scenario := range result.Scenarios {
			successRate := float64(scenario.SuccessRequests) / float64(scenario.TotalRequests) * 100
			scenarioData := fmt.Sprintf("%s,%d,%d,%d,%d,%.2f%%,%.2fms,%.2f\n",
				name,
				scenario.ExecutedUsers,
				scenario.TotalRequests,
				scenario.SuccessRequests,
				scenario.FailedRequests,
				successRate,
				float64(scenario.AvgLatency.Nanoseconds())/1e6,
				scenario.Throughput,
			)
			if _, err := file.WriteString(scenarioData); err != nil {
				return fmt.Errorf("failed to write scenario data: %w", err)
			}
		}
	}

	fmt.Printf("CSV report generated: %s\n", outputFile)
	return nil
}

func (r *JSONReporter) prepareReportData(result *loadtest.LoadTestResult, config ReportConfig) *ReportData {
	return prepareReportData(result, config)
}

func (r *HTMLReporter) prepareReportData(result *loadtest.LoadTestResult, config ReportConfig) *ReportData {
	return prepareReportData(result, config)
}

func prepareReportData(result *loadtest.LoadTestResult, config ReportConfig) *ReportData {
	duration := result.EndTime.Sub(result.StartTime)
	successRate := float64(result.SuccessRequests) / float64(result.TotalRequests) * 100
	errorRate := float64(result.FailedRequests) / float64(result.TotalRequests) * 100

	// Prepare top errors
	topErrors := make([]ErrorSummary, 0)
	for errMsg, count := range result.Errors {
		rate := float64(count) / float64(result.TotalRequests) * 100
		topErrors = append(topErrors, ErrorSummary{
			Error: errMsg,
			Count: count,
			Rate:  rate,
		})
	}

	// Prepare scenario summary
	scenarioSummary := make([]ScenarioSummary, 0)
	for name, scenario := range result.Scenarios {
		scenarioSuccessRate := float64(scenario.SuccessRequests) / float64(scenario.TotalRequests) * 100
		scenarioSummary = append(scenarioSummary, ScenarioSummary{
			Name:        name,
			Users:       scenario.ExecutedUsers,
			Requests:    scenario.TotalRequests,
			SuccessRate: scenarioSuccessRate,
			Throughput:  scenario.Throughput,
			AvgLatency:  scenario.AvgLatency.String(),
		})
	}

	return &ReportData{
		Title:       config.Title,
		Description: config.Description,
		Timestamp:   time.Now(),
		Result:      result,
		Summary: TestSummary{
			Duration:        duration.String(),
			TotalRequests:   result.TotalRequests,
			SuccessRate:     successRate,
			ErrorRate:       errorRate,
			Throughput:      result.Throughput,
			AvgLatency:      result.Latencies.Mean.String(),
			P95Latency:      result.Latencies.P95.String(),
			P99Latency:      result.Latencies.P99.String(),
			TopErrors:       topErrors,
			ScenarioSummary: scenarioSummary,
		},
	}
}

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}} - Load Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f4f4f4; padding: 20px; border-radius: 5px; }
        .summary { display: flex; flex-wrap: wrap; gap: 20px; margin: 20px 0; }
        .metric { background-color: #e9e9e9; padding: 15px; border-radius: 5px; min-width: 200px; }
        .metric h3 { margin: 0 0 10px 0; color: #333; }
        .metric .value { font-size: 24px; font-weight: bold; color: #007acc; }
        .section { margin: 30px 0; }
        .section h2 { color: #333; border-bottom: 2px solid #007acc; padding-bottom: 5px; }
        table { width: 100%; border-collapse: collapse; margin: 10px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .error { color: #d32f2f; }
        .success { color: #388e3c; }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{.Title}}</h1>
        <p>{{.Description}}</p>
        <p><strong>Generated:</strong> {{.Timestamp.Format "2006-01-02 15:04:05"}}</p>
    </div>

    <div class="section">
        <h2>Test Summary</h2>
        <div class="summary">
            <div class="metric">
                <h3>Duration</h3>
                <div class="value">{{.Summary.Duration}}</div>
            </div>
            <div class="metric">
                <h3>Total Requests</h3>
                <div class="value">{{.Summary.TotalRequests}}</div>
            </div>
            <div class="metric">
                <h3>Success Rate</h3>
                <div class="value success">{{printf "%.2f%%" .Summary.SuccessRate}}</div>
            </div>
            <div class="metric">
                <h3>Error Rate</h3>
                <div class="value error">{{printf "%.2f%%" .Summary.ErrorRate}}</div>
            </div>
            <div class="metric">
                <h3>Throughput</h3>
                <div class="value">{{printf "%.2f req/s" .Summary.Throughput}}</div>
            </div>
            <div class="metric">
                <h3>Avg Latency</h3>
                <div class="value">{{.Summary.AvgLatency}}</div>
            </div>
            <div class="metric">
                <h3>P95 Latency</h3>
                <div class="value">{{.Summary.P95Latency}}</div>
            </div>
            <div class="metric">
                <h3>P99 Latency</h3>
                <div class="value">{{.Summary.P99Latency}}</div>
            </div>
        </div>
    </div>

    <div class="section">
        <h2>Latency Distribution</h2>
        <table>
            <tr>
                <th>Percentile</th>
                <th>Latency</th>
            </tr>
            <tr><td>Min</td><td>{{.Result.Latencies.Min}}</td></tr>
            <tr><td>P50</td><td>{{.Result.Latencies.P50}}</td></tr>
            <tr><td>P90</td><td>{{.Result.Latencies.P90}}</td></tr>
            <tr><td>P95</td><td>{{.Result.Latencies.P95}}</td></tr>
            <tr><td>P99</td><td>{{.Result.Latencies.P99}}</td></tr>
            <tr><td>P99.9</td><td>{{.Result.Latencies.P999}}</td></tr>
            <tr><td>Max</td><td>{{.Result.Latencies.Max}}</td></tr>
        </table>
    </div>

    {{if .Summary.ScenarioSummary}}
    <div class="section">
        <h2>Scenario Results</h2>
        <table>
            <tr>
                <th>Scenario</th>
                <th>Users</th>
                <th>Requests</th>
                <th>Success Rate</th>
                <th>Throughput</th>
                <th>Avg Latency</th>
            </tr>
            {{range .Summary.ScenarioSummary}}
            <tr>
                <td>{{.Name}}</td>
                <td>{{.Users}}</td>
                <td>{{.Requests}}</td>
                <td>{{printf "%.2f%%" .SuccessRate}}</td>
                <td>{{printf "%.2f req/s" .Throughput}}</td>
                <td>{{.AvgLatency}}</td>
            </tr>
            {{end}}
        </table>
    </div>
    {{end}}

    {{if .Summary.TopErrors}}
    <div class="section">
        <h2>Top Errors</h2>
        <table>
            <tr>
                <th>Error</th>
                <th>Count</th>
                <th>Rate</th>
            </tr>
            {{range .Summary.TopErrors}}
            <tr>
                <td class="error">{{.Error}}</td>
                <td>{{.Count}}</td>
                <td>{{printf "%.2f%%" .Rate}}</td>
            </tr>
            {{end}}
        </table>
    </div>
    {{end}}
</body>
</html>
`
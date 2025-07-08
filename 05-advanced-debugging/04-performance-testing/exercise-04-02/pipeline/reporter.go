// pipeline/reporter.go
package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type HTMLReporter struct {
	config PipelineConfig
}

func NewHTMLReporter(config PipelineConfig) *HTMLReporter {
	return &HTMLReporter{
		config: config,
	}
}

func (r *HTMLReporter) GenerateReport(results []TestResult, baseline PerformanceBaseline) error {
	// Generate HTML report
	if contains(r.config.ReportFormats, "html") {
		if err := r.generateHTMLReport(results, baseline); err != nil {
			return fmt.Errorf("failed to generate HTML report: %w", err)
		}
	}
	
	// Generate JSON report
	if contains(r.config.ReportFormats, "json") {
		if err := r.generateJSONReport(results, baseline); err != nil {
			return fmt.Errorf("failed to generate JSON report: %w", err)
		}
	}
	
	return nil
}

func (r *HTMLReporter) generateHTMLReport(results []TestResult, baseline PerformanceBaseline) error {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>Performance Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .header { text-align: center; margin-bottom: 30px; }
        .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .metric-card { background: #f8f9fa; padding: 20px; border-radius: 8px; border-left: 4px solid #007bff; }
        .metric-value { font-size: 2em; font-weight: bold; color: #007bff; }
        .metric-label { color: #6c757d; margin-top: 5px; }
        .test-results { margin-bottom: 30px; }
        .test-card { background: white; border: 1px solid #dee2e6; border-radius: 8px; margin-bottom: 15px; overflow: hidden; }
        .test-header { background: #f8f9fa; padding: 15px; border-bottom: 1px solid #dee2e6; }
        .test-content { padding: 15px; }
        .status-passed { color: #28a745; }
        .status-failed { color: #dc3545; }
        .status-skipped { color: #ffc107; }
        .regression-warning { background: #fff3cd; border: 1px solid #ffeaa7; padding: 10px; border-radius: 4px; margin: 10px 0; }
        .regression-critical { background: #f8d7da; border: 1px solid #f5c6cb; padding: 10px; border-radius: 4px; margin: 10px 0; }
        .metrics-table { width: 100%; border-collapse: collapse; margin-top: 10px; }
        .metrics-table th, .metrics-table td { border: 1px solid #dee2e6; padding: 8px; text-align: left; }
        .metrics-table th { background-color: #f8f9fa; }
        .baseline-comparison { margin-top: 20px; }
        .comparison-table { width: 100%; border-collapse: collapse; }
        .comparison-table th, .comparison-table td { border: 1px solid #dee2e6; padding: 8px; text-align: center; }
        .comparison-table th { background-color: #f8f9fa; }
        .improvement { color: #28a745; }
        .degradation { color: #dc3545; }
        .stable { color: #6c757d; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Performance Test Report</h1>
            <p><strong>Generated:</strong> {{.Timestamp}}</p>
            <p><strong>Environment:</strong> {{.Environment}}</p>
        </div>
        
        <div class="summary">
            <div class="metric-card">
                <div class="metric-value">{{.TotalTests}}</div>
                <div class="metric-label">Total Tests</div>
            </div>
            <div class="metric-card">
                <div class="metric-value">{{.PassedTests}}</div>
                <div class="metric-label">Passed Tests</div>
            </div>
            <div class="metric-card">
                <div class="metric-value">{{.FailedTests}}</div>
                <div class="metric-label">Failed Tests</div>
            </div>
            <div class="metric-card">
                <div class="metric-value">{{.TotalDuration}}</div>
                <div class="metric-label">Total Duration</div>
            </div>
        </div>
        
        <div class="test-results">
            <h2>Test Results</h2>
            {{range .Results}}
            <div class="test-card">
                <div class="test-header">
                    <h3>{{.TestName}} <span class="status-{{.Status}}">{{.Status | title}}</span></h3>
                    <p>Duration: {{.Duration}} | Type: {{.Type}}</p>
                </div>
                <div class="test-content">
                    {{if .Regression}}
                    <div class="regression-{{.Regression.Severity}}">
                        <strong>⚠️ Performance Regression Detected ({{.Regression.Severity | title}})</strong>
                        <p>{{.Regression.Recommendation}}</p>
                        <p>Affected Metrics: {{join .Regression.AffectedMetrics ", "}}</p>
                    </div>
                    {{end}}
                    
                    {{if .Errors}}
                    <div class="errors">
                        <h4>Errors:</h4>
                        <ul>
                        {{range .Errors}}
                            <li>{{.}}</li>
                        {{end}}
                        </ul>
                    </div>
                    {{end}}
                    
                    <h4>Metrics:</h4>
                    <table class="metrics-table">
                        <thead>
                            <tr><th>Metric</th><th>Value</th><th>Unit</th></tr>
                        </thead>
                        <tbody>
                        {{range $key, $value := .Metrics}}
                            <tr><td>{{$key}}</td><td>{{$value}}</td><td>{{getUnit $key}}</td></tr>
                        {{end}}
                        </tbody>
                    </table>
                </div>
            </div>
            {{end}}
        </div>
        
        {{if .HasBaseline}}
        <div class="baseline-comparison">
            <h2>Baseline Comparison</h2>
            <table class="comparison-table">
                <thead>
                    <tr><th>Metric</th><th>Current</th><th>Baseline</th><th>Change</th><th>Status</th></tr>
                </thead>
                <tbody>
                {{range .BaselineComparisons}}
                    <tr>
                        <td>{{.Metric}}</td>
                        <td>{{.Current}}</td>
                        <td>{{.Baseline}}</td>
                        <td class="{{.ChangeClass}}">{{.ChangePercent}}%</td>
                        <td class="{{.StatusClass}}">{{.Status}}</td>
                    </tr>
                {{end}}
                </tbody>
            </table>
        </div>
        {{end}}
        
        <div class="footer">
            <p><em>Report generated by Performance Testing Pipeline</em></p>
        </div>
    </div>
</body>
</html>`
	
	// Prepare template data
	data := r.prepareTemplateData(results, baseline)
	
	// Parse and execute template
	t, err := template.New("report").Funcs(template.FuncMap{
		"title": strings.Title,
		"join":  strings.Join,
		"getUnit": func(metric string) string {
			units := map[string]string{
				"throughput":   "req/s",
				"avg_latency":  "seconds",
				"p95_latency":  "seconds",
				"p99_latency":  "seconds",
				"memory_usage": "MB",
				"error_rate":   "%",
			}
			if unit, exists := units[metric]; exists {
				return unit
			}
			return ""
		},
	}).Parse(tmpl)
	if err != nil {
		return err
	}
	
	// Create output file
	reportPath := filepath.Join(r.config.ResultsPath, fmt.Sprintf("performance_report_%s.html", time.Now().Format("20060102_150405")))
	file, err := os.Create(reportPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	return t.Execute(file, data)
}

func (r *HTMLReporter) generateJSONReport(results []TestResult, baseline PerformanceBaseline) error {
	report := map[string]interface{}{
		"timestamp":   time.Now(),
		"environment": r.config.Environment,
		"results":     results,
		"baseline":    baseline,
		"summary": map[string]interface{}{
			"total_tests":  len(results),
			"passed_tests": r.countTestsByStatus(results, "passed"),
			"failed_tests": r.countTestsByStatus(results, "failed"),
		},
	}
	
	reportPath := filepath.Join(r.config.ResultsPath, fmt.Sprintf("performance_report_%s.json", time.Now().Format("20060102_150405")))
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(reportPath, data, 0644)
}

func (r *HTMLReporter) prepareTemplateData(results []TestResult, baseline PerformanceBaseline) map[string]interface{} {
	totalDuration := time.Duration(0)
	for _, result := range results {
		totalDuration += result.Duration
	}
	
	data := map[string]interface{}{
		"Timestamp":     time.Now().Format("2006-01-02 15:04:05"),
		"Environment":   r.config.Environment,
		"TotalTests":    len(results),
		"PassedTests":   r.countTestsByStatus(results, "passed"),
		"FailedTests":   r.countTestsByStatus(results, "failed"),
		"TotalDuration": totalDuration.String(),
		"Results":       r.enrichResults(results),
		"HasBaseline":   len(baseline.Metrics) > 0,
	}
	
	if len(baseline.Metrics) > 0 {
		data["BaselineComparisons"] = r.prepareBaselineComparisons(results, baseline)
	}
	
	return data
}

func (r *HTMLReporter) enrichResults(results []TestResult) []map[string]interface{} {
	enriched := make([]map[string]interface{}, len(results))
	for i, result := range results {
		enriched[i] = map[string]interface{}{
			"TestName":   result.TestName,
			"Status":     result.Status,
			"Duration":   result.Duration.String(),
			"Type":       r.getTestType(result.TestName),
			"Metrics":    result.Metrics,
			"Errors":     result.Errors,
			"Regression": result.Regression,
		}
	}
	return enriched
}

func (r *HTMLReporter) getTestType(testName string) string {
	if strings.Contains(strings.ToLower(testName), "load") {
		return "Load Test"
	}
	if strings.Contains(strings.ToLower(testName), "stress") {
		return "Stress Test"
	}
	if strings.Contains(strings.ToLower(testName), "spike") {
		return "Spike Test"
	}
	if strings.Contains(strings.ToLower(testName), "volume") {
		return "Volume Test"
	}
	return "Performance Test"
}

func (r *HTMLReporter) prepareBaselineComparisons(results []TestResult, baseline PerformanceBaseline) []map[string]interface{} {
	comparisons := make([]map[string]interface{}, 0)
	
	// Aggregate current metrics from all results
	currentMetrics := make(map[string]float64)
	for _, result := range results {
		for metricName, value := range result.Metrics {
			if floatValue, ok := value.(float64); ok {
				currentMetrics[metricName] = floatValue
			}
		}
	}
	
	for metricName, baselineMetric := range baseline.Metrics {
		if currentValue, exists := currentMetrics[metricName]; exists {
			change := currentValue - baselineMetric.Value
			changePercent := (change / baselineMetric.Value) * 100
			
			var changeClass, statusClass, status string
			if abs(changePercent) <= baselineMetric.Tolerance {
				changeClass = "stable"
				statusClass = "stable"
				status = "Stable"
			} else if changePercent > 0 {
				if metricName == "throughput" {
					changeClass = "improvement"
					statusClass = "improvement"
					status = "Improved"
				} else {
					changeClass = "degradation"
					statusClass = "degradation"
					status = "Degraded"
				}
			} else {
				if metricName == "throughput" {
					changeClass = "degradation"
					statusClass = "degradation"
					status = "Degraded"
				} else {
					changeClass = "improvement"
					statusClass = "improvement"
					status = "Improved"
				}
			}
			
			comparisons = append(comparisons, map[string]interface{}{
				"Metric":        metricName,
				"Current":       fmt.Sprintf("%.2f", currentValue),
				"Baseline":      fmt.Sprintf("%.2f", baselineMetric.Value),
				"ChangePercent": fmt.Sprintf("%.2f", changePercent),
				"ChangeClass":   changeClass,
				"StatusClass":   statusClass,
				"Status":        status,
			})
		}
	}
	
	return comparisons
}

func (r *HTMLReporter) countTestsByStatus(results []TestResult, status string) int {
	count := 0
	for _, result := range results {
		if result.Status == status {
			count++
		}
	}
	return count
}

func (r *HTMLReporter) SendNotification(result TestResult, regression *RegressionAnalysis) error {
	// Send Slack notification
	if r.config.Notifications.Slack.WebhookURL != "" {
		if err := r.sendSlackNotification(result, regression); err != nil {
			return fmt.Errorf("failed to send Slack notification: %w", err)
		}
	}
	
	// Send webhook notification
	if r.config.Notifications.Webhook.URL != "" {
		if err := r.sendWebhookNotification(result, regression); err != nil {
			return fmt.Errorf("failed to send webhook notification: %w", err)
		}
	}
	
	return nil
}

func (r *HTMLReporter) sendSlackNotification(result TestResult, regression *RegressionAnalysis) error {
	var color, title, message string
	
	if regression != nil {
		if regression.Severity == "critical" {
			color = "danger"
			title = "🚨 Critical Performance Regression Detected"
		} else {
			color = "warning"
			title = "⚠️ Performance Regression Detected"
		}
		message = fmt.Sprintf("Test: %s\nSeverity: %s\nAffected Metrics: %s\nRecommendation: %s",
			result.TestName, regression.Severity, strings.Join(regression.AffectedMetrics, ", "), regression.Recommendation)
	} else if result.Status == "failed" {
		color = "danger"
		title = "❌ Performance Test Failed"
		message = fmt.Sprintf("Test: %s\nErrors: %s", result.TestName, strings.Join(result.Errors, ", "))
	} else {
		color = "good"
		title = "✅ Performance Test Passed"
		message = fmt.Sprintf("Test: %s\nDuration: %s", result.TestName, result.Duration.String())
	}
	
	payload := map[string]interface{}{
		"channel":  r.config.Notifications.Slack.Channel,
		"username": r.config.Notifications.Slack.Username,
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"title": title,
				"text":  message,
				"ts":    time.Now().Unix(),
			},
		},
	}
	
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	
	resp, err := http.Post(r.config.Notifications.Slack.WebhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack notification failed with status: %d", resp.StatusCode)
	}
	
	return nil
}

func (r *HTMLReporter) sendWebhookNotification(result TestResult, regression *RegressionAnalysis) error {
	payload := map[string]interface{}{
		"timestamp":  time.Now(),
		"test_name":  result.TestName,
		"status":     result.Status,
		"duration":   result.Duration.String(),
		"metrics":    result.Metrics,
		"errors":     result.Errors,
		"regression": regression,
	}
	
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	
	req, err := http.NewRequest("POST", r.config.Notifications.Webhook.URL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/json")
	for key, value := range r.config.Notifications.Webhook.Headers {
		req.Header.Set(key, value)
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook notification failed with status: %d", resp.StatusCode)
	}
	
	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
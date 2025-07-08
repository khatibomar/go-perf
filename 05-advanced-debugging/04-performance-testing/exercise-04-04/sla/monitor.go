package sla

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LoadSLAConfig loads SLA configuration from file
func LoadSLAConfig(path string) (*SLAConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config SLAConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SLAMonitor manages SLA monitoring and alerting
type SLAMonitor struct {
	config      *SLAConfig
	outputDir   string
	verbose     bool
	alerts      []Alert
	mu          sync.RWMutex
	metrics     map[string]*SLAMetric
	violations  []SLAViolation
	lastCheck   time.Time
}

// SLAConfig defines SLA configuration
type SLAConfig struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Enabled     bool          `json:"enabled"`
	Interval    time.Duration `json:"interval"`
	SLAs        []SLA         `json:"slas"`
	Alerting    AlertConfig   `json:"alerting"`
	Reporting   ReportConfig  `json:"reporting"`
}

// SLA defines a Service Level Agreement
type SLA struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Enabled     bool          `json:"enabled"`
	Metric      string        `json:"metric"`
	Objective   SLAObjective  `json:"objective"`
	Window      time.Duration `json:"window"`
	Budget      SLABudget     `json:"budget"`
	Severity    string        `json:"severity"` // "critical", "warning", "info"
	Tags        []string      `json:"tags"`
}

// SLAObjective defines the target for an SLA
type SLAObjective struct {
	Type      string  `json:"type"`      // "availability", "latency", "throughput", "error_rate"
	Target    float64 `json:"target"`    // Target value (e.g., 99.9 for 99.9% availability)
	Operator  string  `json:"operator"`  // "<", "<=", ">", ">=", "=="
	Unit      string  `json:"unit"`      // "percent", "ms", "rps", etc.
	Threshold float64 `json:"threshold"` // Threshold for alerting
}

// SLABudget defines error budget for the SLA
type SLABudget struct {
	Total     float64 `json:"total"`     // Total error budget (e.g., 0.1 for 0.1%)
	Remaining float64 `json:"remaining"` // Remaining error budget
	BurnRate  float64 `json:"burn_rate"` // Current burn rate
	Window    string  `json:"window"`    // "1h", "24h", "7d", "30d"
}

// AlertConfig defines alerting configuration
type AlertConfig struct {
	Enabled     bool              `json:"enabled"`
	Channels    []AlertChannel    `json:"channels"`
	Rules       []AlertRule       `json:"rules"`
	Cooldown    time.Duration     `json:"cooldown"`
	Escalation  EscalationConfig  `json:"escalation"`
}

// AlertChannel defines an alerting channel
type AlertChannel struct {
	Name    string                 `json:"name"`
	Type    string                 `json:"type"` // "email", "slack", "webhook", "pagerduty"
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
}

// AlertRule defines when to trigger alerts
type AlertRule struct {
	Name        string        `json:"name"`
	Condition   string        `json:"condition"`   // "sla_violation", "budget_exhausted", "burn_rate_high"
	Severity    string        `json:"severity"`    // "critical", "warning", "info"
	Threshold   float64       `json:"threshold"`   // Threshold value
	Duration    time.Duration `json:"duration"`    // How long condition must persist
	Channels    []string      `json:"channels"`    // Which channels to alert
	Cooldown    time.Duration `json:"cooldown"`    // Cooldown between alerts
	Enabled     bool          `json:"enabled"`
}

// EscalationConfig defines alert escalation
type EscalationConfig struct {
	Enabled bool                   `json:"enabled"`
	Levels  []EscalationLevel      `json:"levels"`
}

// EscalationLevel defines an escalation level
type EscalationLevel struct {
	Level    int           `json:"level"`
	Delay    time.Duration `json:"delay"`
	Channels []string      `json:"channels"`
}

// ReportConfig defines reporting configuration
type ReportConfig struct {
	Enabled   bool          `json:"enabled"`
	Interval  time.Duration `json:"interval"`
	Format    string        `json:"format"` // "json", "html", "csv"
	Channels  []string      `json:"channels"`
	Retention time.Duration `json:"retention"`
}

// SLAMetric represents a metric used in SLA monitoring
type SLAMetric struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Value     float64                `json:"value"`
	Unit      string                 `json:"unit"`
	Timestamp time.Time              `json:"timestamp"`
	Tags      map[string]string      `json:"tags"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// SLAViolation represents an SLA violation
type SLAViolation struct {
	SLAName     string                 `json:"sla_name"`
	Metric      string                 `json:"metric"`
	ActualValue float64                `json:"actual_value"`
	TargetValue float64                `json:"target_value"`
	Severity    string                 `json:"severity"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Resolved    bool                   `json:"resolved"`
	Message     string                 `json:"message"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Alert represents an alert
type Alert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	SLAName     string                 `json:"sla_name"`
	Metric      string                 `json:"metric"`
	Value       float64                `json:"value"`
	Threshold   float64                `json:"threshold"`
	Channels    []string               `json:"channels"`
	Sent        bool                   `json:"sent"`
	Acknowledged bool                  `json:"acknowledged"`
	Resolved    bool                   `json:"resolved"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SLAReport represents an SLA monitoring report
type SLAReport struct {
	Name        string                    `json:"name"`
	Timestamp   time.Time                 `json:"timestamp"`
	Period      string                    `json:"period"`
	SLAs        map[string]*SLAStatus     `json:"slas"`
	Violations  []SLAViolation            `json:"violations"`
	Alerts      []Alert                   `json:"alerts"`
	Summary     *SLASummary               `json:"summary"`
	Metadata    map[string]interface{}    `json:"metadata"`
}

// SLAStatus represents the status of an SLA
type SLAStatus struct {
	Name            string    `json:"name"`
	Status          string    `json:"status"` // "healthy", "warning", "critical"
	CurrentValue    float64   `json:"current_value"`
	TargetValue     float64   `json:"target_value"`
	Compliance      float64   `json:"compliance"`
	ErrorBudget     SLABudget `json:"error_budget"`
	ViolationCount  int       `json:"violation_count"`
	LastViolation   *time.Time `json:"last_violation,omitempty"`
	Uptime          float64   `json:"uptime"`
	Availability    float64   `json:"availability"`
	LastUpdated     time.Time `json:"last_updated"`
}

// SLASummary provides a high-level summary
type SLASummary struct {
	TotalSLAs       int     `json:"total_slas"`
	HealthySLAs     int     `json:"healthy_slas"`
	WarningSLAs     int     `json:"warning_slas"`
	CriticalSLAs    int     `json:"critical_slas"`
	OverallHealth   string  `json:"overall_health"`
	ComplianceRate  float64 `json:"compliance_rate"`
	TotalViolations int     `json:"total_violations"`
	ActiveAlerts    int     `json:"active_alerts"`
}

// NewSLAMonitor creates a new SLA monitor
func NewSLAMonitor(config *SLAConfig, outputDir string) *SLAMonitor {
	return &SLAMonitor{
		config:     config,
		outputDir:  outputDir,
		metrics:    make(map[string]*SLAMetric),
		violations: make([]SLAViolation, 0),
		alerts:     make([]Alert, 0),
	}
}

// SetVerbose enables or disables verbose output
func (m *SLAMonitor) SetVerbose(verbose bool) {
	m.verbose = verbose
}

// Start starts the SLA monitoring
func (m *SLAMonitor) Start(ctx context.Context) error {
	if !m.config.Enabled {
		return fmt.Errorf("SLA monitoring is disabled")
	}

	if m.verbose {
		fmt.Printf("Starting SLA monitor: %s\n", m.config.Name)
		fmt.Printf("Monitoring %d SLAs with %v interval\n", len(m.config.SLAs), m.config.Interval)
	}

	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()

	// Initial check
	if err := m.checkSLAs(ctx); err != nil {
		if m.verbose {
			fmt.Printf("Error in initial SLA check: %v\n", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := m.checkSLAs(ctx); err != nil {
				if m.verbose {
					fmt.Printf("Error checking SLAs: %v\n", err)
				}
			}
		}
	}
}

// checkSLAs performs SLA checks
func (m *SLAMonitor) checkSLAs(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastCheck = time.Now()

	for _, sla := range m.config.SLAs {
		if !sla.Enabled {
			continue
		}

		if err := m.checkSLA(ctx, sla); err != nil {
			if m.verbose {
				fmt.Printf("Error checking SLA %s: %v\n", sla.Name, err)
			}
		}
	}

	// Process alerts
	if err := m.processAlerts(ctx); err != nil {
		if m.verbose {
			fmt.Printf("Error processing alerts: %v\n", err)
		}
	}

	return nil
}

// checkSLA checks a specific SLA
func (m *SLAMonitor) checkSLA(ctx context.Context, sla SLA) error {
	// Get current metric value
	metric, exists := m.metrics[sla.Metric]
	if !exists {
		// Create a placeholder metric if it doesn't exist
		metric = &SLAMetric{
			Name:      sla.Metric,
			Type:      sla.Objective.Type,
			Value:     0,
			Unit:      sla.Objective.Unit,
			Timestamp: time.Now(),
			Tags:      make(map[string]string),
			Metadata:  make(map[string]interface{}),
		}
		m.metrics[sla.Metric] = metric
	}

	// Check if SLA is violated
	violated := m.isSLAViolated(sla, metric)

	if violated {
		// Record violation
		violation := SLAViolation{
			SLAName:     sla.Name,
			Metric:      sla.Metric,
			ActualValue: metric.Value,
			TargetValue: sla.Objective.Target,
			Severity:    sla.Severity,
			StartTime:   time.Now(),
			Duration:    0,
			Resolved:    false,
			Message:     m.generateViolationMessage(sla, metric),
			Metadata:    make(map[string]interface{}),
		}

		m.violations = append(m.violations, violation)

		// Generate alert
		alert := m.generateAlert(sla, metric, violation)
		m.alerts = append(m.alerts, alert)

		if m.verbose {
			fmt.Printf("SLA violation detected: %s (actual: %.2f, target: %.2f)\n",
				sla.Name, metric.Value, sla.Objective.Target)
		}
	}

	return nil
}

// isSLAViolated checks if an SLA is violated
func (m *SLAMonitor) isSLAViolated(sla SLA, metric *SLAMetric) bool {
	switch sla.Objective.Operator {
	case "<":
		return metric.Value >= sla.Objective.Target
	case "<=":
		return metric.Value > sla.Objective.Target
	case ">":
		return metric.Value <= sla.Objective.Target
	case ">=":
		return metric.Value < sla.Objective.Target
	case "==":
		return metric.Value != sla.Objective.Target
	default:
		// Default: assume target is maximum allowed value
		return metric.Value > sla.Objective.Target
	}
}

// generateViolationMessage generates a human-readable violation message
func (m *SLAMonitor) generateViolationMessage(sla SLA, metric *SLAMetric) string {
	return fmt.Sprintf("SLA '%s' violated: %s is %.2f %s (target: %s %.2f %s)",
		sla.Name, sla.Metric, metric.Value, metric.Unit,
		sla.Objective.Operator, sla.Objective.Target, sla.Objective.Unit)
}

// generateAlert generates an alert for an SLA violation
func (m *SLAMonitor) generateAlert(sla SLA, metric *SLAMetric, violation SLAViolation) Alert {
	return Alert{
		ID:        fmt.Sprintf("%s-%d", sla.Name, time.Now().Unix()),
		Type:      "sla_violation",
		Severity:  sla.Severity,
		Title:     fmt.Sprintf("SLA Violation: %s", sla.Name),
		Message:   violation.Message,
		Timestamp: time.Now(),
		SLAName:   sla.Name,
		Metric:    sla.Metric,
		Value:     metric.Value,
		Threshold: sla.Objective.Target,
		Channels:  m.getAlertChannels(sla.Severity),
		Sent:      false,
		Acknowledged: false,
		Resolved:  false,
		Metadata:  make(map[string]interface{}),
	}
}

// getAlertChannels returns the appropriate alert channels for a severity level
func (m *SLAMonitor) getAlertChannels(severity string) []string {
	channels := make([]string, 0)

	for _, rule := range m.config.Alerting.Rules {
		if rule.Enabled && rule.Severity == severity {
			channels = append(channels, rule.Channels...)
		}
	}

	return channels
}

// processAlerts processes pending alerts
func (m *SLAMonitor) processAlerts(ctx context.Context) error {
	for i := range m.alerts {
		alert := &m.alerts[i]
		if !alert.Sent {
			if err := m.sendAlert(ctx, alert); err != nil {
				if m.verbose {
					fmt.Printf("Failed to send alert %s: %v\n", alert.ID, err)
				}
			} else {
				alert.Sent = true
				if m.verbose {
					fmt.Printf("Alert sent: %s\n", alert.Title)
				}
			}
		}
	}

	return nil
}

// sendAlert sends an alert through configured channels
func (m *SLAMonitor) sendAlert(ctx context.Context, alert *Alert) error {
	if !m.config.Alerting.Enabled {
		return nil
	}

	for _, channelName := range alert.Channels {
		channel := m.findAlertChannel(channelName)
		if channel == nil || !channel.Enabled {
			continue
		}

		switch channel.Type {
		case "console":
			fmt.Printf("[ALERT] %s: %s\n", alert.Severity, alert.Message)
		case "file":
			if err := m.writeAlertToFile(alert, channel); err != nil {
				return err
			}
		default:
			if m.verbose {
				fmt.Printf("Unsupported alert channel type: %s\n", channel.Type)
			}
		}
	}

	return nil
}

// findAlertChannel finds an alert channel by name
func (m *SLAMonitor) findAlertChannel(name string) *AlertChannel {
	for _, channel := range m.config.Alerting.Channels {
		if channel.Name == name {
			return &channel
		}
	}
	return nil
}

// writeAlertToFile writes an alert to a file
func (m *SLAMonitor) writeAlertToFile(alert *Alert, channel *AlertChannel) error {
	filename, ok := channel.Config["filename"].(string)
	if !ok {
		filename = "alerts.json"
	}

	if m.outputDir != "" {
		filename = filepath.Join(m.outputDir, filename)
	}

	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create alert directory: %w", err)
	}

	// Read existing alerts
	var alerts []Alert
	if data, err := os.ReadFile(filename); err == nil {
		json.Unmarshal(data, &alerts)
	}

	// Append new alert
	alerts = append(alerts, *alert)

	// Write back to file
	data, err := json.MarshalIndent(alerts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal alerts: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// UpdateMetric updates a metric value
func (m *SLAMonitor) UpdateMetric(name string, value float64, unit string, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metric := &SLAMetric{
		Name:      name,
		Value:     value,
		Unit:      unit,
		Timestamp: time.Now(),
		Tags:      tags,
		Metadata:  make(map[string]interface{}),
	}

	m.metrics[name] = metric

	if m.verbose {
		fmt.Printf("Metric updated: %s = %.2f %s\n", name, value, unit)
	}
}

// GenerateReport generates an SLA monitoring report
func (m *SLAMonitor) GenerateReport() *SLAReport {
	m.mu.RLock()
	defer m.mu.RUnlock()

	report := &SLAReport{
		Name:       m.config.Name,
		Timestamp:  time.Now(),
		Period:     "current",
		SLAs:       make(map[string]*SLAStatus),
		Violations: m.violations,
		Alerts:     m.alerts,
		Metadata:   make(map[string]interface{}),
	}

	// Generate SLA status for each SLA
	healthyCount := 0
	warningCount := 0
	criticalCount := 0

	for _, sla := range m.config.SLAs {
		status := m.generateSLAStatus(sla)
		report.SLAs[sla.Name] = status

		switch status.Status {
		case "healthy":
			healthyCount++
		case "warning":
			warningCount++
		case "critical":
			criticalCount++
		}
	}

	// Generate summary
	totalSLAs := len(m.config.SLAs)
	complianceRate := float64(healthyCount) / float64(totalSLAs) * 100

	overallHealth := "healthy"
	if criticalCount > 0 {
		overallHealth = "critical"
	} else if warningCount > 0 {
		overallHealth = "warning"
	}

	activeAlerts := 0
	for _, alert := range m.alerts {
		if !alert.Resolved {
			activeAlerts++
		}
	}

	report.Summary = &SLASummary{
		TotalSLAs:       totalSLAs,
		HealthySLAs:     healthyCount,
		WarningSLAs:     warningCount,
		CriticalSLAs:    criticalCount,
		OverallHealth:   overallHealth,
		ComplianceRate:  complianceRate,
		TotalViolations: len(m.violations),
		ActiveAlerts:    activeAlerts,
	}

	return report
}

// generateSLAStatus generates status for a specific SLA
func (m *SLAMonitor) generateSLAStatus(sla SLA) *SLAStatus {
	metric, exists := m.metrics[sla.Metric]
	if !exists {
		return &SLAStatus{
			Name:         sla.Name,
			Status:       "unknown",
			CurrentValue: 0,
			TargetValue:  sla.Objective.Target,
			Compliance:   0,
			LastUpdated:  time.Now(),
		}
	}

	// Determine status
	status := "healthy"
	if m.isSLAViolated(sla, metric) {
		if sla.Severity == "critical" {
			status = "critical"
		} else {
			status = "warning"
		}
	}

	// Calculate compliance (simplified)
	compliance := 100.0
	if m.isSLAViolated(sla, metric) {
		compliance = 0.0
	}

	// Count violations for this SLA
	violationCount := 0
	var lastViolation *time.Time
	for _, violation := range m.violations {
		if violation.SLAName == sla.Name {
			violationCount++
			if lastViolation == nil || violation.StartTime.After(*lastViolation) {
				lastViolation = &violation.StartTime
			}
		}
	}

	return &SLAStatus{
		Name:           sla.Name,
		Status:         status,
		CurrentValue:   metric.Value,
		TargetValue:    sla.Objective.Target,
		Compliance:     compliance,
		ViolationCount: violationCount,
		LastViolation:  lastViolation,
		Uptime:         compliance, // Simplified
		Availability:   compliance, // Simplified
		LastUpdated:    metric.Timestamp,
	}
}

// SaveReport saves an SLA report to a file
func (m *SLAMonitor) SaveReport(report *SLAReport, filename string) error {
	if m.outputDir != "" {
		if err := os.MkdirAll(m.outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		filename = filepath.Join(m.outputDir, filename)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	if m.verbose {
		fmt.Printf("SLA report saved to: %s\n", filename)
	}

	return nil
}
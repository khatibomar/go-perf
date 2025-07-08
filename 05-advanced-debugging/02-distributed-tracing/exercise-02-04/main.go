package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Alert represents an alert instance
type Alert struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Severity    string            `json:"severity"`
	Status      string            `json:"status"`
	Message     string            `json:"message"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     *time.Time        `json:"end_time,omitempty"`
	MTTD        time.Duration     `json:"mttd"`
	MTTR        time.Duration     `json:"mttr"`
}

// AlertRule defines alert conditions
type AlertRule struct {
	Name        string            `json:"name"`
	Expression  string            `json:"expression"`
	Duration    string            `json:"duration"`
	Severity    string            `json:"severity"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Threshold   float64           `json:"threshold"`
	Enabled     bool              `json:"enabled"`
}

// Incident represents an incident
type Incident struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Severity    string            `json:"severity"`
	Status      string            `json:"status"`
	Assignee    string            `json:"assignee"`
	Labels      map[string]string `json:"labels"`
	Alerts      []string          `json:"alerts"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     *time.Time        `json:"end_time,omitempty"`
	MTTD        time.Duration     `json:"mttd"`
	MTTR        time.Duration     `json:"mttr"`
	Escalated   bool              `json:"escalated"`
	PostMortem  *PostMortem       `json:"post_mortem,omitempty"`
}

// PostMortem represents post-incident analysis
type PostMortem struct {
	ID           string    `json:"id"`
	IncidentID   string    `json:"incident_id"`
	RootCause    string    `json:"root_cause"`
	Impact       string    `json:"impact"`
	Resolution   string    `json:"resolution"`
	LessonsLearned []string `json:"lessons_learned"`
	ActionItems  []string  `json:"action_items"`
	CreatedAt    time.Time `json:"created_at"`
}

// EscalationRule defines escalation procedures
type EscalationRule struct {
	Severity     string        `json:"severity"`
	TimeToEscalate time.Duration `json:"time_to_escalate"`
	EscalateTo   []string      `json:"escalate_to"`
	NotificationChannels []string `json:"notification_channels"`
}

// AlertManager manages alerts and incidents
type AlertManager struct {
	mu              sync.RWMutex
	alerts          map[string]*Alert
	incidents       map[string]*Incident
	alertRules      map[string]*AlertRule
	escalationRules map[string]*EscalationRule
	metrics         *AlertMetrics
	ctx             context.Context
	cancel          context.CancelFunc
	disableSimulation bool
}

// AlertMetrics contains Prometheus metrics for alerting
type AlertMetrics struct {
	alertsTotal        *prometheus.CounterVec
	alertsActive       *prometheus.GaugeVec
	incidentsTotal     *prometheus.CounterVec
	incidentsActive    *prometheus.GaugeVec
	alertAccuracy      *prometheus.GaugeVec
	falsePositiveRate  *prometheus.GaugeVec
	mttd               *prometheus.HistogramVec
	mttr               *prometheus.HistogramVec
	escalationsTotal   *prometheus.CounterVec
	responseTime       *prometheus.HistogramVec
}

// NewAlertMetrics creates new alert metrics with custom registry
func NewAlertMetrics() *AlertMetrics {
	reg := prometheus.NewRegistry()
	factory := promauto.With(reg)

	return &AlertMetrics{
		alertsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alerts_total",
				Help: "Total number of alerts fired",
			},
			[]string{"name", "severity", "status"},
		),
		alertsActive: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "alerts_active",
				Help: "Number of active alerts",
			},
			[]string{"name", "severity"},
		),
		incidentsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "incidents_total",
				Help: "Total number of incidents created",
			},
			[]string{"severity", "status"},
		),
		incidentsActive: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "incidents_active",
				Help: "Number of active incidents",
			},
			[]string{"severity"},
		),
		alertAccuracy: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "alert_accuracy_percent",
				Help: "Alert accuracy percentage",
			},
			[]string{"rule_name"},
		),
		falsePositiveRate: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "false_positive_rate_percent",
				Help: "False positive rate percentage",
			},
			[]string{"rule_name"},
		),
		mttd: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "mean_time_to_detect_seconds",
				Help:    "Mean time to detect incidents",
				Buckets: prometheus.ExponentialBuckets(1, 2, 10),
			},
			[]string{"severity"},
		),
		mttr: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "mean_time_to_resolve_seconds",
				Help:    "Mean time to resolve incidents",
				Buckets: prometheus.ExponentialBuckets(60, 2, 12),
			},
			[]string{"severity"},
		),
		escalationsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "escalations_total",
				Help: "Total number of escalations",
			},
			[]string{"severity", "escalation_level"},
		),
		responseTime: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "alert_response_time_seconds",
				Help:    "Alert response time",
				Buckets: prometheus.ExponentialBuckets(1, 2, 8),
			},
			[]string{"alert_name"},
		),
	}
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	return NewAlertManagerWithOptions(false)
}

// NewAlertManagerWithOptions creates a new alert manager with options
func NewAlertManagerWithOptions(disableSimulation bool) *AlertManager {
	ctx, cancel := context.WithCancel(context.Background())
	am := &AlertManager{
		alerts:          make(map[string]*Alert),
		incidents:       make(map[string]*Incident),
		alertRules:      make(map[string]*AlertRule),
		escalationRules: make(map[string]*EscalationRule),
		metrics:         NewAlertMetrics(),
		ctx:             ctx,
		cancel:          cancel,
		disableSimulation: disableSimulation,
	}

	// Initialize default alert rules
	am.initializeDefaultRules()

	// Start background processes only if simulation is not disabled
	if !disableSimulation {
		go am.alertProcessor()
		go am.escalationProcessor()
		go am.metricsUpdater()
	}

	return am
}

// initializeDefaultRules sets up default alert rules and escalation rules
func (am *AlertManager) initializeDefaultRules() {
	// Default alert rules
	alertRules := []*AlertRule{
		{
			Name:       "HighErrorRate",
			Expression: "rate(http_requests_total{status=~\"5..\"}[5m]) > 0.1",
			Duration:   "2m",
			Severity:   "critical",
			Threshold:  0.1,
			Enabled:    true,
			Labels: map[string]string{
				"team": "platform",
				"service": "api",
			},
			Annotations: map[string]string{
				"summary":     "High error rate detected",
				"description": "Error rate is above 10% for more than 2 minutes",
				"runbook":     "https://runbooks.example.com/high-error-rate",
			},
		},
		{
			Name:       "HighLatency",
			Expression: "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 0.5",
			Duration:   "5m",
			Severity:   "warning",
			Threshold:  0.5,
			Enabled:    true,
			Labels: map[string]string{
				"team": "platform",
				"service": "api",
			},
			Annotations: map[string]string{
				"summary":     "High latency detected",
				"description": "95th percentile latency is above 500ms for more than 5 minutes",
				"runbook":     "https://runbooks.example.com/high-latency",
			},
		},
		{
			Name:       "HighMemoryUsage",
			Expression: "process_resident_memory_bytes / 1024 / 1024 > 512",
			Duration:   "10m",
			Severity:   "warning",
			Threshold:  512,
			Enabled:    true,
			Labels: map[string]string{
				"team": "platform",
				"service": "api",
			},
			Annotations: map[string]string{
				"summary":     "High memory usage detected",
				"description": "Memory usage is above 512MB for more than 10 minutes",
				"runbook":     "https://runbooks.example.com/high-memory",
			},
		},
		{
			Name:       "ServiceDown",
			Expression: "up == 0",
			Duration:   "1m",
			Severity:   "critical",
			Threshold:  0,
			Enabled:    true,
			Labels: map[string]string{
				"team": "platform",
				"service": "api",
			},
			Annotations: map[string]string{
				"summary":     "Service is down",
				"description": "Service has been down for more than 1 minute",
				"runbook":     "https://runbooks.example.com/service-down",
			},
		},
		{
			Name:       "DiskSpaceLow",
			Expression: "(node_filesystem_avail_bytes / node_filesystem_size_bytes) * 100 < 10",
			Duration:   "5m",
			Severity:   "warning",
			Threshold:  10,
			Enabled:    true,
			Labels: map[string]string{
				"team": "infrastructure",
				"service": "storage",
			},
			Annotations: map[string]string{
				"summary":     "Low disk space",
				"description": "Available disk space is below 10%",
				"runbook":     "https://runbooks.example.com/disk-space",
			},
		},
	}

	for _, rule := range alertRules {
		am.alertRules[rule.Name] = rule
	}

	// Default escalation rules
	escalationRules := []*EscalationRule{
		{
			Severity:       "critical",
			TimeToEscalate: 5 * time.Minute,
			EscalateTo:     []string{"oncall-engineer", "team-lead"},
			NotificationChannels: []string{"slack", "pagerduty", "email"},
		},
		{
			Severity:       "warning",
			TimeToEscalate: 15 * time.Minute,
			EscalateTo:     []string{"team-lead"},
			NotificationChannels: []string{"slack", "email"},
		},
		{
			Severity:       "info",
			TimeToEscalate: 30 * time.Minute,
			EscalateTo:     []string{"team-lead"},
			NotificationChannels: []string{"email"},
		},
	}

	for _, rule := range escalationRules {
		am.escalationRules[rule.Severity] = rule
	}
}

// FireAlert creates and fires a new alert
func (am *AlertManager) FireAlert(name, severity, message string, labels map[string]string) *Alert {
	am.mu.Lock()
	defer am.mu.Unlock()

	alertID := fmt.Sprintf("%s-%d", name, time.Now().UnixNano())
	alert := &Alert{
		ID:       alertID,
		Name:     name,
		Severity: severity,
		Status:   "firing",
		Message:  message,
		Labels:   labels,
		StartTime: time.Now(),
		MTTD:     time.Since(time.Now()), // Will be updated when detected
	}

	// Add annotations from alert rule if exists
	if rule, exists := am.alertRules[name]; exists {
		alert.Annotations = rule.Annotations
	}

	am.alerts[alertID] = alert

	// Update metrics if available
	if am.metrics != nil {
		am.metrics.alertsTotal.WithLabelValues(name, severity, "firing").Inc()
		am.metrics.alertsActive.WithLabelValues(name, severity).Inc()
	}

	// Check if incident should be created
	if severity == "critical" || am.shouldCreateIncident(alert) {
		am.createIncidentFromAlert(alert)
	}

	return alert
}

// ResolveAlert resolves an existing alert
func (am *AlertManager) ResolveAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert %s not found", alertID)
	}

	if alert.Status == "resolved" {
		return fmt.Errorf("alert %s already resolved", alertID)
	}

	now := time.Now()
	alert.Status = "resolved"
	alert.EndTime = &now
	alert.MTTR = time.Since(alert.StartTime)

	// Update metrics if available
	if am.metrics != nil {
		am.metrics.alertsTotal.WithLabelValues(alert.Name, alert.Severity, "resolved").Inc()
		am.metrics.alertsActive.WithLabelValues(alert.Name, alert.Severity).Dec()
		am.metrics.responseTime.WithLabelValues(alert.Name).Observe(alert.MTTR.Seconds())
	}

	return nil
}

// shouldCreateIncident determines if an incident should be created for an alert
func (am *AlertManager) shouldCreateIncident(alert *Alert) bool {
	// Create incident for critical alerts or multiple related alerts
	if alert.Severity == "critical" {
		return true
	}

	// Check for multiple related alerts
	relatedCount := 0
	for _, a := range am.alerts {
		if a.Status == "firing" && a.Name == alert.Name {
			relatedCount++
		}
	}

	return relatedCount >= 3
}

// createIncidentFromAlert creates an incident from an alert
func (am *AlertManager) createIncidentFromAlert(alert *Alert) *Incident {
	incidentID := fmt.Sprintf("INC-%d", time.Now().UnixNano())
	incident := &Incident{
		ID:          incidentID,
		Title:       fmt.Sprintf("Incident: %s", alert.Name),
		Description: alert.Message,
		Severity:    alert.Severity,
		Status:      "open",
		Assignee:    "oncall-engineer",
		Labels:      alert.Labels,
		Alerts:      []string{alert.ID},
		StartTime:   time.Now(),
		MTTD:        time.Since(alert.StartTime),
		Escalated:   false,
	}

	am.incidents[incidentID] = incident

	// Update metrics if available
	if am.metrics != nil {
		am.metrics.incidentsTotal.WithLabelValues(incident.Severity, "open").Inc()
		am.metrics.incidentsActive.WithLabelValues(incident.Severity).Inc()
		am.metrics.mttd.WithLabelValues(incident.Severity).Observe(incident.MTTD.Seconds())
	}

	return incident
}

// ResolveIncident resolves an incident
func (am *AlertManager) ResolveIncident(incidentID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	incident, exists := am.incidents[incidentID]
	if !exists {
		return fmt.Errorf("incident %s not found", incidentID)
	}

	if incident.Status == "resolved" {
		return fmt.Errorf("incident %s already resolved", incidentID)
	}

	now := time.Now()
	incident.Status = "resolved"
	incident.EndTime = &now
	incident.MTTR = time.Since(incident.StartTime)

	// Update metrics if available
	if am.metrics != nil {
		am.metrics.incidentsTotal.WithLabelValues(incident.Severity, "resolved").Inc()
		am.metrics.incidentsActive.WithLabelValues(incident.Severity).Dec()
		am.metrics.mttr.WithLabelValues(incident.Severity).Observe(incident.MTTR.Seconds())
	}

	return nil
}

// GetAlerts returns all alerts
func (am *AlertManager) GetAlerts() map[string]*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make(map[string]*Alert)
	for k, v := range am.alerts {
		alerts[k] = v
	}
	return alerts
}

// GetIncidents returns all incidents
func (am *AlertManager) GetIncidents() map[string]*Incident {
	am.mu.RLock()
	defer am.mu.RUnlock()

	incidents := make(map[string]*Incident)
	for k, v := range am.incidents {
		incidents[k] = v
	}
	return incidents
}

// GetAlertRules returns all alert rules
func (am *AlertManager) GetAlertRules() map[string]*AlertRule {
	am.mu.RLock()
	defer am.mu.RUnlock()

	rules := make(map[string]*AlertRule)
	for k, v := range am.alertRules {
		rules[k] = v
	}
	return rules
}

// alertProcessor processes alerts in the background
func (am *AlertManager) alertProcessor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-am.ctx.Done():
			return
		case <-ticker.C:
			am.processAlerts()
		}
	}
}

// processAlerts processes active alerts
func (am *AlertManager) processAlerts() {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	for _, alert := range am.alerts {
		if alert.Status == "firing" {
			// Update MTTD
			alert.MTTD = now.Sub(alert.StartTime)

			// Check for auto-resolution (simulate)
			if rand.Float64() < 0.1 && alert.MTTD > 2*time.Minute {
				// Resolve alert without acquiring lock again (already have it)
				if alert.Status != "resolved" {
					alert.Status = "resolved"
					alert.EndTime = &now
					alert.MTTR = time.Since(alert.StartTime)

					// Update metrics if available
					if am.metrics != nil {
						am.metrics.alertsTotal.WithLabelValues(alert.Name, alert.Severity, "resolved").Inc()
						am.metrics.alertsActive.WithLabelValues(alert.Name, alert.Severity).Dec()
						am.metrics.responseTime.WithLabelValues(alert.Name).Observe(alert.MTTR.Seconds())
					}
				}
			}
		}
	}
}

// escalationProcessor handles alert escalations
func (am *AlertManager) escalationProcessor() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-am.ctx.Done():
			return
		case <-ticker.C:
			am.processEscalations()
		}
	}
}

// processEscalations processes incident escalations
func (am *AlertManager) processEscalations() {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	for _, incident := range am.incidents {
		if incident.Status == "open" && !incident.Escalated {
			escalationRule, exists := am.escalationRules[incident.Severity]
			if exists && now.Sub(incident.StartTime) > escalationRule.TimeToEscalate {
				incident.Escalated = true
				if am.metrics != nil {
					am.metrics.escalationsTotal.WithLabelValues(incident.Severity, "level1").Inc()
				}
				// In a real system, this would trigger notifications
				log.Printf("Escalating incident %s to %v", incident.ID, escalationRule.EscalateTo)
			}
		}
	}
}

// metricsUpdater updates alert accuracy and false positive metrics
func (am *AlertManager) metricsUpdater() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-am.ctx.Done():
			return
		case <-ticker.C:
			am.updateAccuracyMetrics()
		}
	}
}

// updateAccuracyMetrics calculates and updates accuracy metrics
func (am *AlertManager) updateAccuracyMetrics() {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// Skip if metrics not available
	if am.metrics == nil {
		return
	}

	// Calculate accuracy per alert rule
	ruleStats := make(map[string]struct {
		total         int
		truePositives int
		falsePositives int
	})

	for _, alert := range am.alerts {
		stats := ruleStats[alert.Name]
		stats.total++

		// Simulate true/false positive detection
		// In a real system, this would be based on actual incident correlation
		if alert.Status == "resolved" && alert.MTTR < 15*time.Minute {
			stats.truePositives++
		} else if alert.Status == "resolved" && alert.MTTR < 1*time.Minute {
			stats.falsePositives++
		}

		ruleStats[alert.Name] = stats
	}

	// Update metrics
	for ruleName, stats := range ruleStats {
		if stats.total > 0 {
			accuracy := float64(stats.truePositives) / float64(stats.total) * 100
			falsePositiveRate := float64(stats.falsePositives) / float64(stats.total) * 100

			am.metrics.alertAccuracy.WithLabelValues(ruleName).Set(accuracy)
			am.metrics.falsePositiveRate.WithLabelValues(ruleName).Set(falsePositiveRate)
		}
	}
}

// CreatePostMortem creates a post-mortem for an incident
func (am *AlertManager) CreatePostMortem(incidentID, rootCause, impact, resolution string, lessonsLearned, actionItems []string) (*PostMortem, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	incident, exists := am.incidents[incidentID]
	if !exists {
		return nil, fmt.Errorf("incident %s not found", incidentID)
	}

	postMortem := &PostMortem{
		ID:             fmt.Sprintf("PM-%d", time.Now().UnixNano()),
		IncidentID:     incidentID,
		RootCause:      rootCause,
		Impact:         impact,
		Resolution:     resolution,
		LessonsLearned: lessonsLearned,
		ActionItems:    actionItems,
		CreatedAt:      time.Now(),
	}

	incident.PostMortem = postMortem

	return postMortem, nil
}

// Shutdown gracefully shuts down the alert manager
func (am *AlertManager) Shutdown() {
	am.cancel()
}

// HTTP Handlers

// alertsHandler returns all alerts
func (am *AlertManager) alertsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		alerts := am.GetAlerts()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"alerts": alerts,
			"count":  len(alerts),
		})

	case http.MethodPost:
		var req struct {
			Name     string            `json:"name"`
			Severity string            `json:"severity"`
			Message  string            `json:"message"`
			Labels   map[string]string `json:"labels"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		alert := am.FireAlert(req.Name, req.Severity, req.Message, req.Labels)
		json.NewEncoder(w).Encode(alert)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// incidentsHandler returns all incidents
func (am *AlertManager) incidentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		incidents := am.GetIncidents()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"incidents": incidents,
			"count":     len(incidents),
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// alertRulesHandler returns all alert rules
func (am *AlertManager) alertRulesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		rules := am.GetAlertRules()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rules": rules,
			"count": len(rules),
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// resolveAlertHandler resolves an alert
func (am *AlertManager) resolveAlertHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	alertID := r.URL.Query().Get("id")
	if alertID == "" {
		http.Error(w, "Alert ID required", http.StatusBadRequest)
		return
	}

	err := am.ResolveAlert(alertID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "resolved",
		"alert_id": alertID,
	})
}

// resolveIncidentHandler resolves an incident
func (am *AlertManager) resolveIncidentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	incidentID := r.URL.Query().Get("id")
	if incidentID == "" {
		http.Error(w, "Incident ID required", http.StatusBadRequest)
		return
	}

	err := am.ResolveIncident(incidentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "resolved",
		"incident_id": incidentID,
	})
}

// postMortemHandler creates a post-mortem
func (am *AlertManager) postMortemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IncidentID     string   `json:"incident_id"`
		RootCause      string   `json:"root_cause"`
		Impact         string   `json:"impact"`
		Resolution     string   `json:"resolution"`
		LessonsLearned []string `json:"lessons_learned"`
		ActionItems    []string `json:"action_items"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	postMortem, err := am.CreatePostMortem(
		req.IncidentID,
		req.RootCause,
		req.Impact,
		req.Resolution,
		req.LessonsLearned,
		req.ActionItems,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(postMortem)
}

// healthHandler returns service health
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"service":   "alerting-incident-response",
		"version":   "1.0.0",
		"timestamp": time.Now().Unix(),
	})
}

// simulateAlerts generates sample alerts for testing
func (am *AlertManager) simulateAlerts() {
	alertTypes := []struct {
		name     string
		severity string
		message  string
		prob     float64
	}{
		{"HighErrorRate", "critical", "Error rate exceeded threshold", 0.1},
		{"HighLatency", "warning", "Response time is high", 0.2},
		{"HighMemoryUsage", "warning", "Memory usage is high", 0.15},
		{"ServiceDown", "critical", "Service is not responding", 0.05},
		{"DiskSpaceLow", "warning", "Disk space is running low", 0.1},
	}

	for {
		select {
		case <-am.ctx.Done():
			return
		default:
			time.Sleep(time.Duration(rand.Intn(30)+10) * time.Second)

			for _, alertType := range alertTypes {
				if rand.Float64() < alertType.prob {
					labels := map[string]string{
						"instance": fmt.Sprintf("server-%d", rand.Intn(5)+1),
						"job":      "api-server",
					}
					am.FireAlert(alertType.name, alertType.severity, alertType.message, labels)
				}
			}
		}
	}
}

// loadTestHandler simulates load for testing
func (am *AlertManager) loadTestHandler(w http.ResponseWriter, r *http.Request) {
	operationsStr := r.URL.Query().Get("operations")
	operations := 10
	if operationsStr != "" {
		if ops, err := strconv.Atoi(operationsStr); err == nil {
			operations = ops
		}
	}

	start := time.Now()

	// Simulate operations that might trigger alerts
	for i := 0; i < operations; i++ {
		// Simulate some processing time
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

		// Randomly trigger alerts
		if rand.Float64() < 0.3 {
			severity := "warning"
			if rand.Float64() < 0.2 {
				severity = "critical"
			}

			alertName := "LoadTestAlert"
			message := fmt.Sprintf("Load test operation %d triggered alert", i+1)
			labels := map[string]string{
				"test_run": fmt.Sprintf("load-test-%d", time.Now().Unix()),
				"operation": fmt.Sprintf("op-%d", i+1),
			}

			am.FireAlert(alertName, severity, message, labels)
		}
	}

	duration := time.Since(start)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"operations":     operations,
		"duration_ms":    duration.Milliseconds(),
		"ops_per_second": float64(operations) / duration.Seconds(),
		"alerts_fired":   "check /alerts endpoint",
	})
}

func main() {
	// Create alert manager
	alertManager := NewAlertManager()
	defer alertManager.Shutdown()

	// Start alert simulation only if not disabled
	if !alertManager.disableSimulation {
		go alertManager.simulateAlerts()
	}

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", healthHandler)

	// Alert management endpoints
	mux.HandleFunc("/alerts", alertManager.alertsHandler)
	mux.HandleFunc("/alerts/resolve", alertManager.resolveAlertHandler)
	mux.HandleFunc("/alert-rules", alertManager.alertRulesHandler)

	// Incident management endpoints
	mux.HandleFunc("/incidents", alertManager.incidentsHandler)
	mux.HandleFunc("/incidents/resolve", alertManager.resolveIncidentHandler)
	mux.HandleFunc("/post-mortem", alertManager.postMortemHandler)

	// Load testing endpoint
	mux.HandleFunc("/load-test", alertManager.loadTestHandler)

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Start server
	log.Println("Starting alerting and incident response server on :8080")
	log.Println("Endpoints:")
	log.Println("  GET  /health - Health check")
	log.Println("  GET  /alerts - List all alerts")
	log.Println("  POST /alerts - Fire new alert")
	log.Println("  POST /alerts/resolve?id={alert_id} - Resolve alert")
	log.Println("  GET  /alert-rules - List alert rules")
	log.Println("  GET  /incidents - List all incidents")
	log.Println("  POST /incidents/resolve?id={incident_id} - Resolve incident")
	log.Println("  POST /post-mortem - Create post-mortem")
	log.Println("  GET  /load-test?operations={n} - Run load test")
	log.Println("  GET  /metrics - Prometheus metrics")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
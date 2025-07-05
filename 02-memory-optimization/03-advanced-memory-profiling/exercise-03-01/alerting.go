package main

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

// AlertLevel represents the severity of an alert
type AlertLevel int

const (
	AlertLevelInfo AlertLevel = iota
	AlertLevelWarning
	AlertLevelCritical
)

func (al AlertLevel) String() string {
	switch al {
	case AlertLevelInfo:
		return "INFO"
	case AlertLevelWarning:
		return "WARNING"
	case AlertLevelCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Alert represents a monitoring alert
type Alert struct {
	ID          string      `json:"id"`
	Level       AlertLevel  `json:"level"`
	Title       string      `json:"title"`
	Message     string      `json:"message"`
	Timestamp   time.Time   `json:"timestamp"`
	Metrics     interface{} `json:"metrics,omitempty"`
	Resolved    bool        `json:"resolved"`
	ResolvedAt  *time.Time  `json:"resolved_at,omitempty"`
}

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	Name        string
	Level       AlertLevel
	Condition   func(*MemoryMetrics) bool
	Message     string
	Cooldown    time.Duration
	lastFired   time.Time
}

// AlertManager manages memory monitoring alerts
type AlertManager struct {
	rules       []*AlertRule
	alerts      map[string]*Alert
	mu          sync.RWMutex
	handlers    []AlertHandler
	isRunning   bool
	stopChan    chan struct{}
	profiler    *ContinuousProfiler
}

// AlertHandler defines interface for handling alerts
type AlertHandler interface {
	HandleAlert(alert *Alert) error
}

// LogAlertHandler logs alerts to standard output
type LogAlertHandler struct{}

func (h *LogAlertHandler) HandleAlert(alert *Alert) error {
	log.Printf("[%s] %s: %s", alert.Level, alert.Title, alert.Message)
	return nil
}

// NewAlertManager creates a new alert manager
func NewAlertManager(profiler *ContinuousProfiler) *AlertManager {
	am := &AlertManager{
		rules:    make([]*AlertRule, 0),
		alerts:   make(map[string]*Alert),
		handlers: make([]AlertHandler, 0),
		stopChan: make(chan struct{}),
		profiler: profiler,
	}
	
	// Add default alert handlers
	am.AddHandler(&LogAlertHandler{})
	
	// Add default rules
	am.addDefaultRules()
	
	return am
}

// AddHandler adds an alert handler
func (am *AlertManager) AddHandler(handler AlertHandler) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.handlers = append(am.handlers, handler)
}

// AddRule adds a custom alert rule
func (am *AlertManager) AddRule(rule *AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.rules = append(am.rules, rule)
}

// Start begins monitoring and alert checking
func (am *AlertManager) Start(checkInterval time.Duration) {
	am.mu.Lock()
	if am.isRunning {
		am.mu.Unlock()
		return
	}
	am.isRunning = true
	am.mu.Unlock()
	
	go am.monitoringLoop(checkInterval)
	log.Println("Alert manager started")
}

// Stop stops the alert manager
func (am *AlertManager) Stop() {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	if !am.isRunning {
		return
	}
	
	am.isRunning = false
	close(am.stopChan)
	log.Println("Alert manager stopped")
}

// GetAlerts returns all alerts
func (am *AlertManager) GetAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	alerts := make([]*Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		alerts = append(alerts, alert)
	}
	
	return alerts
}

// GetActiveAlerts returns unresolved alerts
func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	var activeAlerts []*Alert
	for _, alert := range am.alerts {
		if !alert.Resolved {
			activeAlerts = append(activeAlerts, alert)
		}
	}
	
	return activeAlerts
}

// ResolveAlert marks an alert as resolved
func (am *AlertManager) ResolveAlert(alertID string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	if alert, exists := am.alerts[alertID]; exists {
		alert.Resolved = true
		now := time.Now()
		alert.ResolvedAt = &now
	}
}

// CheckMemoryAnomaly checks for memory anomalies and triggers alerts if needed
func (am *AlertManager) CheckMemoryAnomaly(metrics *MemoryMetrics) error {
	am.checkRules()
	return nil
}

// monitoringLoop runs the main monitoring loop
func (am *AlertManager) monitoringLoop(checkInterval time.Duration) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			am.checkRules()
		case <-am.stopChan:
			return
		}
	}
}

// checkRules evaluates all alert rules
func (am *AlertManager) checkRules() {
	metrics := am.profiler.GetCurrentMetrics()
	
	am.mu.Lock()
	defer am.mu.Unlock()
	
	for _, rule := range am.rules {
		// Check cooldown
		if time.Since(rule.lastFired) < rule.Cooldown {
			continue
		}
		
		// Evaluate condition
		if rule.Condition(metrics) {
			alert := &Alert{
				ID:        fmt.Sprintf("%s_%d", rule.Name, time.Now().Unix()),
				Level:     rule.Level,
				Title:     rule.Name,
				Message:   rule.Message,
				Timestamp: time.Now(),
				Metrics:   metrics,
				Resolved:  false,
			}
			
			am.alerts[alert.ID] = alert
			rule.lastFired = time.Now()
			
			// Send alert to handlers
			for _, handler := range am.handlers {
				go func(h AlertHandler, a *Alert) {
					if err := h.HandleAlert(a); err != nil {
						log.Printf("Alert handler error: %v", err)
					}
				}(handler, alert)
			}
		}
	}
}

// addDefaultRules adds standard memory monitoring rules
func (am *AlertManager) addDefaultRules() {
	// High memory usage warning
	am.rules = append(am.rules, &AlertRule{
		Name:     "HighMemoryUsage",
		Level:    AlertLevelWarning,
		Cooldown: 5 * time.Minute,
		Condition: func(m *MemoryMetrics) bool {
			return m.Alloc > 100*1024*1024 // 100MB
		},
		Message: "Memory usage is high (>100MB)",
	})
	
	// Critical memory usage
	am.rules = append(am.rules, &AlertRule{
		Name:     "CriticalMemoryUsage",
		Level:    AlertLevelCritical,
		Cooldown: 2 * time.Minute,
		Condition: func(m *MemoryMetrics) bool {
			return m.Alloc > 500*1024*1024 // 500MB
		},
		Message: "Memory usage is critical (>500MB)",
	})
	
	// High GC frequency
	am.rules = append(am.rules, &AlertRule{
		Name:     "HighGCFrequency",
		Level:    AlertLevelWarning,
		Cooldown: 10 * time.Minute,
		Condition: func(m *MemoryMetrics) bool {
			// This is a simplified check - in practice you'd track GC rate over time
			return m.NumGC > 1000
		},
		Message: "High garbage collection frequency detected",
	})
	
	// Memory leak detection (simplified)
	am.rules = append(am.rules, &AlertRule{
		Name:     "PotentialMemoryLeak",
		Level:    AlertLevelCritical,
		Cooldown: 30 * time.Minute,
		Condition: func(m *MemoryMetrics) bool {
			// Simplified leak detection: high heap objects count
			return m.HeapObjects > 1000000 // 1M objects
		},
		Message: "Potential memory leak detected (high object count)",
	})
	
	// Profiler health check
		am.rules = append(am.rules, &AlertRule{
			Name:     "ProfilerNotRunning",
			Level:    AlertLevelWarning,
			Cooldown: 15 * time.Minute,
			Condition: func(m *MemoryMetrics) bool {
				stats := am.profiler.GetStats()
				return !stats.Running
			},
			Message: "Continuous profiler is not running",
		})
}

// EmailAlertHandler sends alerts via email (placeholder implementation)
type EmailAlertHandler struct {
	SMTPServer string
	From       string
	To         []string
}

func (h *EmailAlertHandler) HandleAlert(alert *Alert) error {
	// Placeholder implementation
	log.Printf("EMAIL ALERT: [%s] %s - %s", alert.Level, alert.Title, alert.Message)
	return nil
}

// SlackAlertHandler sends alerts to Slack (placeholder implementation)
type SlackAlertHandler struct {
	WebhookURL string
	Channel    string
}

func (h *SlackAlertHandler) HandleAlert(alert *Alert) error {
	// Placeholder implementation
	log.Printf("SLACK ALERT: [%s] %s - %s", alert.Level, alert.Title, alert.Message)
	return nil
}

// AlertingConfig contains alerting configuration
type AlertingConfig struct {
	Enabled       bool          `json:"enabled"`
	CheckInterval time.Duration `json:"check_interval"`
	EmailConfig   *EmailConfig  `json:"email_config,omitempty"`
	SlackConfig   *SlackConfig  `json:"slack_config,omitempty"`
}

type EmailConfig struct {
	SMTPServer string   `json:"smtp_server"`
	Port       int      `json:"port"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	From       string   `json:"from"`
	To         []string `json:"to"`
}

type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel"`
	Username   string `json:"username"`
}

// DefaultAlertingConfig returns default alerting configuration
func DefaultAlertingConfig() *AlertingConfig {
	return &AlertingConfig{
		Enabled:       true,
		CheckInterval: 30 * time.Second,
	}
}

// GetMemoryPressure calculates current memory pressure as a percentage
func GetMemoryPressure() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Simple calculation: allocated / system memory
	if m.Sys == 0 {
		return 0
	}
	return float64(m.Alloc) / float64(m.Sys) * 100
}

// AlertingExample demonstrates how to use the alerting system
func AlertingExample() {
	// This is an example function, not meant to be called from main
	profiler := NewContinuousProfiler(30*time.Second, "./profiles")
	alertManager := NewAlertManager(profiler)
	
	// Add custom alert handler
	emailHandler := &EmailAlertHandler{
		SMTPServer: "smtp.example.com",
		From:       "alerts@example.com",
		To:         []string{"admin@example.com"},
	}
	alertManager.AddHandler(emailHandler)
	
	// Add custom rule
	customRule := &AlertRule{
		Name:     "CustomMemoryRule",
		Level:    AlertLevelWarning,
		Cooldown: 5 * time.Minute,
		Condition: func(m *MemoryMetrics) bool {
			return GetMemoryPressure() > 80 // 80% memory pressure
		},
		Message: "Memory pressure is above 80%",
	}
	alertManager.AddRule(customRule)
	
	// Start monitoring
	alertManager.Start(30 * time.Second)
	
	// In a real application, you would keep this running
	// and handle shutdown gracefully
}
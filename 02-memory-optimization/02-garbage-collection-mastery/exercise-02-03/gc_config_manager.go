package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// GCConfigManager manages GC configurations for different environments and scenarios
type GCConfigManager struct {
	mu                sync.RWMutex
	configurations    map[string]*GCConfig
	environments      map[string]*EnvironmentConfig
	scenarios         map[string]*ScenarioConfig
	currentProfile    string
	defaultConfig     *GCConfig
	configHistory     []*ConfigChange
	validationRules   []*ValidationRule
}

// EnvironmentConfig represents environment-specific GC settings
type EnvironmentConfig struct {
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	BaseConfig        *GCConfig         `json:"base_config"`
	Constraints       *Constraints      `json:"constraints"`
	Monitoring        *MonitoringConfig `json:"monitoring"`
	AutoTuning        bool              `json:"auto_tuning"`
	TuningInterval    time.Duration     `json:"tuning_interval"`
	EnvironmentVars   map[string]string `json:"environment_vars"`
}

// ScenarioConfig represents scenario-specific GC configurations
type ScenarioConfig struct {
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	TriggerConditions []*TriggerCondition    `json:"trigger_conditions"`
	TargetConfig      *GCConfig              `json:"target_config"`
	FallbackConfig    *GCConfig              `json:"fallback_config"`
	Duration          time.Duration          `json:"duration"`
	Priority          int                    `json:"priority"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// Constraints define limits and requirements for GC configuration
type Constraints struct {
	MaxPauseTime      time.Duration `json:"max_pause_time"`
	MaxMemoryUsage    int64         `json:"max_memory_usage"`
	MinThroughput     float64       `json:"min_throughput"`
	MaxGCOverhead     float64       `json:"max_gc_overhead"`
	MaxHeapSize       int64         `json:"max_heap_size"`
	MinHeapUtilization float64      `json:"min_heap_utilization"`
	MaxAllocationRate float64       `json:"max_allocation_rate"`
}

// MonitoringConfig defines monitoring and alerting settings
type MonitoringConfig struct {
	Enabled           bool              `json:"enabled"`
	Interval          time.Duration     `json:"interval"`
	Metrics           []string          `json:"metrics"`
	Alerts            []*AlertRule      `json:"alerts"`
	Dashboard         *DashboardConfig  `json:"dashboard"`
	ExportFormat      string            `json:"export_format"`
	RetentionPeriod   time.Duration     `json:"retention_period"`
}

// TriggerCondition defines when a scenario should be activated
type TriggerCondition struct {
	Metric            string      `json:"metric"`
	Operator          string      `json:"operator"` // ">", "<", ">=", "<=", "==", "!="
	Threshold         float64     `json:"threshold"`
	Duration          time.Duration `json:"duration"`
	ConsecutiveChecks int         `json:"consecutive_checks"`
}

// ConfigChange represents a configuration change event
type ConfigChange struct {
	Timestamp     time.Time   `json:"timestamp"`
	FromConfig    *GCConfig   `json:"from_config"`
	ToConfig      *GCConfig   `json:"to_config"`
	Reason        string      `json:"reason"`
	TriggeredBy   string      `json:"triggered_by"`
	Environment   string      `json:"environment"`
	Scenario      string      `json:"scenario"`
	Metrics       *GCMetrics  `json:"metrics"`
}

// ValidationRule defines rules for validating GC configurations
type ValidationRule struct {
	Name        string                           `json:"name"`
	Description string                           `json:"description"`
	Validator   func(*GCConfig) (bool, string)   `json:"-"`
	Severity    string                           `json:"severity"` // "error", "warning", "info"
	Enabled     bool                             `json:"enabled"`
}

// AlertRule defines alerting rules for GC metrics
type AlertRule struct {
	Name        string        `json:"name"`
	Metric      string        `json:"metric"`
	Condition   string        `json:"condition"`
	Threshold   float64       `json:"threshold"`
	Duration    time.Duration `json:"duration"`
	Severity    string        `json:"severity"`
	Message     string        `json:"message"`
	Enabled     bool          `json:"enabled"`
}

// DashboardConfig defines dashboard configuration
type DashboardConfig struct {
	Enabled     bool     `json:"enabled"`
	Port        int      `json:"port"`
	RefreshRate int      `json:"refresh_rate"`
	Charts      []string `json:"charts"`
	Theme       string   `json:"theme"`
}

// NewGCConfigManager creates a new GC configuration manager
func NewGCConfigManager() *GCConfigManager {
	manager := &GCConfigManager{
		configurations:  make(map[string]*GCConfig),
		environments:    make(map[string]*EnvironmentConfig),
		scenarios:       make(map[string]*ScenarioConfig),
		configHistory:   make([]*ConfigChange, 0),
		validationRules: make([]*ValidationRule, 0),
		defaultConfig:   getCurrentGCConfig(),
	}
	
	// Initialize with default configurations
	manager.initializeDefaultConfigurations()
	manager.initializeValidationRules()
	
	return manager
}

// initializeDefaultConfigurations sets up default configurations for common scenarios
func (manager *GCConfigManager) initializeDefaultConfigurations() {
	// Development environment
	manager.environments["development"] = &EnvironmentConfig{
		Name:        "Development",
		Description: "Development environment with debugging enabled",
		BaseConfig: &GCConfig{
			GOGC:             100,
			GOMAXPROCS:       runtime.NumCPU(),
			GCPercent:        100,
			MemoryLimit:      -1, // No limit
			GCCPUFraction:    0.25,
			EnableGCTrace:    true,
			EnableMemProfile: true,
		},
		Constraints: &Constraints{
			MaxPauseTime:   time.Millisecond * 100,
			MaxGCOverhead:  30.0,
			MaxHeapSize:    1024 * 1024 * 1024, // 1GB
		},
		Monitoring: &MonitoringConfig{
			Enabled:  true,
			Interval: time.Second * 5,
			Metrics:  []string{"pause_time", "gc_frequency", "heap_size"},
		},
		AutoTuning:     false,
		EnvironmentVars: map[string]string{
			"GODEBUG": "gctrace=1",
		},
	}
	
	// Production environment
	manager.environments["production"] = &EnvironmentConfig{
		Name:        "Production",
		Description: "Production environment optimized for performance",
		BaseConfig: &GCConfig{
			GOGC:             100,
			GOMAXPROCS:       runtime.NumCPU(),
			GCPercent:        100,
			MemoryLimit:      -1,
			GCCPUFraction:    0.25,
			EnableGCTrace:    false,
			EnableMemProfile: false,
		},
		Constraints: &Constraints{
			MaxPauseTime:      time.Millisecond * 10,
			MaxGCOverhead:     5.0,
			MinThroughput:     1000.0,
			MinHeapUtilization: 70.0,
		},
		Monitoring: &MonitoringConfig{
			Enabled:  true,
			Interval: time.Second * 30,
			Metrics:  []string{"pause_time", "throughput", "gc_overhead"},
			Alerts: []*AlertRule{
				{
					Name:      "HighPauseTime",
					Metric:    "pause_time",
					Condition: ">",
					Threshold: 10.0,
					Duration:  time.Minute,
					Severity:  "critical",
					Message:   "GC pause time exceeded 10ms",
					Enabled:   true,
				},
			},
		},
		AutoTuning:     true,
		TuningInterval: time.Minute * 5,
	}
	
	// Testing environment
	manager.environments["testing"] = &EnvironmentConfig{
		Name:        "Testing",
		Description: "Testing environment with consistent performance",
		BaseConfig: &GCConfig{
			GOGC:             100,
			GOMAXPROCS:       2, // Limited for consistent results
			GCPercent:        100,
			MemoryLimit:      512 * 1024 * 1024, // 512MB
			GCCPUFraction:    0.25,
			EnableGCTrace:    false,
			EnableMemProfile: false,
		},
		Constraints: &Constraints{
			MaxPauseTime:   time.Millisecond * 50,
			MaxMemoryUsage: 512 * 1024 * 1024,
			MaxGCOverhead:  15.0,
		},
		Monitoring: &MonitoringConfig{
			Enabled:  true,
			Interval: time.Second,
			Metrics:  []string{"pause_time", "memory_usage", "gc_frequency"},
		},
		AutoTuning: false,
	}
	
	// High-throughput scenario
	manager.scenarios["high_throughput"] = &ScenarioConfig{
		Name:        "High Throughput",
		Description: "Optimize for maximum throughput",
		TriggerConditions: []*TriggerCondition{
			{
				Metric:            "throughput",
				Operator:          "<",
				Threshold:         500.0,
				Duration:          time.Minute,
				ConsecutiveChecks: 3,
			},
		},
		TargetConfig: &GCConfig{
			GOGC:          200,
			GCPercent:     200,
			GCCPUFraction: 0.5,
		},
		Duration: time.Minute * 10,
		Priority: 1,
	}
	
	// Low-latency scenario
	manager.scenarios["low_latency"] = &ScenarioConfig{
		Name:        "Low Latency",
		Description: "Optimize for minimal pause times",
		TriggerConditions: []*TriggerCondition{
			{
				Metric:            "pause_time",
				Operator:          ">",
				Threshold:         5.0,
				Duration:          time.Second * 30,
				ConsecutiveChecks: 2,
			},
		},
		TargetConfig: &GCConfig{
			GOGC:          50,
			GCPercent:     50,
			GCCPUFraction: 0.1,
		},
		Duration: time.Minute * 5,
		Priority: 2,
	}
	
	// Memory pressure scenario
	manager.scenarios["memory_pressure"] = &ScenarioConfig{
		Name:        "Memory Pressure",
		Description: "Aggressive GC under memory pressure",
		TriggerConditions: []*TriggerCondition{
			{
				Metric:            "heap_utilization",
				Operator:          ">",
				Threshold:         85.0,
				Duration:          time.Second * 10,
				ConsecutiveChecks: 1,
			},
		},
		TargetConfig: &GCConfig{
			GOGC:          25,
			GCPercent:     25,
			GCCPUFraction: 0.5,
		},
		Duration: time.Minute * 2,
		Priority: 3,
	}
}

// initializeValidationRules sets up validation rules for GC configurations
func (manager *GCConfigManager) initializeValidationRules() {
	manager.validationRules = []*ValidationRule{
		{
			Name:        "ValidGOGC",
			Description: "GOGC should be between 10 and 1000",
			Validator: func(config *GCConfig) (bool, string) {
				if config.GOGC < 10 || config.GOGC > 1000 {
					return false, fmt.Sprintf("GOGC value %d is outside valid range [10, 1000]", config.GOGC)
				}
				return true, ""
			},
			Severity: "error",
			Enabled:  true,
		},
		{
			Name:        "ValidGOMAXPROCS",
			Description: "GOMAXPROCS should be between 1 and available CPUs",
			Validator: func(config *GCConfig) (bool, string) {
				if config.GOMAXPROCS < 1 || config.GOMAXPROCS > runtime.NumCPU()*2 {
					return false, fmt.Sprintf("GOMAXPROCS value %d is outside valid range [1, %d]", 
						config.GOMAXPROCS, runtime.NumCPU()*2)
				}
				return true, ""
			},
			Severity: "warning",
			Enabled:  true,
		},
		{
			Name:        "ValidGCCPUFraction",
			Description: "GC CPU fraction should be between 0.01 and 0.8",
			Validator: func(config *GCConfig) (bool, string) {
				if config.GCCPUFraction < 0.01 || config.GCCPUFraction > 0.8 {
					return false, fmt.Sprintf("GC CPU fraction %.2f is outside valid range [0.01, 0.8]", 
						config.GCCPUFraction)
				}
				return true, ""
			},
			Severity: "warning",
			Enabled:  true,
		},
		{
			Name:        "MemoryLimitCheck",
			Description: "Memory limit should be reasonable if set",
			Validator: func(config *GCConfig) (bool, string) {
				if config.MemoryLimit > 0 && config.MemoryLimit < 64*1024*1024 {
					return false, fmt.Sprintf("Memory limit %d bytes is too low (minimum 64MB)", 
						config.MemoryLimit)
				}
				return true, ""
			},
			Severity: "warning",
			Enabled:  true,
		},
	}
}

// LoadConfiguration loads a configuration from file
func (manager *GCConfigManager) LoadConfiguration(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config GCConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// Validate configuration
	if valid, message := manager.ValidateConfiguration(&config); !valid {
		return fmt.Errorf("invalid configuration: %s", message)
	}
	
	manager.mu.Lock()
	manager.configurations[filename] = &config
	manager.mu.Unlock()
	
	return nil
}

// SaveConfiguration saves a configuration to file
func (manager *GCConfigManager) SaveConfiguration(config *GCConfig, filename string) error {
	// Validate configuration before saving
	if valid, message := manager.ValidateConfiguration(config); !valid {
		return fmt.Errorf("invalid configuration: %s", message)
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	manager.mu.Lock()
	manager.configurations[filename] = config
	manager.mu.Unlock()
	
	return nil
}

// ValidateConfiguration validates a GC configuration against all rules
func (manager *GCConfigManager) ValidateConfiguration(config *GCConfig) (bool, string) {
	var errors []string
	var warnings []string
	
	for _, rule := range manager.validationRules {
		if !rule.Enabled {
			continue
		}
		
		valid, message := rule.Validator(config)
		if !valid {
			switch rule.Severity {
			case "error":
				errors = append(errors, fmt.Sprintf("[%s] %s", rule.Name, message))
			case "warning":
				warnings = append(warnings, fmt.Sprintf("[%s] %s", rule.Name, message))
			}
		}
	}
	
	if len(errors) > 0 {
		return false, strings.Join(errors, "; ")
	}
	
	if len(warnings) > 0 {
		fmt.Printf("Configuration warnings: %s\n", strings.Join(warnings, "; "))
	}
	
	return true, ""
}

// ApplyEnvironmentConfiguration applies configuration for a specific environment
func (manager *GCConfigManager) ApplyEnvironmentConfiguration(envName string) error {
	manager.mu.RLock()
	env, exists := manager.environments[envName]
	manager.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("environment '%s' not found", envName)
	}
	
	// Apply environment variables
	for key, value := range env.EnvironmentVars {
		os.Setenv(key, value)
	}
	
	// Apply GC configuration
	err := manager.applyGCConfig(env.BaseConfig)
	if err != nil {
		return fmt.Errorf("failed to apply environment config: %w", err)
	}
	
	// Record configuration change
	change := &ConfigChange{
		Timestamp:   time.Now(),
		FromConfig:  manager.defaultConfig,
		ToConfig:    env.BaseConfig,
		Reason:      "Environment configuration applied",
		TriggeredBy: "manual",
		Environment: envName,
	}
	
	manager.mu.Lock()
	manager.configHistory = append(manager.configHistory, change)
	manager.currentProfile = envName
	manager.mu.Unlock()
	
	return nil
}

// applyGCConfig applies a GC configuration to the runtime
func (manager *GCConfigManager) applyGCConfig(config *GCConfig) error {
	// Apply GOGC
	if config.GOGC >= 0 {
		debug.SetGCPercent(config.GOGC)
	}
	
	// Apply GOMAXPROCS
	if config.GOMAXPROCS > 0 {
		runtime.GOMAXPROCS(config.GOMAXPROCS)
	}
	
	// Apply memory limit
	if config.MemoryLimit > 0 {
		debug.SetMemoryLimit(config.MemoryLimit)
	}
	
	// Apply GC percent (alternative to GOGC)
	if config.GCPercent >= 0 && config.GCPercent != config.GOGC {
		debug.SetGCPercent(config.GCPercent)
	}
	
	return nil
}

// GetEnvironmentConfiguration returns configuration for a specific environment
func (manager *GCConfigManager) GetEnvironmentConfiguration(envName string) (*EnvironmentConfig, error) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	
	env, exists := manager.environments[envName]
	if !exists {
		return nil, fmt.Errorf("environment '%s' not found", envName)
	}
	
	return env, nil
}

// ListEnvironments returns all available environment names
func (manager *GCConfigManager) ListEnvironments() []string {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	
	envs := make([]string, 0, len(manager.environments))
	for name := range manager.environments {
		envs = append(envs, name)
	}
	
	return envs
}

// ListScenarios returns all available scenario names
func (manager *GCConfigManager) ListScenarios() []string {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	
	scenarios := make([]string, 0, len(manager.scenarios))
	for name := range manager.scenarios {
		scenarios = append(scenarios, name)
	}
	
	return scenarios
}

// CheckScenarioTriggers checks if any scenario should be triggered based on current metrics
func (manager *GCConfigManager) CheckScenarioTriggers(metrics *GCMetrics) []*ScenarioConfig {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	
	triggeredScenarios := make([]*ScenarioConfig, 0)
	
	for _, scenario := range manager.scenarios {
		if manager.shouldTriggerScenario(scenario, metrics) {
			triggeredScenarios = append(triggeredScenarios, scenario)
		}
	}
	
	return triggeredScenarios
}

// shouldTriggerScenario checks if a scenario should be triggered
func (manager *GCConfigManager) shouldTriggerScenario(scenario *ScenarioConfig, metrics *GCMetrics) bool {
	for _, condition := range scenario.TriggerConditions {
		if !manager.evaluateCondition(condition, metrics) {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates a trigger condition against metrics
func (manager *GCConfigManager) evaluateCondition(condition *TriggerCondition, metrics *GCMetrics) bool {
	var value float64
	
	// Extract metric value
	switch condition.Metric {
	case "pause_time":
		value = metrics.AveragePause
	case "gc_frequency":
		value = metrics.GCFrequency
	case "heap_utilization":
		value = metrics.HeapUtilization
	case "gc_overhead":
		value = metrics.GCOverhead
	case "throughput":
		value = metrics.Throughput
	case "allocation_rate":
		value = metrics.AllocationRate
	default:
		return false
	}
	
	// Evaluate condition
	switch condition.Operator {
	case ">":
		return value > condition.Threshold
	case "<":
		return value < condition.Threshold
	case ">=":
		return value >= condition.Threshold
	case "<=":
		return value <= condition.Threshold
	case "==":
		return value == condition.Threshold
	case "!=":
		return value != condition.Threshold
	default:
		return false
	}
}

// GetConfigurationHistory returns the configuration change history
func (manager *GCConfigManager) GetConfigurationHistory() []*ConfigChange {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	
	history := make([]*ConfigChange, len(manager.configHistory))
	copy(history, manager.configHistory)
	
	return history
}

// GenerateConfigurationReport generates a comprehensive configuration report
func (manager *GCConfigManager) GenerateConfigurationReport() string {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	
	report := "\n=== GC Configuration Report ===\n\n"
	
	// Current profile
	report += fmt.Sprintf("Current Profile: %s\n\n", manager.currentProfile)
	
	// Available environments
	report += "Available Environments:\n"
	for name, env := range manager.environments {
		report += fmt.Sprintf("  %s: %s\n", name, env.Description)
		report += fmt.Sprintf("    GOGC: %d, GOMAXPROCS: %d\n", 
			env.BaseConfig.GOGC, env.BaseConfig.GOMAXPROCS)
		report += fmt.Sprintf("    Auto-tuning: %v\n", env.AutoTuning)
	}
	report += "\n"
	
	// Available scenarios
	report += "Available Scenarios:\n"
	for name, scenario := range manager.scenarios {
		report += fmt.Sprintf("  %s: %s\n", name, scenario.Description)
		report += fmt.Sprintf("    Priority: %d, Duration: %v\n", 
			scenario.Priority, scenario.Duration)
		report += fmt.Sprintf("    Triggers: %d conditions\n", len(scenario.TriggerConditions))
	}
	report += "\n"
	
	// Configuration history
	if len(manager.configHistory) > 0 {
		report += "Recent Configuration Changes:\n"
		for i, change := range manager.configHistory {
			if i >= 5 { // Show only last 5 changes
				break
			}
			report += fmt.Sprintf("  %s: %s\n", 
				change.Timestamp.Format("2006-01-02 15:04:05"), change.Reason)
			report += fmt.Sprintf("    Environment: %s, Scenario: %s\n", 
				change.Environment, change.Scenario)
		}
		report += "\n"
	}
	
	// Validation rules
	report += "Active Validation Rules:\n"
	for _, rule := range manager.validationRules {
		if rule.Enabled {
			report += fmt.Sprintf("  %s (%s): %s\n", 
				rule.Name, rule.Severity, rule.Description)
		}
	}
	
	return report
}

// DemonstrateConfigurationManagement demonstrates configuration management capabilities
func DemonstrateConfigurationManagement() {
	fmt.Println("=== GC Configuration Management Demonstration ===")
	
	manager := NewGCConfigManager()
	
	// List available environments
	fmt.Println("\nAvailable environments:")
	for _, env := range manager.ListEnvironments() {
		fmt.Printf("  - %s\n", env)
	}
	
	// Apply development environment
	fmt.Println("\nApplying development environment...")
	err := manager.ApplyEnvironmentConfiguration("development")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Development environment applied successfully")
	}
	
	// Test configuration validation
	fmt.Println("\nTesting configuration validation...")
	invalidConfig := &GCConfig{
		GOGC:          5, // Invalid: too low
		GOMAXPROCS:    1000, // Invalid: too high
		GCCPUFraction: 0.9, // Invalid: too high
	}
	
	valid, message := manager.ValidateConfiguration(invalidConfig)
	if !valid {
		fmt.Printf("Validation failed (as expected): %s\n", message)
	}
	
	// Create and save a custom configuration
	fmt.Println("\nCreating custom configuration...")
	customConfig := &GCConfig{
		GOGC:          150,
		GOMAXPROCS:    runtime.NumCPU(),
		GCPercent:     150,
		MemoryLimit:   1024 * 1024 * 1024, // 1GB
		GCCPUFraction: 0.3,
	}
	
	err = manager.SaveConfiguration(customConfig, "custom_config.json")
	if err != nil {
		fmt.Printf("Error saving config: %v\n", err)
	} else {
		fmt.Println("Custom configuration saved successfully")
	}
	
	// Generate and display report
	fmt.Println(manager.GenerateConfigurationReport())
	
	// Simulate scenario triggering
	fmt.Println("\nSimulating scenario triggering...")
	mockMetrics := &GCMetrics{
		AveragePause:     15.0, // High pause time
		GCFrequency:      5.0,
		HeapUtilization:  90.0, // High heap utilization
		GCOverhead:       8.0,
		Throughput:       300.0, // Low throughput
	}
	
	triggeredScenarios := manager.CheckScenarioTriggers(mockMetrics)
	if len(triggeredScenarios) > 0 {
		fmt.Printf("Triggered scenarios: %d\n", len(triggeredScenarios))
		for _, scenario := range triggeredScenarios {
			fmt.Printf("  - %s: %s\n", scenario.Name, scenario.Description)
		}
	} else {
		fmt.Println("No scenarios triggered")
	}
	
	// Clean up
	os.Remove("custom_config.json")
}
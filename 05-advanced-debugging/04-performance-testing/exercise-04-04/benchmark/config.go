package benchmark

import (
	"encoding/json"
	"os"
	"time"
)

// BenchmarkConfig defines the configuration for performance benchmarks
type BenchmarkConfig struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Duration    time.Duration `json:"duration"`
	WarmupTime  time.Duration `json:"warmup_time"`
	CooldownTime time.Duration `json:"cooldown_time"`
	Targets     []Target      `json:"targets"`
	Scenarios   []Scenario    `json:"scenarios"`
	Metrics     []MetricConfig `json:"metrics"`
	Thresholds  Thresholds    `json:"thresholds"`
	Baseline    BaselineConfig `json:"baseline"`
}

// Target represents a system or endpoint to benchmark
type Target struct {
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Type        string            `json:"type"` // "http", "grpc", "database", "custom"
	Headers     map[string]string `json:"headers,omitempty"`
	Timeout     time.Duration     `json:"timeout"`
	HealthCheck string            `json:"health_check,omitempty"`
}

// Scenario defines a specific test scenario
type Scenario struct {
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Weight        float64       `json:"weight"` // Percentage of total load
	Concurrency   int           `json:"concurrency"`
	RequestRate   int           `json:"request_rate"` // requests per second
	Duration      time.Duration `json:"duration,omitempty"`
	RampUpTime    time.Duration `json:"ramp_up_time"`
	RampDownTime  time.Duration `json:"ramp_down_time"`
	ThinkTime     time.Duration `json:"think_time"`
	Requests      []Request     `json:"requests"`
	DataSets      []DataSet     `json:"data_sets,omitempty"`
}

// Request defines a specific request pattern
type Request struct {
	Name     string            `json:"name"`
	Method   string            `json:"method"`
	Path     string            `json:"path"`
	Headers  map[string]string `json:"headers,omitempty"`
	Body     string            `json:"body,omitempty"`
	Weight   float64           `json:"weight"`
	Validation Validation      `json:"validation,omitempty"`
}

// DataSet defines test data for scenarios
type DataSet struct {
	Name   string                 `json:"name"`
	Type   string                 `json:"type"` // "csv", "json", "generator"
	Source string                 `json:"source"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// Validation defines response validation rules
type Validation struct {
	StatusCode   int               `json:"status_code,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	BodyContains []string          `json:"body_contains,omitempty"`
	BodyRegex    string            `json:"body_regex,omitempty"`
	JSONPath     map[string]string `json:"json_path,omitempty"`
}

// MetricConfig defines which metrics to collect
type MetricConfig struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"` // "latency", "throughput", "error_rate", "custom"
	Unit        string  `json:"unit"`
	Aggregation string  `json:"aggregation"` // "avg", "p50", "p95", "p99", "max", "min"
	Threshold   float64 `json:"threshold"`   // Threshold value for this metric
	Tolerance   float64 `json:"tolerance"`   // Percentage tolerance for baseline comparison
	Critical    bool    `json:"critical"`    // Whether this metric is critical for SLA
}

// Thresholds define performance thresholds
type Thresholds struct {
	Latency     LatencyThresholds     `json:"latency"`
	Throughput  ThroughputThresholds  `json:"throughput"`
	ErrorRate   ErrorRateThresholds   `json:"error_rate"`
	Resources   ResourceThresholds    `json:"resources"`
}

// LatencyThresholds define latency-related thresholds
type LatencyThresholds struct {
	P50  time.Duration `json:"p50"`
	P90  time.Duration `json:"p90"`
	P95  time.Duration `json:"p95"`
	P99  time.Duration `json:"p99"`
	Max  time.Duration `json:"max"`
}

// ThroughputThresholds define throughput-related thresholds
type ThroughputThresholds struct {
	MinRPS float64 `json:"min_rps"`
	MaxRPS float64 `json:"max_rps"`
}

// ErrorRateThresholds define error rate thresholds
type ErrorRateThresholds struct {
	MaxPercent float64 `json:"max_percent"`
	MaxCount   int64   `json:"max_count"`
}

// ResourceThresholds define resource usage thresholds
type ResourceThresholds struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskIOPS      float64 `json:"disk_iops"`
	NetworkMbps   float64 `json:"network_mbps"`
}

// BaselineConfig defines baseline management configuration
type BaselineConfig struct {
	Enabled     bool   `json:"enabled"`
	Strategy    string `json:"strategy"` // "auto", "manual", "branch", "tag"
	Path        string `json:"path"`
	Version     string `json:"version,omitempty"`
	AutoUpdate  bool   `json:"auto_update"`
	Tolerance   float64 `json:"tolerance"` // Default tolerance percentage
	Environment string `json:"environment,omitempty"` // Environment identifier
}

// CapacityConfig defines capacity planning configuration
type CapacityConfig struct {
	Name         string              `json:"name"`
	Description  string              `json:"description"`
	Targets      []Target            `json:"targets"`
	LoadProfiles []LoadProfile       `json:"load_profiles"`
	Metrics      []MetricConfig      `json:"metrics"`
	Prediction   PredictionConfig    `json:"prediction"`
	Scaling      ScalingConfig       `json:"scaling"`
}

// LoadProfile defines different load patterns for capacity testing
type LoadProfile struct {
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Pattern      string        `json:"pattern"` // "constant", "ramp", "spike", "step", "wave"
	StartLoad    int           `json:"start_load"`
	EndLoad      int           `json:"end_load"`
	Duration     time.Duration `json:"duration"`
	StepSize     int           `json:"step_size,omitempty"`
	StepDuration time.Duration `json:"step_duration,omitempty"`
}

// PredictionConfig defines prediction parameters
type PredictionConfig struct {
	Enabled    bool    `json:"enabled"`
	Algorithm  string  `json:"algorithm"` // "linear", "polynomial", "exponential"
	Confidence float64 `json:"confidence"`
	Horizon    string  `json:"horizon"` // "1d", "1w", "1m", "3m", "6m", "1y"
}

// ScalingConfig defines auto-scaling parameters
type ScalingConfig struct {
	Enabled       bool    `json:"enabled"`
	MinInstances  int     `json:"min_instances"`
	MaxInstances  int     `json:"max_instances"`
	TargetCPU     float64 `json:"target_cpu"`
	TargetMemory  float64 `json:"target_memory"`
	ScaleUpDelay  time.Duration `json:"scale_up_delay"`
	ScaleDownDelay time.Duration `json:"scale_down_delay"`
}

// LoadConfig loads benchmark configuration from file
func LoadConfig(path string) (*BenchmarkConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config BenchmarkConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Set defaults
	if config.Duration == 0 {
		config.Duration = 5 * time.Minute
	}
	if config.WarmupTime == 0 {
		config.WarmupTime = 30 * time.Second
	}
	if config.CooldownTime == 0 {
		config.CooldownTime = 30 * time.Second
	}

	return &config, nil
}

// LoadBenchmarkConfig loads benchmark configuration from file (alias for LoadConfig)
func LoadBenchmarkConfig(path string) (*BenchmarkConfig, error) {
	return LoadConfig(path)
}

// SaveBenchmarkConfig saves benchmark configuration to file
func SaveBenchmarkConfig(config *BenchmarkConfig, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LoadCapacityConfig loads capacity planning configuration from file
func LoadCapacityConfig(path string) (*CapacityConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config CapacityConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig saves benchmark configuration to file
func (c *BenchmarkConfig) SaveConfig(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
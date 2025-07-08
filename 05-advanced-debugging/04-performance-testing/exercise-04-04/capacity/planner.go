package capacity

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"
)

// LoadPlanningConfig loads capacity planning configuration from file
func LoadPlanningConfig(path string) (*PlanningConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config PlanningConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// CapacityPlanner manages capacity planning and prediction
type CapacityPlanner struct {
	config    *PlanningConfig
	outputDir string
	verbose   bool
	data      []DataPoint
	models    map[string]PredictionModel
}

// PlanningConfig defines capacity planning configuration
type PlanningConfig struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Enabled     bool                `json:"enabled"`
	Metrics     []MetricConfig      `json:"metrics"`
	Models      []ModelConfig       `json:"models"`
	Scenarios   []ScenarioConfig    `json:"scenarios"`
	Thresholds  ThresholdConfig     `json:"thresholds"`
	Reporting   ReportingConfig     `json:"reporting"`
	DataSources []DataSourceConfig  `json:"data_sources"`
}

// MetricConfig defines a metric for capacity planning
type MetricConfig struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"` // "cpu", "memory", "disk", "network", "latency", "throughput"
	Unit        string  `json:"unit"`
	Weight      float64 `json:"weight"`
	Critical    bool    `json:"critical"`
	Threshold   float64 `json:"threshold"`
	Aggregation string  `json:"aggregation"` // "avg", "max", "p95", "p99"
}

// ModelConfig defines a prediction model
type ModelConfig struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // "linear", "polynomial", "exponential", "seasonal"
	Enabled    bool                   `json:"enabled"`
	Parameters map[string]interface{} `json:"parameters"`
	Horizon    time.Duration          `json:"horizon"` // Prediction horizon
	Confidence float64                `json:"confidence"` // Confidence level (0.0-1.0)
}

// ScenarioConfig defines a capacity planning scenario
type ScenarioConfig struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Enabled     bool                   `json:"enabled"`
	GrowthRate  float64                `json:"growth_rate"` // Expected growth rate (e.g., 0.1 for 10%)
	Timeframe   time.Duration          `json:"timeframe"`   // Planning timeframe
	Factors     map[string]interface{} `json:"factors"`     // Additional factors
}

// ThresholdConfig defines capacity thresholds
type ThresholdConfig struct {
	CPU     ResourceThreshold `json:"cpu"`
	Memory  ResourceThreshold `json:"memory"`
	Disk    ResourceThreshold `json:"disk"`
	Network ResourceThreshold `json:"network"`
	Latency LatencyThreshold  `json:"latency"`
}

// ResourceThreshold defines thresholds for a resource
type ResourceThreshold struct {
	Warning  float64 `json:"warning"`  // Warning threshold (e.g., 70%)
	Critical float64 `json:"critical"` // Critical threshold (e.g., 90%)
	Max      float64 `json:"max"`      // Maximum capacity
}

// LatencyThreshold defines latency thresholds
type LatencyThreshold struct {
	P50Warning  float64 `json:"p50_warning"`
	P50Critical float64 `json:"p50_critical"`
	P95Warning  float64 `json:"p95_warning"`
	P95Critical float64 `json:"p95_critical"`
	P99Warning  float64 `json:"p99_warning"`
	P99Critical float64 `json:"p99_critical"`
}

// ReportingConfig defines reporting configuration
type ReportingConfig struct {
	Enabled   bool          `json:"enabled"`
	Interval  time.Duration `json:"interval"`
	Format    string        `json:"format"` // "json", "html", "csv"
	Retention time.Duration `json:"retention"`
}

// DataSourceConfig defines a data source
type DataSourceConfig struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"` // "prometheus", "influxdb", "file", "api"
	Enabled  bool                   `json:"enabled"`
	Config   map[string]interface{} `json:"config"`
	Interval time.Duration          `json:"interval"`
}

// DataPoint represents a single data point
type DataPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]float64     `json:"metrics"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// PredictionModel interface for prediction models
type PredictionModel interface {
	Name() string
	Train(data []DataPoint) error
	Predict(horizon time.Duration) (*Prediction, error)
	Accuracy() float64
}

// Prediction represents a capacity prediction
type Prediction struct {
	Model       string                 `json:"model"`
	Metric      string                 `json:"metric"`
	Timestamp   time.Time              `json:"timestamp"`
	Horizon     time.Duration          `json:"horizon"`
	Values      []PredictionPoint      `json:"values"`
	Confidence  float64                `json:"confidence"`
	Accuracy    float64                `json:"accuracy"`
	Trend       string                 `json:"trend"` // "increasing", "decreasing", "stable"
	Metadata    map[string]interface{} `json:"metadata"`
}

// PredictionPoint represents a single prediction point
type PredictionPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	LowerBound float64  `json:"lower_bound"`
	UpperBound float64  `json:"upper_bound"`
}

// CapacityReport represents a capacity planning report
type CapacityReport struct {
	Name          string                    `json:"name"`
	Timestamp     time.Time                 `json:"timestamp"`
	Period        string                    `json:"period"`
	Predictions   map[string]*Prediction    `json:"predictions"`
	Recommendations []Recommendation        `json:"recommendations"`
	Scenarios     map[string]*ScenarioResult `json:"scenarios"`
	Risks         []Risk                    `json:"risks"`
	Summary       *CapacitySummary          `json:"summary"`
	Metadata      map[string]interface{}    `json:"metadata"`
}

// Recommendation represents a capacity recommendation
type Recommendation struct {
	Type        string                 `json:"type"` // "scale_up", "scale_out", "optimize", "monitor"
	Priority    string                 `json:"priority"` // "high", "medium", "low"
	Resource    string                 `json:"resource"`
	Current     float64                `json:"current"`
	Recommended float64                `json:"recommended"`
	Reason      string                 `json:"reason"`
	Timeframe   time.Duration          `json:"timeframe"`
	Cost        *CostEstimate          `json:"cost,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// CostEstimate represents a cost estimate
type CostEstimate struct {
	Currency string  `json:"currency"`
	Monthly  float64 `json:"monthly"`
	Annual   float64 `json:"annual"`
	OneTime  float64 `json:"one_time"`
}

// ScenarioResult represents the result of a scenario analysis
type ScenarioResult struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Predictions map[string]*Prediction `json:"predictions"`
	CapacityGap float64                `json:"capacity_gap"`
	TimeToLimit time.Duration          `json:"time_to_limit"`
	Risk        string                 `json:"risk"` // "low", "medium", "high"
	Actions     []string               `json:"actions"`
}

// Risk represents a capacity risk
type Risk struct {
	Type        string        `json:"type"` // "capacity_exhaustion", "performance_degradation", "cost_overrun"
	Severity    string        `json:"severity"` // "low", "medium", "high", "critical"
	Probability float64       `json:"probability"` // 0.0-1.0
	Impact      string        `json:"impact"`
	Timeframe   time.Duration `json:"timeframe"`
	Mitigation  []string      `json:"mitigation"`
}

// CapacitySummary provides a high-level summary
type CapacitySummary struct {
	OverallHealth     string        `json:"overall_health"`
	CapacityUtilization float64     `json:"capacity_utilization"`
	TimeToCapacity    time.Duration `json:"time_to_capacity"`
	GrowthRate        float64       `json:"growth_rate"`
	HighRiskMetrics   []string      `json:"high_risk_metrics"`
	RecommendationCount int         `json:"recommendation_count"`
	CostImpact        *CostEstimate `json:"cost_impact,omitempty"`
}

// LinearModel implements a simple linear regression model
type LinearModel struct {
	name       string
	metric     string
	slope      float64
	intercept  float64
	accuracy   float64
	trained    bool
}

// NewCapacityPlanner creates a new capacity planner
func NewCapacityPlanner(config *PlanningConfig, outputDir string) *CapacityPlanner {
	return &CapacityPlanner{
		config:    config,
		outputDir: outputDir,
		data:      make([]DataPoint, 0),
		models:    make(map[string]PredictionModel),
	}
}

// SetVerbose enables or disables verbose output
func (p *CapacityPlanner) SetVerbose(verbose bool) {
	p.verbose = verbose
}

// AddDataPoint adds a data point for analysis
func (p *CapacityPlanner) AddDataPoint(timestamp time.Time, metrics map[string]float64, metadata map[string]interface{}) {
	dataPoint := DataPoint{
		Timestamp: timestamp,
		Metrics:   metrics,
		Metadata:  metadata,
	}

	p.data = append(p.data, dataPoint)

	// Keep only recent data (e.g., last 30 days)
	cutoff := time.Now().AddDate(0, 0, -30)
	filtered := make([]DataPoint, 0)
	for _, dp := range p.data {
		if dp.Timestamp.After(cutoff) {
			filtered = append(filtered, dp)
		}
	}
	p.data = filtered

	if p.verbose {
		fmt.Printf("Added data point: %d metrics at %v\n", len(metrics), timestamp)
	}
}

// RunAnalysis performs capacity planning analysis
func (p *CapacityPlanner) RunAnalysis(ctx context.Context) (*CapacityReport, error) {
	if !p.config.Enabled {
		return nil, fmt.Errorf("capacity planning is disabled")
	}

	if p.verbose {
		fmt.Printf("Running capacity analysis: %s\n", p.config.Name)
		fmt.Printf("Analyzing %d data points\n", len(p.data))
	}

	// Initialize models
	if err := p.initializeModels(); err != nil {
		return nil, fmt.Errorf("failed to initialize models: %w", err)
	}

	// Train models
	if err := p.trainModels(); err != nil {
		return nil, fmt.Errorf("failed to train models: %w", err)
	}

	// Generate predictions
	predictions, err := p.generatePredictions()
	if err != nil {
		return nil, fmt.Errorf("failed to generate predictions: %w", err)
	}

	// Analyze scenarios
	scenarios, err := p.analyzeScenarios(predictions)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze scenarios: %w", err)
	}

	// Generate recommendations
	recommendations := p.generateRecommendations(predictions, scenarios)

	// Assess risks
	risks := p.assessRisks(predictions, scenarios)

	// Create report
	report := &CapacityReport{
		Name:            p.config.Name,
		Timestamp:       time.Now(),
		Period:          "current",
		Predictions:     predictions,
		Recommendations: recommendations,
		Scenarios:       scenarios,
		Risks:           risks,
		Metadata:        make(map[string]interface{}),
	}

	// Generate summary
	report.Summary = p.generateSummary(predictions, recommendations, risks)

	if p.verbose {
		fmt.Printf("Analysis complete: %d predictions, %d recommendations, %d risks\n",
			len(predictions), len(recommendations), len(risks))
	}

	return report, nil
}

// initializeModels initializes prediction models
func (p *CapacityPlanner) initializeModels() error {
	for _, modelConfig := range p.config.Models {
		if !modelConfig.Enabled {
			continue
		}

		switch modelConfig.Type {
		case "linear":
			for _, metricConfig := range p.config.Metrics {
				model := &LinearModel{
					name:   fmt.Sprintf("%s_%s", modelConfig.Name, metricConfig.Name),
					metric: metricConfig.Name,
				}
				p.models[model.name] = model
			}
		default:
			if p.verbose {
				fmt.Printf("Unsupported model type: %s\n", modelConfig.Type)
			}
		}
	}

	return nil
}

// trainModels trains all prediction models
func (p *CapacityPlanner) trainModels() error {
	if len(p.data) < 2 {
		return fmt.Errorf("insufficient data for training (need at least 2 points)")
	}

	for name, model := range p.models {
		if err := model.Train(p.data); err != nil {
			if p.verbose {
				fmt.Printf("Failed to train model %s: %v\n", name, err)
			}
			continue
		}

		if p.verbose {
			fmt.Printf("Trained model %s (accuracy: %.2f)\n", name, model.Accuracy())
		}
	}

	return nil
}

// generatePredictions generates predictions using trained models
func (p *CapacityPlanner) generatePredictions() (map[string]*Prediction, error) {
	predictions := make(map[string]*Prediction)

	for name, model := range p.models {
		// Use default horizon if not specified
		horizon := 7 * 24 * time.Hour // 7 days
		for _, modelConfig := range p.config.Models {
			if modelConfig.Horizon > 0 {
				horizon = modelConfig.Horizon
				break
			}
		}

		prediction, err := model.Predict(horizon)
		if err != nil {
			if p.verbose {
				fmt.Printf("Failed to generate prediction for %s: %v\n", name, err)
			}
			continue
		}

		predictions[name] = prediction
	}

	return predictions, nil
}

// analyzeScenarios analyzes different capacity scenarios
func (p *CapacityPlanner) analyzeScenarios(predictions map[string]*Prediction) (map[string]*ScenarioResult, error) {
	scenarios := make(map[string]*ScenarioResult)

	for _, scenarioConfig := range p.config.Scenarios {
		if !scenarioConfig.Enabled {
			continue
		}

		result := &ScenarioResult{
			Name:        scenarioConfig.Name,
			Description: scenarioConfig.Description,
			Predictions: make(map[string]*Prediction),
			Actions:     make([]string, 0),
		}

		// Apply growth rate to predictions
		for name, prediction := range predictions {
			adjustedPrediction := p.applyGrowthRate(prediction, scenarioConfig.GrowthRate)
			result.Predictions[name] = adjustedPrediction
		}

		// Analyze capacity gap and time to limit
		result.CapacityGap = p.calculateCapacityGap(result.Predictions)
		result.TimeToLimit = p.calculateTimeToLimit(result.Predictions)
		result.Risk = p.assessScenarioRisk(result)

		// Generate actions
		result.Actions = p.generateScenarioActions(result)

		scenarios[scenarioConfig.Name] = result
	}

	return scenarios, nil
}

// applyGrowthRate applies a growth rate to a prediction
func (p *CapacityPlanner) applyGrowthRate(prediction *Prediction, growthRate float64) *Prediction {
	adjusted := *prediction // Copy
	adjusted.Values = make([]PredictionPoint, len(prediction.Values))

	for i, point := range prediction.Values {
		multiplier := 1.0 + growthRate
		adjusted.Values[i] = PredictionPoint{
			Timestamp:  point.Timestamp,
			Value:      point.Value * multiplier,
			LowerBound: point.LowerBound * multiplier,
			UpperBound: point.UpperBound * multiplier,
		}
	}

	return &adjusted
}

// calculateCapacityGap calculates the capacity gap
func (p *CapacityPlanner) calculateCapacityGap(predictions map[string]*Prediction) float64 {
	// Simplified calculation - would be more sophisticated in practice
	maxGap := 0.0
	for _, prediction := range predictions {
		if len(prediction.Values) > 0 {
			lastValue := prediction.Values[len(prediction.Values)-1].Value
			// Assume 100% is the capacity limit
			gap := lastValue - 100.0
			if gap > maxGap {
				maxGap = gap
			}
		}
	}
	return maxGap
}

// calculateTimeToLimit calculates time until capacity limit is reached
func (p *CapacityPlanner) calculateTimeToLimit(predictions map[string]*Prediction) time.Duration {
	// Simplified calculation
	minTime := time.Duration(math.MaxInt64)
	for _, prediction := range predictions {
		for _, point := range prediction.Values {
			if point.Value >= 100.0 { // Assume 100% is the limit
				timeToLimit := point.Timestamp.Sub(time.Now())
				if timeToLimit < minTime {
					minTime = timeToLimit
				}
				break
			}
		}
	}

	if minTime == time.Duration(math.MaxInt64) {
		return 0 // No limit reached in prediction horizon
	}
	return minTime
}

// assessScenarioRisk assesses the risk level of a scenario
func (p *CapacityPlanner) assessScenarioRisk(scenario *ScenarioResult) string {
	if scenario.CapacityGap > 50 {
		return "high"
	} else if scenario.CapacityGap > 20 {
		return "medium"
	}
	return "low"
}

// generateScenarioActions generates recommended actions for a scenario
func (p *CapacityPlanner) generateScenarioActions(scenario *ScenarioResult) []string {
	actions := make([]string, 0)

	if scenario.Risk == "high" {
		actions = append(actions, "Immediate capacity expansion required")
		actions = append(actions, "Consider emergency scaling procedures")
	} else if scenario.Risk == "medium" {
		actions = append(actions, "Plan capacity expansion within next quarter")
		actions = append(actions, "Monitor growth trends closely")
	} else {
		actions = append(actions, "Continue monitoring")
		actions = append(actions, "Review capacity planning quarterly")
	}

	return actions
}

// generateRecommendations generates capacity recommendations
func (p *CapacityPlanner) generateRecommendations(predictions map[string]*Prediction, scenarios map[string]*ScenarioResult) []Recommendation {
	recommendations := make([]Recommendation, 0)

	// Analyze each metric
	for _, metricConfig := range p.config.Metrics {
		// Find predictions for this metric
		for _, prediction := range predictions {
			if prediction.Metric == metricConfig.Name {
				rec := p.generateMetricRecommendation(metricConfig, prediction)
				if rec != nil {
					recommendations = append(recommendations, *rec)
				}
				break
			}
		}
	}

	return recommendations
}

// generateMetricRecommendation generates a recommendation for a specific metric
func (p *CapacityPlanner) generateMetricRecommendation(metricConfig MetricConfig, prediction *Prediction) *Recommendation {
	if len(prediction.Values) == 0 {
		return nil
	}

	// Get current and predicted values
	currentValue := prediction.Values[0].Value
	futureValue := prediction.Values[len(prediction.Values)-1].Value

	// Check if threshold will be exceeded
	if futureValue > metricConfig.Threshold {
		priority := "medium"
		if metricConfig.Critical {
			priority = "high"
		}

		// Calculate recommended capacity
		recommended := futureValue * 1.2 // 20% buffer

		return &Recommendation{
			Type:        "scale_up",
			Priority:    priority,
			Resource:    metricConfig.Name,
			Current:     currentValue,
			Recommended: recommended,
			Reason:      fmt.Sprintf("Predicted to exceed threshold (%.2f) in %v", metricConfig.Threshold, prediction.Horizon),
			Timeframe:   prediction.Horizon,
			Metadata:    make(map[string]interface{}),
		}
	}

	return nil
}

// assessRisks assesses capacity planning risks
func (p *CapacityPlanner) assessRisks(predictions map[string]*Prediction, scenarios map[string]*ScenarioResult) []Risk {
	risks := make([]Risk, 0)

	// Check for capacity exhaustion risk
	for _, scenario := range scenarios {
		if scenario.Risk == "high" {
			risk := Risk{
				Type:        "capacity_exhaustion",
				Severity:    "high",
				Probability: 0.8,
				Impact:      "Service degradation or outage",
				Timeframe:   scenario.TimeToLimit,
				Mitigation:  []string{"Scale up resources", "Optimize performance", "Implement load balancing"},
			}
			risks = append(risks, risk)
		}
	}

	// Check for performance degradation risk
	for _, prediction := range predictions {
		if prediction.Trend == "increasing" && len(prediction.Values) > 0 {
			lastValue := prediction.Values[len(prediction.Values)-1].Value
			if lastValue > 80 { // Assume 80% is warning threshold
				risk := Risk{
					Type:        "performance_degradation",
					Severity:    "medium",
					Probability: 0.6,
					Impact:      "Increased latency and reduced throughput",
					Timeframe:   prediction.Horizon / 2,
					Mitigation:  []string{"Performance tuning", "Resource optimization", "Caching improvements"},
				}
				risks = append(risks, risk)
			}
		}
	}

	return risks
}

// generateSummary generates a capacity planning summary
func (p *CapacityPlanner) generateSummary(predictions map[string]*Prediction, recommendations []Recommendation, risks []Risk) *CapacitySummary {
	// Calculate overall health
	overallHealth := "healthy"
	highRiskCount := 0
	for _, risk := range risks {
		if risk.Severity == "high" || risk.Severity == "critical" {
			highRiskCount++
		}
	}

	if highRiskCount > 0 {
		overallHealth = "critical"
	} else if len(risks) > 0 {
		overallHealth = "warning"
	}

	// Calculate average capacity utilization
	totalUtilization := 0.0
	count := 0
	for _, prediction := range predictions {
		if len(prediction.Values) > 0 {
			totalUtilization += prediction.Values[0].Value
			count++
		}
	}

	avgUtilization := 0.0
	if count > 0 {
		avgUtilization = totalUtilization / float64(count)
	}

	// Find high-risk metrics
	highRiskMetrics := make([]string, 0)
	for _, prediction := range predictions {
		if len(prediction.Values) > 0 {
			lastValue := prediction.Values[len(prediction.Values)-1].Value
			if lastValue > 90 { // High utilization threshold
				highRiskMetrics = append(highRiskMetrics, prediction.Metric)
			}
		}
	}

	return &CapacitySummary{
		OverallHealth:       overallHealth,
		CapacityUtilization: avgUtilization,
		TimeToCapacity:      p.calculateMinTimeToCapacity(predictions),
		GrowthRate:          p.calculateAverageGrowthRate(predictions),
		HighRiskMetrics:     highRiskMetrics,
		RecommendationCount: len(recommendations),
	}
}

// calculateMinTimeToCapacity calculates the minimum time to capacity limit
func (p *CapacityPlanner) calculateMinTimeToCapacity(predictions map[string]*Prediction) time.Duration {
	minTime := time.Duration(math.MaxInt64)
	for _, prediction := range predictions {
		for _, point := range prediction.Values {
			if point.Value >= 100.0 {
				timeToCapacity := point.Timestamp.Sub(time.Now())
				if timeToCapacity < minTime {
					minTime = timeToCapacity
				}
				break
			}
		}
	}

	if minTime == time.Duration(math.MaxInt64) {
		return 0
	}
	return minTime
}

// calculateAverageGrowthRate calculates the average growth rate
func (p *CapacityPlanner) calculateAverageGrowthRate(predictions map[string]*Prediction) float64 {
	totalGrowthRate := 0.0
	count := 0

	for _, prediction := range predictions {
		if len(prediction.Values) >= 2 {
			firstValue := prediction.Values[0].Value
			lastValue := prediction.Values[len(prediction.Values)-1].Value
			if firstValue > 0 {
				growthRate := (lastValue - firstValue) / firstValue
				totalGrowthRate += growthRate
				count++
			}
		}
	}

	if count == 0 {
		return 0
	}
	return totalGrowthRate / float64(count)
}

// SaveReport saves a capacity planning report
func (p *CapacityPlanner) SaveReport(report *CapacityReport, filename string) error {
	if p.outputDir != "" {
		if err := os.MkdirAll(p.outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		filename = filepath.Join(p.outputDir, filename)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	if p.verbose {
		fmt.Printf("Capacity report saved to: %s\n", filename)
	}

	return nil
}

// Implementation of LinearModel

// Name returns the model name
func (m *LinearModel) Name() string {
	return m.name
}

// Train trains the linear model
func (m *LinearModel) Train(data []DataPoint) error {
	if len(data) < 2 {
		return fmt.Errorf("insufficient data for training")
	}

	// Extract metric values and timestamps
	var x, y []float64
	for _, point := range data {
		if value, exists := point.Metrics[m.metric]; exists {
			x = append(x, float64(point.Timestamp.Unix()))
			y = append(y, value)
		}
	}

	if len(x) < 2 {
		return fmt.Errorf("insufficient data points for metric %s", m.metric)
	}

	// Simple linear regression
	n := float64(len(x))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumXX += x[i] * x[i]
	}

	// Calculate slope and intercept
	m.slope = (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	m.intercept = (sumY - m.slope*sumX) / n

	// Calculate R-squared for accuracy
	meanY := sumY / n
	ssRes := 0.0
	ssTot := 0.0

	for i := 0; i < len(x); i++ {
		predicted := m.slope*x[i] + m.intercept
		ssRes += math.Pow(y[i]-predicted, 2)
		ssTot += math.Pow(y[i]-meanY, 2)
	}

	if ssTot > 0 {
		m.accuracy = 1 - (ssRes / ssTot)
	} else {
		m.accuracy = 0
	}

	m.trained = true
	return nil
}

// Predict generates predictions using the linear model
func (m *LinearModel) Predict(horizon time.Duration) (*Prediction, error) {
	if !m.trained {
		return nil, fmt.Errorf("model not trained")
	}

	now := time.Now()
	prediction := &Prediction{
		Model:      m.name,
		Metric:     m.metric,
		Timestamp:  now,
		Horizon:    horizon,
		Confidence: m.accuracy,
		Accuracy:   m.accuracy,
		Values:     make([]PredictionPoint, 0),
		Metadata:   make(map[string]interface{}),
	}

	// Generate prediction points (hourly)
	steps := int(horizon.Hours())
	if steps > 168 { // Limit to 1 week
		steps = 168
	}

	for i := 0; i <= steps; i++ {
		timestamp := now.Add(time.Duration(i) * time.Hour)
		x := float64(timestamp.Unix())
		value := m.slope*x + m.intercept

		// Add some uncertainty bounds (simplified)
		errorMargin := math.Abs(value) * 0.1 // 10% error margin
		lowerBound := value - errorMargin
		upperBound := value + errorMargin

		point := PredictionPoint{
			Timestamp:  timestamp,
			Value:      value,
			LowerBound: lowerBound,
			UpperBound: upperBound,
		}

		prediction.Values = append(prediction.Values, point)
	}

	// Determine trend
	if m.slope > 0.01 {
		prediction.Trend = "increasing"
	} else if m.slope < -0.01 {
		prediction.Trend = "decreasing"
	} else {
		prediction.Trend = "stable"
	}

	return prediction, nil
}

// Accuracy returns the model accuracy
func (m *LinearModel) Accuracy() float64 {
	return m.accuracy
}
// chaos/framework.go
package chaos

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

type ChaosExperiment struct {
	Name          string
	Description   string
	Hypothesis    string
	Duration      time.Duration
	SteadyState   SteadyStateCheck
	Perturbations []Perturbation
	Rollback      RollbackAction
	SafetyChecks  []SafetyCheck
	Metrics       []MetricCollector
}

type SteadyStateCheck struct {
	Name      string
	Check     func(context.Context) (bool, error)
	Tolerance float64
	Interval  time.Duration
}

type Perturbation struct {
	Name        string
	Type        string // "network", "cpu", "memory", "disk", "service"
	Action      func(context.Context) error
	Cleanup     func(context.Context) error
	Duration    time.Duration
	Probability float64
	Scope       PerturbationScope
}

type PerturbationScope struct {
	Targets    []string // service names, hostnames, etc.
	Percentage float64  // percentage of targets to affect
	Selection  string   // "random", "all", "specific"
}

type RollbackAction struct {
	Name   string
	Action func(context.Context) error
}

type SafetyCheck struct {
	Name     string
	Check    func(context.Context) (bool, string, error)
	Critical bool
	Interval time.Duration
}

type MetricCollector struct {
	Name     string
	Collect  func(context.Context) (map[string]float64, error)
	Interval time.Duration
}

type ExperimentResult struct {
	ExperimentName  string
	StartTime       time.Time
	EndTime         time.Time
	Status          string // "success", "failed", "aborted"
	Hypothesis      string
	HypothesisValid bool
	SteadyState     SteadyStateResult
	Perturbations   []PerturbationResult
	Metrics         map[string][]MetricPoint
	Insights        []string
	Recommendations []string
}

type SteadyStateResult struct {
	InitialCheck bool
	FinalCheck   bool
	Maintained   bool
	Violations   []SteadyStateViolation
}

type SteadyStateViolation struct {
	Timestamp time.Time
	Message   string
	Severity  string
}

type PerturbationResult struct {
	Name      string
	Applied   bool
	StartTime time.Time
	EndTime   time.Time
	Error     string
	Impact    PerturbationImpact
}

type PerturbationImpact struct {
	RecoveryTime       time.Duration
	ServiceDegradation float64
	ErrorRateIncrease  float64
	LatencyIncrease    float64
}

type MetricPoint struct {
	Timestamp time.Time
	Value     float64
}

type ChaosEngine struct {
	config      ChaosConfig
	experiments []ChaosExperiment
	results     []ExperimentResult
	mu          sync.RWMutex
	running     bool
}

type ChaosConfig struct {
	Environment     string
	SafetyMode      bool
	MaxConcurrent   int
	DefaultTimeout  time.Duration
	MetricsInterval time.Duration
	ReportPath      string
}

func NewChaosEngine(config ChaosConfig) *ChaosEngine {
	return &ChaosEngine{
		config:      config,
		experiments: make([]ChaosExperiment, 0),
		results:     make([]ExperimentResult, 0),
	}
}

func (ce *ChaosEngine) AddExperiment(experiment ChaosExperiment) {
	ce.experiments = append(ce.experiments, experiment)
}

func (ce *ChaosEngine) RunExperiment(ctx context.Context, experimentName string) (*ExperimentResult, error) {
	// Find experiment
	var experiment *ChaosExperiment
	for _, exp := range ce.experiments {
		if exp.Name == experimentName {
			experiment = &exp
			break
		}
	}

	if experiment == nil {
		return nil, fmt.Errorf("experiment not found: %s", experimentName)
	}

	result := ExperimentResult{
		ExperimentName:  experiment.Name,
		StartTime:       time.Now(),
		Hypothesis:      experiment.Hypothesis,
		Metrics:         make(map[string][]MetricPoint),
		Insights:        make([]string, 0),
		Recommendations: make([]string, 0),
	}

	defer func() {
		result.EndTime = time.Now()
		ce.mu.Lock()
		ce.results = append(ce.results, result)
		ce.mu.Unlock()
	}()

	// Create experiment context with timeout
	expCtx, cancel := context.WithTimeout(ctx, experiment.Duration)
	defer cancel()

	// Start metric collection
	metricCtx, metricCancel := context.WithCancel(expCtx)
	defer metricCancel()

	var wg sync.WaitGroup
	for _, collector := range experiment.Metrics {
		wg.Add(1)
		go func(c MetricCollector) {
			defer wg.Done()
			ce.collectMetrics(metricCtx, c, &result)
		}(collector)
	}

	// Check initial steady state
	initialSteady, err := experiment.SteadyState.Check(expCtx)
	if err != nil {
		result.Status = "failed"
		return &result, fmt.Errorf("initial steady state check failed: %w", err)
	}

	result.SteadyState.InitialCheck = initialSteady
	if !initialSteady {
		result.Status = "aborted"
		result.Insights = append(result.Insights, "System was not in steady state before experiment")
		return &result, nil
	}

	// Start safety monitoring
	safetyCtx, safetyCancel := context.WithCancel(expCtx)
	defer safetyCancel()

	safetyViolation := make(chan string, 1)
	go ce.monitorSafety(safetyCtx, experiment.SafetyChecks, safetyViolation)

	// Apply perturbations
	perturbationResults := make([]PerturbationResult, 0)

	for _, perturbation := range experiment.Perturbations {
		// Check if we should apply this perturbation (probability)
		if rand.Float64() > perturbation.Probability {
			continue
		}

		pertResult := ce.runPerturbation(expCtx, perturbation)
		perturbationResults = append(perturbationResults, pertResult)

		// Check for safety violations
		select {
		case violation := <-safetyViolation:
			result.Status = "aborted"
			result.Insights = append(result.Insights, fmt.Sprintf("Safety violation: %s", violation))
			ce.rollbackPerturbations(expCtx, experiment.Perturbations)
			return &result, nil
		default:
		}
	}

	result.Perturbations = perturbationResults

	// Wait for experiment duration
	select {
	case <-expCtx.Done():
	case violation := <-safetyViolation:
		result.Status = "aborted"
		result.Insights = append(result.Insights, fmt.Sprintf("Safety violation during experiment: %s", violation))
		ce.rollbackPerturbations(expCtx, experiment.Perturbations)
		return &result, nil
	}

	// Rollback perturbations
	if err := experiment.Rollback.Action(expCtx); err != nil {
		result.Insights = append(result.Insights, fmt.Sprintf("Rollback warning: %v", err))
	}

	// Check final steady state
	finalSteady, err := experiment.SteadyState.Check(expCtx)
	if err != nil {
		result.Status = "failed"
		return &result, fmt.Errorf("final steady state check failed: %w", err)
	}

	result.SteadyState.FinalCheck = finalSteady
	result.SteadyState.Maintained = initialSteady && finalSteady

	// Validate hypothesis
	result.HypothesisValid = result.SteadyState.Maintained

	// Generate insights
	ce.generateInsights(&result)

	result.Status = "success"

	// Stop metric collection
	metricCancel()
	wg.Wait()

	return &result, nil
}

func (ce *ChaosEngine) runPerturbation(ctx context.Context, perturbation Perturbation) PerturbationResult {
	result := PerturbationResult{
		Name:      perturbation.Name,
		StartTime: time.Now(),
	}

	defer func() {
		result.EndTime = time.Now()
	}()

	// Apply perturbation
	if err := perturbation.Action(ctx); err != nil {
		result.Error = err.Error()
		return result
	}

	result.Applied = true

	// Wait for perturbation duration
	select {
	case <-ctx.Done():
	case <-time.After(perturbation.Duration):
	}

	// Cleanup perturbation
	if perturbation.Cleanup != nil {
		if err := perturbation.Cleanup(ctx); err != nil {
			result.Error = fmt.Sprintf("cleanup failed: %v", err)
		}
	}

	return result
}

func (ce *ChaosEngine) collectMetrics(ctx context.Context, collector MetricCollector, result *ExperimentResult) {
	ticker := time.NewTicker(collector.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics, err := collector.Collect(ctx)
			if err != nil {
				continue
			}

			ce.mu.Lock()
			for name, value := range metrics {
				if result.Metrics[name] == nil {
					result.Metrics[name] = make([]MetricPoint, 0)
				}
				result.Metrics[name] = append(result.Metrics[name], MetricPoint{
					Timestamp: time.Now(),
					Value:     value,
				})
			}
			ce.mu.Unlock()
		}
	}
}

func (ce *ChaosEngine) monitorSafety(ctx context.Context, checks []SafetyCheck, violation chan<- string) {
	for _, check := range checks {
		go func(c SafetyCheck) {
			ticker := time.NewTicker(c.Interval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					safe, message, err := c.Check(ctx)
					if err != nil || !safe {
						select {
						case violation <- fmt.Sprintf("%s: %s", c.Name, message):
						default:
						}
						if c.Critical {
							return
						}
					}
				}
			}
		}(check)
	}
}

func (ce *ChaosEngine) rollbackPerturbations(ctx context.Context, perturbations []Perturbation) {
	for _, perturbation := range perturbations {
		if perturbation.Cleanup != nil {
			perturbation.Cleanup(ctx)
		}
	}
}

func (ce *ChaosEngine) generateInsights(result *ExperimentResult) {
	// Analyze metrics for insights
	for metricName, points := range result.Metrics {
		if len(points) < 2 {
			continue
		}

		// Calculate trend
		first := points[0].Value
		last := points[len(points)-1].Value
		change := ((last - first) / first) * 100

		if math.Abs(change) > 10 {
			if change > 0 {
				result.Insights = append(result.Insights,
					fmt.Sprintf("%s increased by %.2f%% during experiment", metricName, change))
			} else {
				result.Insights = append(result.Insights,
					fmt.Sprintf("%s decreased by %.2f%% during experiment", metricName, math.Abs(change)))
			}
		}
	}

	// Generate recommendations
	if !result.SteadyState.Maintained {
		result.Recommendations = append(result.Recommendations,
			"System did not maintain steady state - consider improving resilience")
	}

	if len(result.Perturbations) == 0 {
		result.Recommendations = append(result.Recommendations,
			"No perturbations were applied - consider adjusting probability settings")
	}
}

func (ce *ChaosEngine) GetResults() []ExperimentResult {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return append([]ExperimentResult(nil), ce.results...)
}

func (ce *ChaosEngine) GetExperiments() []ChaosExperiment {
	return append([]ChaosExperiment(nil), ce.experiments...)
}
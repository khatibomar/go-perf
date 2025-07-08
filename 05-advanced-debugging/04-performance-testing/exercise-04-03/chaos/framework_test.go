// chaos/framework_test.go
package chaos

import (
	"context"
	"testing"
	"time"
)

func TestNewChaosEngine(t *testing.T) {
	config := ChaosConfig{
		Environment:     "test",
		SafetyMode:      true,
		MaxConcurrent:   5,
		DefaultTimeout:  10 * time.Minute,
		MetricsInterval: 1 * time.Second,
		ReportPath:      "./reports",
	}
	engine := NewChaosEngine(config)
	if engine == nil {
		t.Fatal("NewChaosEngine returned nil")
	}

	if len(engine.experiments) != 0 {
		t.Errorf("Expected 0 experiments, got %d", len(engine.experiments))
	}
}

func TestChaosEngine_AddExperiment(t *testing.T) {
	config := ChaosConfig{
		Environment:     "test",
		SafetyMode:      true,
		MaxConcurrent:   5,
		DefaultTimeout:  10 * time.Minute,
		MetricsInterval: 1 * time.Second,
		ReportPath:      "./reports",
	}
	engine := NewChaosEngine(config)
	
	experiment := ChaosExperiment{
		Name:        "Test Experiment",
		Description: "Test experiment description",
		Hypothesis:  "Test hypothesis",
		Duration:    1 * time.Minute,
	}
	
	engine.AddExperiment(experiment)
	
	if len(engine.experiments) != 1 {
		t.Errorf("Expected 1 experiment, got %d", len(engine.experiments))
	}
	
	if engine.experiments[0].Name != "Test Experiment" {
		t.Errorf("Expected experiment name 'Test Experiment', got '%s'", 
			engine.experiments[0].Name)
	}
}

func TestChaosEngine_RunExperiment(t *testing.T) {
	config := ChaosConfig{
		Environment:     "test",
		SafetyMode:      true,
		MaxConcurrent:   5,
		DefaultTimeout:  10 * time.Minute,
		MetricsInterval: 1 * time.Second,
		ReportPath:      "./reports",
	}
	engine := NewChaosEngine(config)
	ctx := context.Background()
	
	// Create a simple test experiment
	experiment := ChaosExperiment{
		Name:        "Simple Test",
		Description: "Simple test experiment",
		Hypothesis:  "System should remain stable",
		Duration:    100 * time.Millisecond,
		SteadyState: SteadyStateCheck{
			Name: "Response Time Check",
			Check: func(ctx context.Context) (bool, error) {
				return true, nil
			},
			Interval:  10 * time.Millisecond,
			Tolerance: 0.1,
		},
		Perturbations: []Perturbation{
			{
				Name: "No-op Perturbation",
				Type: "test",
				Action: func(ctx context.Context) error {
					return nil
				},
				Cleanup: func(ctx context.Context) error {
					return nil
				},
				Duration:    50 * time.Millisecond,
				Probability: 1.0,
			},
		},
		Rollback: RollbackAction{
			Name: "No-op Rollback",
			Action: func(ctx context.Context) error {
				return nil
			},
		},
		SafetyChecks: []SafetyCheck{
			{
				Name: "Always Safe",
				Check: func(ctx context.Context) (bool, string, error) {
					return true, "system is safe", nil
				},
				Interval: 25 * time.Millisecond,
				Critical: false,
			},
		},
		Metrics: []MetricCollector{
			{
				Name: "Test Metric",
				Collect: func(ctx context.Context) (map[string]float64, error) {
					return map[string]float64{"test_metric": 42.0}, nil
				},
				Interval: 25 * time.Millisecond,
			},
		},
	}
	
	engine.AddExperiment(experiment)
	
	result, err := engine.RunExperiment(ctx, "Simple Test")
	if err != nil {
		t.Fatalf("RunExperiment failed: %v", err)
	}
	
	if result == nil {
		t.Fatal("RunExperiment returned nil result")
	}
	
	if result.ExperimentName != "Simple Test" {
		t.Errorf("Expected experiment name 'Simple Test', got '%s'", result.ExperimentName)
	}
	
	if result.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", result.Status)
	}
	
	duration := result.EndTime.Sub(result.StartTime)
	if duration <= 0 {
		t.Errorf("Expected positive duration, got %v", duration)
	}
}

func TestChaosEngine_RunExperiment_NonExistent(t *testing.T) {
	config := ChaosConfig{
		Environment:     "test",
		SafetyMode:      true,
		MaxConcurrent:   5,
		DefaultTimeout:  10 * time.Minute,
		MetricsInterval: 1 * time.Second,
		ReportPath:      "./reports",
	}
	engine := NewChaosEngine(config)
	ctx := context.Background()
	
	_, err := engine.RunExperiment(ctx, "NonExistent")
	if err == nil {
		t.Error("Expected error for non-existent experiment")
	}
}

func TestChaosEngine_RunExperiment_WithFailingSteadyState(t *testing.T) {
	config := ChaosConfig{
		Environment:     "test",
		SafetyMode:      true,
		MaxConcurrent:   5,
		DefaultTimeout:  10 * time.Minute,
		MetricsInterval: 1 * time.Second,
		ReportPath:      "./reports",
	}
	engine := NewChaosEngine(config)
	ctx := context.Background()
	
	experiment := ChaosExperiment{
		Name:        "Failing Steady State",
		Description: "Test with failing steady state",
		Hypothesis:  "This should fail",
		Duration:    100 * time.Millisecond,
		SteadyState: SteadyStateCheck{
			Name: "Always Fail",
			Check: func(ctx context.Context) (bool, error) {
				return false, nil
			},
			Interval:  10 * time.Millisecond,
			Tolerance: 0.1,
		},
		Perturbations: []Perturbation{},
		Rollback: RollbackAction{
			Name: "No-op Rollback",
			Action: func(ctx context.Context) error {
				return nil
			},
		},
	}
	
	engine.AddExperiment(experiment)
	
	result, err := engine.RunExperiment(ctx, "Failing Steady State")
	if err != nil {
		t.Fatalf("RunExperiment failed: %v", err)
	}
	
	if result.Status != "aborted" {
		t.Errorf("Expected status 'aborted', got '%s'", result.Status)
	}
	
	if result.SteadyState.InitialCheck {
		t.Error("Expected initial steady state to be false")
	}
}

func TestChaosEngine_CollectMetrics(t *testing.T) {
	config := ChaosConfig{
		Environment:     "test",
		SafetyMode:      true,
		MaxConcurrent:   5,
		DefaultTimeout:  10 * time.Minute,
		MetricsInterval: 1 * time.Second,
		ReportPath:      "./reports",
	}
	engine := NewChaosEngine(config)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	metrics := []MetricCollector{
		{
			Name: "Test Metric",
			Collect: func(ctx context.Context) (map[string]float64, error) {
				return map[string]float64{"test_metric": 123.45}, nil
			},
			Interval: 25 * time.Millisecond,
		},
	}
	
	result := &ExperimentResult{
		Metrics: make(map[string][]MetricPoint),
	}
	for _, metric := range metrics {
		engine.collectMetrics(ctx, metric, result)
	}
	
	if len(result.Metrics) == 0 {
		t.Error("Expected at least one metric point")
	}
	
	// Check that we collected multiple points
	if metricPoints, exists := result.Metrics["test_metric"]; !exists {
		t.Error("Expected test_metric to be present")
	} else if len(metricPoints) < 2 {
		t.Errorf("Expected at least 2 metric points, got %d", len(metricPoints))
	} else {
		// Verify metric content
		for _, metric := range metricPoints {
			if metric.Value != 123.45 {
				t.Errorf("Expected metric value 123.45, got %f", metric.Value)
			}
		}
	}
}

func TestChaosEngine_MonitorSafety(t *testing.T) {
	config := ChaosConfig{
		Environment:     "test",
		SafetyMode:      true,
		MaxConcurrent:   5,
		DefaultTimeout:  10 * time.Minute,
		MetricsInterval: 1 * time.Second,
		ReportPath:      "./reports",
	}
	engine := NewChaosEngine(config)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	safetyChecks := []SafetyCheck{
		{
			Name: "Safe Check",
			Check: func(ctx context.Context) (bool, string, error) {
				return true, "system is safe", nil
			},
			Interval: 25 * time.Millisecond,
			Critical: false,
		},
	}
	
	// This should not trigger emergency stop
	violation := make(chan string, 1)
	go engine.monitorSafety(ctx, safetyChecks, violation)

	// Wait for context to finish
	<-ctx.Done()

	// Check if any violations were reported
	select {
	case msg := <-violation:
		t.Errorf("Unexpected safety violation: %s", msg)
	default:
		// No violations, which is expected for this test
	}
}

func TestChaosEngine_MonitorSafety_EmergencyStop(t *testing.T) {
	config := ChaosConfig{
		Environment:     "test",
		SafetyMode:      true,
		MaxConcurrent:   5,
		DefaultTimeout:  10 * time.Minute,
		MetricsInterval: 1 * time.Second,
		ReportPath:      "./reports",
	}
	engine := NewChaosEngine(config)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	
	callCount := 0
	safetyChecks := []SafetyCheck{
		{
			Name: "Unsafe Check",
			Check: func(ctx context.Context) (bool, string, error) {
				callCount++
				if callCount <= 2 {
					return true, "system is safe", nil
				}
				return false, "system is unsafe", nil
			},
			Interval: 25 * time.Millisecond,
			Critical: true,
		},
	}
	
	violation := make(chan string, 1)
	go engine.monitorSafety(ctx, safetyChecks, violation)

	// Wait for a violation to be reported
	select {
	case msg := <-violation:
		if msg == "" {
			t.Error("Expected non-empty violation message")
		}
	case <-time.After(300 * time.Millisecond):
		t.Error("Expected safety violation to be reported")
	}
}

func TestChaosEngine_GenerateInsights(t *testing.T) {
	config := ChaosConfig{
		Environment:     "test",
		SafetyMode:      true,
		MaxConcurrent:   5,
		DefaultTimeout:  10 * time.Minute,
		MetricsInterval: 1 * time.Second,
		ReportPath:      "./reports",
	}
	engine := NewChaosEngine(config)
	
	result := &ExperimentResult{
		ExperimentName: "Test Experiment",
		Status:         "completed",
		StartTime:      time.Now().Add(-1 * time.Minute),
		EndTime:        time.Now(),
		SteadyState: SteadyStateResult{
			InitialCheck: true,
			FinalCheck:   true,
			Violations:   []SteadyStateViolation{},
		},
		Perturbations: []PerturbationResult{
			{
				Name:      "Test Perturbation",
				Applied:   true,
				StartTime: time.Now().Add(-30 * time.Second),
				EndTime:   time.Now(),
			},
		},
		Metrics: map[string][]MetricPoint{
			"CPU": {
				{Value: 50.0, Timestamp: time.Now().Add(-1 * time.Minute)},
				{Value: 70.0, Timestamp: time.Now()},
			},
			"Memory": {
				{Value: 1024.0, Timestamp: time.Now().Add(-1 * time.Minute)},
				{Value: 1200.0, Timestamp: time.Now()},
			},
		},
	}
	
	engine.generateInsights(result)

	if len(result.Insights) == 0 {
		t.Error("Expected at least one insight")
	}

	// Check that insights were generated
	foundInsight := false
	for _, insight := range result.Insights {
		if insight != "" {
			foundInsight = true
			break
		}
	}

	if !foundInsight {
		t.Error("Expected non-empty insights")
	}
}

func TestChaosEngine_RollbackPerturbations(t *testing.T) {
	config := ChaosConfig{
		Environment:     "test",
		SafetyMode:      true,
		MaxConcurrent:   5,
		DefaultTimeout:  10 * time.Minute,
		MetricsInterval: 1 * time.Second,
		ReportPath:      "./reports",
	}
	engine := NewChaosEngine(config)
	ctx := context.Background()
	
	cleanupCalled := false
	perturbations := []Perturbation{
		{
			Name: "Test Perturbation",
			Type: "test",
			Action: func(ctx context.Context) error {
				return nil
			},
			Cleanup: func(ctx context.Context) error {
				cleanupCalled = true
				return nil
			},
			Duration:    10 * time.Millisecond,
			Probability: 1.0,
		},
	}
	
	engine.rollbackPerturbations(ctx, perturbations)
	
	if !cleanupCalled {
		t.Error("Cleanup action was not called")
	}
}

func TestSteadyStateCheck_Execution(t *testing.T) {
	ctx := context.Background()
	
	check := SteadyStateCheck{
		Name: "Test Check",
		Check: func(ctx context.Context) (bool, error) {
			return true, nil
		},
		Interval:  10 * time.Millisecond,
		Tolerance: 0.1,
	}
	
	result, err := check.Check(ctx)
	if err != nil {
		t.Errorf("Check failed: %v", err)
	}

	if !result {
		t.Error("Expected check to return true")
	}
}

func TestPerturbation_Execution(t *testing.T) {
	ctx := context.Background()
	
	actionCalled := false
	cleanupCalled := false
	
	perturbation := Perturbation{
		Name: "Test Perturbation",
		Type: "test",
		Action: func(ctx context.Context) error {
			actionCalled = true
			return nil
		},
		Cleanup: func(ctx context.Context) error {
			cleanupCalled = true
			return nil
		},
		Duration:    10 * time.Millisecond,
		Probability: 1.0,
	}
	
	// Test action
	err := perturbation.Action(ctx)
	if err != nil {
		t.Errorf("Action failed: %v", err)
	}
	if !actionCalled {
		t.Error("Action was not called")
	}
	
	// Test cleanup
	err = perturbation.Cleanup(ctx)
	if err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}
	if !cleanupCalled {
		t.Error("Cleanup was not called")
	}
}
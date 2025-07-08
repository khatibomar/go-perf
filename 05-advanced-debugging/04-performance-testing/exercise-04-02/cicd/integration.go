// cicd/integration.go
package cicd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"exercise-04-02/pipeline"
)

type CICDIntegration struct {
	config     CICDConfig
	pipeline   *pipeline.PerformancePipeline
	environment map[string]string
}

type CICDConfig struct {
	Provider        string // "github", "gitlab", "jenkins", "azure"
	Workspace       string
	Branch          string
	CommitSHA       string
	PullRequestID   string
	BuildNumber     string
	ArtifactsPath   string
	TestTimeout     time.Duration
	FailOnRegression bool
	BaselineStrategy string // "branch", "tag", "manual"
	Triggers        []TriggerConfig
}

type TriggerConfig struct {
	Event       string   // "push", "pull_request", "schedule", "manual"
	Branches    []string
	Paths       []string
	Schedule    string   // cron expression
	TestSuite   string
}

type BuildContext struct {
	Provider      string
	Workspace     string
	Branch        string
	CommitSHA     string
	CommitMessage string
	Author        string
	PullRequest   *PullRequestInfo
	BuildNumber   string
	BuildURL      string
	ArtifactsURL  string
}

type PullRequestInfo struct {
	ID          string
	Title       string
	SourceBranch string
	TargetBranch string
	Author      string
}

type TestExecutionResult struct {
	BuildContext    BuildContext
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	Status          string // "success", "failure", "cancelled"
	TestResults     []pipeline.TestResult
	Regressions     []pipeline.RegressionAnalysis
	Artifacts       []string
	Reports         []string
	ExitCode        int
	ErrorMessage    string
}

func NewCICDIntegration(config CICDConfig) *CICDIntegration {
	pipelineConfig := pipeline.PipelineConfig{
		Environment:   config.Provider,
		TestTimeout:   config.TestTimeout,
		ParallelTests: 1,
		ResultsPath:   filepath.Join(config.ArtifactsPath, "results"),
		BaselinePath:  filepath.Join(config.ArtifactsPath, "baseline"),
		ReportFormats: []string{"html", "json"},
		Notifications: pipeline.NotificationConfig{
			OnFailure:    true,
			OnRegression: true,
			OnSuccess:    false,
		},
	}
	
	return &CICDIntegration{
		config:      config,
		pipeline:    pipeline.NewPerformancePipeline(pipelineConfig),
		environment: make(map[string]string),
	}
}

func (ci *CICDIntegration) DetectEnvironment() BuildContext {
	ctx := BuildContext{
		Provider:    ci.config.Provider,
		Workspace:   ci.config.Workspace,
		Branch:      ci.getBranchName(),
		CommitSHA:   ci.getCommitSHA(),
		BuildNumber: ci.config.BuildNumber,
	}
	
	// Detect CI/CD provider specific information
	switch ci.config.Provider {
	case "github":
		ci.detectGitHubEnvironment(&ctx)
	case "gitlab":
		ci.detectGitLabEnvironment(&ctx)
	case "jenkins":
		ci.detectJenkinsEnvironment(&ctx)
	case "azure":
		ci.detectAzureEnvironment(&ctx)
	}
	
	return ctx
}

func (ci *CICDIntegration) detectGitHubEnvironment(ctx *BuildContext) {
	ctx.CommitMessage = os.Getenv("GITHUB_EVENT_HEAD_COMMIT_MESSAGE")
	ctx.Author = os.Getenv("GITHUB_ACTOR")
	ctx.BuildURL = fmt.Sprintf("https://github.com/%s/actions/runs/%s", os.Getenv("GITHUB_REPOSITORY"), os.Getenv("GITHUB_RUN_ID"))
	
	if prNumber := os.Getenv("GITHUB_EVENT_PULL_REQUEST_NUMBER"); prNumber != "" {
		ctx.PullRequest = &PullRequestInfo{
			ID:           prNumber,
			Title:        os.Getenv("GITHUB_EVENT_PULL_REQUEST_TITLE"),
			SourceBranch: os.Getenv("GITHUB_HEAD_REF"),
			TargetBranch: os.Getenv("GITHUB_BASE_REF"),
			Author:       os.Getenv("GITHUB_EVENT_PULL_REQUEST_USER_LOGIN"),
		}
	}
}

func (ci *CICDIntegration) detectGitLabEnvironment(ctx *BuildContext) {
	ctx.CommitMessage = os.Getenv("CI_COMMIT_MESSAGE")
	ctx.Author = os.Getenv("GITLAB_USER_NAME")
	ctx.BuildURL = os.Getenv("CI_PIPELINE_URL")
	
	if mrIID := os.Getenv("CI_MERGE_REQUEST_IID"); mrIID != "" {
		ctx.PullRequest = &PullRequestInfo{
			ID:           mrIID,
			Title:        os.Getenv("CI_MERGE_REQUEST_TITLE"),
			SourceBranch: os.Getenv("CI_MERGE_REQUEST_SOURCE_BRANCH_NAME"),
			TargetBranch: os.Getenv("CI_MERGE_REQUEST_TARGET_BRANCH_NAME"),
			Author:       os.Getenv("CI_MERGE_REQUEST_AUTHOR"),
		}
	}
}

func (ci *CICDIntegration) detectJenkinsEnvironment(ctx *BuildContext) {
	ctx.CommitMessage = os.Getenv("GIT_COMMIT_MSG")
	ctx.Author = os.Getenv("GIT_AUTHOR_NAME")
	ctx.BuildURL = os.Getenv("BUILD_URL")
	
	if changeID := os.Getenv("CHANGE_ID"); changeID != "" {
		ctx.PullRequest = &PullRequestInfo{
			ID:           changeID,
			Title:        os.Getenv("CHANGE_TITLE"),
			SourceBranch: os.Getenv("CHANGE_BRANCH"),
			TargetBranch: os.Getenv("CHANGE_TARGET"),
			Author:       os.Getenv("CHANGE_AUTHOR"),
		}
	}
}

func (ci *CICDIntegration) detectAzureEnvironment(ctx *BuildContext) {
	ctx.CommitMessage = os.Getenv("BUILD_SOURCEVERSIONMESSAGE")
	ctx.Author = os.Getenv("BUILD_REQUESTEDFOR")
	ctx.BuildURL = fmt.Sprintf("%s/_build/results?buildId=%s", os.Getenv("SYSTEM_TEAMFOUNDATIONCOLLECTIONURI"), os.Getenv("BUILD_BUILDID"))
	
	if prID := os.Getenv("SYSTEM_PULLREQUEST_PULLREQUESTID"); prID != "" {
		ctx.PullRequest = &PullRequestInfo{
			ID:           prID,
			SourceBranch: os.Getenv("SYSTEM_PULLREQUEST_SOURCEBRANCH"),
			TargetBranch: os.Getenv("SYSTEM_PULLREQUEST_TARGETBRANCH"),
		}
	}
}

func (ci *CICDIntegration) SetupTestEnvironment(ctx context.Context) error {
	// Create necessary directories
	dirs := []string{
		ci.config.ArtifactsPath,
		filepath.Join(ci.config.ArtifactsPath, "results"),
		filepath.Join(ci.config.ArtifactsPath, "baseline"),
		filepath.Join(ci.config.ArtifactsPath, "reports"),
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	// Load baseline based on strategy
	if err := ci.loadBaseline(ctx); err != nil {
		return fmt.Errorf("failed to load baseline: %w", err)
	}
	
	// Setup test data and dependencies
	if err := ci.setupTestDependencies(ctx); err != nil {
		return fmt.Errorf("failed to setup test dependencies: %w", err)
	}
	
	return nil
}

func (ci *CICDIntegration) loadBaseline(ctx context.Context) error {
	switch ci.config.BaselineStrategy {
	case "branch":
		return ci.loadBaselineFromBranch(ctx, "main")
	case "tag":
		return ci.loadBaselineFromTag(ctx, "latest")
	case "manual":
		// Baseline is manually managed, just load if exists
		return ci.pipeline.LoadBaseline()
	default:
		return ci.pipeline.LoadBaseline()
	}
}

func (ci *CICDIntegration) loadBaselineFromBranch(ctx context.Context, branch string) error {
	// In a real implementation, this would fetch baseline from the specified branch
	// For this exercise, we'll simulate it
	baselinePath := filepath.Join(ci.config.ArtifactsPath, "baseline", "baseline.json")
	
	// Check if baseline exists locally first
	if _, err := os.Stat(baselinePath); err == nil {
		return ci.pipeline.LoadBaseline()
	}
	
	// Create a default baseline if none exists
	defaultBaseline := pipeline.PerformanceBaseline{
		Version:     "default",
		Timestamp:   time.Now(),
		Environment: ci.config.Provider,
		Metrics: map[string]pipeline.BaselineMetric{
			"throughput": {
				Name:      "throughput",
				Value:     100.0,
				Unit:      "req/s",
				Tolerance: 5.0,
				Trend:     "stable",
			},
			"avg_latency": {
				Name:      "avg_latency",
				Value:     0.050,
				Unit:      "seconds",
				Tolerance: 10.0,
				Trend:     "stable",
			},
		},
	}
	
	data, err := json.MarshalIndent(defaultBaseline, "", "  ")
	if err != nil {
		return err
	}
	
	if err := os.WriteFile(baselinePath, data, 0644); err != nil {
		return err
	}
	
	return ci.pipeline.LoadBaseline()
}

func (ci *CICDIntegration) loadBaselineFromTag(ctx context.Context, tag string) error {
	// Similar to branch-based loading, but from a specific tag
	return ci.loadBaselineFromBranch(ctx, tag)
}

func (ci *CICDIntegration) setupTestDependencies(ctx context.Context) error {
	// Start required services (databases, caches, etc.)
	services := []string{
		"docker-compose up -d postgres redis",
		"sleep 10", // Wait for services to start
	}
	
	for _, service := range services {
		if err := ci.runCommand(ctx, service); err != nil {
			// Log error but don't fail - services might already be running
			fmt.Printf("Warning: failed to start service: %v\n", err)
		}
	}
	
	return nil
}

func (ci *CICDIntegration) ExecuteTests(ctx context.Context) (*TestExecutionResult, error) {
	buildCtx := ci.DetectEnvironment()
	
	result := &TestExecutionResult{
		BuildContext: buildCtx,
		StartTime:    time.Now(),
		Status:       "running",
		Artifacts:    make([]string, 0),
		Reports:      make([]string, 0),
	}
	
	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
	}()
	
	// Setup test environment
	if err := ci.SetupTestEnvironment(ctx); err != nil {
		result.Status = "failure"
		result.ErrorMessage = err.Error()
		result.ExitCode = 1
		return result, err
	}
	
	// Configure test suite based on trigger
	ci.configureTestSuite(buildCtx)
	
	// Execute performance tests
	if err := ci.pipeline.Run(ctx); err != nil {
		result.Status = "failure"
		result.ErrorMessage = err.Error()
		result.ExitCode = 1
		return result, err
	}
	
	// Collect results
	result.TestResults = ci.pipeline.GetResults()
	result.Regressions = ci.collectRegressions(result.TestResults)
	
	// Determine overall status
	if ci.hasFailures(result.TestResults) {
		result.Status = "failure"
		result.ExitCode = 1
	} else if ci.config.FailOnRegression && len(result.Regressions) > 0 {
		result.Status = "failure"
		result.ExitCode = 2 // Different exit code for regressions
	} else {
		result.Status = "success"
		result.ExitCode = 0
	}
	
	// Collect artifacts
	result.Artifacts = ci.collectArtifacts()
	result.Reports = ci.collectReports()
	
	return result, nil
}

func (ci *CICDIntegration) configureTestSuite(buildCtx BuildContext) {
	// Add different test configurations based on the build context
	if buildCtx.PullRequest != nil {
		// For pull requests, run lighter test suite
		ci.pipeline.AddTest(pipeline.PerformanceTest{
			Name:         "PR_LoadTest",
			Type:         "load",
			Config:       map[string]interface{}{"duration": "30s", "users": 10},
			Timeout:      2 * time.Minute,
			RetryCount:   1,
			CriticalPath: true,
		})
	} else if buildCtx.Branch == "main" || buildCtx.Branch == "master" {
		// For main branch, run comprehensive test suite
		ci.pipeline.AddTest(pipeline.PerformanceTest{
			Name:         "Main_LoadTest",
			Type:         "load",
			Config:       map[string]interface{}{"duration": "5m", "users": 50},
			Timeout:      10 * time.Minute,
			RetryCount:   2,
			CriticalPath: true,
		})
		
		ci.pipeline.AddTest(pipeline.PerformanceTest{
			Name:         "Main_StressTest",
			Type:         "stress",
			Config:       map[string]interface{}{"max_users": 200, "ramp_duration": "2m"},
			Timeout:      15 * time.Minute,
			RetryCount:   1,
			CriticalPath: false,
		})
	} else {
		// For feature branches, run basic test suite
		ci.pipeline.AddTest(pipeline.PerformanceTest{
			Name:         "Feature_LoadTest",
			Type:         "load",
			Config:       map[string]interface{}{"duration": "1m", "users": 20},
			Timeout:      5 * time.Minute,
			RetryCount:   1,
			CriticalPath: true,
		})
	}
}

func (ci *CICDIntegration) collectRegressions(results []pipeline.TestResult) []pipeline.RegressionAnalysis {
	regressions := make([]pipeline.RegressionAnalysis, 0)
	for _, result := range results {
		if result.Regression != nil {
			regressions = append(regressions, *result.Regression)
		}
	}
	return regressions
}

func (ci *CICDIntegration) hasFailures(results []pipeline.TestResult) bool {
	for _, result := range results {
		if result.Status == "failed" {
			return true
		}
	}
	return false
}

func (ci *CICDIntegration) collectArtifacts() []string {
	artifacts := make([]string, 0)
	
	// Collect test result files
	resultsDir := filepath.Join(ci.config.ArtifactsPath, "results")
	if files, err := filepath.Glob(filepath.Join(resultsDir, "*.json")); err == nil {
		artifacts = append(artifacts, files...)
	}
	
	// Collect baseline files
	baselineDir := filepath.Join(ci.config.ArtifactsPath, "baseline")
	if files, err := filepath.Glob(filepath.Join(baselineDir, "*.json")); err == nil {
		artifacts = append(artifacts, files...)
	}
	
	return artifacts
}

func (ci *CICDIntegration) collectReports() []string {
	reports := make([]string, 0)
	
	// Collect HTML and JSON reports
	reportsDir := filepath.Join(ci.config.ArtifactsPath, "results")
	patterns := []string{"*.html", "*.json"}
	
	for _, pattern := range patterns {
		if files, err := filepath.Glob(filepath.Join(reportsDir, pattern)); err == nil {
			reports = append(reports, files...)
		}
	}
	
	return reports
}

func (ci *CICDIntegration) PublishResults(ctx context.Context, result *TestExecutionResult) error {
	// Publish test results to CI/CD system
	switch ci.config.Provider {
	case "github":
		return ci.publishToGitHub(ctx, result)
	case "gitlab":
		return ci.publishToGitLab(ctx, result)
	case "jenkins":
		return ci.publishToJenkins(ctx, result)
	case "azure":
		return ci.publishToAzure(ctx, result)
	default:
		return ci.publishGeneric(ctx, result)
	}
}

func (ci *CICDIntegration) publishToGitHub(ctx context.Context, result *TestExecutionResult) error {
	// In a real implementation, this would use GitHub API to:
	// - Create check runs
	// - Add PR comments
	// - Upload artifacts
	fmt.Printf("Publishing results to GitHub: Status=%s, Duration=%s\n", result.Status, result.Duration)
	return nil
}

func (ci *CICDIntegration) publishToGitLab(ctx context.Context, result *TestExecutionResult) error {
	// Similar to GitHub but using GitLab API
	fmt.Printf("Publishing results to GitLab: Status=%s, Duration=%s\n", result.Status, result.Duration)
	return nil
}

func (ci *CICDIntegration) publishToJenkins(ctx context.Context, result *TestExecutionResult) error {
	// Publish using Jenkins API or plugins
	fmt.Printf("Publishing results to Jenkins: Status=%s, Duration=%s\n", result.Status, result.Duration)
	return nil
}

func (ci *CICDIntegration) publishToAzure(ctx context.Context, result *TestExecutionResult) error {
	// Publish using Azure DevOps API
	fmt.Printf("Publishing results to Azure DevOps: Status=%s, Duration=%s\n", result.Status, result.Duration)
	return nil
}

func (ci *CICDIntegration) publishGeneric(ctx context.Context, result *TestExecutionResult) error {
	// Generic publishing - just log results
	fmt.Printf("Test Execution Complete:\n")
	fmt.Printf("  Status: %s\n", result.Status)
	fmt.Printf("  Duration: %s\n", result.Duration)
	fmt.Printf("  Tests: %d\n", len(result.TestResults))
	fmt.Printf("  Regressions: %d\n", len(result.Regressions))
	fmt.Printf("  Artifacts: %d\n", len(result.Artifacts))
	fmt.Printf("  Reports: %d\n", len(result.Reports))
	return nil
}

func (ci *CICDIntegration) getBranchName() string {
	if branch := ci.config.Branch; branch != "" {
		return branch
	}
	
	// Try to detect from environment variables
	envVars := []string{
		"GITHUB_REF_NAME",
		"CI_COMMIT_REF_NAME",
		"BRANCH_NAME",
		"BUILD_SOURCEBRANCH",
	}
	
	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			// Clean up refs/heads/ prefix if present
			return strings.TrimPrefix(value, "refs/heads/")
		}
	}
	
	// Fallback to git command
	if output, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		return strings.TrimSpace(string(output))
	}
	
	return "unknown"
}

func (ci *CICDIntegration) getCommitSHA() string {
	if sha := ci.config.CommitSHA; sha != "" {
		return sha
	}
	
	// Try to detect from environment variables
	envVars := []string{
		"GITHUB_SHA",
		"CI_COMMIT_SHA",
		"GIT_COMMIT",
		"BUILD_SOURCEVERSION",
	}
	
	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			return value
		}
	}
	
	// Fallback to git command
	if output, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		return strings.TrimSpace(string(output))
	}
	
	return "unknown"
}

func (ci *CICDIntegration) runCommand(ctx context.Context, command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}
	
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = ci.config.Workspace
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}
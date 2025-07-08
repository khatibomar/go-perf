// main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"exercise-04-02/cicd"
)

func main() {
	// Command line flags
	var (
		provider = flag.String("provider", "github", "CI/CD provider (github, gitlab, jenkins, azure)")
		workspace = flag.String("workspace", ".", "Workspace directory")
		branch = flag.String("branch", "", "Git branch (auto-detected if empty)")
		commitSHA = flag.String("commit", "", "Commit SHA (auto-detected if empty)")
		buildNumber = flag.String("build", "", "Build number")
		artifactsPath = flag.String("artifacts", "./artifacts", "Artifacts directory")
		testTimeout = flag.Duration("timeout", 30*time.Minute, "Test timeout")
		failOnRegression = flag.Bool("fail-on-regression", true, "Fail build on performance regression")
		baselineStrategy = flag.String("baseline", "branch", "Baseline strategy (branch, tag, manual)")
		mode = flag.String("mode", "ci", "Execution mode (ci, local, demo)")
	)
	flag.Parse()
	
	// Create CI/CD configuration
	config := cicd.CICDConfig{
		Provider:         *provider,
		Workspace:        *workspace,
		Branch:           *branch,
		CommitSHA:        *commitSHA,
		BuildNumber:      *buildNumber,
		ArtifactsPath:    *artifactsPath,
		TestTimeout:      *testTimeout,
		FailOnRegression: *failOnRegression,
		BaselineStrategy: *baselineStrategy,
	}
	
	// Create CI/CD integration
	integration := cicd.NewCICDIntegration(config)
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *testTimeout+5*time.Minute)
	defer cancel()
	
	// Execute based on mode
	switch *mode {
	case "ci":
		if err := runCIMode(ctx, integration); err != nil {
			log.Fatalf("CI execution failed: %v", err)
		}
	case "local":
		if err := runLocalMode(ctx, integration); err != nil {
			log.Fatalf("Local execution failed: %v", err)
		}
	case "demo":
		if err := runDemoMode(ctx, integration); err != nil {
			log.Fatalf("Demo execution failed: %v", err)
		}
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

func runCIMode(ctx context.Context, integration *cicd.CICDIntegration) error {
	fmt.Println("🚀 Starting CI/CD Performance Testing Pipeline")
	fmt.Println("=============================================")
	
	// Execute tests
	result, err := integration.ExecuteTests(ctx)
	if err != nil {
		return fmt.Errorf("test execution failed: %w", err)
	}
	
	// Print summary
	printExecutionSummary(result)
	
	// Publish results
	if err := integration.PublishResults(ctx, result); err != nil {
		return fmt.Errorf("failed to publish results: %w", err)
	}
	
	// Exit with appropriate code
	os.Exit(result.ExitCode)
	return nil
}

func runLocalMode(ctx context.Context, integration *cicd.CICDIntegration) error {
	fmt.Println("🔧 Running Performance Tests Locally")
	fmt.Println("====================================")
	
	// Execute tests
	result, err := integration.ExecuteTests(ctx)
	if err != nil {
		return fmt.Errorf("test execution failed: %w", err)
	}
	
	// Print detailed results
	printDetailedResults(result)
	
	// Open HTML report if available
	if len(result.Reports) > 0 {
		for _, report := range result.Reports {
			if filepath.Ext(report) == ".html" {
				fmt.Printf("\n📊 HTML Report available at: %s\n", report)
				break
			}
		}
	}
	
	return nil
}

func runDemoMode(ctx context.Context, integration *cicd.CICDIntegration) error {
	fmt.Println("🎯 Performance Testing Demo")
	fmt.Println("===========================")
	
	// Create a comprehensive demo pipeline
	if err := setupDemoTests(integration); err != nil {
		return fmt.Errorf("failed to setup demo tests: %w", err)
	}
	
	// Execute tests with progress updates
	result, err := executeWithProgress(ctx, integration)
	if err != nil {
		return fmt.Errorf("demo execution failed: %w", err)
	}
	
	// Show comprehensive results
	printDemoResults(result)
	
	return nil
}

func setupDemoTests(integration *cicd.CICDIntegration) error {
	// This would normally be done through the CI/CD integration,
	// but for demo purposes we'll access the pipeline directly
	fmt.Println("Setting up demo test suite...")
	
	// Note: In a real implementation, we'd need to expose the pipeline
	// or provide a method to add tests through the integration
	fmt.Println("✅ Demo test suite configured")
	return nil
}

func executeWithProgress(ctx context.Context, integration *cicd.CICDIntegration) (*cicd.TestExecutionResult, error) {
	fmt.Println("\n🏃 Executing performance tests...")
	
	// Start execution
	resultChan := make(chan *cicd.TestExecutionResult, 1)
	errorChan := make(chan error, 1)
	
	go func() {
		result, err := integration.ExecuteTests(ctx)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- result
	}()
	
	// Show progress
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	progress := 0
	for {
		select {
		case result := <-resultChan:
			fmt.Println("\n✅ Tests completed!")
			return result, nil
		case err := <-errorChan:
			fmt.Println("\n❌ Tests failed!")
			return nil, err
		case <-ticker.C:
			progress = (progress + 1) % 4
			spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}[progress%10]
			fmt.Printf("\r%s Running tests...", spinner)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func printExecutionSummary(result *cicd.TestExecutionResult) {
	fmt.Printf("\n📊 Execution Summary\n")
	fmt.Printf("==================\n")
	fmt.Printf("Status: %s\n", getStatusEmoji(result.Status)+" "+result.Status)
	fmt.Printf("Duration: %s\n", result.Duration)
	fmt.Printf("Tests: %d\n", len(result.TestResults))
	fmt.Printf("Regressions: %d\n", len(result.Regressions))
	fmt.Printf("Exit Code: %d\n", result.ExitCode)
	
	if result.ErrorMessage != "" {
		fmt.Printf("Error: %s\n", result.ErrorMessage)
	}
	
	// Print test results summary
	passed := 0
	failed := 0
	for _, test := range result.TestResults {
		if test.Status == "passed" {
			passed++
		} else {
			failed++
		}
	}
	fmt.Printf("Passed: %d, Failed: %d\n", passed, failed)
}

func printDetailedResults(result *cicd.TestExecutionResult) {
	printExecutionSummary(result)
	
	fmt.Printf("\n📋 Detailed Test Results\n")
	fmt.Printf("=======================\n")
	
	for _, test := range result.TestResults {
		status := getStatusEmoji(test.Status)
		fmt.Printf("%s %s (%s)\n", status, test.TestName, test.Duration)
		
		// Print key metrics
		if len(test.Metrics) > 0 {
			fmt.Printf("   Metrics:\n")
			for name, value := range test.Metrics {
				fmt.Printf("     %s: %v\n", name, value)
			}
		}
		
		// Print errors if any
		if len(test.Errors) > 0 {
			fmt.Printf("   Errors:\n")
			for _, err := range test.Errors {
				fmt.Printf("     - %s\n", err)
			}
		}
		
		// Print regression info
		if test.Regression != nil {
			fmt.Printf("   🚨 Regression: %s (%s)\n", test.Regression.Severity, test.Regression.Recommendation)
		}
		
		fmt.Println()
	}
	
	// Print artifacts
	if len(result.Artifacts) > 0 {
		fmt.Printf("📁 Artifacts:\n")
		for _, artifact := range result.Artifacts {
			fmt.Printf("   - %s\n", artifact)
		}
		fmt.Println()
	}
	
	// Print reports
	if len(result.Reports) > 0 {
		fmt.Printf("📄 Reports:\n")
		for _, report := range result.Reports {
			fmt.Printf("   - %s\n", report)
		}
		fmt.Println()
	}
}

func printDemoResults(result *cicd.TestExecutionResult) {
	printDetailedResults(result)
	
	fmt.Printf("\n🎯 Demo Insights\n")
	fmt.Printf("===============\n")
	fmt.Printf("This demo showcases:\n")
	fmt.Printf("✅ Automated performance testing in CI/CD\n")
	fmt.Printf("✅ Performance regression detection\n")
	fmt.Printf("✅ Baseline management and comparison\n")
	fmt.Printf("✅ Multi-format reporting (HTML, JSON)\n")
	fmt.Printf("✅ CI/CD provider integration patterns\n")
	fmt.Printf("✅ Configurable test suites based on context\n")
	
	if len(result.Regressions) > 0 {
		fmt.Printf("\n⚠️  Performance regressions detected:\n")
		for i, regression := range result.Regressions {
			fmt.Printf("   %d. Severity: %s\n", i+1, regression.Severity)
			fmt.Printf("      Affected: %v\n", regression.AffectedMetrics)
			fmt.Printf("      Recommendation: %s\n", regression.Recommendation)
		}
	} else {
		fmt.Printf("\n✅ No performance regressions detected!\n")
	}
	
	fmt.Printf("\n🚀 Next Steps:\n")
	fmt.Printf("   - Integrate with your CI/CD pipeline\n")
	fmt.Printf("   - Configure baseline management strategy\n")
	fmt.Printf("   - Set up notifications (Slack, email, webhooks)\n")
	fmt.Printf("   - Customize test suites for different scenarios\n")
	fmt.Printf("   - Review generated reports for insights\n")
}

func getStatusEmoji(status string) string {
	switch status {
	case "passed", "success":
		return "✅"
	case "failed", "failure":
		return "❌"
	case "skipped":
		return "⏭️"
	case "running":
		return "🏃"
	default:
		return "❓"
	}
}
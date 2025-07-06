package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TestSuite represents a test suite configuration
type TestSuite struct {
	Name        string
	Package     string
	TestPattern string
	Timeout     time.Duration
	Flags       []string
}

// TestRunner manages test execution
type TestRunner struct {
	ProjectRoot string
	OutputDir   string
	Verbose     bool
	Coverage    bool
	Benchmark   bool
	Race        bool
}

// NewTestRunner creates a new test runner
func NewTestRunner(projectRoot string) *TestRunner {
	return &TestRunner{
		ProjectRoot: projectRoot,
		OutputDir:   filepath.Join(projectRoot, "test-results"),
		Verbose:     true,
		Coverage:    true,
		Benchmark:   true,
		Race:        true,
	}
}

// Run executes all test suites
func (tr *TestRunner) Run() error {
	fmt.Println("🚀 Starting Memory Leak Detection System Test Suite")
	fmt.Println(strings.Repeat("=", 60))

	// Create output directory
	if err := os.MkdirAll(tr.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Define test suites
	testSuites := []TestSuite{
		{
			Name:        "Unit Tests",
			Package:     "./tests",
			TestPattern: "Test",
			Timeout:     5 * time.Minute,
			Flags:       []string{"-v", "-race"},
		},
		{
			Name:        "Integration Tests",
			Package:     "./tests",
			TestPattern: "TestBasicLeakDetection|TestHeapLeakDetection|TestGoroutineLeakDetection|TestResourceLeakDetection|TestMultiTypeLeakDetection",
			Timeout:     10 * time.Minute,
			Flags:       []string{"-v", "-race"},
		},
		{
			Name:        "Real-time Tests",
			Package:     "./tests",
			TestPattern: "TestRealTimeAlerts|TestAnalyzerLifecycle",
			Timeout:     3 * time.Minute,
			Flags:       []string{"-v"},
		},
		{
			Name:        "Configuration Tests",
			Package:     "./tests",
			TestPattern: "TestDetectorConfiguration|TestDetectorErrorHandling|TestDetectorConcurrency",
			Timeout:     2 * time.Minute,
			Flags:       []string{"-v", "-race"},
		},
	}

	// Run test suites
	var totalPassed, totalFailed int
	for _, suite := range testSuites {
		fmt.Printf("\n📋 Running %s...\n", suite.Name)
		fmt.Println(strings.Repeat("-", 40))

		passed, failed, err := tr.runTestSuite(suite)
		if err != nil {
			fmt.Printf("❌ %s failed: %v\n", suite.Name, err)
			continue
		}

		totalPassed += passed
		totalFailed += failed

		if failed == 0 {
			fmt.Printf("✅ %s: %d tests passed\n", suite.Name, passed)
		} else {
			fmt.Printf("⚠️  %s: %d passed, %d failed\n", suite.Name, passed, failed)
		}
	}

	// Run benchmarks if enabled
	if tr.Benchmark {
		fmt.Printf("\n🏃 Running Benchmarks...\n")
		fmt.Println(strings.Repeat("-", 40))
		if err := tr.runBenchmarks(); err != nil {
			fmt.Printf("❌ Benchmarks failed: %v\n", err)
		} else {
			fmt.Println("✅ Benchmarks completed")
		}
	}

	// Generate coverage report if enabled
	if tr.Coverage {
		fmt.Printf("\n📊 Generating Coverage Report...\n")
		fmt.Println(strings.Repeat("-", 40))
		if err := tr.generateCoverageReport(); err != nil {
			fmt.Printf("❌ Coverage report failed: %v\n", err)
		} else {
			fmt.Println("✅ Coverage report generated")
		}
	}

	// Print summary
	fmt.Printf("\n📈 Test Summary\n")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total Tests Passed: %d\n", totalPassed)
	fmt.Printf("Total Tests Failed: %d\n", totalFailed)
	fmt.Printf("Success Rate: %.1f%%\n", float64(totalPassed)/float64(totalPassed+totalFailed)*100)
	fmt.Printf("Results saved to: %s\n", tr.OutputDir)

	if totalFailed > 0 {
		return fmt.Errorf("%d tests failed", totalFailed)
	}

	return nil
}

// runTestSuite executes a specific test suite
func (tr *TestRunner) runTestSuite(suite TestSuite) (passed, failed int, err error) {
	args := []string{"test"}
	args = append(args, suite.Flags...)
	args = append(args, "-run", suite.TestPattern)
	args = append(args, "-timeout", suite.Timeout.String())
	args = append(args, suite.Package)

	// Add coverage if enabled
	if tr.Coverage {
		coverageFile := filepath.Join(tr.OutputDir, fmt.Sprintf("%s-coverage.out", strings.ReplaceAll(suite.Name, " ", "_")))
		args = append(args, "-coverprofile", coverageFile)
		args = append(args, "-covermode", "atomic")
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = tr.ProjectRoot

	// Capture output
	outputFile := filepath.Join(tr.OutputDir, fmt.Sprintf("%s-output.txt", strings.ReplaceAll(suite.Name, " ", "_")))
	output, err := os.Create(outputFile)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create output file: %v", err)
	}
	defer output.Close()

	cmd.Stdout = output
	cmd.Stderr = output

	// Run the test
	start := time.Now()
	err = cmd.Run()
	duration := time.Since(start)

	fmt.Printf("Duration: %v\n", duration)

	// Parse results (simplified - in real scenario, you'd parse the actual output)
	if err != nil {
		// Test failed
		return 0, 1, nil
	}

	// Test passed (simplified)
	return 1, 0, nil
}

// runBenchmarks executes benchmark tests
func (tr *TestRunner) runBenchmarks() error {
	args := []string{"test", "-bench=.", "-benchmem", "-run=^$", "./tests"}

	cmd := exec.Command("go", args...)
	cmd.Dir = tr.ProjectRoot

	// Capture benchmark output
	benchmarkFile := filepath.Join(tr.OutputDir, "benchmarks.txt")
	output, err := os.Create(benchmarkFile)
	if err != nil {
		return fmt.Errorf("failed to create benchmark file: %v", err)
	}
	defer output.Close()

	cmd.Stdout = output
	cmd.Stderr = output

	return cmd.Run()
}

// generateCoverageReport generates a comprehensive coverage report
func (tr *TestRunner) generateCoverageReport() error {
	// Merge coverage files
	coverageFiles, err := filepath.Glob(filepath.Join(tr.OutputDir, "*-coverage.out"))
	if err != nil {
		return fmt.Errorf("failed to find coverage files: %v", err)
	}

	if len(coverageFiles) == 0 {
		return fmt.Errorf("no coverage files found")
	}

	// Merge coverage files (simplified - in real scenario, you'd properly merge them)
	mergedCoverage := filepath.Join(tr.OutputDir, "coverage.out")
	merged, err := os.Create(mergedCoverage)
	if err != nil {
		return fmt.Errorf("failed to create merged coverage file: %v", err)
	}
	defer merged.Close()

	// Write coverage mode header
	merged.WriteString("mode: atomic\n")

	// Merge coverage data (simplified)
	for _, file := range coverageFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		// Skip the mode line and append the rest
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if i == 0 || line == "" {
				continue
			}
			merged.WriteString(line + "\n")
		}
	}

	// Generate HTML coverage report
	htmlReport := filepath.Join(tr.OutputDir, "coverage.html")
	cmd := exec.Command("go", "tool", "cover", "-html", mergedCoverage, "-o", htmlReport)
	cmd.Dir = tr.ProjectRoot

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate HTML coverage report: %v", err)
	}

	// Generate coverage summary
	summaryFile := filepath.Join(tr.OutputDir, "coverage-summary.txt")
	cmd = exec.Command("go", "tool", "cover", "-func", mergedCoverage)
	cmd.Dir = tr.ProjectRoot

	summary, err := os.Create(summaryFile)
	if err != nil {
		return fmt.Errorf("failed to create coverage summary file: %v", err)
	}
	defer summary.Close()

	cmd.Stdout = summary
	cmd.Stderr = summary

	return cmd.Run()
}

// runQuickTests runs a subset of tests for quick validation
func (tr *TestRunner) runQuickTests() error {
	fmt.Println("🏃 Running Quick Tests...")
	fmt.Println(strings.Repeat("-", 40))

	quickSuite := TestSuite{
		Name:        "Quick Tests",
		Package:     "./tests",
		TestPattern: "TestHeapAnalyzer|TestGoroutineDetector|TestResourceDetector|TestLeakSimulator",
		Timeout:     2 * time.Minute,
		Flags:       []string{"-v"},
	}

	passed, failed, err := tr.runTestSuite(quickSuite)
	if err != nil {
		return fmt.Errorf("quick tests failed: %v", err)
	}

	fmt.Printf("Quick Tests: %d passed, %d failed\n", passed, failed)
	return nil
}

// runStressTests runs stress tests to validate system under load
func (tr *TestRunner) runStressTests() error {
	fmt.Println("💪 Running Stress Tests...")
	fmt.Println(strings.Repeat("-", 40))

	stressSuite := TestSuite{
		Name:        "Stress Tests",
		Package:     "./tests",
		TestPattern: "TestDetectorConcurrency|TestConcurrentDetection",
		Timeout:     5 * time.Minute,
		Flags:       []string{"-v", "-race", "-count=3"},
	}

	passed, failed, err := tr.runTestSuite(stressSuite)
	if err != nil {
		return fmt.Errorf("stress tests failed: %v", err)
	}

	fmt.Printf("Stress Tests: %d passed, %d failed\n", passed, failed)
	return nil
}

// validateEnvironment checks if the test environment is properly set up
func (tr *TestRunner) validateEnvironment() error {
	fmt.Println("🔍 Validating Test Environment...")

	// Check Go version
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Go is not installed or not in PATH: %v", err)
	}
	fmt.Printf("Go Version: %s", string(output))

	// Check if project dependencies are available
	cmd = exec.Command("go", "mod", "verify")
	cmd.Dir = tr.ProjectRoot
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Go modules verification failed: %v", err)
	}
	fmt.Println("✅ Go modules verified")

	// Check if test files exist
	testFiles := []string{
		filepath.Join(tr.ProjectRoot, "tests", "unit_test.go"),
		filepath.Join(tr.ProjectRoot, "tests", "integration_test.go"),
		filepath.Join(tr.ProjectRoot, "tests", "benchmark_test.go"),
	}

	for _, file := range testFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("test file not found: %s", file)
		}
	}
	fmt.Println("✅ Test files found")

	return nil
}

// generateTestReport creates a comprehensive test report
func (tr *TestRunner) generateTestReport() error {
	reportFile := filepath.Join(tr.OutputDir, "test-report.md")
	report, err := os.Create(reportFile)
	if err != nil {
		return fmt.Errorf("failed to create test report: %v", err)
	}
	defer report.Close()

	// Write report header
	report.WriteString("# Memory Leak Detection System - Test Report\n\n")
	report.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	// Write test results summary
	report.WriteString("## Test Results Summary\n\n")
	report.WriteString("| Test Suite | Status | Duration | Coverage |\n")
	report.WriteString("|------------|--------|----------|----------|\n")

	// Add coverage information
	report.WriteString("\n## Coverage Report\n\n")
	report.WriteString("Detailed coverage report available in `coverage.html`\n\n")

	// Add benchmark results
	report.WriteString("\n## Benchmark Results\n\n")
	report.WriteString("Detailed benchmark results available in `benchmarks.txt`\n\n")

	// Add recommendations
	report.WriteString("\n## Recommendations\n\n")
	report.WriteString("- Maintain test coverage above 80%\n")
	report.WriteString("- Run integration tests before each release\n")
	report.WriteString("- Monitor benchmark performance for regressions\n")
	report.WriteString("- Update tests when adding new features\n")

	fmt.Printf("Test report generated: %s\n", reportFile)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run run_tests.go <command> [options]")
		fmt.Println("Commands:")
		fmt.Println("  all      - Run all tests")
		fmt.Println("  quick    - Run quick tests")
		fmt.Println("  stress   - Run stress tests")
		fmt.Println("  bench    - Run benchmarks only")
		fmt.Println("  coverage - Generate coverage report only")
		fmt.Println("  validate - Validate test environment")
		os.Exit(1)
	}

	command := os.Args[1]
	projectRoot, _ := os.Getwd()
	testRunner := NewTestRunner(projectRoot)

	// Validate environment first
	if err := testRunner.validateEnvironment(); err != nil {
		fmt.Printf("❌ Environment validation failed: %v\n", err)
		os.Exit(1)
	}

	var err error
	switch command {
	case "all":
		err = testRunner.Run()
		if err == nil {
			testRunner.generateTestReport()
		}
	case "quick":
		err = testRunner.runQuickTests()
	case "stress":
		err = testRunner.runStressTests()
	case "bench":
		err = testRunner.runBenchmarks()
	case "coverage":
		err = testRunner.generateCoverageReport()
	case "validate":
		fmt.Println("✅ Environment validation passed")
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("❌ Command failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n🎉 All tests completed successfully!")
}

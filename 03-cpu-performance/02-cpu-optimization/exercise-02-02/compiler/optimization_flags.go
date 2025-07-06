package compiler

import (
	"fmt"
	"runtime"
	"strings"
)

// ShowBuildFlagsInfo displays information about Go build flags
func ShowBuildFlagsInfo() {
	fmt.Println("Go Build Flags for Optimization:")
	fmt.Println("\nGCFLAGS (Go Compiler Flags):")
	fmt.Println("  -N: Disable optimizations")
	fmt.Println("  -l: Disable inlining (levels 0-4, default varies)")
	fmt.Println("  -l=0: Disable all inlining")
	fmt.Println("  -l=4: Maximum inlining aggressiveness")
	fmt.Println("  -m: Print optimization decisions")
	fmt.Println("  -S: Print assembly listing")
	fmt.Println("  -B: Disable bounds checking")
	fmt.Println("  -C: Disable nil checks")
	fmt.Println("\nLDFLAGS (Linker Flags):")
	fmt.Println("  -s: Strip symbol table and debug info")
	fmt.Println("  -w: Strip DWARF debug info")
	fmt.Println("  -X: Set string variable values")
	fmt.Println("\nBuild Tags and Environment:")
	fmt.Println("  GOOS, GOARCH: Target platform")
	fmt.Println("  CGO_ENABLED: Enable/disable CGO")
	fmt.Println("  -race: Enable race detector (debug builds)")
	fmt.Println("  -tags: Build tags for conditional compilation")
}

// OptimizedFunction - function that benefits from optimizations
func OptimizedFunction(data []int) int {
	result := 0
	
	// Simple loop that can be optimized
	for i := 0; i < len(data); i++ {
		result += optimizedHelper(data[i], i)
	}
	
	return result
}

// optimizedHelper - small function that should be inlined
func optimizedHelper(value, index int) int {
	return value*2 + index
}

// UnoptimizedFunction - function that simulates unoptimized behavior
func UnoptimizedFunction(data []int) int {
	result := 0
	
	// More complex loop that's harder to optimize
	for i := 0; i < len(data); i++ {
		result += unoptimizedHelper(data[i], i)
		
		// Add some complexity to prevent optimization
		if i%2 == 0 {
			result += complexCalculation(data[i])
		}
	}
	
	return result
}

// unoptimizedHelper - function that's less likely to be inlined
//go:noinline
func unoptimizedHelper(value, index int) int {
	// More complex calculation
	temp := value * 2
	if temp > 100 {
		temp /= 2
	}
	return temp + index
}

// complexCalculation - expensive calculation
//go:noinline
func complexCalculation(value int) int {
	result := value
	for i := 0; i < 10; i++ {
		result = (result*31 + 17) % 1000
	}
	return result
}

// BuildFlagsDemo - demonstrates different optimization scenarios
type BuildFlagsDemo struct {
	Name        string
	Description string
	Flags       []string
}

// GetOptimizationScenarios returns different build flag scenarios
func GetOptimizationScenarios() []BuildFlagsDemo {
	return []BuildFlagsDemo{
		{
			Name:        "Debug Build",
			Description: "No optimizations, full debug info",
			Flags:       []string{"-gcflags=-N -l", "-ldflags="},
		},
		{
			Name:        "Default Build",
			Description: "Standard optimizations",
			Flags:       []string{""},
		},
		{
			Name:        "Aggressive Optimization",
			Description: "Maximum inlining and optimizations",
			Flags:       []string{"-gcflags=-l=4"},
		},
		{
			Name:        "Size Optimized",
			Description: "Optimized for binary size",
			Flags:       []string{"-ldflags=-s -w"},
		},
		{
			Name:        "Production Build",
			Description: "Optimized for production deployment",
			Flags:       []string{"-gcflags=-l=4", "-ldflags=-s -w"},
		},
		{
			Name:        "Race Detection",
			Description: "Debug build with race detection",
			Flags:       []string{"-race", "-gcflags=-N -l"},
		},
	}
}

// CompilerOptimizationLevels demonstrates different optimization levels
func CompilerOptimizationLevels() {
	fmt.Println("\n=== Compiler Optimization Levels ===")
	
	levels := map[string]string{
		"-l=0": "No inlining",
		"-l=1": "Conservative inlining",
		"-l=2": "Default inlining",
		"-l=3": "Aggressive inlining",
		"-l=4": "Maximum inlining",
	}
	
	for flag, desc := range levels {
		fmt.Printf("%-6s: %s\n", flag, desc)
	}
}

// MemoryOptimizationFlags demonstrates memory-related flags
func MemoryOptimizationFlags() {
	fmt.Println("\n=== Memory Optimization Flags ===")
	
	flags := map[string]string{
		"-ldflags=-s":     "Strip symbol table",
		"-ldflags=-w":     "Strip debug info",
		"-ldflags=-s -w":  "Strip both (smallest binary)",
		"-gcflags=-B":     "Disable bounds checking (unsafe)",
		"-gcflags=-C":     "Disable nil checks (unsafe)",
	}
	
	for flag, desc := range flags {
		fmt.Printf("%-20s: %s\n", flag, desc)
	}
}

// RuntimeOptimizationInfo shows runtime optimization information
func RuntimeOptimizationInfo() {
	fmt.Println("\n=== Runtime Optimization Information ===")
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Architecture: %s\n", runtime.GOARCH)
	fmt.Printf("Operating System: %s\n", runtime.GOOS)
	fmt.Printf("CPU Count: %d\n", runtime.NumCPU())
	fmt.Printf("Goroutines: %d\n", runtime.NumGoroutine())
	
	// Check if race detector is enabled
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Race Detector: %v\n", isRaceDetectorEnabled())
}

// isRaceDetectorEnabled checks if race detector is enabled
func isRaceDetectorEnabled() bool {
	// This is a heuristic - race detector typically increases memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Sys > 100*1024*1024 // Rough heuristic
}

// ConditionalCompilation demonstrates build tags
func ConditionalCompilation() {
	fmt.Println("\n=== Conditional Compilation Examples ===")
	fmt.Println("Build tags allow conditional compilation:")
	fmt.Println("  //go:build debug")
	fmt.Println("  //go:build !debug")
	fmt.Println("  //go:build linux && amd64")
	fmt.Println("  //go:build cgo")
	fmt.Println("\nUsage: go build -tags debug")
}

// PerformanceFlags demonstrates performance-related flags
type PerformanceFlags struct {
	Flag        string
	Description string
	UseCase     string
	Impact      string
}

func GetPerformanceFlags() []PerformanceFlags {
	return []PerformanceFlags{
		{
			Flag:        "-gcflags=-l=4",
			Description: "Maximum function inlining",
			UseCase:     "CPU-intensive applications",
			Impact:      "Faster execution, larger binary",
		},
		{
			Flag:        "-ldflags=-s -w",
			Description: "Strip debug information",
			UseCase:     "Production deployments",
			Impact:      "Smaller binary, no debugging",
		},
		{
			Flag:        "-gcflags=-B",
			Description: "Disable bounds checking",
			UseCase:     "High-performance critical paths",
			Impact:      "Faster array access, unsafe",
		},
		{
			Flag:        "-gcflags=-N -l",
			Description: "Disable all optimizations",
			UseCase:     "Debugging and development",
			Impact:      "Slower execution, better debugging",
		},
		{
			Flag:        "-race",
			Description: "Enable race detector",
			UseCase:     "Concurrent code testing",
			Impact:      "Much slower, detects races",
		},
	}
}

// ShowPerformanceFlags displays performance flag information
func ShowPerformanceFlags() {
	fmt.Println("\n=== Performance Build Flags ===")
	flags := GetPerformanceFlags()
	
	for _, flag := range flags {
		fmt.Printf("\nFlag: %s\n", flag.Flag)
		fmt.Printf("Description: %s\n", flag.Description)
		fmt.Printf("Use Case: %s\n", flag.UseCase)
		fmt.Printf("Impact: %s\n", flag.Impact)
	}
}

// OptimizationBenchmark - benchmark different optimization levels
func OptimizationBenchmark(data []int, optimized bool) int {
	if optimized {
		return OptimizedFunction(data)
	}
	return UnoptimizedFunction(data)
}

// CompilerDirectives demonstrates compiler directives
func CompilerDirectives() {
	fmt.Println("\n=== Compiler Directives ===")
	directives := []string{
		"//go:noinline - Prevent function inlining",
		"//go:nosplit - Prevent stack splitting",
		"//go:noescape - Mark parameters as non-escaping",
		"//go:linkname - Link to runtime functions",
		"//go:build - Build constraints",
		"//go:generate - Code generation",
	}
	
	for _, directive := range directives {
		fmt.Printf("  %s\n", directive)
	}
}

// EnvironmentVariables shows relevant environment variables
func EnvironmentVariables() {
	fmt.Println("\n=== Optimization Environment Variables ===")
	vars := map[string]string{
		"GOOS":        "Target operating system",
		"GOARCH":      "Target architecture",
		"CGO_ENABLED": "Enable/disable CGO",
		"GOMAXPROCS":  "Maximum number of OS threads",
		"GOGC":        "Garbage collection target percentage",
		"GOMEMLIMIT":  "Memory limit for the Go runtime",
	}
	
	for env, desc := range vars {
		fmt.Printf("%-12s: %s\n", env, desc)
	}
}

// BuildCommandExamples shows example build commands
func BuildCommandExamples() {
	fmt.Println("\n=== Build Command Examples ===")
	
	examples := []struct {
		name    string
		command string
		purpose string
	}{
		{
			name:    "Debug Build",
			command: "go build -gcflags='-N -l' -o app-debug",
			purpose: "Development and debugging",
		},
		{
			name:    "Production Build",
			command: "go build -ldflags='-s -w' -gcflags='-l=4' -o app-prod",
			purpose: "Optimized production deployment",
		},
		{
			name:    "Cross Compilation",
			command: "GOOS=linux GOARCH=amd64 go build -o app-linux",
			purpose: "Build for different platform",
		},
		{
			name:    "Race Detection",
			command: "go build -race -o app-race",
			purpose: "Testing concurrent code",
		},
	}
	
	for _, example := range examples {
		fmt.Printf("\n%s:\n", example.name)
		fmt.Printf("  Command: %s\n", example.command)
		fmt.Printf("  Purpose: %s\n", example.purpose)
	}
}

// AnalyzeBuildFlags analyzes the impact of different build flags
func AnalyzeBuildFlags(data []int) map[string]int {
	results := make(map[string]int)
	
	// Simulate different optimization levels
	results["optimized"] = OptimizedFunction(data)
	results["unoptimized"] = UnoptimizedFunction(data)
	
	// Calculate relative performance
	results["improvement"] = results["unoptimized"] - results["optimized"]
	
	return results
}

// GetRecommendedFlags returns recommended flags for different scenarios
func GetRecommendedFlags() map[string][]string {
	return map[string][]string{
		"development": {"-gcflags=-N -l", "-race"},
		"testing":     {"-race", "-cover"},
		"production":  {"-ldflags=-s -w", "-gcflags=-l=4"},
		"debugging":   {"-gcflags=-N -l"},
		"profiling":   {"-gcflags=-l=4"},
	}
}

// PrintRecommendedFlags displays recommended flags for different scenarios
func PrintRecommendedFlags() {
	fmt.Println("\n=== Recommended Build Flags ===")
	flags := GetRecommendedFlags()
	
	for scenario, flagList := range flags {
		fmt.Printf("\n%s:\n", strings.Title(scenario))
		for _, flag := range flagList {
			fmt.Printf("  %s\n", flag)
		}
	}
}
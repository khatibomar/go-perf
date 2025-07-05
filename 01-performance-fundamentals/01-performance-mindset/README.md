# Chapter 01: Performance Mindset and Methodology

**Duration:** 4-6 hours  
**Difficulty:** Beginner

## Learning Objectives

By the end of this chapter, you will:
- ✅ Understand the performance optimization mindset
- ✅ Learn measurement-driven development principles
- ✅ Identify common performance anti-patterns
- ✅ Set up a proper development environment for performance work
- ✅ Apply systematic performance optimization methodology

## Theory: Performance Optimization Principles

### The Performance Mindset

Performance optimization is not about writing "fast code" from the start. It's about:

1. **Measure First, Optimize Second**
   - Never assume where bottlenecks are
   - Use data to drive decisions
   - Profile before and after changes

2. **Understand Your Requirements**
   - What is "fast enough" for your use case?
   - Latency vs throughput trade-offs
   - Resource constraints (CPU, memory, network)

3. **Optimize the Right Things**
   - Focus on hot paths (code that runs frequently)
   - 80/20 rule: 80% of time spent in 20% of code
   - Don't optimize code that rarely runs

### Performance Optimization Methodology

```
1. BASELINE → 2. MEASURE → 3. ANALYZE → 4. OPTIMIZE → 5. VERIFY → (repeat)
     ↑                                                        |
     └────────────────────────────────────────────────────────┘
```

**Step 1: Establish Baseline**
- Create reproducible test scenarios
- Document current performance metrics
- Set up monitoring and alerting

**Step 2: Measure Performance**
- Use profiling tools (pprof, trace)
- Collect metrics (CPU, memory, I/O)
- Identify bottlenecks

**Step 3: Analyze Results**
- Understand why bottlenecks exist
- Research potential solutions
- Estimate impact of changes

**Step 4: Optimize**
- Make targeted changes
- One optimization at a time
- Keep changes small and focused

**Step 5: Verify Improvements**
- Re-measure performance
- Ensure no regressions
- Document improvements

### Common Performance Anti-Patterns

❌ **Premature Optimization**
```go
// DON'T: Optimizing before measuring
func processData(data []string) []string {
    // Complex optimization that may not be needed
    result := make([]string, 0, len(data)*2) // Pre-allocating too much
    // ... complex logic
    return result
}
```

✅ **Measure-Driven Optimization**
```go
// DO: Start simple, then optimize based on profiling
func processData(data []string) []string {
    var result []string
    for _, item := range data {
        // Simple, readable logic first
        processed := strings.ToUpper(item)
        result = append(result, processed)
    }
    return result
}
```

❌ **Micro-Optimizations Without Context**
```go
// DON'T: Optimizing insignificant code
func rarely_called_function() {
    // Spending time optimizing code that runs once per hour
}
```

❌ **Ignoring Algorithmic Complexity**
```go
// DON'T: O(n²) when O(n) is possible
func findDuplicates(items []string) []string {
    var duplicates []string
    for i := 0; i < len(items); i++ {
        for j := i + 1; j < len(items); j++ {
            if items[i] == items[j] {
                duplicates = append(duplicates, items[i])
            }
        }
    }
    return duplicates
}
```

✅ **Algorithm-First Optimization**
```go
// DO: Better algorithm first
func findDuplicates(items []string) []string {
    seen := make(map[string]bool)
    var duplicates []string
    
    for _, item := range items {
        if seen[item] {
            duplicates = append(duplicates, item)
        } else {
            seen[item] = true
        }
    }
    return duplicates
}
```

## Development Environment Setup

### Essential Tools Installation

```bash
#!/bin/bash
# save as setup-perf-env.sh

echo "Setting up Go Performance Development Environment..."

# Verify Go installation
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Please install Go 1.24+ first."
    exit 1
fi

echo "Go version: $(go version)"

# Install pprof
echo "Installing pprof..."
go install github.com/google/pprof@latest

# Install additional Go tools
echo "Installing Go tools..."
go install golang.org/x/tools/cmd/benchcmp@latest
go install github.com/pkg/profile@latest

# Install system monitoring tools
echo "Installing system tools..."
sudo apt update
sudo apt install -y \
    htop \
    iotop \
    sysstat \
    graphviz \
    curl \
    wget

# Create workspace directories
echo "Creating workspace..."
mkdir -p ~/go-perf-workspace/{benchmarks,profiles,examples}

echo "Setup complete! Environment ready for performance work."
```

### IDE Configuration

For VS Code, install these extensions:
- Go extension (official)
- Go Profiler extension
- Benchmark extension

## Video Resources

📺 **Watch Before Exercises:**
- ["High Performance Go" by Dave Cheney](https://www.youtube.com/watch?v=2557w0qsDV0) (First 20 minutes)

## Exercises

### Exercise 01-01: Environment Setup and First Measurement

**Objective:** Set up your environment and create your first performance measurement.

**Tasks:**
1. Run the environment setup script
2. Create a simple Go program
3. Measure its basic performance
4. Document baseline metrics

**Implementation:**

Create `exercise-01-01/main.go`:
```go
package main

import (
	"fmt"
	"time"
)

func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}

func main() {
	n := 35
	
	// Measure execution time
	start := time.Now()
	result := fibonacci(n)
	duration := time.Since(start)
	
	fmt.Printf("fibonacci(%d) = %d\n", n, result)
	fmt.Printf("Execution time: %v\n", duration)
	fmt.Printf("Nanoseconds: %d\n", duration.Nanoseconds())
}
```

Create `exercise-01-01/benchmark_test.go`:
```go
package main

import "testing"

func BenchmarkFibonacci(b *testing.B) {
	for i := 0; i < b.N; i++ {
		fibonacci(35)
	}
}

func BenchmarkFibonacci20(b *testing.B) {
	for i := 0; i < b.N; i++ {
		fibonacci(20)
	}
}

func BenchmarkFibonacci30(b *testing.B) {
	for i := 0; i < b.N; i++ {
		fibonacci(30)
	}
}
```

**Run the exercise:**
```bash
# Run the program
go run main.go

# Run benchmarks
go test -bench=. -benchmem

# Save benchmark results
go test -bench=. -benchmem > baseline.txt
```

**Expected Output:**
```
fibonacci(35) = 9227465
Execution time: 45.123456ms
Nanoseconds: 45123456

goos: linux
goarch: amd64
BenchmarkFibonacci-8      	      30	  45123456 ns/op	       0 B/op	       0 allocs/op
BenchmarkFibonacci20-8    	   50000	     23456 ns/op	       0 B/op	       0 allocs/op
BenchmarkFibonacci30-8    	    3000	    456789 ns/op	       0 B/op	       0 allocs/op
```

### Exercise 01-02: Identifying Performance Anti-Patterns

**Objective:** Recognize and fix common performance anti-patterns.

**Tasks:**
1. Analyze provided code with anti-patterns
2. Identify the issues
3. Propose improvements
4. Measure the difference

Create `exercise-01-02/antipatterns.go`:
```go
package main

import (
	"fmt"
	"strings"
)

// Anti-pattern 1: String concatenation in loop
func buildStringBad(items []string) string {
	var result string
	for _, item := range items {
		result += item + ","
	}
	return result
}

// Anti-pattern 2: Unnecessary allocations
func processItemsBad(items []string) []string {
	var result []string
	for _, item := range items {
		// Creating new slice every iteration
		temp := []string{strings.ToUpper(item)}
		result = append(result, temp[0])
	}
	return result
}

// Anti-pattern 3: Inefficient data structure usage
func findItemBad(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func main() {
	items := make([]string, 1000)
	for i := range items {
		items[i] = fmt.Sprintf("item%d", i)
	}
	
	// Test the anti-patterns
	result1 := buildStringBad(items)
	result2 := processItemsBad(items)
	found := findItemBad(items, "item500")
	
	fmt.Printf("String length: %d\n", len(result1))
	fmt.Printf("Processed items: %d\n", len(result2))
	fmt.Printf("Found item: %v\n", found)
}
```

Create `exercise-01-02/improved.go`:
```go
package main

import (
	"fmt"
	"strings"
)

// Improvement 1: Use strings.Builder
func buildStringGood(items []string) string {
	var builder strings.Builder
	for _, item := range items {
		builder.WriteString(item)
		builder.WriteString(",")
	}
	return builder.String()
}

// Improvement 2: Pre-allocate and avoid unnecessary allocations
func processItemsGood(items []string) []string {
	result := make([]string, 0, len(items)) // Pre-allocate
	for _, item := range items {
		result = append(result, strings.ToUpper(item))
	}
	return result
}

// Improvement 3: Use map for O(1) lookups
func createLookupMap(items []string) map[string]bool {
	lookup := make(map[string]bool, len(items))
	for _, item := range items {
		lookup[item] = true
	}
	return lookup
}

func findItemGood(lookup map[string]bool, target string) bool {
	return lookup[target]
}
```

Create benchmark comparison in `exercise-01-02/benchmark_test.go`:
```go
package main

import (
	"fmt"
	"testing"
)

var testItems []string
var lookupMap map[string]bool

func init() {
	testItems = make([]string, 1000)
	for i := range testItems {
		testItems[i] = fmt.Sprintf("item%d", i)
	}
	lookupMap = createLookupMap(testItems)
}

func BenchmarkBuildStringBad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buildStringBad(testItems)
	}
}

func BenchmarkBuildStringGood(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buildStringGood(testItems)
	}
}

func BenchmarkProcessItemsBad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		processItemsBad(testItems)
	}
}

func BenchmarkProcessItemsGood(b *testing.B) {
	for i := 0; i < b.N; i++ {
		processItemsGood(testItems)
	}
}

func BenchmarkFindItemBad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		findItemBad(testItems, "item500")
	}
}

func BenchmarkFindItemGood(b *testing.B) {
	for i := 0; i < b.N; i++ {
		findItemGood(lookupMap, "item500")
	}
}
```

**Run the comparison:**
```bash
go test -bench=. -benchmem
```

### Exercise 01-03: Performance Measurement Tools

**Objective:** Learn to use basic performance measurement tools.

Create `exercise-01-03/measurement.go`:
```go
package main

import (
	"fmt"
	"runtime"
	"time"
)

func measureMemory() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	fmt.Printf("Allocated memory: %d KB\n", m.Alloc/1024)
	fmt.Printf("Total allocations: %d\n", m.TotalAlloc/1024)
	fmt.Printf("System memory: %d KB\n", m.Sys/1024)
	fmt.Printf("Number of GC cycles: %d\n", m.NumGC)
}

func allocateMemory() {
	// Allocate some memory
	data := make([][]int, 1000)
	for i := range data {
		data[i] = make([]int, 1000)
		for j := range data[i] {
			data[i][j] = i * j
		}
	}
	
	// Keep reference to prevent GC
	_ = data
}

func timeFunction(name string, fn func()) {
	start := time.Now()
	fn()
	duration := time.Since(start)
	fmt.Printf("%s took: %v\n", name, duration)
}

func main() {
	fmt.Println("=== Initial Memory State ===")
	measureMemory()
	
	fmt.Println("\n=== Allocating Memory ===")
	timeFunction("Memory allocation", allocateMemory)
	
	fmt.Println("\n=== After Allocation ===")
	measureMemory()
	
	fmt.Println("\n=== Forcing GC ===")
	runtime.GC()
	time.Sleep(100 * time.Millisecond) // Let GC complete
	
	fmt.Println("\n=== After GC ===")
	measureMemory()
}
```

**Run and analyze:**
```bash
go run measurement.go
```

## Chapter Summary

In this chapter, you learned:

✅ **Performance Mindset:** Always measure before optimizing  
✅ **Methodology:** Systematic approach to performance improvement  
✅ **Anti-Patterns:** Common mistakes that hurt performance  
✅ **Environment:** Essential tools for performance work  
✅ **Measurement:** Basic techniques for gathering performance data  

## Key Takeaways

1. **Measure First:** Never assume where bottlenecks are
2. **Start Simple:** Write clear code first, optimize later
3. **Use Data:** Let profiling guide your optimization decisions
4. **One Change:** Make one optimization at a time
5. **Verify:** Always confirm improvements with measurements

## Next Steps

Proceed to [Chapter 02: Go Runtime and Memory Model](../02-go-runtime-memory/) to understand the foundation of Go's performance characteristics.

---

**Checkpoint:** Can you explain the performance optimization methodology and identify common anti-patterns? If yes, you're ready for the next chapter!
package compiler

import (
	"fmt"
	"math"
)

// ShowDeadCodeAnalysis displays information about dead code elimination
func ShowDeadCodeAnalysis() {
	fmt.Println("Dead Code Elimination Analysis:")
	fmt.Println("- Compiler removes code that doesn't affect program output")
	fmt.Println("- Unused variables, functions, and computations are eliminated")
	fmt.Println("- Use -gcflags='-S' to see generated assembly")
	fmt.Println("- Use -gcflags='-d=ssa/check/on' for SSA debugging")
	fmt.Println("- Dead code elimination happens during SSA optimization")
}

// ProcessWithDeadCode - contains code that should be eliminated
func ProcessWithDeadCode(data []int) int {
	result := 0
	
	// Dead variable - never used
	unusedVar := 42
	_ = unusedVar // Prevent compiler warning
	
	// Dead computation - result not used
	deadComputation := 0
	for i := 0; i < 1000; i++ {
		deadComputation += i * i
	}
	_ = deadComputation
	
	// Actual useful computation
	for _, v := range data {
		result += v
	}
	
	// Dead branch - condition always false
	if false {
		result *= 1000
		fmt.Println("This will never execute")
	}
	
	// Dead function call - result not used
	_ = expensiveDeadFunction()
	
	return result
}

// ProcessWithoutDeadCode - optimized version
func ProcessWithoutDeadCode(data []int) int {
	result := 0
	
	// Only the essential computation
	for _, v := range data {
		result += v
	}
	
	return result
}

// expensiveDeadFunction - expensive computation that might be eliminated
func expensiveDeadFunction() int {
	sum := 0
	for i := 0; i < 10000; i++ {
		sum += int(math.Sqrt(float64(i)))
	}
	return sum
}

// DeadCodeInLoop - dead code inside loops
func DeadCodeInLoop(data []int) int {
	result := 0
	
	for i, v := range data {
		// Dead variable in loop
		deadLoopVar := i * 2
		_ = deadLoopVar
		
		// Dead conditional computation
		if false {
			v *= 100
		}
		
		// Useful computation
		result += v
		
		// Dead assignment
		deadAssignment := result * 2
		_ = deadAssignment
	}
	
	return result
}

// OptimizedLoop - same logic without dead code
func OptimizedLoop(data []int) int {
	result := 0
	
	for _, v := range data {
		result += v
	}
	
	return result
}

// ConditionalDeadCode - dead code in conditionals
func ConditionalDeadCode(data []int, flag bool) int {
	result := 0
	
	for _, v := range data {
		result += v
		
		// Dead code in conditional
		if flag {
			// This might be eliminated if flag is constant false
			deadInCondition := v * v
			_ = deadInCondition
		}
		
		// Always false condition
		if 1 > 2 {
			result *= 1000 // This will be eliminated
		}
		
		// Always true condition
		if 2 > 1 {
			result += 1 // Condition will be eliminated, code kept
		}
	}
	
	return result
}

// FunctionWithUnusedParameters - parameters might be eliminated
func FunctionWithUnusedParameters(used int, unused1 string, unused2 []int) int {
	// unused1 and unused2 are dead parameters
	_ = unused1
	_ = unused2
	
	return used * 2
}

// DeadStructFields - unused struct fields
type StructWithDeadFields struct {
	UsedField   int
	UnusedField string
	DeadField   []int
}

func ProcessStructWithDeadFields(s StructWithDeadFields) int {
	// Only UsedField is accessed, others are dead
	return s.UsedField * 2
}

// DeadInterfaceMethod - unused interface methods
type ProcessorInterface interface {
	Process(data []int) int
	UnusedMethod() string // This might be eliminated if never called
}

type ConcreteProcessor struct{}

func (p ConcreteProcessor) Process(data []int) int {
	sum := 0
	for _, v := range data {
		sum += v
	}
	return sum
}

func (p ConcreteProcessor) UnusedMethod() string {
	// This method is never called, so it's dead code
	return "unused"
}

// DeadGoroutine - goroutines that don't affect output
func ProcessWithDeadGoroutine(data []int) int {
	result := 0
	
	// Dead goroutine - doesn't affect result
	go func() {
		deadSum := 0
		for i := 0; i < 1000; i++ {
			deadSum += i
		}
		// Result is never used
	}()
	
	// Actual computation
	for _, v := range data {
		result += v
	}
	
	return result
}

// ConstantFolding - constants that can be folded
func ConstantFolding(data []int) int {
	result := 0
	
	// These constants will be folded at compile time
	const a = 10
	const b = 20
	const c = a + b // Will become 30 at compile time
	
	// Dead computation with constants
	deadConstComputation := a * b * c // Computed at compile time, then eliminated
	_ = deadConstComputation
	
	for _, v := range data {
		result += v * c // c is constant 30
	}
	
	return result
}

// UnreachableCode - code after return statements
func UnreachableCode(data []int) int {
	result := 0
	
	for _, v := range data {
		result += v
		if result > 1000 {
			return result
			// Dead code after return
			// fmt.Println("This will never execute")
			// result *= 2
		}
	}
	
	return result
	// More dead code after return
	// fmt.Println("This is also unreachable")
}

// DeadRecursion - recursive calls that don't contribute
func DeadRecursion(n int) int {
	if n <= 0 {
		return 1
	}
	
	// Dead recursive call - result not used
	_ = DeadRecursion(n - 1)
	
	// Only this matters
	return n * 2
}

// OptimizedRecursion - without dead recursion
func OptimizedRecursion(n int) int {
	if n <= 0 {
		return 1
	}
	return n * 2
}

// DeadMapOperations - unused map operations
func DeadMapOperations(data []int) int {
	result := 0
	
	// Dead map - created but never used meaningfully
	deadMap := make(map[int]int)
	for i, v := range data {
		deadMap[i] = v // Dead assignment
		result += v
	}
	_ = deadMap
	
	return result
}

// DeadSliceOperations - unused slice operations
func DeadSliceOperations(data []int) int {
	result := 0
	
	// Dead slice operations
	deadSlice := make([]int, 0, len(data))
	for _, v := range data {
		deadSlice = append(deadSlice, v*2) // Dead append
		result += v
	}
	_ = deadSlice
	
	return result
}

// PureFunction - function with no side effects
func PureFunction(x, y int) int {
	return x + y
}

// DeadPureFunctionCall - pure function calls that don't affect output
func DeadPureFunctionCall(data []int) int {
	result := 0
	
	for i, v := range data {
		// Dead pure function call
		_ = PureFunction(i, v)
		
		result += v
	}
	
	return result
}

// SideEffectFunction - function with side effects (can't be eliminated)
func SideEffectFunction(x int) int {
	fmt.Printf("Side effect: %d\n", x) // Side effect prevents elimination
	return x * 2
}

// DeadCodeWithSideEffects - side effects prevent elimination
func DeadCodeWithSideEffects(data []int) int {
	result := 0
	
	for _, v := range data {
		// This can't be eliminated due to side effect
		_ = SideEffectFunction(v)
		
		result += v
	}
	
	return result
}

// BenchmarkDeadCodeElimination - helper for benchmarking
func BenchmarkDeadCodeElimination(data []int, withDeadCode bool) int {
	if withDeadCode {
		return ProcessWithDeadCode(data)
	}
	return ProcessWithoutDeadCode(data)
}

// AnalyzeDeadCodeImpact - measure impact of dead code elimination
func AnalyzeDeadCodeImpact(size int) (withDead, withoutDead int) {
	data := make([]int, size)
	for i := range data {
		data[i] = i
	}
	
	withDead = ProcessWithDeadCode(data)
	withoutDead = ProcessWithoutDeadCode(data)
	
	return
}
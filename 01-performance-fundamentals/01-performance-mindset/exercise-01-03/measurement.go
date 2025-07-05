package main

import (
	"fmt"
	"runtime"
	"time"
)

func measureMemory() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

func printMemStats(label string, m runtime.MemStats) {
	fmt.Printf("=== %s ===\n", label)
	fmt.Printf("Allocated memory: %d KB\n", m.Alloc/1024)
	fmt.Printf("Total allocations: %d KB\n", m.TotalAlloc/1024)
	fmt.Printf("System memory: %d KB\n", m.Sys/1024)
	fmt.Printf("Number of GC cycles: %d\n", m.NumGC)
	fmt.Printf("GC CPU fraction: %.4f\n", m.GCCPUFraction)
	fmt.Printf("Heap objects: %d\n", m.HeapObjects)
	fmt.Printf("Stack in use: %d KB\n", m.StackInuse/1024)
	fmt.Println()
}

func allocateMemory() {
	fmt.Println("Allocating memory...")
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
	fmt.Printf("Allocated %d x %d matrix\n", len(data), len(data[0]))
}

func allocateStrings() {
	fmt.Println("Allocating strings...")
	// Allocate many small strings
	strings := make([]string, 100000)
	for i := range strings {
		strings[i] = fmt.Sprintf("String number %d with some extra text to make it longer", i)
	}
	
	// Keep reference
	_ = strings
	fmt.Printf("Allocated %d strings\n", len(strings))
}

func timeFunction(name string, fn func()) time.Duration {
	start := time.Now()
	fn()
	duration := time.Since(start)
	fmt.Printf("%s took: %v\n", name, duration)
	return duration
}

func cpuIntensiveTask() {
	fmt.Println("Running CPU intensive task...")
	// Simulate CPU-intensive work
	sum := 0
	for i := 0; i < 10000000; i++ {
		sum += i * i
	}
	fmt.Printf("CPU task result: %d\n", sum)
}

func measureGoroutines() {
	fmt.Printf("Number of goroutines: %d\n", runtime.NumGoroutine())
	fmt.Printf("Number of CPUs: %d\n", runtime.NumCPU())
	fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
}

func demonstrateGC() {
	fmt.Println("\n=== Garbage Collection Demo ===")
	
	// Force GC and measure
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	before := measureMemory()
	printMemStats("Before allocation", before)
	
	// Allocate memory that will become garbage
	func() {
		garbage := make([][]byte, 1000)
		for i := range garbage {
			garbage[i] = make([]byte, 1024)
		}
		// garbage goes out of scope here
	}()
	
	after := measureMemory()
	printMemStats("After allocation (before GC)", after)
	
	// Force GC
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	afterGC := measureMemory()
	printMemStats("After GC", afterGC)
	
	fmt.Printf("Memory freed by GC: %d KB\n", (after.Alloc-afterGC.Alloc)/1024)
}

func demonstrateStackVsHeap() {
	fmt.Println("\n=== Stack vs Heap Demo ===")
	
	// Stack allocation (small, known size)
	timeFunction("Stack allocation", func() {
		for i := 0; i < 1000000; i++ {
			var arr [10]int
			for j := range arr {
				arr[j] = j
			}
			_ = arr
		}
	})
	
	// Heap allocation (dynamic size)
	timeFunction("Heap allocation", func() {
		for i := 0; i < 1000000; i++ {
			arr := make([]int, 10)
			for j := range arr {
				arr[j] = j
			}
			_ = arr
		}
	})
}

func main() {
	fmt.Println("=== Go Performance Measurement Demo ===")
	fmt.Println()
	
	// Initial system info
	fmt.Println("=== System Information ===")
	measureGoroutines()
	fmt.Println()
	
	// Initial memory state
	initial := measureMemory()
	printMemStats("Initial Memory State", initial)
	
	// CPU intensive task
	fmt.Println("=== CPU Performance ===")
	timeFunction("CPU intensive task", cpuIntensiveTask)
	fmt.Println()
	
	// Memory allocation patterns
	fmt.Println("=== Memory Allocation Patterns ===")
	timeFunction("Integer matrix allocation", allocateMemory)
	after1 := measureMemory()
	printMemStats("After Matrix Allocation", after1)
	
	timeFunction("String allocation", allocateStrings)
	after2 := measureMemory()
	printMemStats("After String Allocation", after2)
	
	// Demonstrate GC
	demonstrateGC()
	
	// Stack vs Heap
	demonstrateStackVsHeap()
	
	// Final memory state
	final := measureMemory()
	printMemStats("Final Memory State", final)
	
	// Summary
	fmt.Println("=== Summary ===")
	fmt.Printf("Total memory allocated during execution: %d KB\n", (final.TotalAlloc-initial.TotalAlloc)/1024)
	fmt.Printf("Memory currently in use: %d KB\n", final.Alloc/1024)
	fmt.Printf("Total GC cycles: %d\n", final.NumGC-initial.NumGC)
	fmt.Printf("Average GC pause: %.2f ms\n", float64(final.PauseTotalNs-initial.PauseTotalNs)/float64(final.NumGC-initial.NumGC)/1000000)
}
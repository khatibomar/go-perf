package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"exercise-04-03/backoff"
	"exercise-04-03/cas"
	"exercise-04-03/list"
	"exercise-04-03/stack"
)

func main() {
	fmt.Println("Lock-Free Programming Exercise 04-03")
	fmt.Println("=====================================")

	// Demonstrate CAS patterns
	demonstrateCASPatterns()

	// Demonstrate stack implementations
	demonstrateStacks()

	// Demonstrate list implementations
	demonstrateLists()

	// Demonstrate backoff strategies
	demonstrateBackoffStrategies()

	// Performance comparison
	performanceComparison()
}

func demonstrateCASPatterns() {
	fmt.Println("\n1. CAS Patterns Demonstration")
	fmt.Println("------------------------------")

	// Simple CAS loop
	var counter int64
	casResult := cas.SimpleCASLoop(&counter, func(old int64) int64 {
		return old + 10
	})
	fmt.Printf("Simple CAS Loop: %v\n", casResult)

	// Optimistic update
	success := cas.OptimisticUpdate(&counter, func(old int64) (int64, bool) {
		return old + 5, true
	})
	fmt.Printf("Optimistic Update Success: %v, Counter: %d\n", success, atomic.LoadInt64(&counter))

	// Atomic increment
	result := cas.AtomicIncrement(&counter)
	fmt.Printf("Atomic Increment: %d\n", result)

	// Atomic max
	cas.AtomicMax(&counter, 20)
	fmt.Printf("Atomic Max (20): %d\n", atomic.LoadInt64(&counter))

	// Optimistic counter
	optCounter := cas.NewOptimisticCounter()
	optCounter.Add(100)
	fmt.Printf("Optimistic Counter: %d\n", optCounter.Get())

	// Versioned counter
	verCounter := cas.NewVersionedCounter()
	verCounter.Add(50)
	value, version := verCounter.Get()
	fmt.Printf("Versioned Counter: %d (version: %d)\n", value, version)

	// Helping counter
	helpCounter := cas.NewHelpingCounter()
	helpCounter.Add(25)
	fmt.Printf("Helping Counter: %d\n", helpCounter.Get())
}

func demonstrateStacks() {
	fmt.Println("\n2. Stack Implementations Demonstration")
	fmt.Println("---------------------------------------")

	stacks := []struct {
		name  string
		stack interface{}
	}{
		{"Lock-Free Stack", stack.NewLockFreeStack()},
		{"ABA-Safe Stack", stack.NewABASafeStack()},
		{"Versioned Stack", stack.NewVersionedStack()},
		{"Helping Stack", stack.NewHelpingStack()},
	}

	for _, s := range stacks {
		fmt.Printf("\n%s:\n", s.name)
		
		// Push elements
		for i := 1; i <= 5; i++ {
			switch st := s.stack.(type) {
			case *stack.LockFreeStack:
				st.Push(i * 10)
			case *stack.ABASafeStack:
				st.Push(i * 10)
			case *stack.VersionedStack:
				st.Push(i * 10)
			case *stack.HelpingStack:
				st.Push(i * 10)
			}
		}
		
		// Show size
		var size int
		switch st := s.stack.(type) {
		case *stack.LockFreeStack:
			size = st.Size()
		case *stack.ABASafeStack:
			size = st.Size()
		case *stack.VersionedStack:
			size = st.Size()
		case *stack.HelpingStack:
			size = st.Size()
		}
		fmt.Printf("  Size after pushes: %d\n", size)
		
		// Pop elements
		fmt.Print("  Popped values: ")
		for i := 0; i < 3; i++ {
			var value interface{}
			var ok bool
			switch st := s.stack.(type) {
			case *stack.LockFreeStack:
				value, ok = st.Pop()
			case *stack.ABASafeStack:
				value, ok = st.Pop()
			case *stack.VersionedStack:
				value, ok = st.Pop()
			case *stack.HelpingStack:
				value, ok = st.Pop()
			}
			if ok {
				fmt.Printf("%v ", value)
			}
		}
		fmt.Println()
	}
}

func demonstrateLists() {
	fmt.Println("\n3. List Implementations Demonstration")
	fmt.Println("--------------------------------------")

	lists := []struct {
		name string
		list interface{}
	}{
		{"Lock-Free List", list.NewLockFreeList()},
		{"Optimistic List", list.NewOptimisticList()},
		{"Lazy List", list.NewLazyList()},
		{"Harris List", list.NewHarrisLockFreeList()},
	}

	for _, l := range lists {
		fmt.Printf("\n%s:\n", l.name)
		
		// Insert elements
		keys := []int64{5, 2, 8, 1, 9}
		for _, key := range keys {
			switch lst := l.list.(type) {
			case *list.LockFreeList:
				lst.Insert(key, fmt.Sprintf("value_%d", key))
			case *list.OptimisticList:
				lst.Insert(key, fmt.Sprintf("value_%d", key))
			case *list.LazyList:
				lst.Insert(key, fmt.Sprintf("value_%d", key))
			case *list.HarrisLockFreeList:
				lst.Insert(key, fmt.Sprintf("value_%d", key))
			}
		}
		
		// Show size
		var size int
		switch lst := l.list.(type) {
		case *list.LockFreeList:
			size = lst.Size()
		case *list.OptimisticList:
			size = lst.Size()
		case *list.LazyList:
			size = lst.Size()
		case *list.HarrisLockFreeList:
			size = lst.Size()
		}
		fmt.Printf("  Size after inserts: %d\n", size)
		
		// Test contains
		fmt.Print("  Contains check (1,3,5,7): ")
		for _, key := range []int64{1, 3, 5, 7} {
			var contains bool
			switch lst := l.list.(type) {
			case *list.LockFreeList:
				contains = lst.Contains(key)
			case *list.OptimisticList:
				contains = lst.Contains(key)
			case *list.LazyList:
				contains = lst.Contains(key)
			case *list.HarrisLockFreeList:
				contains = lst.Contains(key)
			}
			fmt.Printf("%d:%v ", key, contains)
		}
		fmt.Println()
		
		// Delete some elements
		for _, key := range []int64{2, 8} {
			switch lst := l.list.(type) {
			case *list.LockFreeList:
				lst.Delete(key)
			case *list.OptimisticList:
				lst.Delete(key)
			case *list.LazyList:
				lst.Delete(key)
			case *list.HarrisLockFreeList:
				lst.Delete(key)
			}
		}
		
		// Show final size
		switch lst := l.list.(type) {
		case *list.LockFreeList:
			size = lst.Size()
		case *list.OptimisticList:
			size = lst.Size()
		case *list.LazyList:
			size = lst.Size()
		case *list.HarrisLockFreeList:
			size = lst.Size()
		}
		fmt.Printf("  Size after deletes: %d\n", size)
	}
}

func demonstrateBackoffStrategies() {
	fmt.Println("\n4. Backoff Strategies Demonstration")
	fmt.Println("------------------------------------")

	// Exponential backoff
	fmt.Println("\nExponential Backoff:")
	expBackoff := backoff.NewExponentialBackoff(
		time.Microsecond,
		time.Millisecond,
		2.0,
		false,
	)
	for i := 0; i < 5; i++ {
		fmt.Printf("  Attempt %d: delay=%v\n", expBackoff.GetAttempts(), expBackoff.GetCurrentDelay())
		expBackoff.Backoff()
	}

	// Adaptive backoff
	fmt.Println("\nAdaptive Backoff:")
	adaptiveBackoff := backoff.NewAdaptiveBackoff(
		time.Microsecond,
		time.Millisecond,
	)
	for i := 0; i < 3; i++ {
		adaptiveBackoff.RecordFailure()
		adaptiveBackoff.Backoff()
	}
	adaptiveBackoff.RecordSuccess()
	stats := adaptiveBackoff.GetStats()
	fmt.Printf("  Stats: Success=%d, Failure=%d, SuccessRate=%.2f\n",
		stats.SuccessCount, stats.FailureCount, stats.SuccessRate)

	// Yield strategies
	fmt.Println("\nYield Strategies:")
	strategies := []backoff.YieldStrategy{
		backoff.YieldGosched,
		backoff.YieldSleep,
		backoff.YieldSpinWait,
		backoff.YieldHybrid,
	}
	for _, strategy := range strategies {
		ym := backoff.NewYieldManager(strategy)
		fmt.Printf("  %s: ", strategy.String())
		start := time.Now()
		ym.Yield()
		duration := time.Since(start)
		fmt.Printf("took %v\n", duration)
	}

	// Contention manager
	fmt.Println("\nContention Manager:")
	cm := backoff.NewContentionManager(time.Microsecond, 2.0)
	for i := 0; i < 3; i++ {
		cm.RecordFailure()
		cm.Backoff()
	}
	cm.RecordSuccess()
	cmStats := cm.GetStats()
	fmt.Printf("  Contention Stats: Success=%d, Failure=%d\n",
		cmStats.SuccessCount, cmStats.FailureCount)
}

func performanceComparison() {
	fmt.Println("\n5. Performance Comparison")
	fmt.Println("--------------------------")

	// Compare CAS patterns under contention
	compareCASPatterns()

	// Compare stack implementations
	compareStackImplementations()

	// Compare list implementations
	compareListImplementations()
}

func compareCASPatterns() {
	fmt.Println("\nCAS Patterns Performance:")
	goroutines := runtime.NumCPU()
	operations := 10000

	tests := []struct {
		name string
		fn   func()
	}{
		{"Simple CAS", func() {
			var counter int64
			var wg sync.WaitGroup
			for i := 0; i < goroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < operations/goroutines; j++ {
						cas.AtomicIncrement(&counter)
					}
				}()
			}
			wg.Wait()
		}},
		{"Optimistic Counter", func() {
			counter := cas.NewOptimisticCounter()
			var wg sync.WaitGroup
			for i := 0; i < goroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < operations/goroutines; j++ {
						counter.Increment()
					}
				}()
			}
			wg.Wait()
		}},
		{"Versioned Counter", func() {
			counter := cas.NewVersionedCounter()
			var wg sync.WaitGroup
			for i := 0; i < goroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < operations/goroutines; j++ {
						counter.Increment()
					}
				}()
			}
			wg.Wait()
		}},
	}

	for _, test := range tests {
		start := time.Now()
		test.fn()
		duration := time.Since(start)
		fmt.Printf("  %s: %v\n", test.name, duration)
	}
}

func compareStackImplementations() {
	fmt.Println("\nStack Implementations Performance:")
	goroutines := runtime.NumCPU()
	operations := 1000

	stacks := []struct {
		name  string
		stack interface{}
	}{
		{"Lock-Free", stack.NewLockFreeStack()},
		{"ABA-Safe", stack.NewABASafeStack()},
		{"Versioned", stack.NewVersionedStack()},
		{"Helping", stack.NewHelpingStack()},
	}

	for _, s := range stacks {
		start := time.Now()
		var wg sync.WaitGroup
		
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < operations/goroutines; j++ {
					switch st := s.stack.(type) {
					case *stack.LockFreeStack:
						st.Push(j)
						st.Pop()
					case *stack.ABASafeStack:
						st.Push(j)
						st.Pop()
					case *stack.VersionedStack:
						st.Push(j)
						st.Pop()
					case *stack.HelpingStack:
						st.Push(j)
						st.Pop()
					}
				}
			}()
		}
		
		wg.Wait()
		duration := time.Since(start)
		fmt.Printf("  %s Stack: %v\n", s.name, duration)
	}
}

func compareListImplementations() {
	fmt.Println("\nList Implementations Performance:")
	goroutines := runtime.NumCPU()
	operations := 1000

	lists := []struct {
		name string
		list interface{}
	}{
		{"Lock-Free", list.NewLockFreeList()},
		{"Optimistic", list.NewOptimisticList()},
		{"Lazy", list.NewLazyList()},
		{"Harris", list.NewHarrisLockFreeList()},
	}

	for _, l := range lists {
		start := time.Now()
		var wg sync.WaitGroup
		
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < operations/goroutines; j++ {
					key := int64(id*operations + j)
					switch lst := l.list.(type) {
					case *list.LockFreeList:
						lst.Insert(key, key)
						lst.Contains(key)
						lst.Delete(key)
					case *list.OptimisticList:
						lst.Insert(key, key)
						lst.Contains(key)
						lst.Delete(key)
					case *list.LazyList:
						lst.Insert(key, key)
						lst.Contains(key)
						lst.Delete(key)
					case *list.HarrisLockFreeList:
						lst.Insert(key, key)
						lst.Contains(key)
						lst.Delete(key)
					}
				}
			}(i)
		}
		
		wg.Wait()
		duration := time.Since(start)
		fmt.Printf("  %s List: %v\n", l.name, duration)
	}

	fmt.Println("\nDemonstration completed successfully!")
	fmt.Printf("All implementations are working with %d CPU cores.\n", runtime.NumCPU())
}
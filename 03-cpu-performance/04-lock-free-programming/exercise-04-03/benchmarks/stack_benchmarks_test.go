package benchmarks

import (
	"fmt"
	"runtime"
	"sync"
	"testing"

	"exercise-04-03/stack"
)

// BenchmarkLockFreeStack benchmarks the basic lock-free stack
func BenchmarkLockFreeStack(b *testing.B) {
	s := stack.NewLockFreeStack()
	
	b.Run("Push", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				s.Push(42)
			}
		})
	})
	
	b.Run("Pop", func(b *testing.B) {
		// Pre-populate stack
		for i := 0; i < b.N*2; i++ {
			s.Push(i)
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				s.Pop()
			}
		})
	})
	
	b.Run("Mixed", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				s.Push(42)
				s.Pop()
			}
		})
	})
}

// BenchmarkABASafeStack benchmarks the ABA-safe stack
func BenchmarkABASafeStack(b *testing.B) {
	s := stack.NewABASafeStack()
	
	b.Run("Push", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				s.Push(42)
			}
		})
	})
	
	b.Run("Pop", func(b *testing.B) {
		// Pre-populate stack
		for i := 0; i < b.N*2; i++ {
			s.Push(i)
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				s.Pop()
			}
		})
	})
	
	b.Run("Mixed", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				s.Push(42)
				s.Pop()
			}
		})
	})
}

// BenchmarkVersionedStack benchmarks the versioned stack
func BenchmarkVersionedStack(b *testing.B) {
	s := stack.NewVersionedStack()
	
	b.Run("Push", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				s.Push(42)
			}
		})
	})
	
	b.Run("Pop", func(b *testing.B) {
		// Pre-populate stack
		for i := 0; i < b.N*2; i++ {
			s.Push(i)
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				s.Pop()
			}
		})
	})
	
	b.Run("Mixed", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				s.Push(42)
				s.Pop()
			}
		})
	})
}

// BenchmarkHelpingStack benchmarks the helping stack
func BenchmarkHelpingStack(b *testing.B) {
	s := stack.NewHelpingStack()
	
	b.Run("Push", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				s.Push(42)
			}
		})
	})
	
	b.Run("Pop", func(b *testing.B) {
		// Pre-populate stack
		for i := 0; i < b.N*2; i++ {
			s.Push(i)
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				s.Pop()
			}
		})
	})
	
	b.Run("Mixed", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				s.Push(42)
				s.Pop()
			}
		})
	})
}

// BenchmarkStackComparison compares all stack implementations
func BenchmarkStackComparison(b *testing.B) {
	operations := []struct {
		name string
		fn   func(interface{})
	}{
		{"Push", func(s interface{}) {
			switch stack := s.(type) {
			case *stack.LockFreeStack:
				stack.Push(42)
			case *stack.ABASafeStack:
				stack.Push(42)
			case *stack.VersionedStack:
				stack.Push(42)
			case *stack.HelpingStack:
				stack.Push(42)
			}
		}},
		{"Pop", func(s interface{}) {
			switch stack := s.(type) {
			case *stack.LockFreeStack:
				stack.Pop()
			case *stack.ABASafeStack:
				stack.Pop()
			case *stack.VersionedStack:
				stack.Pop()
			case *stack.HelpingStack:
				stack.Pop()
			}
		}},
	}
	
	stacks := []struct {
		name  string
		stack interface{}
	}{
		{"LockFree", stack.NewLockFreeStack()},
		{"ABASafe", stack.NewABASafeStack()},
		{"Versioned", stack.NewVersionedStack()},
		{"Helping", stack.NewHelpingStack()},
	}
	
	for _, op := range operations {
		for _, s := range stacks {
			b.Run(fmt.Sprintf("%s-%s", op.name, s.name), func(b *testing.B) {
				// Pre-populate for pop operations
				if op.name == "Pop" {
					for i := 0; i < b.N*2; i++ {
						switch stack := s.stack.(type) {
						case *stack.LockFreeStack:
							stack.Push(i)
						case *stack.ABASafeStack:
							stack.Push(i)
						case *stack.VersionedStack:
							stack.Push(i)
						case *stack.HelpingStack:
							stack.Push(i)
						}
					}
				}
				
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						op.fn(s.stack)
					}
				})
			})
		}
	}
}

// BenchmarkStackContention benchmarks stack under different contention levels
func BenchmarkStackContention(b *testing.B) {
	contentionLevels := []int{1, 2, 4, 8, 16}
	
	for _, level := range contentionLevels {
		b.Run(fmt.Sprintf("Contention-%d", level), func(b *testing.B) {
			s := stack.NewLockFreeStack()
			var wg sync.WaitGroup
			
			b.ResetTimer()
			
			for i := 0; i < level; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < b.N/level; j++ {
						s.Push(j)
						s.Pop()
					}
				}()
			}
			
			wg.Wait()
		})
	}
}

// BenchmarkStackMemoryUsage benchmarks memory usage patterns
func BenchmarkStackMemoryUsage(b *testing.B) {
	s := stack.NewLockFreeStack()
	
	b.Run("GrowShrink", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Grow phase
			for j := 0; j < 1000; j++ {
				s.Push(j)
			}
			// Shrink phase
			for j := 0; j < 1000; j++ {
				s.Pop()
			}
			runtime.GC()
		}
	})
}
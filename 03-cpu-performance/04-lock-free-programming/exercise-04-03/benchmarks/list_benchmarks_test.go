package benchmarks

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"exercise-04-03/list"
)

// BenchmarkLockFreeList benchmarks the basic lock-free list
func BenchmarkLockFreeList(b *testing.B) {
	l := list.NewLockFreeList()
	
	b.Run("Insert", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				l.Insert(int64(i), i*10)
				i++
			}
		})
	})
	
	b.Run("Contains", func(b *testing.B) {
		// Pre-populate list
		for i := 0; i < 1000; i++ {
			l.Insert(int64(i), i*10)
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				key := rand.Intn(1000)
				l.Contains(int64(key))
			}
		})
	})
	
	b.Run("Delete", func(b *testing.B) {
		// Pre-populate list
		for i := 0; i < b.N*2; i++ {
			l.Insert(int64(i), i*10)
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				l.Delete(int64(i))
				i++
			}
		})
	})
	
	b.Run("Mixed", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				switch i % 3 {
				case 0:
					l.Insert(int64(i), i*10)
				case 1:
					l.Contains(int64(i))
				case 2:
					l.Delete(int64(i))
				}
				i++
			}
		})
	})
}

// BenchmarkOptimisticList benchmarks the optimistic list
func BenchmarkOptimisticList(b *testing.B) {
	l := list.NewOptimisticList()
	
	b.Run("Insert", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				l.Insert(int64(i), i*10)
				i++
			}
		})
	})
	
	b.Run("Contains", func(b *testing.B) {
		// Pre-populate list
		for i := 0; i < 1000; i++ {
			l.Insert(int64(i), i*10)
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				key := rand.Intn(1000)
				l.Contains(int64(key))
			}
		})
	})
	
	b.Run("Delete", func(b *testing.B) {
		// Pre-populate list
		for i := 0; i < b.N*2; i++ {
			l.Insert(int64(i), i*10)
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				l.Delete(int64(i))
				i++
			}
		})
	})
}

// BenchmarkLazyList benchmarks the lazy list
func BenchmarkLazyList(b *testing.B) {
	l := list.NewLazyList()
	
	b.Run("Insert", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				l.Insert(int64(i), i*10)
				i++
			}
		})
	})
	
	b.Run("Contains", func(b *testing.B) {
		// Pre-populate list
		for i := 0; i < 1000; i++ {
			l.Insert(int64(i), i*10)
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				key := rand.Intn(1000)
				l.Contains(int64(key))
			}
		})
	})
	
	b.Run("Delete", func(b *testing.B) {
		// Pre-populate list
		for i := 0; i < b.N*2; i++ {
			l.Insert(int64(i), i*10)
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				l.Delete(int64(i))
				i++
			}
		})
	})
}

// BenchmarkHarrisLockFreeList benchmarks Harris's lock-free list
func BenchmarkHarrisLockFreeList(b *testing.B) {
	l := list.NewHarrisLockFreeList()
	
	b.Run("Insert", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				l.Insert(int64(i), i*10)
				i++
			}
		})
	})
	
	b.Run("Contains", func(b *testing.B) {
		// Pre-populate list
		for i := 0; i < 1000; i++ {
			l.Insert(int64(i), i*10)
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				key := rand.Intn(1000)
				l.Contains(int64(key))
			}
		})
	})
	
	b.Run("Delete", func(b *testing.B) {
		// Pre-populate list
		for i := 0; i < b.N*2; i++ {
			l.Insert(int64(i), i*10)
		}
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				l.Delete(int64(i))
				i++
			}
		})
	})
}

// BenchmarkListComparison compares all list implementations
func BenchmarkListComparison(b *testing.B) {
	operations := []struct {
		name string
		fn   func(interface{}, int)
	}{
		{"Insert", func(l interface{}, key int) {
			switch list := l.(type) {
			case *list.LockFreeList:
				list.Insert(int64(key), key*10)
			case *list.OptimisticList:
				list.Insert(int64(key), key*10)
			case *list.LazyList:
				list.Insert(int64(key), key*10)
			case *list.HarrisLockFreeList:
				list.Insert(int64(key), key*10)
			}
		}},
		{"Contains", func(l interface{}, key int) {
			switch list := l.(type) {
			case *list.LockFreeList:
				list.Contains(int64(key))
			case *list.OptimisticList:
				list.Contains(int64(key))
			case *list.LazyList:
				list.Contains(int64(key))
			case *list.HarrisLockFreeList:
				list.Contains(int64(key))
			}
		}},
		{"Delete", func(l interface{}, key int) {
			switch list := l.(type) {
			case *list.LockFreeList:
				list.Delete(int64(key))
			case *list.OptimisticList:
				list.Delete(int64(key))
			case *list.LazyList:
				list.Delete(int64(key))
			case *list.HarrisLockFreeList:
				list.Delete(int64(key))
			}
		}},
	}
	
	lists := []struct {
		name string
		list interface{}
	}{
		{"LockFree", list.NewLockFreeList()},
		{"Optimistic", list.NewOptimisticList()},
		{"Lazy", list.NewLazyList()},
		{"Harris", list.NewHarrisLockFreeList()},
	}
	
	for _, op := range operations {
		for _, l := range lists {
			b.Run(fmt.Sprintf("%s-%s", op.name, l.name), func(b *testing.B) {
				// Pre-populate for contains and delete operations
				if op.name == "Contains" || op.name == "Delete" {
					for i := 0; i < 1000; i++ {
						switch list := l.list.(type) {
						case *list.LockFreeList:
						list.Insert(int64(i), i*10)
					case *list.OptimisticList:
						list.Insert(int64(i), i*10)
					case *list.LazyList:
						list.Insert(int64(i), i*10)
					case *list.HarrisLockFreeList:
						list.Insert(int64(i), i*10)
						}
					}
				}
				
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						key := i
						if op.name == "Contains" {
							key = rand.Intn(1000)
						}
						op.fn(l.list, key)
						i++
					}
				})
			})
		}
	}
}

// BenchmarkListContention benchmarks list under different contention levels
func BenchmarkListContention(b *testing.B) {
	contentionLevels := []int{1, 2, 4, 8, 16}
	
	for _, level := range contentionLevels {
		b.Run(fmt.Sprintf("Contention-%d", level), func(b *testing.B) {
			l := list.NewLockFreeList()
			var wg sync.WaitGroup
			
			b.ResetTimer()
			
			for i := 0; i < level; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < b.N/level; j++ {
					key := id*1000 + j
					l.Insert(int64(key), key*10)
					l.Contains(int64(key))
					l.Delete(int64(key))
					}
				}(i)
			}
			
			wg.Wait()
		})
	}
}

// BenchmarkListTraversal benchmarks list traversal patterns
func BenchmarkListTraversal(b *testing.B) {
	l := list.NewLockFreeList()
	
	// Pre-populate list
	for i := 0; i < 10000; i++ {
		l.Insert(int64(i), i*10)
	}
	
	b.Run("Sequential", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j := 0; j < 100; j++ {
				l.Contains(int64(j))
			}
		}
	})
	
	b.Run("Random", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j := 0; j < 100; j++ {
				key := rand.Intn(10000)
				l.Contains(int64(key))
			}
		}
	})
	
	b.Run("Locality", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			base := rand.Intn(9900)
			for j := 0; j < 100; j++ {
				key := base + j
				l.Contains(int64(key))
			}
		}
	})
}

// BenchmarkListMemoryUsage benchmarks memory usage patterns
func BenchmarkListMemoryUsage(b *testing.B) {
	l := list.NewLockFreeList()
	
	b.Run("GrowShrink", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Grow phase
			for j := 0; j < 1000; j++ {
				l.Insert(int64(j), j*10)
			}
			// Shrink phase
			for j := 0; j < 1000; j++ {
				l.Delete(int64(j))
			}
			runtime.GC()
		}
	})
}

// BenchmarkListWorkloads benchmarks different workload patterns
func BenchmarkListWorkloads(b *testing.B) {
	workloads := []struct {
		name   string
		insert int
		delete int
		lookup int
	}{
		{"ReadHeavy", 10, 10, 80},
		{"WriteHeavy", 40, 40, 20},
		{"Balanced", 33, 33, 34},
		{"InsertOnly", 100, 0, 0},
		{"LookupOnly", 0, 0, 100},
	}
	
	for _, workload := range workloads {
		b.Run(workload.name, func(b *testing.B) {
			l := list.NewLockFreeList()
			
			// Pre-populate for delete/lookup operations
		for i := 0; i < 1000; i++ {
			l.Insert(int64(i), i*10)
		}
			
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 1000
				for pb.Next() {
					rand.Seed(time.Now().UnixNano())
					op := rand.Intn(100)
					
					if op < workload.insert {
					l.Insert(int64(i), i*10)
					i++
				} else if op < workload.insert+workload.delete {
					key := rand.Intn(i)
					l.Delete(int64(key))
				} else {
					key := rand.Intn(i)
					l.Contains(int64(key))
					}
				}
			})
		})
	}
}
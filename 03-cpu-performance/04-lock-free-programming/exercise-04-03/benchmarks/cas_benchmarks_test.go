package benchmarks

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"exercise-04-03/cas"
	"exercise-04-03/backoff"
)

// BenchmarkSimpleCAS benchmarks basic CAS operations
func BenchmarkSimpleCAS(b *testing.B) {
	var counter int64
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for {
				old := atomic.LoadInt64(&counter)
				if atomic.CompareAndSwapInt64(&counter, old, old+1) {
					break
				}
			}
		}
	})
}

// BenchmarkCASWithBackoff benchmarks CAS with exponential backoff
func BenchmarkCASWithBackoff(b *testing.B) {
	var counter int64
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		backoffMgr := backoff.NewExponentialBackoff(
			time.Nanosecond,
			time.Microsecond,
			2.0,
			true,
		)
		
		for pb.Next() {
			for {
				old := atomic.LoadInt64(&counter)
				if atomic.CompareAndSwapInt64(&counter, old, old+1) {
					backoffMgr.Reset()
					break
				}
				backoffMgr.Backoff()
			}
		}
	})
}

// BenchmarkOptimisticUpdate benchmarks optimistic update pattern
func BenchmarkOptimisticUpdate(b *testing.B) {
	var counter int64
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cas.OptimisticUpdate(&counter, func(old int64) (int64, bool) {
				return old + 1, true
			})
		}
	})
}

// BenchmarkCASLoop benchmarks simple CAS loop
func BenchmarkCASLoop(b *testing.B) {
	var counter int64
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cas.SimpleCASLoop(&counter, func(old int64) int64 {
				return old + 1
			})
		}
	})
}

// BenchmarkVersionedCAS benchmarks versioned CAS operations
func BenchmarkVersionedCAS(b *testing.B) {
	counter := cas.NewVersionedCounter()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			counter.Increment()
		}
	})
}

// BenchmarkHelpingCounter benchmarks helping mechanism
func BenchmarkHelpingCounter(b *testing.B) {
	counter := cas.NewHelpingCounter()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			counter.Increment()
		}
	})
}

// BenchmarkContentionManager benchmarks contention management
func BenchmarkContentionManager(b *testing.B) {
	var counter int64
	cm := backoff.NewContentionManager(time.Millisecond, 2.0)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for {
				old := atomic.LoadInt64(&counter)
				if atomic.CompareAndSwapInt64(&counter, old, old+1) {
					cm.RecordSuccess()
					break
				}
				cm.RecordFailure()
				cm.Backoff()
			}
		}
	})
}

// BenchmarkAdaptiveBackoff benchmarks adaptive backoff
func BenchmarkAdaptiveBackoff(b *testing.B) {
	var counter int64
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		backoffMgr := backoff.NewAdaptiveBackoff(
			time.Nanosecond,
			time.Microsecond,
		)
		
		for pb.Next() {
			for {
				old := atomic.LoadInt64(&counter)
				if atomic.CompareAndSwapInt64(&counter, old, old+1) {
					backoffMgr.RecordSuccess()
					break
				}
				backoffMgr.RecordFailure()
				backoffMgr.Backoff()
			}
		}
	})
}

// BenchmarkYieldStrategies benchmarks different yield strategies
func BenchmarkYieldStrategies(b *testing.B) {
	strategies := []backoff.YieldStrategy{
		backoff.YieldGosched,
		backoff.YieldSleep,
		backoff.YieldSpinWait,
		backoff.YieldHybrid,
	}
	
	for _, strategy := range strategies {
		b.Run(strategy.String(), func(b *testing.B) {
			var counter int64
			ym := backoff.NewYieldManager(strategy)
			
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					for {
						old := atomic.LoadInt64(&counter)
						if atomic.CompareAndSwapInt64(&counter, old, old+1) {
							break
						}
						ym.Yield()
					}
				}
			})
		})
	}
}

// BenchmarkCASContention benchmarks CAS under different contention levels
func BenchmarkCASContention(b *testing.B) {
	contentionLevels := []int{1, 2, 4, 8, 16, 32}
	
	for _, level := range contentionLevels {
		b.Run(fmt.Sprintf("Contention-%d", level), func(b *testing.B) {
			var counter int64
			var wg sync.WaitGroup
			
			b.ResetTimer()
			
			for i := 0; i < level; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < b.N/level; j++ {
						for {
							old := atomic.LoadInt64(&counter)
							if atomic.CompareAndSwapInt64(&counter, old, old+1) {
								break
							}
							runtime.Gosched()
						}
					}
				}()
			}
			
			wg.Wait()
		})
	}
}

// BenchmarkCASRetryLimit benchmarks CAS with retry limits
func BenchmarkCASRetryLimit(b *testing.B) {
	retryLimits := []int{1, 5, 10, 50, 100}
	
	for _, limit := range retryLimits {
		b.Run(fmt.Sprintf("Retry-%d", limit), func(b *testing.B) {
			var counter int64
			
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					old := atomic.LoadInt64(&counter)
					cas.CASWithRetryLimit(&counter, old, old+1, limit)
				}
			})
		})
	}
}
package benchmarks

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"exercise-02-01/algorithms"
)

// Fibonacci benchmarks

// BenchmarkFibonacciNaive benchmarks naive recursive Fibonacci
func BenchmarkFibonacciNaive(b *testing.B) {
	sizes := []int{10, 15, 20, 25} // Smaller sizes due to exponential complexity
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("N%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				algorithms.FibonacciNaive(size)
			}
		})
	}
}

// BenchmarkFibonacciMemoized benchmarks memoized Fibonacci
func BenchmarkFibonacciMemoized(b *testing.B) {
	sizes := []int{10, 50, 100, 500, 1000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("N%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				algorithms.FibonacciMemoized(size)
			}
		})
	}
}

// BenchmarkFibonacciBottomUp benchmarks bottom-up Fibonacci
func BenchmarkFibonacciBottomUp(b *testing.B) {
	sizes := []int{10, 50, 100, 500, 1000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("N%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				algorithms.FibonacciBottomUp(size)
			}
		})
	}
}

// BenchmarkFibonacciOptimized benchmarks space-optimized Fibonacci
func BenchmarkFibonacciOptimized(b *testing.B) {
	sizes := []int{10, 50, 100, 500, 1000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("N%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				algorithms.FibonacciOptimized(size)
			}
		})
	}
}

// Longest Common Subsequence benchmarks

// BenchmarkLongestCommonSubsequence benchmarks LCS algorithm
func BenchmarkLongestCommonSubsequence(b *testing.B) {
	sizes := []int{10, 50, 100, 200}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			str1 := generateRandomString(size)
			str2 := generateRandomString(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.LongestCommonSubsequence(str1, str2)
			}
		})
	}
}

// BenchmarkLongestCommonSubsequenceOptimized benchmarks space-optimized LCS
func BenchmarkLongestCommonSubsequenceOptimized(b *testing.B) {
	sizes := []int{10, 50, 100, 200}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			str1 := generateRandomString(size)
			str2 := generateRandomString(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.LongestCommonSubsequenceOptimized(str1, str2)
			}
		})
	}
}

// Knapsack Problem benchmarks

// BenchmarkKnapsackProblem benchmarks 0/1 knapsack problem
func BenchmarkKnapsackProblem(b *testing.B) {
	sizes := []int{10, 50, 100, 200}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			weights, values := generateKnapsackData(size)
			capacity := size * 5 // Reasonable capacity
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.KnapsackProblem(weights, values, capacity)
			}
		})
	}
}

// BenchmarkKnapsackOptimized benchmarks space-optimized knapsack
func BenchmarkKnapsackOptimized(b *testing.B) {
	sizes := []int{10, 50, 100, 200}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			weights, values := generateKnapsackData(size)
			capacity := size * 5 // Reasonable capacity
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.KnapsackOptimized(weights, values, capacity)
			}
		})
	}
}

// Longest Increasing Subsequence benchmarks

// BenchmarkLongestIncreasingSubsequence benchmarks LIS O(n²) algorithm
func BenchmarkLongestIncreasingSubsequence(b *testing.B) {
	sizes := []int{100, 500, 1000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateBenchmarkData(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.LongestIncreasingSubsequence(data)
			}
		})
	}
}

// BenchmarkLongestIncreasingSubsequenceOptimized benchmarks LIS O(n log n) algorithm
func BenchmarkLongestIncreasingSubsequenceOptimized(b *testing.B) {
	sizes := []int{100, 500, 1000, 5000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateBenchmarkData(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.LongestIncreasingSubsequenceOptimized(data)
			}
		})
	}
}

// Edit Distance benchmarks

// BenchmarkEditDistance benchmarks edit distance algorithm
func BenchmarkEditDistance(b *testing.B) {
	sizes := []int{10, 50, 100, 200}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			str1 := generateRandomString(size)
			str2 := generateRandomString(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.EditDistance(str1, str2)
			}
		})
	}
}

// BenchmarkEditDistanceOptimized benchmarks space-optimized edit distance
func BenchmarkEditDistanceOptimized(b *testing.B) {
	sizes := []int{10, 50, 100, 200}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			str1 := generateRandomString(size)
			str2 := generateRandomString(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.EditDistanceOptimized(str1, str2)
			}
		})
	}
}

// Coin Change benchmarks

// BenchmarkCoinChange benchmarks coin change problem
func BenchmarkCoinChange(b *testing.B) {
	coins := []int{1, 5, 10, 25}
	amounts := []int{50, 100, 500, 1000}
	
	for _, amount := range amounts {
		b.Run(fmt.Sprintf("Amount%d", amount), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				algorithms.CoinChange(coins, amount)
			}
		})
	}
}

// BenchmarkCoinChangeWays benchmarks coin change ways counting
func BenchmarkCoinChangeWays(b *testing.B) {
	coins := []int{1, 5, 10, 25}
	amounts := []int{50, 100, 500, 1000}
	
	for _, amount := range amounts {
		b.Run(fmt.Sprintf("Amount%d", amount), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				algorithms.CoinChangeWays(coins, amount)
			}
		})
	}
}

// Maximum Subarray benchmarks

// BenchmarkMaxSubarraySum benchmarks Kadane's algorithm
func BenchmarkMaxSubarraySum(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateMixedData(size) // Mix of positive and negative numbers
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.MaxSubarraySum(data)
			}
		})
	}
}

// BenchmarkMaxSubarrayProduct benchmarks maximum subarray product
func BenchmarkMaxSubarrayProduct(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateMixedData(size) // Mix of positive and negative numbers
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.MaxSubarrayProduct(data)
			}
		})
	}
}

// House Robber benchmarks

// BenchmarkHouseRobber benchmarks house robber problem
func BenchmarkHouseRobber(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			houses := generateBenchmarkData(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.HouseRobber(houses)
			}
		})
	}
}

// Climbing Stairs benchmarks

// BenchmarkClimbingStairs benchmarks climbing stairs problem
func BenchmarkClimbingStairs(b *testing.B) {
	steps := []int{10, 50, 100, 500, 1000}
	
	for _, step := range steps {
		b.Run(fmt.Sprintf("Steps%d", step), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				algorithms.ClimbingStairs(step)
			}
		})
	}
}

// Palindrome Partitioning benchmarks

// BenchmarkPalindromePartitioning benchmarks palindrome partitioning
func BenchmarkPalindromePartitioning(b *testing.B) {
	sizes := []int{10, 20, 50, 100}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			str := generateRandomString(size)
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				algorithms.PalindromePartitioning(str)
			}
		})
	}
}

// Comparative benchmarks

// BenchmarkFibonacciComparison compares different Fibonacci implementations
func BenchmarkFibonacciComparison(b *testing.B) {
	n := 30
	
	b.Run("Naive", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.FibonacciNaive(n)
		}
	})
	
	b.Run("Memoized", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.FibonacciMemoized(n)
		}
	})
	
	b.Run("BottomUp", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.FibonacciBottomUp(n)
		}
	})
	
	b.Run("Optimized", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.FibonacciOptimized(n)
		}
	})
}

// BenchmarkLCSComparison compares LCS implementations
func BenchmarkLCSComparison(b *testing.B) {
	str1 := generateRandomString(100)
	str2 := generateRandomString(100)
	
	b.Run("Standard", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.LongestCommonSubsequence(str1, str2)
		}
	})
	
	b.Run("Optimized", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			algorithms.LongestCommonSubsequenceOptimized(str1, str2)
		}
	})
}

// Helper functions

// generateRandomString creates a random string of specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// generateKnapsackData creates weights and values for knapsack problem
func generateKnapsackData(size int) ([]int, []int) {
	rand.Seed(time.Now().UnixNano())
	weights := make([]int, size)
	values := make([]int, size)
	
	for i := 0; i < size; i++ {
		weights[i] = rand.Intn(20) + 1 // Weight between 1 and 20
		values[i] = rand.Intn(100) + 1 // Value between 1 and 100
	}
	
	return weights, values
}

// generateMixedData creates data with positive and negative numbers
func generateMixedData(size int) []int {
	rand.Seed(time.Now().UnixNano())
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Intn(200) - 100 // Numbers between -100 and 99
	}
	return data
}
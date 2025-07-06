package algorithms

import (
	"math"
)

// FibonacciNaive implements naive recursive Fibonacci (O(2^n))
// This is intentionally inefficient for comparison
func FibonacciNaive(n int) int {
	if n <= 1 {
		return n
	}
	return FibonacciNaive(n-1) + FibonacciNaive(n-2)
}

// FibonacciMemoized implements memoized Fibonacci (O(n))
func FibonacciMemoized(n int) int {
	memo := make(map[int]int)
	return fibMemoHelper(n, memo)
}

func fibMemoHelper(n int, memo map[int]int) int {
	if n <= 1 {
		return n
	}
	if val, exists := memo[n]; exists {
		return val
	}
	memo[n] = fibMemoHelper(n-1, memo) + fibMemoHelper(n-2, memo)
	return memo[n]
}

// FibonacciBottomUp implements bottom-up Fibonacci (O(n))
func FibonacciBottomUp(n int) int {
	if n <= 1 {
		return n
	}

	dp := make([]int, n+1)
	dp[0] = 0
	dp[1] = 1

	for i := 2; i <= n; i++ {
		dp[i] = dp[i-1] + dp[i-2]
	}

	return dp[n]
}

// FibonacciOptimized implements space-optimized Fibonacci (O(n) time, O(1) space)
func FibonacciOptimized(n int) int {
	if n <= 1 {
		return n
	}

	prev2, prev1 := 0, 1
	for i := 2; i <= n; i++ {
		current := prev1 + prev2
		prev2 = prev1
		prev1 = current
	}

	return prev1
}

// LongestCommonSubsequence finds the length of LCS between two strings
func LongestCommonSubsequence(str1, str2 string) int {
	m, n := len(str1), len(str2)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if str1[i-1] == str2[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	return dp[m][n]
}

// LongestCommonSubsequenceOptimized uses space optimization (O(min(m,n)) space)
func LongestCommonSubsequenceOptimized(str1, str2 string) int {
	m, n := len(str1), len(str2)
	
	// Ensure str2 is the shorter string for space optimization
	if m < n {
		str1, str2 = str2, str1
		m, n = n, m
	}

	prev := make([]int, n+1)
	curr := make([]int, n+1)

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if str1[i-1] == str2[j-1] {
				curr[j] = prev[j-1] + 1
			} else {
				curr[j] = max(prev[j], curr[j-1])
			}
		}
		prev, curr = curr, prev
	}

	return prev[n]
}

// KnapsackProblem solves the 0/1 knapsack problem
func KnapsackProblem(weights, values []int, capacity int) int {
	n := len(weights)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, capacity+1)
	}

	for i := 1; i <= n; i++ {
		for w := 1; w <= capacity; w++ {
			if weights[i-1] <= w {
				// Include or exclude the item
				include := values[i-1] + dp[i-1][w-weights[i-1]]
				exclude := dp[i-1][w]
				dp[i][w] = max(include, exclude)
			} else {
				// Can't include the item
				dp[i][w] = dp[i-1][w]
			}
		}
	}

	return dp[n][capacity]
}

// KnapsackOptimized uses space-optimized knapsack (O(capacity) space)
func KnapsackOptimized(weights, values []int, capacity int) int {
	n := len(weights)
	dp := make([]int, capacity+1)

	for i := 0; i < n; i++ {
		// Traverse backwards to avoid using updated values
		for w := capacity; w >= weights[i]; w-- {
			dp[w] = max(dp[w], dp[w-weights[i]]+values[i])
		}
	}

	return dp[capacity]
}

// LongestIncreasingSubsequence finds the length of LIS
func LongestIncreasingSubsequence(arr []int) int {
	n := len(arr)
	if n == 0 {
		return 0
	}

	dp := make([]int, n)
	for i := range dp {
		dp[i] = 1
	}

	for i := 1; i < n; i++ {
		for j := 0; j < i; j++ {
			if arr[i] > arr[j] {
				dp[i] = max(dp[i], dp[j]+1)
			}
		}
	}

	maxLength := 0
	for _, length := range dp {
		maxLength = max(maxLength, length)
	}

	return maxLength
}

// LongestIncreasingSubsequenceOptimized uses binary search (O(n log n))
func LongestIncreasingSubsequenceOptimized(arr []int) int {
	n := len(arr)
	if n == 0 {
		return 0
	}

	tails := make([]int, 0, n)

	for _, num := range arr {
		// Binary search for the position to insert/replace
		left, right := 0, len(tails)
		for left < right {
			mid := left + (right-left)/2
			if tails[mid] < num {
				left = mid + 1
			} else {
				right = mid
			}
		}

		if left == len(tails) {
			tails = append(tails, num)
		} else {
			tails[left] = num
		}
	}

	return len(tails)
}

// EditDistance calculates the minimum edit distance between two strings
func EditDistance(str1, str2 string) int {
	m, n := len(str1), len(str2)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	// Initialize base cases
	for i := 0; i <= m; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= n; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if str1[i-1] == str2[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				// Min of insert, delete, replace
				dp[i][j] = 1 + min3(dp[i-1][j], dp[i][j-1], dp[i-1][j-1])
			}
		}
	}

	return dp[m][n]
}

// EditDistanceOptimized uses space optimization
func EditDistanceOptimized(str1, str2 string) int {
	m, n := len(str1), len(str2)
	
	// Ensure str2 is the shorter string
	if m < n {
		str1, str2 = str2, str1
		m, n = n, m
	}

	prev := make([]int, n+1)
	curr := make([]int, n+1)

	// Initialize first row
	for j := 0; j <= n; j++ {
		prev[j] = j
	}

	for i := 1; i <= m; i++ {
		curr[0] = i
		for j := 1; j <= n; j++ {
			if str1[i-1] == str2[j-1] {
				curr[j] = prev[j-1]
			} else {
				curr[j] = 1 + min3(prev[j], curr[j-1], prev[j-1])
			}
		}
		prev, curr = curr, prev
	}

	return prev[n]
}

// CoinChange finds the minimum number of coins needed to make the amount
func CoinChange(coins []int, amount int) int {
	dp := make([]int, amount+1)
	for i := 1; i <= amount; i++ {
		dp[i] = math.MaxInt32
	}

	for i := 1; i <= amount; i++ {
		for _, coin := range coins {
			if coin <= i && dp[i-coin] != math.MaxInt32 {
				dp[i] = Min(dp[i], dp[i-coin]+1)
			}
		}
	}

	if dp[amount] == math.MaxInt32 {
		return -1
	}
	return dp[amount]
}

// CoinChangeWays counts the number of ways to make the amount
func CoinChangeWays(coins []int, amount int) int {
	dp := make([]int, amount+1)
	dp[0] = 1

	for _, coin := range coins {
		for i := coin; i <= amount; i++ {
			dp[i] += dp[i-coin]
		}
	}

	return dp[amount]
}

// MaxSubarraySum finds the maximum sum of a contiguous subarray (Kadane's algorithm)
func MaxSubarraySum(arr []int) int {
	if len(arr) == 0 {
		return 0
	}

	maxSoFar := arr[0]
	maxEndingHere := arr[0]

	for i := 1; i < len(arr); i++ {
		maxEndingHere = max(arr[i], maxEndingHere+arr[i])
		maxSoFar = max(maxSoFar, maxEndingHere)
	}

	return maxSoFar
}

// MaxSubarrayProduct finds the maximum product of a contiguous subarray
func MaxSubarrayProduct(arr []int) int {
	if len(arr) == 0 {
		return 0
	}

	maxSoFar := arr[0]
	maxEndingHere := arr[0]
	minEndingHere := arr[0]

	for i := 1; i < len(arr); i++ {
		if arr[i] < 0 {
			maxEndingHere, minEndingHere = minEndingHere, maxEndingHere
		}

		maxEndingHere = max(arr[i], maxEndingHere*arr[i])
		minEndingHere = Min(arr[i], minEndingHere*arr[i])

		maxSoFar = max(maxSoFar, maxEndingHere)
	}

	return maxSoFar
}

// HouseRobber solves the house robber problem
func HouseRobber(houses []int) int {
	n := len(houses)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return houses[0]
	}

	prev2 := houses[0]
	prev1 := max(houses[0], houses[1])

	for i := 2; i < n; i++ {
		current := max(prev1, prev2+houses[i])
		prev2 = prev1
		prev1 = current
	}

	return prev1
}

// ClimbingStairs counts the number of ways to climb n stairs
func ClimbingStairs(n int) int {
	if n <= 2 {
		return n
	}

	prev2, prev1 := 1, 2
	for i := 3; i <= n; i++ {
		current := prev1 + prev2
		prev2 = prev1
		prev1 = current
	}

	return prev1
}

// PalindromePartitioning finds minimum cuts needed for palindrome partitioning
func PalindromePartitioning(s string) int {
	n := len(s)
	if n <= 1 {
		return 0
	}

	// Precompute palindrome table
	isPalindrome := make([][]bool, n)
	for i := range isPalindrome {
		isPalindrome[i] = make([]bool, n)
	}

	// Every single character is a palindrome
	for i := 0; i < n; i++ {
		isPalindrome[i][i] = true
	}

	// Check for palindromes of length 2
	for i := 0; i < n-1; i++ {
		isPalindrome[i][i+1] = (s[i] == s[i+1])
	}

	// Check for palindromes of length 3 and more
	for length := 3; length <= n; length++ {
		for i := 0; i <= n-length; i++ {
			j := i + length - 1
			isPalindrome[i][j] = (s[i] == s[j]) && isPalindrome[i+1][j-1]
		}
	}

	// DP for minimum cuts
	cuts := make([]int, n)
	for i := 0; i < n; i++ {
		cuts[i] = i // Maximum cuts needed
		if isPalindrome[0][i] {
			cuts[i] = 0
		} else {
			for j := 0; j < i; j++ {
				if isPalindrome[j+1][i] {
					cuts[i] = Min(cuts[i], cuts[j]+1)
				}
			}
		}
	}

	return cuts[n-1]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func min3(a, b, c int) int {
	return Min(Min(a, b), c)
}
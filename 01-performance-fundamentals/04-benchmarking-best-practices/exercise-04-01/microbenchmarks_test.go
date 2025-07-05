package benchmarks

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Microbenchmarks for basic algorithms and operations

// String operations benchmarks
func BenchmarkStringConcatenation(b *testing.B) {
	strings := []string{"hello", "world", "this", "is", "a", "test"}
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		var result string
		for _, s := range strings {
			result += s
		}
		_ = result
	}
}

func BenchmarkStringBuilder(b *testing.B) {
	strs := []string{"hello", "world", "this", "is", "a", "test"}
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		var builder strings.Builder
		for _, s := range strs {
			builder.WriteString(s)
		}
		result := builder.String()
		_ = result
	}
}

func BenchmarkStringJoin(b *testing.B) {
	strs := []string{"hello", "world", "this", "is", "a", "test"}
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		result := strings.Join(strs, "")
		_ = result
	}
}

// Sorting algorithm benchmarks
func BenchmarkBubbleSort(b *testing.B) {
	data := generateRandomSlice(1000)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Copy data for each iteration
		dataCopy := make([]int, len(data))
		copy(dataCopy, data)
		
		bubbleSort(dataCopy)
	}
}

func BenchmarkQuickSort(b *testing.B) {
	data := generateRandomSlice(1000)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Copy data for each iteration
		dataCopy := make([]int, len(data))
		copy(dataCopy, data)
		
		quickSort(dataCopy, 0, len(dataCopy)-1)
	}
}

func BenchmarkStandardSort(b *testing.B) {
	data := generateRandomSlice(1000)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Copy data for each iteration
		dataCopy := make([]int, len(data))
		copy(dataCopy, data)
		
		sort.Ints(dataCopy)
	}
}

// Search algorithm benchmarks
func BenchmarkLinearSearch(b *testing.B) {
	data := generateSortedSlice(10000)
	target := data[len(data)/2] // Middle element
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		result := linearSearch(data, target)
		_ = result
	}
}

func BenchmarkBinarySearch(b *testing.B) {
	data := generateSortedSlice(10000)
	target := data[len(data)/2] // Middle element
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		result := binarySearch(data, target)
		_ = result
	}
}

func BenchmarkStandardBinarySearch(b *testing.B) {
	data := generateSortedSlice(10000)
	target := data[len(data)/2] // Middle element
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		result := sort.SearchInts(data, target)
		_ = result
	}
}

// Hash function benchmarks
func BenchmarkMD5Hash(b *testing.B) {
	data := []byte("Hello, World! This is a test string for hashing.")
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		hash := md5.Sum(data)
		_ = hash
	}
}

func BenchmarkSHA256Hash(b *testing.B) {
	data := []byte("Hello, World! This is a test string for hashing.")
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		hash := sha256.Sum256(data)
		_ = hash
	}
}

// Number conversion benchmarks
func BenchmarkIntToString(b *testing.B) {
	num := 12345
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		result := strconv.Itoa(num)
		_ = result
	}
}

func BenchmarkIntToStringFmt(b *testing.B) {
	num := 12345
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		result := fmt.Sprintf("%d", num)
		_ = result
	}
}

func BenchmarkStringToInt(b *testing.B) {
	str := "12345"
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		result, _ := strconv.Atoi(str)
		_ = result
	}
}

// Memory allocation benchmarks
func BenchmarkSliceAppend(b *testing.B) {
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		var slice []int
		for j := 0; j < 1000; j++ {
			slice = append(slice, j)
		}
		_ = slice
	}
}

func BenchmarkSlicePrealloc(b *testing.B) {
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		slice := make([]int, 0, 1000)
		for j := 0; j < 1000; j++ {
			slice = append(slice, j)
		}
		_ = slice
	}
}

func BenchmarkSliceDirect(b *testing.B) {
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		slice := make([]int, 1000)
		for j := 0; j < 1000; j++ {
			slice[j] = j
		}
		_ = slice
	}
}

// Map operations benchmarks
func BenchmarkMapInsert(b *testing.B) {
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		m := make(map[int]int)
		for j := 0; j < 1000; j++ {
			m[j] = j * 2
		}
		_ = m
	}
}

func BenchmarkMapInsertPrealloc(b *testing.B) {
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		m := make(map[int]int, 1000)
		for j := 0; j < 1000; j++ {
			m[j] = j * 2
		}
		_ = m
	}
}

func BenchmarkMapLookup(b *testing.B) {
	m := make(map[int]int, 1000)
	for i := 0; i < 1000; i++ {
		m[i] = i * 2
	}
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		key := i % 1000
		value, ok := m[key]
		_ = value
		_ = ok
	}
}

// Parametric benchmarks for different input sizes
func BenchmarkSortingSizes(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			data := generateRandomSlice(size)
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				dataCopy := make([]int, len(data))
				copy(dataCopy, data)
				sort.Ints(dataCopy)
			}
		})
	}
}

func BenchmarkStringBuildingSizes(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			strs := make([]string, size)
			for i := range strs {
				strs[i] = fmt.Sprintf("string%d", i)
			}
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				var builder strings.Builder
				for _, s := range strs {
					builder.WriteString(s)
				}
				result := builder.String()
				_ = result
			}
		})
	}
}

// Parallel benchmarks
func BenchmarkParallelMapAccess(b *testing.B) {
	m := make(map[int]int, 1000)
	for i := 0; i < 1000; i++ {
		m[i] = i * 2
	}
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := rand.Intn(1000)
			value, ok := m[key]
			_ = value
			_ = ok
		}
	})
}

func BenchmarkParallelSliceAccess(b *testing.B) {
	slice := make([]int, 1000)
	for i := range slice {
		slice[i] = i * 2
	}
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index := rand.Intn(len(slice))
			value := slice[index]
			_ = value
		}
	})
}

// Helper functions for algorithms
func generateRandomSlice(size int) []int {
	rand.Seed(time.Now().UnixNano())
	slice := make([]int, size)
	for i := range slice {
		slice[i] = rand.Intn(size * 10)
	}
	return slice
}

func generateSortedSlice(size int) []int {
	slice := make([]int, size)
	for i := range slice {
		slice[i] = i
	}
	return slice
}

func bubbleSort(arr []int) {
	n := len(arr)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if arr[j] > arr[j+1] {
				arr[j], arr[j+1] = arr[j+1], arr[j]
			}
		}
	}
}

func quickSort(arr []int, low, high int) {
	if low < high {
		pi := partition(arr, low, high)
		quickSort(arr, low, pi-1)
		quickSort(arr, pi+1, high)
	}
}

func partition(arr []int, low, high int) int {
	pivot := arr[high]
	i := low - 1
	
	for j := low; j < high; j++ {
		if arr[j] < pivot {
			i++
			arr[i], arr[j] = arr[j], arr[i]
		}
	}
	arr[i+1], arr[high] = arr[high], arr[i+1]
	return i + 1
}

func linearSearch(arr []int, target int) int {
	for i, v := range arr {
		if v == target {
			return i
		}
	}
	return -1
}

func binarySearch(arr []int, target int) int {
	left, right := 0, len(arr)-1
	
	for left <= right {
		mid := left + (right-left)/2
		
		if arr[mid] == target {
			return mid
		} else if arr[mid] < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return -1
}
package main

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// Test data for benchmarks
var (
	testData = []int{64, 34, 25, 12, 22, 11, 90, 88, 76, 50, 43}
	testString = "The quick brown fox jumps over the lazy dog"
	testStrings = []string{"apple", "banana", "cherry", "date", "elderberry"}
	testBytes = []byte("Hello, World! This is a test string for benchmarking.")
)

// TestZeroAllocProcessor tests the zero allocation processor
func TestZeroAllocProcessor(t *testing.T) {
	processor := NewZeroAllocProcessor(1024)
	
	// Test reverse in place
	data := make([]int, len(testData))
	copy(data, testData)
	processor.ReverseInPlace(data)
	if data[0] != testData[len(testData)-1] {
		t.Errorf("ReverseInPlace failed")
	}
	
	// Test sort in place
	copy(data, testData)
	processor.QuickSortInPlace(data)
	for i := 1; i < len(data); i++ {
		if data[i] < data[i-1] {
			t.Errorf("QuickSortInPlace failed")
		}
	}
	
	// Test filter in place
	copy(data, testData)
	newLen := processor.FilterInPlace(data, func(x int) bool { return x%2 == 0 })
	for i := 0; i < newLen; i++ {
		if data[i]%2 != 0 {
			t.Errorf("FilterInPlace failed")
		}
	}
	
	// Test binary search
	sortedData := []int{1, 3, 5, 7, 9, 11, 13, 15}
	index := processor.BinarySearchInPlace(sortedData, 7)
	if index != 3 {
		t.Errorf("BinarySearchInPlace failed, expected 3, got %d", index)
	}
	
	// Test int to string conversion
	str := processor.IntToStringZeroAlloc(12345)
	if str != "12345" {
		t.Errorf("IntToStringZeroAlloc failed, expected '12345', got '%s'", str)
	}
	
	// Test string to int parsing
	num, err := processor.ParseIntZeroAlloc("12345")
	if err != nil || num != 12345 {
		t.Errorf("ParseIntZeroAlloc failed, expected 12345, got %d, err: %v", num, err)
	}
}

// TestStringProcessor tests the string processor
func TestStringProcessor(t *testing.T) {
	processor := NewStringProcessor(1024)
	
	// Test concatenation
	result := processor.ConcatZeroAlloc("Hello", ", ", "World")
	if result != "Hello, World" {
		t.Errorf("ConcatZeroAlloc failed, expected 'Hello, World', got '%s'", result)
	}
	
	// Test join
	joined := processor.JoinZeroAlloc([]string{"a", "b", "c"}, ",")
	if joined != "a,b,c" {
		t.Errorf("JoinZeroAlloc failed, expected 'a,b,c', got '%s'", joined)
	}
	
	// Test split
	parts := processor.SplitZeroAlloc("a,b,c", ",")
	if len(parts) != 3 || parts[0] != "a" || parts[1] != "b" || parts[2] != "c" {
		t.Errorf("SplitZeroAlloc failed")
	}
	
	// Test case conversion
	upper := processor.ToUpperZeroAlloc("hello")
	if upper != "HELLO" {
		t.Errorf("ToUpperZeroAlloc failed, expected 'HELLO', got '%s'", upper)
	}
	
	lower := processor.ToLowerZeroAlloc("HELLO")
	if lower != "hello" {
		t.Errorf("ToLowerZeroAlloc failed, expected 'hello', got '%s'", lower)
	}
	
	// Test replace
	replaced := processor.ReplaceZeroAlloc("hello world", "world", "Go")
	if replaced != "hello Go" {
		t.Errorf("ReplaceZeroAlloc failed, expected 'hello Go', got '%s'", replaced)
	}
}

// Benchmark zero-allocation vs standard library functions

// BenchmarkReverseInPlace benchmarks in-place reverse
func BenchmarkReverseInPlace(b *testing.B) {
	processor := NewZeroAllocProcessor(1024)
	data := make([]int, len(testData))
	
	b.Run("ZeroAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			copy(data, testData)
			processor.ReverseInPlace(data)
			SinkInt(data[0])
		}
	})
	
	b.Run("WithAllocation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := make([]int, len(testData))
			for j, v := range testData {
				result[len(testData)-1-j] = v
			}
			SinkInt(result[0])
		}
	})
}

// BenchmarkSortInPlace benchmarks in-place sorting
func BenchmarkSortInPlace(b *testing.B) {
	processor := NewZeroAllocProcessor(1024)
	data := make([]int, len(testData))
	
	b.Run("ZeroAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			copy(data, testData)
			processor.QuickSortInPlace(data)
			SinkInt(data[0])
		}
	})
}

// BenchmarkFilterInPlace benchmarks in-place filtering
func BenchmarkFilterInPlace(b *testing.B) {
	processor := NewZeroAllocProcessor(1024)
	data := make([]int, len(testData))
	predicate := func(x int) bool { return x%2 == 0 }
	
	b.Run("ZeroAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			copy(data, testData)
			newLen := processor.FilterInPlace(data, predicate)
			SinkInt(newLen)
		}
	})
	
	b.Run("WithAllocation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var result []int
			for _, v := range testData {
				if predicate(v) {
					result = append(result, v)
				}
			}
			SinkInt(len(result))
		}
	})
}

// BenchmarkBinarySearch benchmarks binary search
func BenchmarkBinarySearch(b *testing.B) {
	processor := NewZeroAllocProcessor(1024)
	sortedData := []int{1, 3, 5, 7, 9, 11, 13, 15, 17, 19}
	target := 13
	
	b.Run("ZeroAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			index := processor.BinarySearchInPlace(sortedData, target)
			SinkInt(index)
		}
	})
}

// BenchmarkIntToString benchmarks integer to string conversion
func BenchmarkIntToString(b *testing.B) {
	processor := NewZeroAllocProcessor(1024)
	number := 12345
	
	b.Run("ZeroAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			str := processor.IntToStringZeroAlloc(number)
			SinkString(str)
		}
	})
	
	b.Run("Strconv", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			str := strconv.Itoa(number)
			SinkString(str)
		}
	})
	
	b.Run("Sprintf", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			str := fmt.Sprintf("%d", number)
			SinkString(str)
		}
	})
}

// BenchmarkStringConcat benchmarks string concatenation
func BenchmarkStringConcat(b *testing.B) {
	processor := NewStringProcessor(1024)
	strs := []string{"Hello", ", ", "World", "!"}
	
	b.Run("ZeroAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := processor.ConcatZeroAlloc(strs...)
			SinkString(result)
		}
	})
	
	b.Run("PlusOperator", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := strs[0] + strs[1] + strs[2] + strs[3]
			SinkString(result)
		}
	})
	
	b.Run("StringBuilder", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var builder strings.Builder
			for _, s := range strs {
				builder.WriteString(s)
			}
			result := builder.String()
			SinkString(result)
		}
	})
}

// BenchmarkStringJoin benchmarks string joining
func BenchmarkStringJoin(b *testing.B) {
	processor := NewStringProcessor(1024)
	strs := []string{"apple", "banana", "cherry", "date"}
	sep := ", "
	
	b.Run("ZeroAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := processor.JoinZeroAlloc(strs, sep)
			SinkString(result)
		}
	})
	
	b.Run("StringsJoin", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := strings.Join(strs, sep)
			SinkString(result)
		}
	})
}

// BenchmarkStringSplit benchmarks string splitting
func BenchmarkStringSplit(b *testing.B) {
	processor := NewStringProcessor(1024)
	text := "apple,banana,cherry,date,elderberry"
	sep := ","
	
	b.Run("ZeroAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			parts := processor.SplitZeroAlloc(text, sep)
			SinkString(parts[0])
		}
	})
	
	b.Run("StringsSplit", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			parts := strings.Split(text, sep)
			SinkString(parts[0])
		}
	})
}

// BenchmarkStringCase benchmarks case conversion
func BenchmarkStringCase(b *testing.B) {
	processor := NewStringProcessor(1024)
	text := "Hello World This Is A Test String"
	
	b.Run("ToUpperZeroAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := processor.ToUpperZeroAlloc(text)
			SinkString(result)
		}
	})
	
	b.Run("StringsToUpper", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := strings.ToUpper(text)
			SinkString(result)
		}
	})
	
	b.Run("ToLowerZeroAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := processor.ToLowerZeroAlloc(text)
			SinkString(result)
		}
	})
	
	b.Run("StringsToLower", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := strings.ToLower(text)
			SinkString(result)
		}
	})
}

// BenchmarkStringReplace benchmarks string replacement
func BenchmarkStringReplace(b *testing.B) {
	processor := NewStringProcessor(1024)
	text := "hello world hello universe hello galaxy"
	old := "hello"
	new := "hi"
	
	b.Run("ZeroAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := processor.ReplaceZeroAlloc(text, old, new)
			SinkString(result)
		}
	})
	
	b.Run("StringsReplace", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := strings.ReplaceAll(text, old, new)
			SinkString(result)
		}
	})
}

// BenchmarkStringFormat benchmarks string formatting
func BenchmarkStringFormat(b *testing.B) {
	processor := NewStringProcessor(1024)
	template := "Hello %s, you have %d messages"
	name := "Alice"
	count := 42
	
	b.Run("ZeroAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := processor.FormatZeroAlloc(template, name, count)
			SinkString(result)
		}
	})
	
	b.Run("Sprintf", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := fmt.Sprintf(template, name, count)
			SinkString(result)
		}
	})
}

// BenchmarkByteStringConversion benchmarks unsafe conversions
func BenchmarkByteStringConversion(b *testing.B) {
	testBytes := []byte("Hello, World! This is a test.")
	testString := "Hello, World! This is a test."
	
	b.Run("BytesToStringZeroAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			str := BytesToStringZeroAlloc(testBytes)
			SinkString(str)
		}
	})
	
	b.Run("BytesToStringStandard", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			str := string(testBytes)
			SinkString(str)
		}
	})
	
	b.Run("StringToBytesZeroAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			bytes := StringToBytesZeroAlloc(testString)
			SinkBytes(bytes)
		}
	})
	
	b.Run("StringToBytesStandard", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			bytes := []byte(testString)
			SinkBytes(bytes)
		}
	})
}

// BenchmarkHashCalculation benchmarks hash calculation
func BenchmarkHashCalculation(b *testing.B) {
	processor := NewZeroAllocProcessor(1024)
	data := []byte("The quick brown fox jumps over the lazy dog")
	
	b.Run("ZeroAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			hash := processor.HashZeroAlloc(data)
			SinkInt(int(hash))
		}
	})
}

// BenchmarkMemoryComparison benchmarks memory comparison
func BenchmarkMemoryComparison(b *testing.B) {
	processor := NewZeroAllocProcessor(1024)
	data1 := []byte("Hello, World!")
	data2 := []byte("Hello, World!")
	
	b.Run("ZeroAlloc", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := processor.CompareZeroAlloc(data1, data2)
			SinkInt(result)
		}
	})
}
package benchmarks

import (
	"math"
	"math/rand"
	"time"
)

// Initialize random seed
func init() {
	rand.Seed(time.Now().UnixNano())
}

// GenerateRandomData creates an array of random integers
func GenerateRandomData(size int) []int {
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Intn(size * 10)
	}
	return data
}

// GenerateSortedData creates an already sorted array
func GenerateSortedData(size int) []int {
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = i
	}
	return data
}

// GenerateReverseSortedData creates a reverse sorted array
func GenerateReverseSortedData(size int) []int {
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = size - i - 1
	}
	return data
}

// GeneratePartiallySortedData creates a partially sorted array
func GeneratePartiallySortedData(size int) []int {
	data := GenerateSortedData(size)
	
	// Shuffle 10% of the elements
	shuffleCount := size / 10
	for i := 0; i < shuffleCount; i++ {
		idx1 := rand.Intn(size)
		idx2 := rand.Intn(size)
		data[idx1], data[idx2] = data[idx2], data[idx1]
	}
	
	return data
}

// GenerateManyDuplicatesData creates an array with many duplicate values
func GenerateManyDuplicatesData(size int) []int {
	data := make([]int, size)
	uniqueValues := size / 10 // Only 10% unique values
	if uniqueValues < 1 {
		uniqueValues = 1
	}
	
	for i := 0; i < size; i++ {
		data[i] = rand.Intn(uniqueValues)
	}
	return data
}

// GenerateFewUniqueData creates an array with very few unique values
func GenerateFewUniqueData(size int) []int {
	data := make([]int, size)
	uniqueValues := []int{1, 2, 3, 4, 5} // Only 5 unique values
	
	for i := 0; i < size; i++ {
		data[i] = uniqueValues[rand.Intn(len(uniqueValues))]
	}
	return data
}

// GenerateNearlySortedData creates an array that is nearly sorted
func GenerateNearlySortedData(size int) []int {
	data := GenerateSortedData(size)
	
	// Add small perturbations
	perturbations := size / 100 // 1% perturbations
	if perturbations < 1 {
		perturbations = 1
	}
	
	for i := 0; i < perturbations; i++ {
		idx := rand.Intn(size)
		if idx > 0 {
			data[idx], data[idx-1] = data[idx-1], data[idx]
		}
	}
	
	return data
}

// GenerateOrganPipeData creates an organ pipe pattern (ascending then descending)
func GenerateOrganPipeData(size int) []int {
	data := make([]int, size)
	mid := size / 2
	
	// Ascending part
	for i := 0; i < mid; i++ {
		data[i] = i
	}
	
	// Descending part
	for i := mid; i < size; i++ {
		data[i] = size - i - 1
	}
	
	return data
}

// GenerateSawtoothData creates a sawtooth pattern
func GenerateSawtoothData(size int) []int {
	data := make([]int, size)
	period := size / 10 // 10 periods
	if period < 1 {
		period = 1
	}
	
	for i := 0; i < size; i++ {
		data[i] = i % period
	}
	
	return data
}

// GenerateRandomRangeData creates random data within a specific range
func GenerateRandomRangeData(size int, min, max int) []int {
	data := make([]int, size)
	rangeSize := max - min + 1
	
	for i := 0; i < size; i++ {
		data[i] = min + rand.Intn(rangeSize)
	}
	
	return data
}

// GenerateGaussianData creates data following a Gaussian distribution
func GenerateGaussianData(size int) []int {
	data := make([]int, size)
	mean := float64(size) / 2
	stddev := float64(size) / 6
	
	for i := 0; i < size; i++ {
		// Box-Muller transform for Gaussian distribution
		u1 := rand.Float64()
		u2 := rand.Float64()
		z0 := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
		value := int(mean + stddev*z0)
		
		// Clamp to reasonable range
		if value < 0 {
			value = 0
		} else if value >= size*2 {
			value = size*2 - 1
		}
		
		data[i] = value
	}
	
	return data
}

// GenerateStringData creates random string data for string sorting tests
func GenerateStringData(size int, maxLength int) []string {
	data := make([]string, size)
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	
	for i := 0; i < size; i++ {
		length := rand.Intn(maxLength) + 1
		str := make([]byte, length)
		
		for j := 0; j < length; j++ {
			str[j] = charset[rand.Intn(len(charset))]
		}
		
		data[i] = string(str)
	}
	
	return data
}

// GenerateInt32Data creates random int32 data
func GenerateInt32Data(size int) []int32 {
	data := make([]int32, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Int31()
	}
	return data
}

// GenerateInt64Data creates random int64 data
func GenerateInt64Data(size int) []int64 {
	data := make([]int64, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Int63()
	}
	return data
}

// GenerateNegativeData creates data with negative numbers
func GenerateNegativeData(size int) []int {
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = -rand.Intn(size * 10)
	}
	return data
}

// GenerateMixedSignData creates data with mixed positive and negative numbers
func GenerateMixedSignData(size int) []int {
	data := make([]int, size)
	for i := 0; i < size; i++ {
		value := rand.Intn(size * 10)
		if rand.Float32() < 0.5 {
			value = -value
		}
		data[i] = value
	}
	return data
}



// DataPattern represents different data patterns for testing
type DataPattern struct {
	Name        string
	Description string
	Generator   func(int) []int
}

// GetAllDataPatterns returns all available data patterns
func GetAllDataPatterns() []DataPattern {
	return []DataPattern{
		{"Random", "Completely random data", GenerateRandomData},
		{"Sorted", "Already sorted data", GenerateSortedData},
		{"Reverse", "Reverse sorted data", GenerateReverseSortedData},
		{"PartiallySorted", "90% sorted with 10% shuffled", GeneratePartiallySortedData},
		{"ManyDuplicates", "Many duplicate values (10% unique)", GenerateManyDuplicatesData},
		{"FewUnique", "Only 5 unique values", GenerateFewUniqueData},
		{"NearlySorted", "Nearly sorted with 1% perturbations", GenerateNearlySortedData},
		{"OrganPipe", "Ascending then descending pattern", GenerateOrganPipeData},
		{"Sawtooth", "Repeating sawtooth pattern", GenerateSawtoothData},
		{"Gaussian", "Gaussian distribution", GenerateGaussianData},
		{"Negative", "Data with negative numbers", GenerateNegativeData},
		{"MixedSign", "Mixed positive and negative", GenerateMixedSignData},
	}
}
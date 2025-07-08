package tests

import (
	"math/rand"
	"sort"
)

// TestHelper provides utility functions for testing
type TestHelper struct{}

// NewTestHelper creates a new test helper
func NewTestHelper() *TestHelper {
	return &TestHelper{}
}

// GenerateRandomData creates random test data
func (th *TestHelper) GenerateRandomData(size int) []int {
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Intn(size * 10)
	}
	return data
}

// GenerateSortedData creates sorted test data
func (th *TestHelper) GenerateSortedData(size int) []int {
	data := th.GenerateRandomData(size)
	sort.Ints(data)
	return data
}

// GenerateStringData creates random string test data
func (th *TestHelper) GenerateStringData(size int) []string {
	data := make([]string, size)
	for i := 0; i < size; i++ {
		data[i] = randomString(8)
	}
	return data
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
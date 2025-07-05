package benchmarks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// End-to-end benchmarks for complete workflows and applications

// Data processing pipeline benchmark
type DataRecord struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	Tags      []string  `json:"tags"`
}

type ProcessingResult struct {
	TotalRecords int            `json:"total_records"`
	AverageValue float64        `json:"average_value"`
	MaxValue     float64        `json:"max_value"`
	MinValue     float64        `json:"min_value"`
	TagCounts    map[string]int `json:"tag_counts"`
}

func generateDataRecords(count int) []DataRecord {
	records := make([]DataRecord, count)
	tags := []string{"urgent", "normal", "low", "critical", "info"}

	for i := 0; i < count; i++ {
		records[i] = DataRecord{
			ID:        i,
			Name:      fmt.Sprintf("record_%d", i),
			Value:     float64(i%1000) + 0.5,
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Tags:      []string{tags[i%len(tags)], tags[(i+1)%len(tags)]},
		}
	}
	return records
}

func processDataRecords(records []DataRecord) ProcessingResult {
	if len(records) == 0 {
		return ProcessingResult{}
	}

	result := ProcessingResult{
		TotalRecords: len(records),
		TagCounts:    make(map[string]int),
		MaxValue:     records[0].Value,
		MinValue:     records[0].Value,
	}

	var totalValue float64
	for _, record := range records {
		totalValue += record.Value

		if record.Value > result.MaxValue {
			result.MaxValue = record.Value
		}
		if record.Value < result.MinValue {
			result.MinValue = record.Value
		}

		for _, tag := range record.Tags {
			result.TagCounts[tag]++
		}
	}

	result.AverageValue = totalValue / float64(len(records))
	return result
}

func BenchmarkDataProcessingPipeline(b *testing.B) {
	records := generateDataRecords(10000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate complete data processing workflow

		// 1. JSON serialization
		jsonData, _ := json.Marshal(records)

		// 2. JSON deserialization
		var deserializedRecords []DataRecord
		json.Unmarshal(jsonData, &deserializedRecords)

		// 3. Data processing
		result := processDataRecords(deserializedRecords)

		// 4. Result serialization
		resultJSON, _ := json.Marshal(result)

		_ = resultJSON
	}
}

func BenchmarkConcurrentDataProcessing(b *testing.B) {
	records := generateDataRecords(10000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Process data in parallel chunks
		numWorkers := 4
		chunks := make([][]DataRecord, numWorkers)

		// Divide records into chunks
		for j, record := range records {
			chunkIndex := j % numWorkers
			chunks[chunkIndex] = append(chunks[chunkIndex], record)
		}

		// Process chunks concurrently
		var wg sync.WaitGroup
		results := make([]ProcessingResult, numWorkers)

		for j := 0; j < numWorkers; j++ {
			wg.Add(1)
			go func(index int, chunk []DataRecord) {
				defer wg.Done()
				results[index] = processDataRecords(chunk)
			}(j, chunks[j])
		}

		wg.Wait()

		// Merge results
		finalResult := mergeProcessingResults(results)
		_ = finalResult
	}
}

func mergeProcessingResults(results []ProcessingResult) ProcessingResult {
	final := ProcessingResult{
		TagCounts: make(map[string]int),
	}

	var totalValue float64
	var totalRecords int

	for _, result := range results {
		totalRecords += result.TotalRecords
		totalValue += result.AverageValue * float64(result.TotalRecords)

		if result.MaxValue > final.MaxValue {
			final.MaxValue = result.MaxValue
		}
		if final.MinValue == 0 || result.MinValue < final.MinValue {
			final.MinValue = result.MinValue
		}

		for tag, count := range result.TagCounts {
			final.TagCounts[tag] += count
		}
	}

	final.TotalRecords = totalRecords
	if totalRecords > 0 {
		final.AverageValue = totalValue / float64(totalRecords)
	}

	return final
}

// HTTP API workflow benchmark
type APIServer struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

func NewAPIServer() *APIServer {
	return &APIServer{
		data: make(map[string]interface{}),
	}
}

func (s *APIServer) handleGet(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "key parameter required", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	value, exists := s.data[key]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "key not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": value})
}

func (s *APIServer) handlePost(w http.ResponseWriter, r *http.Request) {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	key, ok := payload["key"].(string)
	if !ok {
		http.Error(w, "key must be a string", http.StatusBadRequest)
		return
	}

	value := payload["value"]

	s.mu.Lock()
	s.data[key] = value
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *APIServer) handleList(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	keys := make([]string, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}
	s.mu.RUnlock()

	sort.Strings(keys)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{"keys": keys})
}

func (s *APIServer) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/get", s.handleGet)
	mux.HandleFunc("/post", s.handlePost)
	mux.HandleFunc("/list", s.handleList)
	return mux
}

func BenchmarkHTTPAPIWorkflow(b *testing.B) {
	server := NewAPIServer()
	mux := server.setupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Complete API workflow
		key := fmt.Sprintf("key_%d", i%100)
		value := fmt.Sprintf("value_%d", i)

		// 1. POST request to store data
		payload := map[string]interface{}{
			"key":   key,
			"value": value,
		}
		jsonPayload, _ := json.Marshal(payload)

		resp, err := client.Post(testServer.URL+"/post", "application/json", bytes.NewBuffer(jsonPayload))
		if err == nil {
			resp.Body.Close()
		}

		// 2. GET request to retrieve data
		resp, err = client.Get(testServer.URL + "/get?key=" + key)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}

		// 3. LIST request every 10 iterations
		if i%10 == 0 {
			resp, err = client.Get(testServer.URL + "/list")
			if err == nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	}
}

func BenchmarkConcurrentHTTPRequests(b *testing.B) {
	server := NewAPIServer()
	mux := server.setupRoutes()
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Pre-populate some data
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key_%d", i)
		server.data[key] = fmt.Sprintf("value_%d", i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		client := &http.Client{Timeout: 5 * time.Second}
		for pb.Next() {
			key := fmt.Sprintf("key_%d", rand.Intn(100))

			// 80% reads, 20% writes
			if rand.Float32() < 0.8 {
				// GET request
				resp, err := client.Get(testServer.URL + "/get?key=" + key)
				if err == nil {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
			} else {
				// POST request
				payload := map[string]interface{}{
					"key":   key,
					"value": fmt.Sprintf("updated_value_%d", rand.Intn(1000)),
				}
				jsonPayload, _ := json.Marshal(payload)

				resp, err := client.Post(testServer.URL+"/post", "application/json", bytes.NewBuffer(jsonPayload))
				if err == nil {
					resp.Body.Close()
				}
			}
		}
	})
}

// Database-like operations workflow
type InMemoryDB struct {
	tables map[string]map[string]interface{}
	mu     sync.RWMutex
}

func NewInMemoryDB() *InMemoryDB {
	return &InMemoryDB{
		tables: make(map[string]map[string]interface{}),
	}
}

func (db *InMemoryDB) CreateTable(tableName string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.tables[tableName] = make(map[string]interface{})
}

func (db *InMemoryDB) Insert(tableName, key string, value interface{}) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	table, exists := db.tables[tableName]
	if !exists {
		return fmt.Errorf("table %s does not exist", tableName)
	}

	table[key] = value
	return nil
}

func (db *InMemoryDB) Select(tableName, key string) (interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	table, exists := db.tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table %s does not exist", tableName)
	}

	value, exists := table[key]
	if !exists {
		return nil, fmt.Errorf("key %s not found", key)
	}

	return value, nil
}

func (db *InMemoryDB) Update(tableName, key string, value interface{}) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	table, exists := db.tables[tableName]
	if !exists {
		return fmt.Errorf("table %s does not exist", tableName)
	}

	if _, exists := table[key]; !exists {
		return fmt.Errorf("key %s not found", key)
	}

	table[key] = value
	return nil
}

func (db *InMemoryDB) Delete(tableName, key string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	table, exists := db.tables[tableName]
	if !exists {
		return fmt.Errorf("table %s does not exist", tableName)
	}

	delete(table, key)
	return nil
}

func (db *InMemoryDB) SelectAll(tableName string) (map[string]interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	table, exists := db.tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table %s does not exist", tableName)
	}

	// Return a copy to avoid race conditions
	result := make(map[string]interface{})
	for k, v := range table {
		result[k] = v
	}

	return result, nil
}

func BenchmarkDatabaseWorkflow(b *testing.B) {
	db := NewInMemoryDB()
	tableName := "users"
	db.CreateTable(tableName)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Complete database workflow
		userID := fmt.Sprintf("user_%d", i%1000)
		userData := map[string]interface{}{
			"name":  fmt.Sprintf("User %d", i),
			"email": fmt.Sprintf("user%d@example.com", i),
			"age":   20 + (i % 50),
		}

		// 1. Insert user
		db.Insert(tableName, userID, userData)

		// 2. Select user
		db.Select(tableName, userID)

		// 3. Update user (every 3rd iteration)
		if i%3 == 0 {
			updatedData := make(map[string]interface{})
			for k, v := range userData {
				updatedData[k] = v
			}
			updatedData["age"] = 25 + (i % 40)
			db.Update(tableName, userID, updatedData)
		}

		// 4. Select all users (every 10th iteration)
		if i%10 == 0 {
			db.SelectAll(tableName)
		}

		// 5. Delete user (every 5th iteration)
		if i%5 == 0 {
			db.Delete(tableName, userID)
		}
	}
}

func BenchmarkConcurrentDatabaseOperations(b *testing.B) {
	db := NewInMemoryDB()
	tableName := "concurrent_users"
	db.CreateTable(tableName)

	// Pre-populate some data
	for i := 0; i < 1000; i++ {
		userID := fmt.Sprintf("user_%d", i)
		userData := map[string]interface{}{
			"name":  fmt.Sprintf("User %d", i),
			"email": fmt.Sprintf("user%d@example.com", i),
			"age":   20 + (i % 50),
		}
		db.Insert(tableName, userID, userData)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			userID := fmt.Sprintf("user_%d", rand.Intn(1000))

			// 60% reads, 30% updates, 10% inserts
			operationType := rand.Float32()

			if operationType < 0.6 {
				// Read operation
				db.Select(tableName, userID)
			} else if operationType < 0.9 {
				// Update operation
				updatedData := map[string]interface{}{
					"name":  fmt.Sprintf("Updated User %d", rand.Intn(1000)),
					"email": fmt.Sprintf("updated%d@example.com", rand.Intn(1000)),
					"age":   20 + rand.Intn(50),
				}
				db.Update(tableName, userID, updatedData)
			} else {
				// Insert operation
				newUserID := fmt.Sprintf("new_user_%d", rand.Intn(10000))
				newUserData := map[string]interface{}{
					"name":  fmt.Sprintf("New User %d", rand.Intn(1000)),
					"email": fmt.Sprintf("new%d@example.com", rand.Intn(1000)),
					"age":   18 + rand.Intn(60),
				}
				db.Insert(tableName, newUserID, newUserData)
			}
		}
	})
}

// File processing workflow
func BenchmarkFileProcessingWorkflow(b *testing.B) {
	// Generate test data
	testData := make([]string, 10000)
	for i := range testData {
		testData[i] = fmt.Sprintf("line_%d: %s", i, strings.Repeat("data", 10))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// 1. Create file content
		var buffer bytes.Buffer
		for _, line := range testData {
			buffer.WriteString(line + "\n")
		}

		// 2. Process file content (simulate reading and parsing)
		content := buffer.String()
		lines := strings.Split(content, "\n")

		// 3. Filter and transform data
		var processedLines []string
		for _, line := range lines {
			if strings.Contains(line, "data") {
				processedLines = append(processedLines, strings.ToUpper(line))
			}
		}

		// 4. Generate output
		var output bytes.Buffer
		for _, line := range processedLines {
			output.WriteString(line + "\n")
		}

		_ = output.String()
	}
}

// Context-aware workflow benchmark
func BenchmarkContextAwareWorkflow(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)

		// Simulate work with context
		result := make(chan string, 1)
		go func() {
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			result <- fmt.Sprintf("work_result_%d", i)
		}()

		// Wait for result or timeout
		select {
		case res := <-result:
			_ = res
		case <-ctx.Done():
			// Handle timeout
		}

		cancel()
	}
}

// Memory-intensive workflow benchmark
func BenchmarkMemoryIntensiveWorkflow(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// 1. Allocate large data structures
		data := make([][]int, 1000)
		for j := range data {
			data[j] = make([]int, 1000)
			for k := range data[j] {
				data[j][k] = j*k + i
			}
		}

		// 2. Process data (matrix operations)
		sum := 0
		for j := range data {
			for k := range data[j] {
				sum += data[j][k]
			}
		}

		// 3. Transform data
		for j := range data {
			for k := range data[j] {
				data[j][k] = data[j][k] * 2
			}
		}

		_ = sum
		_ = data
	}
}

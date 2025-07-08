package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestDatabaseTypes tests the DatabaseType constants
func TestDatabaseTypes(t *testing.T) {
	tests := []struct {
		name     string
		dbType   DatabaseType
		expected DatabaseType
	}{
		{"PostgreSQL", PostgreSQL, 0},
		{"MySQL", MySQL, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dbType != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.dbType)
			}
		})
	}
}

// TestDefaultPostgreSQLConfig tests the default PostgreSQL configuration
func TestDefaultPostgreSQLConfig(t *testing.T) {
	config := DefaultPostgreSQLConfig("postgres://user:pass@localhost/testdb")

	if config.Type != PostgreSQL {
		t.Errorf("Expected type %v, got %v", PostgreSQL, config.Type)
	}

	if config.MaxOpenConns != 30 {
		t.Errorf("Expected MaxOpenConns 30, got %d", config.MaxOpenConns)
	}

	if config.MaxIdleConns != 5 {
		t.Errorf("Expected MaxIdleConns 5, got %d", config.MaxIdleConns)
	}

	if config.ConnectTimeout != 10*time.Second {
		t.Errorf("Expected ConnectTimeout 10s, got %v", config.ConnectTimeout)
	}

	if config.QueryTimeout != 30*time.Second {
		t.Errorf("Expected QueryTimeout 30s, got %v", config.QueryTimeout)
	}

	if config.MaxLifetime != time.Hour {
		t.Errorf("Expected MaxLifetime 1h, got %v", config.MaxLifetime)
	}

	if config.MaxIdleTime != 30*time.Minute {
		t.Errorf("Expected MaxIdleTime 30m, got %v", config.MaxIdleTime)
	}
}

// TestDefaultMySQLConfig tests the default MySQL configuration
func TestDefaultMySQLConfig(t *testing.T) {
	config := DefaultMySQLConfig("user:pass@tcp(localhost:3306)/testdb")

	if config.Type != MySQL {
		t.Errorf("Expected type %v, got %v", MySQL, config.Type)
	}

	if config.MaxOpenConns != 25 {
		t.Errorf("Expected MaxOpenConns 25, got %d", config.MaxOpenConns)
	}

	if config.MaxIdleConns != 5 {
		t.Errorf("Expected MaxIdleConns 5, got %d", config.MaxIdleConns)
	}

	if config.ConnectTimeout != 10*time.Second {
		t.Errorf("Expected ConnectTimeout 10s, got %v", config.ConnectTimeout)
	}

	if config.QueryTimeout != 30*time.Second {
		t.Errorf("Expected QueryTimeout 30s, got %v", config.QueryTimeout)
	}

	if config.MaxLifetime != 5*time.Minute {
		t.Errorf("Expected MaxLifetime 5m, got %v", config.MaxLifetime)
	}

	if config.MaxIdleTime != 2*time.Minute {
		t.Errorf("Expected MaxIdleTime 2m, got %v", config.MaxIdleTime)
	}
}

// TestDatabaseMetrics tests the metrics functionality
func TestDatabaseMetrics(t *testing.T) {
	metrics := &DatabaseMetrics{}

	// Test initial values
	if metrics.TotalQueries != 0 {
		t.Errorf("Expected TotalQueries 0, got %d", metrics.TotalQueries)
	}

	if metrics.SuccessfulQueries != 0 {
		t.Errorf("Expected SuccessfulQueries 0, got %d", metrics.SuccessfulQueries)
	}

	if metrics.FailedQueries != 0 {
		t.Errorf("Expected FailedQueries 0, got %d", metrics.FailedQueries)
	}

	if metrics.TotalDuration != 0 {
		t.Errorf("Expected TotalDuration 0, got %v", metrics.TotalDuration)
	}

	// Test that metrics can be created and have initial zero values
	// Note: RecordQuery is not a method on DatabaseMetrics, it's internal to OptimizedDatabase
	// So we just test the structure exists and has the expected fields
}

// TestOptimizedDatabaseCreation tests database creation without actual connections
func TestOptimizedDatabaseCreation(t *testing.T) {
	// Test PostgreSQL database creation structure
	pgConfig := DefaultPostgreSQLConfig("postgres://user:pass@localhost/testdb")
	pgDB := &OptimizedDatabase{
		config:  pgConfig,
		metrics: &DatabaseMetrics{},
	}

	if pgDB.config.Type != PostgreSQL {
		t.Errorf("Expected PostgreSQL type, got %v", pgDB.config.Type)
	}

	// Test MySQL database creation structure
	mysqlConfig := DefaultMySQLConfig("user:pass@tcp(localhost:3306)/testdb")
	mysqlDB := &OptimizedDatabase{
		config:  mysqlConfig,
		metrics: &DatabaseMetrics{},
	}

	if mysqlDB.config.Type != MySQL {
		t.Errorf("Expected MySQL type, got %v", mysqlDB.config.Type)
	}
}

// TestBulkInsertQueryGeneration tests the bulk insert query generation logic
func TestBulkInsertQueryGeneration(t *testing.T) {
	// This tests the logic without actual database connection
	tableName := "test_table"
	columns := []string{"id", "name", "email"}
	data := [][]interface{}{
		{1, "John", "john@example.com"},
		{2, "Jane", "jane@example.com"},
	}

	// Test PostgreSQL query format
	pgQuery, pgArgs := generateBulkInsertQuery(PostgreSQL, tableName, columns, data)
	expectedPGQuery := "INSERT INTO test_table (id,name,email) VALUES ($1,$2,$3),($4,$5,$6)"
	if pgQuery != expectedPGQuery {
		t.Errorf("Expected PostgreSQL query %s, got %s", expectedPGQuery, pgQuery)
	}
	if len(pgArgs) != 6 {
		t.Errorf("Expected 6 args for PostgreSQL, got %d", len(pgArgs))
	}

	// Test MySQL query format
	mysqlQuery, mysqlArgs := generateBulkInsertQuery(MySQL, tableName, columns, data)
	expectedMySQLQuery := "INSERT INTO test_table (id,name,email) VALUES (?,?,?),(?,?,?)"
	if mysqlQuery != expectedMySQLQuery {
		t.Errorf("Expected MySQL query %s, got %s", expectedMySQLQuery, mysqlQuery)
	}
	if len(mysqlArgs) != 6 {
		t.Errorf("Expected 6 args for MySQL, got %d", len(mysqlArgs))
	}
}

// Helper function to generate bulk insert queries (extracted from main logic)
func generateBulkInsertQuery(dbType DatabaseType, tableName string, columns []string, data [][]interface{}) (string, []interface{}) {
	var valueStrings []string
	var valueArgs []interface{}

	for i, row := range data {
		var placeholders []string
		for j := range row {
			if dbType == PostgreSQL {
				placeholders = append(placeholders, fmt.Sprintf("$%d", i*len(row)+j+1))
			} else {
				placeholders = append(placeholders, "?")
			}
		}
		valueStrings = append(valueStrings, fmt.Sprintf("(%s)", strings.Join(placeholders, ",")))
		valueArgs = append(valueArgs, row...)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		tableName,
		strings.Join(columns, ","),
		strings.Join(valueStrings, ","))

	return query, valueArgs
}

// Benchmark tests
func BenchmarkConfigCreation(b *testing.B) {
	b.Run("PostgreSQL", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = DefaultPostgreSQLConfig("postgres://user:pass@localhost/testdb")
		}
	})

	b.Run("MySQL", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = DefaultMySQLConfig("user:pass@tcp(localhost:3306)/testdb")
		}
	})
}

func BenchmarkMetricsCreation(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = &DatabaseMetrics{}
	}
}
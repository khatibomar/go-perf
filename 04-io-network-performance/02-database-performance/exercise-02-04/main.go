package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

// DatabaseType represents the type of database
type DatabaseType int

const (
	PostgreSQL DatabaseType = iota
	MySQL
)

// DatabaseConfig holds database-specific configuration
type DatabaseConfig struct {
	Type           DatabaseType
	DSN            string
	MaxOpenConns   int
	MaxIdleConns   int
	MaxLifetime    time.Duration
	MaxIdleTime    time.Duration
	ConnectTimeout time.Duration
	QueryTimeout   time.Duration
	CustomParams   map[string]string
}

// DatabaseMetrics holds performance metrics
type DatabaseMetrics struct {
	TotalQueries     int64
	SuccessfulQueries int64
	FailedQueries    int64
	TotalDuration    int64 // nanoseconds
	SlowQueries      int64
	ConnectionErrors int64
	ActiveConnections int64
	IdleConnections  int64
	mu               sync.RWMutex
	QueryTimes       []time.Duration
}

// OptimizedDatabase wraps database connections with optimizations
type OptimizedDatabase struct {
	db      *sql.DB
	pgxPool *pgxpool.Pool
	config  *DatabaseConfig
	metrics *DatabaseMetrics
	mu      sync.RWMutex
}

// NewOptimizedDatabase creates a new optimized database connection
func NewOptimizedDatabase(config *DatabaseConfig) (*OptimizedDatabase, error) {
	odb := &OptimizedDatabase{
		config:  config,
		metrics: &DatabaseMetrics{},
	}

	switch config.Type {
	case PostgreSQL:
		return odb.initPostgreSQL()
	case MySQL:
		return odb.initMySQL()
	default:
		return nil, fmt.Errorf("unsupported database type: %v", config.Type)
	}
}

// initPostgreSQL initializes PostgreSQL connection with pgx driver
func (odb *OptimizedDatabase) initPostgreSQL() (*OptimizedDatabase, error) {
	// Use pgxpool for better performance
	pgxConfig, err := pgxpool.ParseConfig(odb.config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PostgreSQL config: %w", err)
	}

	// Apply optimizations
	pgxConfig.MaxConns = int32(odb.config.MaxOpenConns)
	pgxConfig.MinConns = int32(odb.config.MaxIdleConns)
	pgxConfig.MaxConnLifetime = odb.config.MaxLifetime
	pgxConfig.MaxConnIdleTime = odb.config.MaxIdleTime
	pgxConfig.HealthCheckPeriod = 30 * time.Second

	// Custom connection parameters
	for key, value := range odb.config.CustomParams {
		pgxConfig.ConnConfig.RuntimeParams[key] = value
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), pgxConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostgreSQL pool: %w", err)
	}

	odb.pgxPool = pool

	// Also create standard sql.DB for compatibility
	db, err := sql.Open("postgres", odb.config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	odb.configureStandardDB(db)
	odb.db = db

	return odb, nil
}

// initMySQL initializes MySQL connection with optimizations
func (odb *OptimizedDatabase) initMySQL() (*OptimizedDatabase, error) {
	db, err := sql.Open("mysql", odb.config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	odb.configureStandardDB(db)
	odb.db = db

	return odb, nil
}

// configureStandardDB applies common database configurations
func (odb *OptimizedDatabase) configureStandardDB(db *sql.DB) {
	db.SetMaxOpenConns(odb.config.MaxOpenConns)
	db.SetMaxIdleConns(odb.config.MaxIdleConns)
	db.SetConnMaxLifetime(odb.config.MaxLifetime)
	db.SetConnMaxIdleTime(odb.config.MaxIdleTime)
}

// ExecuteQuery executes a query with performance monitoring
func (odb *OptimizedDatabase) ExecuteQuery(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	atomic.AddInt64(&odb.metrics.TotalQueries, 1)

	ctx, cancel := context.WithTimeout(ctx, odb.config.QueryTimeout)
	defer cancel()

	rows, err := odb.db.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	odb.recordMetrics(err, duration)
	return rows, err
}

// ExecuteQueryPgx executes a query using pgx driver (PostgreSQL only)
func (odb *OptimizedDatabase) ExecuteQueryPgx(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	if odb.pgxPool == nil {
		return nil, fmt.Errorf("pgx pool not available")
	}

	start := time.Now()
	atomic.AddInt64(&odb.metrics.TotalQueries, 1)

	ctx, cancel := context.WithTimeout(ctx, odb.config.QueryTimeout)
	defer cancel()

	rows, err := odb.pgxPool.Query(ctx, query, args...)
	duration := time.Since(start)

	odb.recordMetrics(err, duration)
	return rows, err
}

// BulkInsertPostgreSQL performs optimized bulk insert using COPY
func (odb *OptimizedDatabase) BulkInsertPostgreSQL(ctx context.Context, tableName string, columns []string, rows [][]interface{}) error {
	if odb.pgxPool == nil {
		return fmt.Errorf("pgx pool not available")
	}

	start := time.Now()
	atomic.AddInt64(&odb.metrics.TotalQueries, 1)

	_, err := odb.pgxPool.CopyFrom(
		ctx,
		pgx.Identifier{tableName},
		columns,
		pgx.CopyFromRows(rows),
	)

	duration := time.Since(start)
	odb.recordMetrics(err, duration)

	return err
}

// BulkInsertMySQL performs optimized bulk insert using multi-value INSERT
func (odb *OptimizedDatabase) BulkInsertMySQL(ctx context.Context, tableName string, columns []string, rows [][]interface{}) error {
	if len(rows) == 0 {
		return nil
	}

	start := time.Now()
	atomic.AddInt64(&odb.metrics.TotalQueries, 1)

	// Build bulk insert query
	valueStrings := make([]string, 0, len(rows))
	valueArgs := make([]interface{}, 0, len(rows)*len(columns))

	for _, row := range rows {
		placeholders := make([]string, len(columns))
		for i := range columns {
			placeholders[i] = "?"
		}
		valueStrings = append(valueStrings, fmt.Sprintf("(%s)", strings.Join(placeholders, ",")))
		valueArgs = append(valueArgs, row...)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		tableName,
		strings.Join(columns, ","),
		strings.Join(valueStrings, ","))

	_, err := odb.db.ExecContext(ctx, query, valueArgs...)
	duration := time.Since(start)
	odb.recordMetrics(err, duration)

	return err
}

// recordMetrics records performance metrics
func (odb *OptimizedDatabase) recordMetrics(err error, duration time.Duration) {
	if err != nil {
		atomic.AddInt64(&odb.metrics.FailedQueries, 1)
	} else {
		atomic.AddInt64(&odb.metrics.SuccessfulQueries, 1)
	}

	atomic.AddInt64(&odb.metrics.TotalDuration, int64(duration))

	// Track slow queries (>100ms)
	if duration > 100*time.Millisecond {
		atomic.AddInt64(&odb.metrics.SlowQueries, 1)
	}

	// Store recent query times for analysis
	odb.metrics.mu.Lock()
	odb.metrics.QueryTimes = append(odb.metrics.QueryTimes, duration)
	if len(odb.metrics.QueryTimes) > 1000 {
		odb.metrics.QueryTimes = odb.metrics.QueryTimes[1:]
	}
	odb.metrics.mu.Unlock()
}

// GetMetrics returns current performance metrics
func (odb *OptimizedDatabase) GetMetrics() DatabaseMetrics {
	odb.metrics.mu.RLock()
	defer odb.metrics.mu.RUnlock()

	metrics := *odb.metrics
	metrics.QueryTimes = make([]time.Duration, len(odb.metrics.QueryTimes))
	copy(metrics.QueryTimes, odb.metrics.QueryTimes)

	// Update connection stats
	if odb.db != nil {
		stats := odb.db.Stats()
		metrics.ActiveConnections = int64(stats.InUse)
		metrics.IdleConnections = int64(stats.Idle)
	}

	return metrics
}

// GetAverageQueryTime calculates average query execution time
func (odb *OptimizedDatabase) GetAverageQueryTime() time.Duration {
	totalQueries := atomic.LoadInt64(&odb.metrics.TotalQueries)
	if totalQueries == 0 {
		return 0
	}

	totalDuration := atomic.LoadInt64(&odb.metrics.TotalDuration)
	return time.Duration(totalDuration / totalQueries)
}

// GetSuccessRate calculates query success rate
func (odb *OptimizedDatabase) GetSuccessRate() float64 {
	totalQueries := atomic.LoadInt64(&odb.metrics.TotalQueries)
	if totalQueries == 0 {
		return 0
	}

	successfulQueries := atomic.LoadInt64(&odb.metrics.SuccessfulQueries)
	return float64(successfulQueries) / float64(totalQueries) * 100
}

// HealthCheck performs database health check
func (odb *OptimizedDatabase) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return odb.db.PingContext(ctx)
}

// Close closes database connections
func (odb *OptimizedDatabase) Close() error {
	var err error
	if odb.pgxPool != nil {
		odb.pgxPool.Close()
	}
	if odb.db != nil {
		err = odb.db.Close()
	}
	return err
}

// DefaultPostgreSQLConfig returns optimized PostgreSQL configuration
func DefaultPostgreSQLConfig(dsn string) *DatabaseConfig {
	return &DatabaseConfig{
		Type:           PostgreSQL,
		DSN:            dsn,
		MaxOpenConns:   30,
		MaxIdleConns:   5,
		MaxLifetime:    time.Hour,
		MaxIdleTime:    30 * time.Minute,
		ConnectTimeout: 10 * time.Second,
		QueryTimeout:   30 * time.Second,
		CustomParams: map[string]string{
			"application_name": "optimized-app",
			"statement_timeout": "30000",
			"lock_timeout":      "10000",
		},
	}
}

// DefaultMySQLConfig returns optimized MySQL configuration
func DefaultMySQLConfig(dsn string) *DatabaseConfig {
	return &DatabaseConfig{
		Type:           MySQL,
		DSN:            dsn,
		MaxOpenConns:   25,
		MaxIdleConns:   5,
		MaxLifetime:    5 * time.Minute,
		MaxIdleTime:    2 * time.Minute,
		ConnectTimeout: 10 * time.Second,
		QueryTimeout:   30 * time.Second,
		CustomParams:   map[string]string{},
	}
}

func main() {
	fmt.Println("Database-Specific Optimizations Demo")
	fmt.Println("====================================")

	// PostgreSQL example
	fmt.Println("\n1. PostgreSQL Optimizations:")
	pgConfig := DefaultPostgreSQLConfig("postgres://user:pass@localhost/testdb?sslmode=disable")
	fmt.Printf("   - Max Open Connections: %d\n", pgConfig.MaxOpenConns)
	fmt.Printf("   - Max Idle Connections: %d\n", pgConfig.MaxIdleConns)
	fmt.Printf("   - Connection Max Lifetime: %v\n", pgConfig.MaxLifetime)
	fmt.Printf("   - Custom Parameters: %v\n", pgConfig.CustomParams)

	// MySQL example
	fmt.Println("\n2. MySQL Optimizations:")
	mysqlConfig := DefaultMySQLConfig("user:pass@tcp(localhost:3306)/testdb?parseTime=true&interpolateParams=true")
	fmt.Printf("   - Max Open Connections: %d\n", mysqlConfig.MaxOpenConns)
	fmt.Printf("   - Max Idle Connections: %d\n", mysqlConfig.MaxIdleConns)
	fmt.Printf("   - Connection Max Lifetime: %v\n", mysqlConfig.MaxLifetime)

	fmt.Println("\n3. Performance Features:")
	fmt.Println("   - Connection pool monitoring")
	fmt.Println("   - Query performance metrics")
	fmt.Println("   - Bulk insert optimizations")
	fmt.Println("   - Health check monitoring")
	fmt.Println("   - Database-specific driver optimizations")

	fmt.Println("\n4. Usage Examples:")
	fmt.Println("   // PostgreSQL with pgx driver")
	fmt.Println("   db, err := NewOptimizedDatabase(pgConfig)")
	fmt.Println("   rows, err := db.ExecuteQueryPgx(ctx, query, args...)")
	fmt.Println("   err = db.BulkInsertPostgreSQL(ctx, table, columns, data)")
	fmt.Println("")
	fmt.Println("   // MySQL with optimized bulk insert")
	fmt.Println("   db, err := NewOptimizedDatabase(mysqlConfig)")
	fmt.Println("   err = db.BulkInsertMySQL(ctx, table, columns, data)")
	fmt.Println("")
	fmt.Println("   // Performance monitoring")
	fmt.Println("   metrics := db.GetMetrics()")
	fmt.Println("   avgTime := db.GetAverageQueryTime()")
	fmt.Println("   successRate := db.GetSuccessRate()")

	log.Println("Demo completed. Use with actual database connections for testing.")
}
package main

import (
	"database/sql"
	"fmt"
	"reflect"
	"sync"
)

// CachedRows represents cached query results
type CachedRows struct {
	columns []string
	rows    [][]interface{}
	current int
}

// NewCachedRows creates a new CachedRows from sql.Rows
func NewCachedRows(rows *sql.Rows) (*CachedRows, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	cached := &CachedRows{
		columns: columns,
		rows:    make([][]interface{}, 0),
		current: -1,
	}

	// Read all rows into memory
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		cached.rows = append(cached.rows, values)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
		}

	return cached, nil
}

// ToSQLRows converts CachedRows back to sql.Rows interface
func (c *CachedRows) ToSQLRows() *sql.Rows {
	// Reset current position
	c.current = -1
	// This is a simplified implementation
	// In a real scenario, you'd need a more sophisticated approach
	return &sql.Rows{}
}

// MockRows implements a mock sql.Rows for testing
type MockRows struct {
	columns []string
	rows    [][]interface{}
	current int
	closed  bool
}

// NewMockRows creates a new MockRows
func NewMockRows(columns []string, rows [][]interface{}) *MockRows {
	return &MockRows{
		columns: columns,
		rows:    rows,
		current: -1,
		closed:  false,
	}
}

// Columns returns the column names
func (m *MockRows) Columns() ([]string, error) {
	if m.closed {
		return nil, fmt.Errorf("rows are closed")
	}
	return m.columns, nil
}

// Close closes the rows
func (m *MockRows) Close() error {
	m.closed = true
	return nil
}

// Next advances to the next row
func (m *MockRows) Next() bool {
	if m.closed {
		return false
	}
	m.current++
	return m.current < len(m.rows)
}

// Scan copies the columns in the current row into the values
func (m *MockRows) Scan(dest ...interface{}) error {
	if m.closed {
		return fmt.Errorf("rows are closed")
	}
	if m.current < 0 || m.current >= len(m.rows) {
		return fmt.Errorf("no current row")
	}

	row := m.rows[m.current]
	if len(dest) != len(row) {
		return fmt.Errorf("expected %d arguments, got %d", len(row), len(dest))
	}

	for i, val := range row {
		if err := convertAssign(dest[i], val); err != nil {
			return err
		}
	}

	return nil
}

// Err returns any error that occurred during iteration
func (m *MockRows) Err() error {
	return nil
}

// convertAssign is a simplified version of sql driver's convertAssign
func convertAssign(dest, src interface{}) error {
	switch d := dest.(type) {
	case *string:
		switch s := src.(type) {
		case string:
			*d = s
		case []byte:
			*d = string(s)
		case nil:
			*d = ""
		default:
			*d = fmt.Sprintf("%v", s)
		}
	case *int:
		switch s := src.(type) {
		case int:
			*d = s
		case int64:
			*d = int(s)
		case nil:
			*d = 0
		default:
			return fmt.Errorf("cannot convert %T to int", src)
		}
	case *int64:
		switch s := src.(type) {
		case int64:
			*d = s
		case int:
			*d = int64(s)
		case nil:
			*d = 0
		default:
			return fmt.Errorf("cannot convert %T to int64", src)
		}
	case *bool:
		switch s := src.(type) {
		case bool:
			*d = s
		case nil:
			*d = false
		default:
			return fmt.Errorf("cannot convert %T to bool", src)
		}
	case *interface{}:
		*d = src
	default:
		// Use reflection for other types
		dv := reflect.ValueOf(dest)
		if dv.Kind() != reflect.Ptr {
			return fmt.Errorf("destination not a pointer")
		}
		dv = dv.Elem()
		sv := reflect.ValueOf(src)
		if sv.Type().ConvertibleTo(dv.Type()) {
			dv.Set(sv.Convert(dv.Type()))
		} else {
			return fmt.Errorf("cannot convert %T to %T", src, dest)
		}
	}
	return nil
}

// ResultCache provides a simple interface for caching query results
type ResultCache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Delete(key string)
	Clear()
	Size() int
}

// MemoryCache implements ResultCache using in-memory storage
type MemoryCache struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

// NewMemoryCache creates a new MemoryCache
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		data: make(map[string]interface{}),
	}
}

// Get retrieves a value from the cache
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, exists := c.data[key]
	return val, exists
}

// Set stores a value in the cache
func (c *MemoryCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

// Delete removes a value from the cache
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// Clear removes all values from the cache
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]interface{})
}

// Size returns the number of items in the cache
func (c *MemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}
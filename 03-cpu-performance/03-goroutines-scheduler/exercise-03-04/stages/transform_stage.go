package stages

import (
	"time"
)

// TransformFunc defines a transformation function
type TransformFunc func(interface{}) (interface{}, error)

// TransformStage applies a transformation to input data
type TransformStage struct {
	*BaseStage
	transformFunc TransformFunc
}

// NewTransformStage creates a new transform stage
func NewTransformStage(name string, transformFunc TransformFunc) *TransformStage {
	return &TransformStage{
		BaseStage:     NewBaseStage(name),
		transformFunc: transformFunc,
	}
}

// Process applies the transformation function to the input data
func (s *TransformStage) Process(data interface{}) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		s.UpdateMetrics(duration, true)
	}()
	
	if s.transformFunc == nil {
		return data, nil
	}
	
	return s.transformFunc(data)
}

// MultiplyStage multiplies numeric values by a factor
type MultiplyStage struct {
	*BaseStage
	factor float64
}

// NewMultiplyStage creates a stage that multiplies numbers by a factor
func NewMultiplyStage(name string, factor float64) *MultiplyStage {
	return &MultiplyStage{
		BaseStage: NewBaseStage(name),
		factor:    factor,
	}
}

// Process multiplies the input by the factor
func (s *MultiplyStage) Process(data interface{}) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		s.UpdateMetrics(duration, true)
	}()
	
	switch v := data.(type) {
	case int:
		return int(float64(v) * s.factor), nil
	case int64:
		return int64(float64(v) * s.factor), nil
	case float64:
		return v * s.factor, nil
	case float32:
		return float32(float64(v) * s.factor), nil
	default:
		return data, nil
	}
}

// StringTransformStage transforms strings
type StringTransformStage struct {
	*BaseStage
	transformFunc func(string) string
}

// NewStringTransformStage creates a string transformation stage
func NewStringTransformStage(name string, transformFunc func(string) string) *StringTransformStage {
	return &StringTransformStage{
		BaseStage:     NewBaseStage(name),
		transformFunc: transformFunc,
	}
}

// Process transforms string data
func (s *StringTransformStage) Process(data interface{}) (interface{}, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		s.UpdateMetrics(duration, true)
	}()
	
	if str, ok := data.(string); ok {
		return s.transformFunc(str), nil
	}
	return data, nil
}
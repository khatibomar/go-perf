package service

import (
	"exercise-01-02/data"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Analyzer performs complex data analysis operations
type Analyzer struct {
	statistics    *StatisticsEngine
	patternEngine *PatternEngine
	predictive    *PredictiveEngine
	cache         map[string]*AnalysisResult
	stats         *AnalyzerStats
}

// AnalysisResult contains the results of data analysis
type AnalysisResult struct {
	Statistics      *StatisticalAnalysis
	Patterns        *PatternAnalysis
	Predictions     *PredictiveAnalysis
	Complexity      *ComplexityAnalysis
	Performance     *PerformanceAnalysis
	Recommendations []string
	ConfidenceScore float64
	AnalyzedAt      time.Time
	AnalysisTime    time.Duration
}

// StatisticalAnalysis contains statistical metrics
type StatisticalAnalysis struct {
	Mean         float64
	Median       float64
	Mode         []int
	StandardDev  float64
	Variance     float64
	Skewness     float64
	Kurtosis     float64
	Range        int
	Quartiles    [4]float64
	Outliers     []int
	Distribution string
	Correlations map[string]float64
}

// PatternAnalysis contains pattern detection results
type PatternAnalysis struct {
	Sequences      []Sequence
	Cycles         []Cycle
	Trends         []Trend
	Anomalies      []Anomaly
	Frequencies    map[int]int
	Entropy        float64
	Complexity     float64
	Predictability float64
}

// PredictiveAnalysis contains predictive modeling results
type PredictiveAnalysis struct {
	NextValues     []int
	TrendDirection string
	Confidence     float64
	Seasonality    *SeasonalityInfo
	Forecasts      []Forecast
	ModelAccuracy  float64
}

// ComplexityAnalysis contains complexity metrics
type ComplexityAnalysis struct {
	TimeComplexity        string
	SpaceComplexity       string
	CyclomaticComplexity  int
	DataComplexity        float64
	AlgorithmicComplexity float64
}

// PerformanceAnalysis contains performance metrics
type PerformanceAnalysis struct {
	ProcessingTime   time.Duration
	MemoryUsage      int64
	CPUUtilization   float64
	Throughput       float64
	Bottlenecks      []string
	OptimizationTips []string
}

// Supporting structures
type Sequence struct {
	Pattern []int
	Count   int
	Length  int
}

type Cycle struct {
	Period    int
	Amplitude float64
	Phase     float64
}

type Trend struct {
	Direction string
	Slope     float64
	Strength  float64
}

type Anomaly struct {
	Index     int
	Value     int
	Expected  float64
	Deviation float64
	Severity  string
}

type SeasonalityInfo struct {
	Period   int
	Strength float64
	Detected bool
}

type Forecast struct {
	TimeStep   int
	Value      float64
	Confidence float64
	Range      [2]float64
}

// StatisticsEngine handles statistical computations
type StatisticsEngine struct {
	calculationCache map[string]interface{}
	precision        int
}

// PatternEngine handles pattern detection
type PatternEngine struct {
	patternCache     map[string][]Sequence
	minPatternLength int
	maxPatternLength int
}

// PredictiveEngine handles predictive modeling
type PredictiveEngine struct {
	models              map[string]PredictiveModel
	historySize         int
	confidenceThreshold float64
}

// PredictiveModel interface for different prediction models
type PredictiveModel interface {
	Predict(data []int) []int
	GetAccuracy() float64
	Train(data []int)
}

// AnalyzerStats tracks analyzer performance
type AnalyzerStats struct {
	TotalAnalyses int64
	AverageTime   time.Duration
	CacheHitRate  float64
	ErrorRate     float64
}

// NewAnalyzer creates a new analyzer instance
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		statistics:    NewStatisticsEngine(),
		patternEngine: NewPatternEngine(),
		predictive:    NewPredictiveEngine(),
		cache:         make(map[string]*AnalysisResult),
		stats:         &AnalyzerStats{},
	}
}

// NewStatisticsEngine creates a new statistics engine
func NewStatisticsEngine() *StatisticsEngine {
	return &StatisticsEngine{
		calculationCache: make(map[string]interface{}),
		precision:        6,
	}
}

// NewPatternEngine creates a new pattern engine
func NewPatternEngine() *PatternEngine {
	return &PatternEngine{
		patternCache:     make(map[string][]Sequence),
		minPatternLength: 2,
		maxPatternLength: 10,
	}
}

// NewPredictiveEngine creates a new predictive engine
func NewPredictiveEngine() *PredictiveEngine {
	return &PredictiveEngine{
		models:              make(map[string]PredictiveModel),
		historySize:         100,
		confidenceThreshold: 0.7,
	}
}

// Predict performs predictive analysis on the data
func (pe *PredictiveEngine) Predict(data []int, count int) *PredictiveAnalysis {
	if len(data) == 0 {
		return &PredictiveAnalysis{}
	}

	// Simple prediction based on trend analysis
	nextValues := make([]int, count)
	if len(data) >= 2 {
		// Calculate simple linear trend
		lastValue := data[len(data)-1]
		secondLastValue := data[len(data)-2]
		trend := lastValue - secondLastValue
		
		for i := 0; i < count; i++ {
			nextValues[i] = lastValue + trend*(i+1)
		}
	} else {
		// If insufficient data, use last value
		lastValue := data[len(data)-1]
		for i := 0; i < count; i++ {
			nextValues[i] = lastValue
		}
	}

	// Determine trend direction
	trendDirection := "stable"
	if len(data) >= 2 {
		if data[len(data)-1] > data[len(data)-2] {
			trendDirection = "increasing"
		} else if data[len(data)-1] < data[len(data)-2] {
			trendDirection = "decreasing"
		}
	}

	return &PredictiveAnalysis{
		NextValues:     nextValues,
		TrendDirection: trendDirection,
		Confidence:     0.7,
		ModelAccuracy:  0.8,
	}
}

// AnalyzeData performs comprehensive data analysis
func (a *Analyzer) AnalyzeData(inputData interface{}) *AnalysisResult {
	start := time.Now()
	defer func() {
		a.stats.TotalAnalyses++
		a.stats.AverageTime = time.Duration(int64(a.stats.AverageTime)*int64(a.stats.TotalAnalyses-1)+int64(time.Since(start))) / time.Duration(a.stats.TotalAnalyses)
	}()

	// Convert data to analyzable format
	numericData := a.extractNumericData(inputData)
	if len(numericData) == 0 {
		// Generate sample data for analysis
		dataGen := data.NewDataGenerator()
		intValues := dataGen.GenerateValues(100, "integer")
		numericData = make([]int, len(intValues))
		for i, v := range intValues {
			numericData[i] = v.(int)
		}
	}

	// Check cache
	cacheKey := a.generateCacheKey(numericData)
	if cached := a.getCachedResult(cacheKey); cached != nil {
		a.stats.CacheHitRate = (a.stats.CacheHitRate*float64(a.stats.TotalAnalyses-1) + 1.0) / float64(a.stats.TotalAnalyses)
		return cached
	}

	// Perform analysis
	statistical := a.statistics.AnalyzeStatistics(numericData)
	patterns := a.patternEngine.DetectPatterns(numericData)
	predictiveResult := a.predictive.Predict(numericData, 10)
	complexity := a.analyzeComplexity(numericData)
	performance := a.analyzePerformance(numericData, start)

	// Generate recommendations
	recommendations := a.generateRecommendations(statistical, patterns, predictiveResult)

	// Calculate confidence score
	confidenceScore := a.calculateConfidenceScore(statistical, patterns, predictiveResult)

	result := &AnalysisResult{
		Statistics:      statistical,
		Patterns:        patterns,
		Predictions:     predictiveResult,
		Complexity:      complexity,
		Performance:     performance,
		Recommendations: recommendations,
		ConfidenceScore: confidenceScore,
		AnalyzedAt:      time.Now(),
		AnalysisTime:    time.Since(start),
	}

	// Cache result
	a.cacheResult(cacheKey, result)

	return result
}

// extractNumericData extracts numeric data from various input types
func (a *Analyzer) extractNumericData(inputData interface{}) []int {
	switch d := inputData.(type) {
	case []int:
		return d
	case []float64:
		result := make([]int, len(d))
		for i, v := range d {
			result[i] = int(v)
		}
		return result
	case map[string]interface{}:
		// Extract numbers from map
		numbers := make([]int, 0)
		a.extractNumbersFromMap(d, &numbers)
		return numbers
	case string:
		// Convert string to numeric representation
		return a.stringToNumbers(d)
	default:
		// Generate sample data
		dataGen := data.NewDataGenerator()
		intValues := dataGen.GenerateValues(50, "integer")
		result := make([]int, len(intValues))
		for i, v := range intValues {
			result[i] = v.(int)
		}
		return result
	}
}

// extractNumbersFromMap recursively extracts numbers from a map
func (a *Analyzer) extractNumbersFromMap(m map[string]interface{}, numbers *[]int) {
	for _, value := range m {
		switch v := value.(type) {
		case int:
			*numbers = append(*numbers, v)
		case float64:
			*numbers = append(*numbers, int(v))
		case map[string]interface{}:
			a.extractNumbersFromMap(v, numbers)
		case []interface{}:
			for _, item := range v {
				if num, ok := item.(int); ok {
					*numbers = append(*numbers, num)
				} else if num, ok := item.(float64); ok {
					*numbers = append(*numbers, int(num))
				}
			}
		}
	}
}

// stringToNumbers converts string to numeric representation
func (a *Analyzer) stringToNumbers(s string) []int {
	numbers := make([]int, 0)

	// Convert each character to its ASCII value
	for _, char := range s {
		numbers = append(numbers, int(char))
	}

	// If string is too short, pad with derived values
	if len(numbers) < 20 {
		for i := len(numbers); i < 20; i++ {
			// Generate derived values based on string properties
			value := (len(s)*i + strings.Count(s, string(rune(i%128)))) % 1000
			numbers = append(numbers, value)
		}
	}

	return numbers
}

// AnalyzeStatistics performs statistical analysis
func (se *StatisticsEngine) AnalyzeStatistics(data []int) *StatisticalAnalysis {
	if len(data) == 0 {
		return &StatisticalAnalysis{}
	}

	// Convert to float64 for calculations
	floatData := make([]float64, len(data))
	for i, v := range data {
		floatData[i] = float64(v)
	}

	// Sort data for median and quartile calculations
	sortedData := make([]float64, len(floatData))
	copy(sortedData, floatData)
	sort.Float64s(sortedData)

	// Calculate basic statistics
	mean := se.calculateMean(floatData)
	median := se.calculateMedian(sortedData)
	mode := se.calculateMode(data)
	stdDev := se.calculateStandardDeviation(floatData, mean)
	variance := stdDev * stdDev
	skewness := se.calculateSkewness(floatData, mean, stdDev)
	kurtosis := se.calculateKurtosis(floatData, mean, stdDev)
	dataRange := data[len(sortedData)-1] - data[0]
	quartiles := se.calculateQuartiles(sortedData)
	outliers := se.detectOutliers(data, mean, stdDev)
	distribution := se.identifyDistribution(floatData, mean, stdDev)
	correlations := se.calculateCorrelations(floatData)

	return &StatisticalAnalysis{
		Mean:         mean,
		Median:       median,
		Mode:         mode,
		StandardDev:  stdDev,
		Variance:     variance,
		Skewness:     skewness,
		Kurtosis:     kurtosis,
		Range:        dataRange,
		Quartiles:    quartiles,
		Outliers:     outliers,
		Distribution: distribution,
		Correlations: correlations,
	}
}

// Statistical calculation methods

func (se *StatisticsEngine) calculateMean(data []float64) float64 {
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func (se *StatisticsEngine) calculateMedian(sortedData []float64) float64 {
	n := len(sortedData)
	if n%2 == 0 {
		return (sortedData[n/2-1] + sortedData[n/2]) / 2
	}
	return sortedData[n/2]
}

func (se *StatisticsEngine) calculateMode(data []int) []int {
	frequency := make(map[int]int)
	for _, v := range data {
		frequency[v]++
	}

	maxFreq := 0
	for _, freq := range frequency {
		if freq > maxFreq {
			maxFreq = freq
		}
	}

	modes := make([]int, 0)
	for value, freq := range frequency {
		if freq == maxFreq {
			modes = append(modes, value)
		}
	}

	sort.Ints(modes)
	return modes
}

func (se *StatisticsEngine) calculateStandardDeviation(data []float64, mean float64) float64 {
	sum := 0.0
	for _, v := range data {
		diff := v - mean
		sum += diff * diff
	}
	return math.Sqrt(sum / float64(len(data)))
}

func (se *StatisticsEngine) calculateSkewness(data []float64, mean, stdDev float64) float64 {
	if stdDev == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range data {
		normalized := (v - mean) / stdDev
		sum += normalized * normalized * normalized
	}
	return sum / float64(len(data))
}

func (se *StatisticsEngine) calculateKurtosis(data []float64, mean, stdDev float64) float64 {
	if stdDev == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range data {
		normalized := (v - mean) / stdDev
		sum += normalized * normalized * normalized * normalized
	}
	return (sum / float64(len(data))) - 3 // Excess kurtosis
}

func (se *StatisticsEngine) calculateQuartiles(sortedData []float64) [4]float64 {
	return [4]float64{
		sortedData[0],                          // Min (Q0)
		se.calculatePercentile(sortedData, 25), // Q1
		se.calculatePercentile(sortedData, 50), // Q2 (Median)
		se.calculatePercentile(sortedData, 75), // Q3
	}
}

func (se *StatisticsEngine) calculatePercentile(sortedData []float64, percentile float64) float64 {
	n := len(sortedData)
	index := (percentile / 100.0) * float64(n-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sortedData[lower]
	}

	weight := index - float64(lower)
	return sortedData[lower]*(1-weight) + sortedData[upper]*weight
}

func (se *StatisticsEngine) detectOutliers(data []int, mean, stdDev float64) []int {
	outliers := make([]int, 0)
	threshold := 2.0 * stdDev

	for _, v := range data {
		if math.Abs(float64(v)-mean) > threshold {
			outliers = append(outliers, v)
		}
	}

	return outliers
}

func (se *StatisticsEngine) identifyDistribution(data []float64, mean, stdDev float64) string {
	// Simple distribution identification based on skewness and kurtosis
	skewness := se.calculateSkewness(data, mean, stdDev)
	kurtosis := se.calculateKurtosis(data, mean, stdDev)

	if math.Abs(skewness) < 0.5 && math.Abs(kurtosis) < 0.5 {
		return "normal"
	} else if skewness > 1.0 {
		return "right-skewed"
	} else if skewness < -1.0 {
		return "left-skewed"
	} else if kurtosis > 1.0 {
		return "leptokurtic"
	} else if kurtosis < -1.0 {
		return "platykurtic"
	}
	return "unknown"
}

func (se *StatisticsEngine) calculateCorrelations(data []float64) map[string]float64 {
	correlations := make(map[string]float64)

	// Auto-correlation with lag-1
	if len(data) > 1 {
		correlations["lag1"] = se.calculateAutoCorrelation(data, 1)
	}

	// Auto-correlation with lag-2
	if len(data) > 2 {
		correlations["lag2"] = se.calculateAutoCorrelation(data, 2)
	}

	return correlations
}

func (se *StatisticsEngine) calculateAutoCorrelation(data []float64, lag int) float64 {
	if len(data) <= lag {
		return 0.0
	}

	n := len(data) - lag
	mean1 := se.calculateMean(data[:n])
	mean2 := se.calculateMean(data[lag:])

	numerator := 0.0
	denominator1 := 0.0
	denominator2 := 0.0

	for i := 0; i < n; i++ {
		diff1 := data[i] - mean1
		diff2 := data[i+lag] - mean2
		numerator += diff1 * diff2
		denominator1 += diff1 * diff1
		denominator2 += diff2 * diff2
	}

	denominator := math.Sqrt(denominator1 * denominator2)
	if denominator == 0 {
		return 0.0
	}

	return numerator / denominator
}

// DetectPatterns identifies patterns in the data
func (pe *PatternEngine) DetectPatterns(data []int) *PatternAnalysis {
	sequences := pe.findSequences(data)
	cycles := pe.detectCycles(data)
	trends := pe.analyzeTrends(data)
	anomalies := pe.detectAnomalies(data)
	frequencies := pe.calculateFrequencies(data)
	entropy := pe.calculateEntropy(frequencies)
	complexity := pe.calculateComplexity(data)
	predictability := pe.calculatePredictability(data)

	return &PatternAnalysis{
		Sequences:      sequences,
		Cycles:         cycles,
		Trends:         trends,
		Anomalies:      anomalies,
		Frequencies:    frequencies,
		Entropy:        entropy,
		Complexity:     complexity,
		Predictability: predictability,
	}
}

// Pattern detection methods

func (pe *PatternEngine) findSequences(data []int) []Sequence {
	sequences := make([]Sequence, 0)
	patternCounts := make(map[string]int)

	// Find repeating sequences of different lengths
	for length := pe.minPatternLength; length <= pe.maxPatternLength && length <= len(data)/2; length++ {
		for i := 0; i <= len(data)-length; i++ {
			pattern := data[i : i+length]
			patternStr := pe.intSliceToString(pattern)
			patternCounts[patternStr]++
		}
	}

	// Convert to sequences with count > 1
	for patternStr, count := range patternCounts {
		if count > 1 {
			pattern := pe.stringToIntSlice(patternStr)
			sequences = append(sequences, Sequence{
				Pattern: pattern,
				Count:   count,
				Length:  len(pattern),
			})
		}
	}

	// Sort by count (most frequent first)
	sort.Slice(sequences, func(i, j int) bool {
		return sequences[i].Count > sequences[j].Count
	})

	return sequences
}

func (pe *PatternEngine) detectCycles(data []int) []Cycle {
	cycles := make([]Cycle, 0)

	// Simple cycle detection using autocorrelation
	for period := 2; period <= len(data)/3; period++ {
		correlation := pe.calculatePeriodCorrelation(data, period)
		if correlation > 0.7 { // Threshold for cycle detection
			amplitude := pe.calculateAmplitude(data, period)
			phase := pe.calculatePhase(data, period)
			cycles = append(cycles, Cycle{
				Period:    period,
				Amplitude: amplitude,
				Phase:     phase,
			})
		}
	}

	return cycles
}

func (pe *PatternEngine) calculatePeriodCorrelation(data []int, period int) float64 {
	if len(data) < period*2 {
		return 0.0
	}

	correlationSum := 0.0
	count := 0

	for i := 0; i < len(data)-period; i++ {
		correlationSum += float64(data[i] * data[i+period])
		count++
	}

	if count == 0 {
		return 0.0
	}

	// Normalize correlation
	mean := pe.calculateMean(data)
	normalizedCorr := (correlationSum / float64(count)) / (mean * mean)
	return math.Min(normalizedCorr, 1.0)
}

func (pe *PatternEngine) calculateMean(data []int) float64 {
	sum := 0
	for _, v := range data {
		sum += v
	}
	return float64(sum) / float64(len(data))
}

func (pe *PatternEngine) calculateAmplitude(data []int, period int) float64 {
	if len(data) < period {
		return 0.0
	}

	max := float64(data[0])
	min := float64(data[0])

	for i := 0; i < period && i < len(data); i++ {
		val := float64(data[i])
		if val > max {
			max = val
		}
		if val < min {
			min = val
		}
	}

	return (max - min) / 2.0
}

func (pe *PatternEngine) calculatePhase(data []int, period int) float64 {
	// Simplified phase calculation
	if len(data) < period {
		return 0.0
	}

	// Find the index of maximum value in first period
	maxIndex := 0
	maxValue := data[0]

	for i := 1; i < period && i < len(data); i++ {
		if data[i] > maxValue {
			maxValue = data[i]
			maxIndex = i
		}
	}

	// Phase as fraction of period
	return float64(maxIndex) / float64(period) * 2 * math.Pi
}

func (pe *PatternEngine) analyzeTrends(data []int) []Trend {
	trends := make([]Trend, 0)

	// Analyze overall trend
	overallTrend := pe.calculateLinearTrend(data)
	trends = append(trends, overallTrend)

	// Analyze trends in segments
	segmentSize := len(data) / 4
	if segmentSize > 10 {
		for i := 0; i < 4; i++ {
			start := i * segmentSize
			end := start + segmentSize
			if end > len(data) {
				end = len(data)
			}
			segmentTrend := pe.calculateLinearTrend(data[start:end])
			trends = append(trends, segmentTrend)
		}
	}

	return trends
}

func (pe *PatternEngine) calculateLinearTrend(data []int) Trend {
	n := len(data)
	if n < 2 {
		return Trend{Direction: "insufficient_data", Slope: 0, Strength: 0}
	}

	// Calculate linear regression slope
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, y := range data {
		x := float64(i)
		sumX += x
		sumY += float64(y)
		sumXY += x * float64(y)
		sumXX += x * x
	}

	// Calculate slope
	nFloat := float64(n)
	slope := (nFloat*sumXY - sumX*sumY) / (nFloat*sumXX - sumX*sumX)

	// Determine direction
	direction := "stable"
	if slope > 0.1 {
		direction = "increasing"
	} else if slope < -0.1 {
		direction = "decreasing"
	}

	// Calculate strength (R-squared)
	meanY := sumY / nFloat
	ssTotal := 0.0
	ssRes := 0.0

	for i, y := range data {
		yPred := slope*float64(i) + (sumY-slope*sumX)/nFloat
		ssTotal += (float64(y) - meanY) * (float64(y) - meanY)
		ssRes += (float64(y) - yPred) * (float64(y) - yPred)
	}

	strength := 1.0 - (ssRes / ssTotal)
	if math.IsNaN(strength) || math.IsInf(strength, 0) {
		strength = 0.0
	}

	return Trend{
		Direction: direction,
		Slope:     slope,
		Strength:  math.Max(0, math.Min(1, strength)),
	}
}

func (pe *PatternEngine) detectAnomalies(data []int) []Anomaly {
	anomalies := make([]Anomaly, 0)

	// Calculate mean and standard deviation
	mean := pe.calculateMean(data)
	stdDev := pe.calculateStdDev(data, mean)

	// Detect outliers using z-score
	threshold := 2.0
	for i, value := range data {
		zScore := math.Abs(float64(value)-mean) / stdDev
		if zScore > threshold {
			severity := "mild"
			if zScore > 3.0 {
				severity = "severe"
			} else if zScore > 2.5 {
				severity = "moderate"
			}

			anomalies = append(anomalies, Anomaly{
				Index:     i,
				Value:     value,
				Expected:  mean,
				Deviation: zScore,
				Severity:  severity,
			})
		}
	}

	return anomalies
}

func (pe *PatternEngine) calculateStdDev(data []int, mean float64) float64 {
	sum := 0.0
	for _, v := range data {
		diff := float64(v) - mean
		sum += diff * diff
	}
	return math.Sqrt(sum / float64(len(data)))
}

func (pe *PatternEngine) calculateFrequencies(data []int) map[int]int {
	frequencies := make(map[int]int)
	for _, v := range data {
		frequencies[v]++
	}
	return frequencies
}

func (pe *PatternEngine) calculateEntropy(frequencies map[int]int) float64 {
	total := 0
	for _, freq := range frequencies {
		total += freq
	}

	if total == 0 {
		return 0.0
	}

	entropy := 0.0
	for _, freq := range frequencies {
		if freq > 0 {
			p := float64(freq) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (pe *PatternEngine) calculateComplexity(data []int) float64 {
	// Lempel-Ziv complexity approximation
	if len(data) == 0 {
		return 0.0
	}

	dictionary := make(map[string]bool)
	complexity := 0.0
	current := ""

	for _, v := range data {
		current += string(rune(v))
		if !dictionary[current] {
			dictionary[current] = true
			complexity++
			current = ""
		}
	}

	return complexity / float64(len(data))
}

func (pe *PatternEngine) calculatePredictability(data []int) float64 {
	if len(data) < 3 {
		return 0.0
	}

	// Calculate how well we can predict next value based on previous values
	correctPredictions := 0
	totalPredictions := 0

	for i := 2; i < len(data); i++ {
		// Simple prediction: next value = current + (current - previous)
		predicted := data[i-1] + (data[i-1] - data[i-2])
		actual := data[i]

		// Consider prediction correct if within 10% of actual value
		if actual != 0 {
			error := math.Abs(float64(predicted-actual)) / math.Abs(float64(actual))
			if error < 0.1 {
				correctPredictions++
			}
		} else if predicted == actual {
			correctPredictions++
		}

		totalPredictions++
	}

	if totalPredictions == 0 {
		return 0.0
	}

	return float64(correctPredictions) / float64(totalPredictions)
}

// Additional helper methods for analyzer

func (a *Analyzer) analyzeComplexity(data []int) *ComplexityAnalysis {
	// Simplified complexity analysis
	n := len(data)
	timeComplexity := "O(n)"
	spaceComplexity := "O(1)"
	cyclomaticComplexity := 1 // Base complexity

	// Adjust based on data characteristics
	if n > 1000 {
		timeComplexity = "O(n log n)"
	}
	if n > 10000 {
		spaceComplexity = "O(n)"
	}

	// Calculate data complexity
	uniqueValues := make(map[int]bool)
	for _, v := range data {
		uniqueValues[v] = true
	}
	dataComplexity := float64(len(uniqueValues)) / float64(n)

	// Calculate algorithmic complexity (simplified)
	algorithmicComplexity := dataComplexity * math.Log2(float64(n))

	return &ComplexityAnalysis{
		TimeComplexity:        timeComplexity,
		SpaceComplexity:       spaceComplexity,
		CyclomaticComplexity:  cyclomaticComplexity,
		DataComplexity:        dataComplexity,
		AlgorithmicComplexity: algorithmicComplexity,
	}
}

func (a *Analyzer) analyzePerformance(data []int, startTime time.Time) *PerformanceAnalysis {
	processingTime := time.Since(startTime)
	memoryUsage := int64(len(data) * 8) // Approximate memory usage
	cpuUtilization := 0.5               // Simulated CPU utilization
	throughput := float64(len(data)) / processingTime.Seconds()

	bottlenecks := make([]string, 0)
	optimizationTips := make([]string, 0)

	// Identify potential bottlenecks
	if len(data) > 10000 {
		bottlenecks = append(bottlenecks, "large_dataset")
		optimizationTips = append(optimizationTips, "Consider data sampling or parallel processing")
	}

	if processingTime > time.Millisecond*100 {
		bottlenecks = append(bottlenecks, "slow_processing")
		optimizationTips = append(optimizationTips, "Optimize algorithms or use caching")
	}

	return &PerformanceAnalysis{
		ProcessingTime:   processingTime,
		MemoryUsage:      memoryUsage,
		CPUUtilization:   cpuUtilization,
		Throughput:       throughput,
		Bottlenecks:      bottlenecks,
		OptimizationTips: optimizationTips,
	}
}

func (a *Analyzer) generateRecommendations(stats *StatisticalAnalysis, patterns *PatternAnalysis, predictions *PredictiveAnalysis) []string {
	recommendations := make([]string, 0)

	// Statistical recommendations
	if len(stats.Outliers) > 0 {
		recommendations = append(recommendations, "Consider investigating outliers in the data")
	}

	if stats.StandardDev > stats.Mean {
		recommendations = append(recommendations, "High variability detected - consider data normalization")
	}

	// Pattern recommendations
	if len(patterns.Sequences) > 0 {
		recommendations = append(recommendations, "Repeating patterns detected - consider pattern-based optimization")
	}

	if patterns.Entropy < 2.0 {
		recommendations = append(recommendations, "Low entropy suggests predictable data - compression may be effective")
	}

	// Predictive recommendations
	if predictions.Confidence > 0.8 {
		recommendations = append(recommendations, "High prediction confidence - consider implementing forecasting")
	}

	return recommendations
}

func (a *Analyzer) calculateConfidenceScore(stats *StatisticalAnalysis, patterns *PatternAnalysis, predictions *PredictiveAnalysis) float64 {
	// Combine various confidence metrics
	statConfidence := 1.0 - (stats.StandardDev / (stats.Mean + 1.0))
	patternConfidence := patterns.Predictability
	predictiveConfidence := predictions.Confidence

	// Weighted average
	overallConfidence := (statConfidence*0.3 + patternConfidence*0.3 + predictiveConfidence*0.4)

	return math.Max(0.0, math.Min(1.0, overallConfidence))
}

// Cache management methods

func (a *Analyzer) generateCacheKey(data []int) string {
	// Generate a simple hash of the data
	hash := 0
	for i, v := range data {
		hash = hash*31 + v + i
		if i > 100 { // Limit hash calculation for large datasets
			break
		}
	}
	return fmt.Sprintf("analysis-%d-%d", hash, len(data))
}

func (a *Analyzer) getCachedResult(key string) *AnalysisResult {
	if result, exists := a.cache[key]; exists {
		// Check if cache is still valid (10 minutes)
		if time.Since(result.AnalyzedAt) < time.Minute*10 {
			return result
		}
		// Remove expired cache
		delete(a.cache, key)
	}
	return nil
}

func (a *Analyzer) cacheResult(key string, result *AnalysisResult) {
	a.cache[key] = result

	// Cleanup old cache entries if cache is too large
	if len(a.cache) > 50 {
		a.cleanupCache()
	}
}

func (a *Analyzer) cleanupCache() {
	now := time.Now()
	for key, result := range a.cache {
		if now.Sub(result.AnalyzedAt) > time.Minute*10 {
			delete(a.cache, key)
		}
	}
}

// GetStats returns analyzer statistics
func (a *Analyzer) GetStats() *AnalyzerStats {
	return a.stats
}

// Helper functions for pattern engine
func (pe *PatternEngine) intSliceToString(slice []int) string {
	result := ""
	for i, v := range slice {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%d", v)
	}
	return result
}

func (pe *PatternEngine) stringToIntSlice(s string) []int {
	if s == "" {
		return []int{}
	}
	parts := strings.Split(s, ",")
	result := make([]int, len(parts))
	for i, part := range parts {
		if val, err := strconv.Atoi(part); err == nil {
			result[i] = val
		}
	}
	return result
}

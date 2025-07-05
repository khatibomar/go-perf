package main

import (
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"time"
	"unsafe"
)

// AlignmentAnalyzer provides comprehensive struct alignment analysis
type AlignmentAnalyzer struct {
	cacheLineSize int
	wordSize      int
}

// NewAlignmentAnalyzer creates a new alignment analyzer
func NewAlignmentAnalyzer() *AlignmentAnalyzer {
	return &AlignmentAnalyzer{
		cacheLineSize: 64, // Common cache line size
		wordSize:      int(unsafe.Sizeof(uintptr(0))),
	}
}

// FieldInfo contains detailed field information
type FieldInfo struct {
	Name         string
	Type         string
	Size         int
	Alignment    int
	Offset       int
	Padding      int
	CacheLinePos int     // Position within cache line
	AccessFreq   float64 // Access frequency (0-1)
}

// StructInfo contains comprehensive struct analysis
type StructInfo struct {
	Name            string
	Size            int
	Alignment       int
	Fields          []FieldInfo
	TotalPadding    int
	Efficiency      float64
	CacheLineSpan   int  // Number of cache lines spanned
	CacheLineWaste  int  // Wasted bytes in cache lines
	HotFieldsSpread bool // Whether hot fields span multiple cache lines
}

// AnalyzeStruct performs comprehensive struct analysis
func (aa *AlignmentAnalyzer) AnalyzeStruct(structType interface{}, accessPattern map[string]float64) StructInfo {
	t := reflect.TypeOf(structType)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	info := StructInfo{
		Name:      t.Name(),
		Size:      int(t.Size()),
		Alignment: int(t.Align()),
		Fields:    make([]FieldInfo, t.NumField()),
	}

	var totalFieldSize int
	var lastOffset int

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		offset := int(field.Offset)

		// Calculate padding before this field
		padding := 0
		if i > 0 {
			expectedOffset := lastOffset + int(t.Field(i-1).Type.Size())
			padding = offset - expectedOffset
		}

		accessFreq := 0.0
		if accessPattern != nil {
			accessFreq = accessPattern[field.Name]
		}

		info.Fields[i] = FieldInfo{
			Name:         field.Name,
			Type:         field.Type.String(),
			Size:         int(field.Type.Size()),
			Alignment:    int(field.Type.Align()),
			Offset:       offset,
			Padding:      padding,
			CacheLinePos: offset % aa.cacheLineSize,
			AccessFreq:   accessFreq,
		}

		totalFieldSize += int(field.Type.Size())
		info.TotalPadding += padding
		lastOffset = offset
	}

	// Calculate final padding
	finalPadding := info.Size - (lastOffset + int(t.Field(t.NumField()-1).Type.Size()))
	info.TotalPadding += finalPadding

	// Calculate efficiency
	if info.Size > 0 {
		info.Efficiency = float64(totalFieldSize) / float64(info.Size)
	}

	// Calculate cache line metrics
	info.CacheLineSpan = (info.Size + aa.cacheLineSize - 1) / aa.cacheLineSize
	info.CacheLineWaste = (info.CacheLineSpan * aa.cacheLineSize) - info.Size

	// Check if hot fields span multiple cache lines
	info.HotFieldsSpread = aa.checkHotFieldsSpread(info.Fields)

	return info
}

// checkHotFieldsSpread checks if frequently accessed fields span multiple cache lines
func (aa *AlignmentAnalyzer) checkHotFieldsSpread(fields []FieldInfo) bool {
	hotFields := make([]FieldInfo, 0)
	for _, field := range fields {
		if field.AccessFreq > 0.7 { // Consider >70% access frequency as "hot"
			hotFields = append(hotFields, field)
		}
	}

	if len(hotFields) <= 1 {
		return false
	}

	// Check if hot fields span multiple cache lines
	firstCacheLine := hotFields[0].Offset / aa.cacheLineSize
	lastCacheLine := (hotFields[len(hotFields)-1].Offset + hotFields[len(hotFields)-1].Size - 1) / aa.cacheLineSize

	return lastCacheLine > firstCacheLine
}

// OptimizationStrategy represents different optimization approaches
type OptimizationStrategy int

const (
	MinimizeSize OptimizationStrategy = iota
	MinimizePadding
	OptimizeCacheLocality
	OptimizeAccessPattern
	Balanced
)

// OptimizationResult contains optimization results
type OptimizationResult struct {
	Strategy         OptimizationStrategy
	OriginalInfo     StructInfo
	OptimizedInfo    StructInfo
	SizeSaved        int
	PaddingSaved     int
	CacheImprovement bool
	OptimizedFields  []FieldInfo
	Recommendations  []string
}

// OptimizeStruct optimizes struct layout using the specified strategy
func (aa *AlignmentAnalyzer) OptimizeStruct(info StructInfo, strategy OptimizationStrategy) OptimizationResult {
	result := OptimizationResult{
		Strategy:     strategy,
		OriginalInfo: info,
	}

	// Create a copy of fields for optimization
	optimizedFields := make([]FieldInfo, len(info.Fields))
	copy(optimizedFields, info.Fields)

	switch strategy {
	case MinimizeSize:
		optimizedFields = aa.optimizeForSize(optimizedFields)
	case MinimizePadding:
		optimizedFields = aa.optimizeForPadding(optimizedFields)
	case OptimizeCacheLocality:
		optimizedFields = aa.optimizeForCacheLocality(optimizedFields)
	case OptimizeAccessPattern:
		optimizedFields = aa.optimizeForAccessPattern(optimizedFields)
	case Balanced:
		optimizedFields = aa.optimizeBalanced(optimizedFields)
	}

	// Calculate optimized layout
	optimizedInfo := aa.calculateOptimizedLayout(info.Name, optimizedFields)
	result.OptimizedInfo = optimizedInfo
	result.OptimizedFields = optimizedFields

	// Calculate improvements
	result.SizeSaved = info.Size - optimizedInfo.Size
	result.PaddingSaved = info.TotalPadding - optimizedInfo.TotalPadding
	result.CacheImprovement = optimizedInfo.CacheLineSpan < info.CacheLineSpan ||
		(optimizedInfo.CacheLineSpan == info.CacheLineSpan && !optimizedInfo.HotFieldsSpread && info.HotFieldsSpread)

	// Generate recommendations
	result.Recommendations = aa.generateRecommendations(info, optimizedInfo)

	return result
}

// optimizeForSize optimizes to minimize total struct size
func (aa *AlignmentAnalyzer) optimizeForSize(fields []FieldInfo) []FieldInfo {
	// Sort by alignment (descending), then by size (descending)
	sort.Slice(fields, func(i, j int) bool {
		if fields[i].Alignment != fields[j].Alignment {
			return fields[i].Alignment > fields[j].Alignment
		}
		return fields[i].Size > fields[j].Size
	})
	return fields
}

// optimizeForPadding optimizes to minimize padding
func (aa *AlignmentAnalyzer) optimizeForPadding(fields []FieldInfo) []FieldInfo {
	// Group by alignment and pack efficiently
	alignmentGroups := make(map[int][]FieldInfo)
	for _, field := range fields {
		alignmentGroups[field.Alignment] = append(alignmentGroups[field.Alignment], field)
	}

	var result []FieldInfo

	// Process groups in descending alignment order
	alignments := make([]int, 0, len(alignmentGroups))
	for alignment := range alignmentGroups {
		alignments = append(alignments, alignment)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(alignments)))

	for _, alignment := range alignments {
		group := alignmentGroups[alignment]
		// Sort within group by size (descending)
		sort.Slice(group, func(i, j int) bool {
			return group[i].Size > group[j].Size
		})
		result = append(result, group...)
	}

	return result
}

// optimizeForCacheLocality optimizes for cache line efficiency
func (aa *AlignmentAnalyzer) optimizeForCacheLocality(fields []FieldInfo) []FieldInfo {
	// Group fields that fit together in cache lines
	var result []FieldInfo
	currentCacheLineUsage := 0

	// Sort by size to pack efficiently
	sortedFields := make([]FieldInfo, len(fields))
	copy(sortedFields, fields)
	sort.Slice(sortedFields, func(i, j int) bool {
		return sortedFields[i].Size < sortedFields[j].Size
	})

	for _, field := range sortedFields {
		// Check if field fits in current cache line
		if currentCacheLineUsage+field.Size <= aa.cacheLineSize {
			result = append(result, field)
			currentCacheLineUsage += field.Size
		} else {
			// Start new cache line
			currentCacheLineUsage = field.Size
			result = append(result, field)
		}
	}

	return result
}

// optimizeForAccessPattern optimizes based on field access patterns
func (aa *AlignmentAnalyzer) optimizeForAccessPattern(fields []FieldInfo) []FieldInfo {
	// Sort by access frequency (descending), then by alignment
	sort.Slice(fields, func(i, j int) bool {
		if fields[i].AccessFreq != fields[j].AccessFreq {
			return fields[i].AccessFreq > fields[j].AccessFreq
		}
		return fields[i].Alignment > fields[j].Alignment
	})
	return fields
}

// optimizeBalanced provides a balanced optimization approach
func (aa *AlignmentAnalyzer) optimizeBalanced(fields []FieldInfo) []FieldInfo {
	// Separate hot and cold fields
	var hotFields, coldFields []FieldInfo

	for _, field := range fields {
		if field.AccessFreq > 0.5 {
			hotFields = append(hotFields, field)
		} else {
			coldFields = append(coldFields, field)
		}
	}

	// Optimize hot fields for cache locality
	hotFields = aa.optimizeForCacheLocality(hotFields)

	// Optimize cold fields for size
	coldFields = aa.optimizeForSize(coldFields)

	// Combine: hot fields first, then cold fields
	result := append(hotFields, coldFields...)
	return result
}

// calculateOptimizedLayout calculates the layout for optimized fields
func (aa *AlignmentAnalyzer) calculateOptimizedLayout(name string, fields []FieldInfo) StructInfo {
	info := StructInfo{
		Name:   name,
		Fields: make([]FieldInfo, len(fields)),
	}

	var offset int
	var totalFieldSize int

	for i, field := range fields {
		// Align offset to field alignment
		offset = (offset + field.Alignment - 1) &^ (field.Alignment - 1)

		// Calculate padding
		padding := 0
		if i > 0 {
			expectedOffset := info.Fields[i-1].Offset + info.Fields[i-1].Size
			padding = offset - expectedOffset
		}

		info.Fields[i] = FieldInfo{
			Name:         field.Name,
			Type:         field.Type,
			Size:         field.Size,
			Alignment:    field.Alignment,
			Offset:       offset,
			Padding:      padding,
			CacheLinePos: offset % aa.cacheLineSize,
			AccessFreq:   field.AccessFreq,
		}

		offset += field.Size
		totalFieldSize += field.Size
		info.TotalPadding += padding
	}

	// Calculate struct alignment (max field alignment)
	maxAlignment := 1
	for _, field := range fields {
		if field.Alignment > maxAlignment {
			maxAlignment = field.Alignment
		}
	}
	info.Alignment = maxAlignment

	// Align final size to struct alignment
	info.Size = (offset + info.Alignment - 1) &^ (info.Alignment - 1)
	finalPadding := info.Size - offset
	info.TotalPadding += finalPadding

	// Calculate efficiency
	if info.Size > 0 {
		info.Efficiency = float64(totalFieldSize) / float64(info.Size)
	}

	// Calculate cache line metrics
	info.CacheLineSpan = (info.Size + aa.cacheLineSize - 1) / aa.cacheLineSize
	info.CacheLineWaste = (info.CacheLineSpan * aa.cacheLineSize) - info.Size
	info.HotFieldsSpread = aa.checkHotFieldsSpread(info.Fields)

	return info
}

// generateRecommendations generates optimization recommendations
func (aa *AlignmentAnalyzer) generateRecommendations(original, optimized StructInfo) []string {
	var recommendations []string

	if optimized.Size < original.Size {
		recommendations = append(recommendations,
			fmt.Sprintf("Reorder fields to save %d bytes (%.1f%% reduction)",
				original.Size-optimized.Size,
				float64(original.Size-optimized.Size)/float64(original.Size)*100))
	}

	if optimized.TotalPadding < original.TotalPadding {
		recommendations = append(recommendations,
			fmt.Sprintf("Reduce padding by %d bytes", original.TotalPadding-optimized.TotalPadding))
	}

	if optimized.CacheLineSpan < original.CacheLineSpan {
		recommendations = append(recommendations,
			fmt.Sprintf("Reduce cache line span from %d to %d lines",
				original.CacheLineSpan, optimized.CacheLineSpan))
	}

	if !optimized.HotFieldsSpread && original.HotFieldsSpread {
		recommendations = append(recommendations,
			"Group frequently accessed fields in same cache line")
	}

	if optimized.Efficiency > original.Efficiency {
		recommendations = append(recommendations,
			fmt.Sprintf("Improve memory efficiency from %.1f%% to %.1f%%",
				original.Efficiency*100, optimized.Efficiency*100))
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Struct is already well-optimized")
	}

	return recommendations
}

// PaddingAnalyzer analyzes padding in various scenarios
type PaddingAnalyzer struct {
	analyzer *AlignmentAnalyzer
}

// NewPaddingAnalyzer creates a new padding analyzer
func NewPaddingAnalyzer() *PaddingAnalyzer {
	return &PaddingAnalyzer{
		analyzer: NewAlignmentAnalyzer(),
	}
}

// AnalyzePaddingScenarios analyzes different padding scenarios
func (pa *PaddingAnalyzer) AnalyzePaddingScenarios() {
	fmt.Println("=== Padding Analysis Scenarios ===")

	// Scenario 1: Basic alignment issues
	pa.analyzeBasicAlignment()

	// Scenario 2: Array packing
	pa.analyzeArrayPacking()

	// Scenario 3: Embedded structs
	pa.analyzeEmbeddedStructs()

	// Scenario 4: Interface overhead
	pa.analyzeInterfaceOverhead()

	// Scenario 5: Cache line optimization
	pa.analyzeCacheLineOptimization()
}

// analyzeBasicAlignment demonstrates basic alignment issues
func (pa *PaddingAnalyzer) analyzeBasicAlignment() {
	fmt.Println("\n--- Basic Alignment Issues ---")

	type BadAlignment struct {
		A bool  // 1 byte
		B int64 // 8 bytes
		C bool  // 1 byte
		D int32 // 4 bytes
		E bool  // 1 byte
	}

	type GoodAlignment struct {
		B int64 // 8 bytes
		D int32 // 4 bytes
		A bool  // 1 byte
		C bool  // 1 byte
		E bool  // 1 byte
		// 3 bytes padding
	}

	badInfo := pa.analyzer.AnalyzeStruct(BadAlignment{}, nil)
	goodInfo := pa.analyzer.AnalyzeStruct(GoodAlignment{}, nil)

	fmt.Printf("Bad Alignment: %d bytes (%d padding, %.1f%% efficient)\n",
		badInfo.Size, badInfo.TotalPadding, badInfo.Efficiency*100)
	fmt.Printf("Good Alignment: %d bytes (%d padding, %.1f%% efficient)\n",
		goodInfo.Size, goodInfo.TotalPadding, goodInfo.Efficiency*100)
	fmt.Printf("Space saved: %d bytes (%.1f%% reduction)\n",
		badInfo.Size-goodInfo.Size,
		float64(badInfo.Size-goodInfo.Size)/float64(badInfo.Size)*100)
}

// analyzeArrayPacking demonstrates array packing issues
func (pa *PaddingAnalyzer) analyzeArrayPacking() {
	fmt.Println("\n--- Array Packing Analysis ---")

	type UnpackedElement struct {
		Flag  bool  // 1 byte
		Value int64 // 8 bytes
		// 7 bytes padding per element
	}

	type PackedElement struct {
		Value int64 // 8 bytes
		Flag  bool  // 1 byte
		// 7 bytes padding per element
	}

	// Better: separate arrays
	type SeparateArrays struct {
		Values []int64 // 8 bytes per element
		Flags  []bool  // 1 byte per element
	}

	unpackedInfo := pa.analyzer.AnalyzeStruct(UnpackedElement{}, nil)
	packedInfo := pa.analyzer.AnalyzeStruct(PackedElement{}, nil)
	separateInfo := pa.analyzer.AnalyzeStruct(SeparateArrays{}, nil)

	fmt.Printf("Unpacked Element: %d bytes per element\n", unpackedInfo.Size)
	fmt.Printf("Packed Element: %d bytes per element\n", packedInfo.Size)
	fmt.Printf("Separate Arrays: %d bytes overhead + data\n", separateInfo.Size)

	// Calculate array overhead for 1000 elements
	numElements := 1000
	unpackedTotal := numElements * unpackedInfo.Size
	packedTotal := numElements * packedInfo.Size
	separateTotal := separateInfo.Size + numElements*(8+1) // int64 + bool

	fmt.Printf("\nFor %d elements:\n", numElements)
	fmt.Printf("Unpacked: %d bytes\n", unpackedTotal)
	fmt.Printf("Packed: %d bytes\n", packedTotal)
	fmt.Printf("Separate: %d bytes\n", separateTotal)
	fmt.Printf("Best approach saves: %d bytes\n", unpackedTotal-separateTotal)
}

// analyzeEmbeddedStructs demonstrates embedded struct alignment
func (pa *PaddingAnalyzer) analyzeEmbeddedStructs() {
	fmt.Println("\n--- Embedded Struct Analysis ---")

	type SmallStruct struct {
		A bool
		B bool
	}

	type EmbeddedBad struct {
		X int64       // 8 bytes
		S SmallStruct // 2 bytes + padding
		Y int64       // 8 bytes
	}

	type EmbeddedGood struct {
		X int64       // 8 bytes
		Y int64       // 8 bytes
		S SmallStruct // 2 bytes + padding
	}

	type Flattened struct {
		X int64 // 8 bytes
		Y int64 // 8 bytes
		A bool  // 1 byte
		B bool  // 1 byte
		// 6 bytes padding
	}

	badInfo := pa.analyzer.AnalyzeStruct(EmbeddedBad{}, nil)
	goodInfo := pa.analyzer.AnalyzeStruct(EmbeddedGood{}, nil)
	flatInfo := pa.analyzer.AnalyzeStruct(Flattened{}, nil)

	fmt.Printf("Embedded Bad: %d bytes (%d padding)\n", badInfo.Size, badInfo.TotalPadding)
	fmt.Printf("Embedded Good: %d bytes (%d padding)\n", goodInfo.Size, goodInfo.TotalPadding)
	fmt.Printf("Flattened: %d bytes (%d padding)\n", flatInfo.Size, flatInfo.TotalPadding)
}

// analyzeInterfaceOverhead demonstrates interface overhead
func (pa *PaddingAnalyzer) analyzeInterfaceOverhead() {
	fmt.Println("\n--- Interface Overhead Analysis ---")

	type ConcreteStruct struct {
		Value int64
	}

	type InterfaceContainer struct {
		Data interface{} // 16 bytes (pointer + type info)
	}

	type PointerContainer struct {
		Data *ConcreteStruct // 8 bytes
	}

	concreteInfo := pa.analyzer.AnalyzeStruct(ConcreteStruct{}, nil)
	interfaceInfo := pa.analyzer.AnalyzeStruct(InterfaceContainer{}, nil)
	pointerInfo := pa.analyzer.AnalyzeStruct(PointerContainer{}, nil)

	fmt.Printf("Concrete Struct: %d bytes\n", concreteInfo.Size)
	fmt.Printf("Interface Container: %d bytes\n", interfaceInfo.Size)
	fmt.Printf("Pointer Container: %d bytes\n", pointerInfo.Size)
	fmt.Printf("Interface overhead: %d bytes\n", interfaceInfo.Size-pointerInfo.Size)
}

// analyzeCacheLineOptimization demonstrates cache line optimization
func (pa *PaddingAnalyzer) analyzeCacheLineOptimization() {
	fmt.Println("\n--- Cache Line Optimization ---")

	// Simulate access patterns
	accessPattern := map[string]float64{
		"HotField1":   0.9,
		"HotField2":   0.8,
		"ColdField1":  0.1,
		"ColdField2":  0.2,
		"MediumField": 0.5,
	}

	type CacheUnfriendly struct {
		HotField1   int64    // 8 bytes - frequently accessed
		ColdField1  [32]byte // 32 bytes - rarely accessed
		HotField2   int64    // 8 bytes - frequently accessed
		ColdField2  [32]byte // 32 bytes - rarely accessed
		MediumField int32    // 4 bytes - moderately accessed
	}

	originalInfo := pa.analyzer.AnalyzeStruct(CacheUnfriendly{}, accessPattern)

	fmt.Printf("Original Layout:\n")
	fmt.Printf("  Size: %d bytes\n", originalInfo.Size)
	fmt.Printf("  Cache lines: %d\n", originalInfo.CacheLineSpan)
	fmt.Printf("  Hot fields spread: %v\n", originalInfo.HotFieldsSpread)

	// Optimize for access pattern
	optResult := pa.analyzer.OptimizeStruct(originalInfo, OptimizeAccessPattern)

	fmt.Printf("\nOptimized Layout:\n")
	fmt.Printf("  Size: %d bytes\n", optResult.OptimizedInfo.Size)
	fmt.Printf("  Cache lines: %d\n", optResult.OptimizedInfo.CacheLineSpan)
	fmt.Printf("  Hot fields spread: %v\n", optResult.OptimizedInfo.HotFieldsSpread)
	fmt.Printf("  Cache improvement: %v\n", optResult.CacheImprovement)

	fmt.Println("\nRecommendations:")
	for _, rec := range optResult.Recommendations {
		fmt.Printf("  - %s\n", rec)
	}
}

// PerformanceTester tests performance impact of alignment
type PerformanceTester struct {
	iterations int
}

// NewPerformanceTester creates a new performance tester
func NewPerformanceTester(iterations int) *PerformanceTester {
	return &PerformanceTester{
		iterations: iterations,
	}
}

// TestAlignmentPerformance tests performance impact of struct alignment
func (pt *PerformanceTester) TestAlignmentPerformance() {
	fmt.Println("\n=== Alignment Performance Testing ===")

	type BadStruct struct {
		A bool
		B int64
		C bool
		D int32
		E bool
	}

	type GoodStruct struct {
		B int64
		D int32
		A bool
		C bool
		E bool
	}

	// Test single struct access
	pt.testSingleStructAccess()

	// Test array access
	pt.testArrayAccess()

	// Test cache line effects
	pt.testCacheLineEffects()
}

// testSingleStructAccess tests single struct access performance
func (pt *PerformanceTester) testSingleStructAccess() {
	fmt.Println("\n--- Single Struct Access ---")

	type BadStruct struct {
		A bool
		B int64
		C bool
		D int32
		E bool
	}

	type GoodStruct struct {
		B int64
		D int32
		A bool
		C bool
		E bool
	}

	badStruct := BadStruct{true, 42, false, 123, true}
	goodStruct := GoodStruct{42, 123, true, false, true}

	// Test bad struct
	start := time.Now()
	for i := 0; i < pt.iterations; i++ {
		_ = badStruct.A
		_ = badStruct.B
		_ = badStruct.C
		_ = badStruct.D
		_ = badStruct.E
	}
	badDuration := time.Since(start)

	// Test good struct
	start = time.Now()
	for i := 0; i < pt.iterations; i++ {
		_ = goodStruct.A
		_ = goodStruct.B
		_ = goodStruct.C
		_ = goodStruct.D
		_ = goodStruct.E
	}
	goodDuration := time.Since(start)

	fmt.Printf("Bad alignment: %v\n", badDuration)
	fmt.Printf("Good alignment: %v\n", goodDuration)
	if badDuration > goodDuration {
		fmt.Printf("Improvement: %.2fx faster\n", float64(badDuration)/float64(goodDuration))
	}
}

// testArrayAccess tests array access performance
func (pt *PerformanceTester) testArrayAccess() {
	fmt.Println("\n--- Array Access Performance ---")

	type BadElement struct {
		Flag  bool
		Value int64
	}

	type GoodElement struct {
		Value int64
		Flag  bool
	}

	const arraySize = 10000
	badArray := make([]BadElement, arraySize)
	goodArray := make([]GoodElement, arraySize)

	// Initialize arrays
	for i := 0; i < arraySize; i++ {
		badArray[i] = BadElement{true, int64(i)}
		goodArray[i] = GoodElement{int64(i), true}
	}

	// Test bad array
	start := time.Now()
	sum := int64(0)
	for i := 0; i < pt.iterations/100; i++ {
		for j := 0; j < arraySize; j++ {
			if badArray[j].Flag {
				sum += badArray[j].Value
			}
		}
	}
	badDuration := time.Since(start)

	// Test good array
	start = time.Now()
	sum = 0
	for i := 0; i < pt.iterations/100; i++ {
		for j := 0; j < arraySize; j++ {
			if goodArray[j].Flag {
				sum += goodArray[j].Value
			}
		}
	}
	goodDuration := time.Since(start)

	fmt.Printf("Bad array layout: %v\n", badDuration)
	fmt.Printf("Good array layout: %v\n", goodDuration)
	if badDuration > goodDuration {
		fmt.Printf("Improvement: %.2fx faster\n", float64(badDuration)/float64(goodDuration))
	}

	fmt.Printf("Memory usage - Bad: %d bytes, Good: %d bytes\n",
		unsafe.Sizeof(BadElement{})*arraySize,
		unsafe.Sizeof(GoodElement{})*arraySize)
}

// testCacheLineEffects tests cache line effects
func (pt *PerformanceTester) testCacheLineEffects() {
	fmt.Println("\n--- Cache Line Effects ---")

	type CacheUnfriendly struct {
		Hot1 int64    // Frequently accessed
		Cold [56]byte // Rarely accessed, fills rest of cache line
		Hot2 int64    // Frequently accessed, in next cache line
	}

	type CacheFriendly struct {
		Hot1 int64    // Frequently accessed
		Hot2 int64    // Frequently accessed, same cache line
		Cold [56]byte // Rarely accessed
	}

	const numStructs = 1000
	unfriendlyArray := make([]CacheUnfriendly, numStructs)
	friendlyArray := make([]CacheFriendly, numStructs)

	// Initialize
	for i := 0; i < numStructs; i++ {
		unfriendlyArray[i].Hot1 = int64(i)
		unfriendlyArray[i].Hot2 = int64(i * 2)
		friendlyArray[i].Hot1 = int64(i)
		friendlyArray[i].Hot2 = int64(i * 2)
	}

	// Test cache unfriendly
	start := time.Now()
	sum := int64(0)
	for i := 0; i < pt.iterations/10; i++ {
		for j := 0; j < numStructs; j++ {
			sum += unfriendlyArray[j].Hot1 + unfriendlyArray[j].Hot2
		}
	}
	unfriendlyDuration := time.Since(start)

	// Test cache friendly
	start = time.Now()
	sum = 0
	for i := 0; i < pt.iterations/10; i++ {
		for j := 0; j < numStructs; j++ {
			sum += friendlyArray[j].Hot1 + friendlyArray[j].Hot2
		}
	}
	friendlyDuration := time.Since(start)

	fmt.Printf("Cache unfriendly: %v\n", unfriendlyDuration)
	fmt.Printf("Cache friendly: %v\n", friendlyDuration)
	if unfriendlyDuration > friendlyDuration {
		fmt.Printf("Improvement: %.2fx faster\n", float64(unfriendlyDuration)/float64(friendlyDuration))
	}
}

// DemonstrationSuite orchestrates all demonstrations
type DemonstrationSuite struct {
	analyzer        *AlignmentAnalyzer
	paddingAnalyzer *PaddingAnalyzer
	perfTester      *PerformanceTester
}

// NewDemonstrationSuite creates a new demonstration suite
func NewDemonstrationSuite() *DemonstrationSuite {
	return &DemonstrationSuite{
		analyzer:        NewAlignmentAnalyzer(),
		paddingAnalyzer: NewPaddingAnalyzer(),
		perfTester:      NewPerformanceTester(1000000),
	}
}

// RunAllDemonstrations runs all alignment and padding demonstrations
func (ds *DemonstrationSuite) RunAllDemonstrations() {
	fmt.Println("Memory Alignment and Padding Optimization")
	fmt.Println("=========================================")

	// Run padding analysis
	ds.paddingAnalyzer.AnalyzePaddingScenarios()

	// Run performance tests
	ds.perfTester.TestAlignmentPerformance()

	// Demonstrate optimization strategies
	ds.demonstrateOptimizationStrategies()

	// Show memory statistics
	ds.showMemoryStatistics()
}

// demonstrateOptimizationStrategies demonstrates different optimization strategies
func (ds *DemonstrationSuite) demonstrateOptimizationStrategies() {
	fmt.Println("\n=== Optimization Strategies ===")

	accessPattern := map[string]float64{
		"HotField1":    0.9,
		"HotField2":    0.8,
		"MediumField1": 0.5,
		"MediumField2": 0.4,
		"ColdField1":   0.1,
		"ColdField2":   0.2,
	}

	type TestStruct struct {
		ColdField1   [16]byte // Rarely accessed
		HotField1    int64    // Frequently accessed
		ColdField2   [16]byte // Rarely accessed
		MediumField1 int32    // Moderately accessed
		HotField2    int64    // Frequently accessed
		MediumField2 int32    // Moderately accessed
	}

	originalInfo := ds.analyzer.AnalyzeStruct(TestStruct{}, accessPattern)

	fmt.Printf("Original Struct: %d bytes, %d padding, %.1f%% efficient\n",
		originalInfo.Size, originalInfo.TotalPadding, originalInfo.Efficiency*100)

	strategies := []OptimizationStrategy{
		MinimizeSize,
		MinimizePadding,
		OptimizeCacheLocality,
		OptimizeAccessPattern,
		Balanced,
	}

	strategyNames := []string{
		"Minimize Size",
		"Minimize Padding",
		"Optimize Cache Locality",
		"Optimize Access Pattern",
		"Balanced",
	}

	for i, strategy := range strategies {
		result := ds.analyzer.OptimizeStruct(originalInfo, strategy)
		fmt.Printf("\n%s:\n", strategyNames[i])
		fmt.Printf("  Size: %d bytes (saved %d)\n", result.OptimizedInfo.Size, result.SizeSaved)
		fmt.Printf("  Padding: %d bytes (saved %d)\n", result.OptimizedInfo.TotalPadding, result.PaddingSaved)
		fmt.Printf("  Efficiency: %.1f%%\n", result.OptimizedInfo.Efficiency*100)
		fmt.Printf("  Cache lines: %d\n", result.OptimizedInfo.CacheLineSpan)
		fmt.Printf("  Cache improvement: %v\n", result.CacheImprovement)
	}
}

// showMemoryStatistics shows current memory statistics
func (ds *DemonstrationSuite) showMemoryStatistics() {
	fmt.Println("\n=== Memory Statistics ===")

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("Heap Alloc: %d KB\n", m.HeapAlloc/1024)
	fmt.Printf("Total Alloc: %d KB\n", m.TotalAlloc/1024)
	fmt.Printf("Sys: %d KB\n", m.Sys/1024)
	fmt.Printf("NumGC: %d\n", m.NumGC)
	fmt.Printf("GC CPU Fraction: %.4f\n", m.GCCPUFraction)
}

// main demonstrates all alignment and padding optimization techniques
func main() {
	suite := NewDemonstrationSuite()
	suite.RunAllDemonstrations()

	fmt.Println("\nAlignment and padding optimization demonstration completed!")
}

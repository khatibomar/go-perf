package main

import (
	"fmt"
	"runtime"
	"sort"
	"unsafe"
)

// SizeClassAnalyzer analyzes Go's size class utilization
type SizeClassAnalyzer struct {
	utilization map[int]*SizeClassUtilization
	totalObjects int64
	totalWaste   int64
}

// SizeClassUtilization tracks utilization for a specific size class
type SizeClassUtilization struct {
	SizeClass    int
	Allocations  int64
	TotalBytes   int64
	WastedBytes  int64
	Efficiency   float64
	Objects      []ObjectInfo
}

// ObjectInfo contains information about an allocated object
type ObjectInfo struct {
	ActualSize   uintptr
	AllocatedSize uintptr
	Waste        uintptr
	TypeName     string
}

// NewSizeClassAnalyzer creates a new size class analyzer
func NewSizeClassAnalyzer() *SizeClassAnalyzer {
	return &SizeClassAnalyzer{
		utilization: make(map[int]*SizeClassUtilization),
	}
}

// RecordAllocation records an allocation in the analyzer
func (sca *SizeClassAnalyzer) RecordAllocation(actualSize uintptr, typeName string) {
	sizeClass := GetSizeClass(actualSize)
	allocatedSize := uintptr(sizeClass)
	if sizeClass == -1 {
		// Large object, allocated directly
		allocatedSize = actualSize
	}
	
	waste := allocatedSize - actualSize
	
	if sca.utilization[sizeClass] == nil {
		sca.utilization[sizeClass] = &SizeClassUtilization{
			SizeClass: sizeClass,
			Objects:   make([]ObjectInfo, 0),
		}
	}
	
	util := sca.utilization[sizeClass]
	util.Allocations++
	util.TotalBytes += int64(allocatedSize)
	util.WastedBytes += int64(waste)
	util.Objects = append(util.Objects, ObjectInfo{
		ActualSize:    actualSize,
		AllocatedSize: allocatedSize,
		Waste:         waste,
		TypeName:      typeName,
	})
	
	// Calculate efficiency
	if allocatedSize > 0 {
		util.Efficiency = float64(actualSize) / float64(allocatedSize) * 100
	}
	
	sca.totalObjects++
	sca.totalWaste += int64(waste)
}

// AnalyzeCommonTypes analyzes size class utilization for common Go types
func (sca *SizeClassAnalyzer) AnalyzeCommonTypes() {
	fmt.Println("=== Common Go Types Size Class Analysis ===")
	
	// Analyze basic types
	basicTypes := []struct {
		name string
		size uintptr
		fn   func() interface{}
	}{
		{"bool", unsafe.Sizeof(bool(false)), func() interface{} { return new(bool) }},
		{"int8", unsafe.Sizeof(int8(0)), func() interface{} { return new(int8) }},
		{"int16", unsafe.Sizeof(int16(0)), func() interface{} { return new(int16) }},
		{"int32", unsafe.Sizeof(int32(0)), func() interface{} { return new(int32) }},
		{"int64", unsafe.Sizeof(int64(0)), func() interface{} { return new(int64) }},
		{"float32", unsafe.Sizeof(float32(0)), func() interface{} { return new(float32) }},
		{"float64", unsafe.Sizeof(float64(0)), func() interface{} { return new(float64) }},
		{"string", unsafe.Sizeof(string("")), func() interface{} { return new(string) }},
		{"[]byte(cap=0)", unsafe.Sizeof([]byte{}), func() interface{} { s := make([]byte, 0); return &s }},
		{"[]byte(cap=16)", unsafe.Sizeof([]byte{}) + 16, func() interface{} { s := make([]byte, 0, 16); return &s }},
		{"[]byte(cap=64)", unsafe.Sizeof([]byte{}) + 64, func() interface{} { s := make([]byte, 0, 64); return &s }},
		{"[]byte(cap=256)", unsafe.Sizeof([]byte{}) + 256, func() interface{} { s := make([]byte, 0, 256); return &s }},
		{"map[string]int", unsafe.Sizeof(make(map[string]int)), func() interface{} { m := make(map[string]int); return &m }},
	}
	
	for _, bt := range basicTypes {
		sca.RecordAllocation(bt.size, bt.name)
	}
	
	// Analyze struct types
	structTypes := []struct {
		name string
		size uintptr
	}{
		{"SmallStruct", unsafe.Sizeof(SmallStruct{})},
		{"MediumStruct", unsafe.Sizeof(MediumStruct{})},
		{"LargeStruct", unsafe.Sizeof(LargeStruct{})},
		{"PoorlyAlignedStruct", unsafe.Sizeof(PoorlyAlignedStruct{})},
		{"WellAlignedStruct", unsafe.Sizeof(WellAlignedStruct{})},
		{"PackedStruct", unsafe.Sizeof(PackedStruct{})},
	}
	
	for _, st := range structTypes {
		sca.RecordAllocation(st.size, st.name)
	}
	
	// Print analysis results
	sca.PrintAnalysis()
}

// PrintAnalysis prints detailed size class analysis
func (sca *SizeClassAnalyzer) PrintAnalysis() {
	fmt.Println("\n=== Size Class Utilization Report ===")
	fmt.Printf("Total Objects Analyzed: %d\n", sca.totalObjects)
	fmt.Printf("Total Memory Waste: %d bytes (%.2f KB)\n", sca.totalWaste, float64(sca.totalWaste)/1024)
	
	if sca.totalObjects > 0 {
		avgWaste := float64(sca.totalWaste) / float64(sca.totalObjects)
		fmt.Printf("Average Waste per Object: %.2f bytes\n", avgWaste)
	}
	
	fmt.Println("\nSize Class Details:")
	fmt.Printf("%-15s %-12s %-10s %-12s %-10s %-15s\n", 
		"Size Class", "Allocations", "Total KB", "Wasted KB", "Efficiency", "Waste %")
	fmt.Println("---------------------------------------------------------------------------------")
	
	// Sort size classes for better readability
	var sortedClasses []int
	for sizeClass := range sca.utilization {
		sortedClasses = append(sortedClasses, sizeClass)
	}
	sort.Ints(sortedClasses)
	
	for _, sizeClass := range sortedClasses {
		util := sca.utilization[sizeClass]
		
		sizeClassStr := fmt.Sprintf("%d", sizeClass)
		if sizeClass == -1 {
			sizeClassStr = "Large (>32KB)"
		} else if sizeClass == 0 {
			sizeClassStr = "Zero"
		}
		
		wastePercent := float64(util.WastedBytes) / float64(util.TotalBytes) * 100
		if util.TotalBytes == 0 {
			wastePercent = 0
		}
		
		fmt.Printf("%-15s %-12d %-10.2f %-12.2f %-10.1f%% %-15.1f%%\n",
			sizeClassStr,
			util.Allocations,
			float64(util.TotalBytes)/1024,
			float64(util.WastedBytes)/1024,
			util.Efficiency,
			wastePercent)
	}
	
	// Show most wasteful allocations
	sca.PrintMostWastefulAllocations()
	
	// Show optimization recommendations
	sca.PrintOptimizationRecommendations()
}

// PrintMostWastefulAllocations shows the allocations with highest waste
func (sca *SizeClassAnalyzer) PrintMostWastefulAllocations() {
	fmt.Println("\n=== Most Wasteful Allocations ===")
	fmt.Printf("%-25s %-12s %-15s %-10s %-10s\n", 
		"Type", "Actual Size", "Allocated Size", "Waste", "Efficiency")
	fmt.Println("------------------------------------------------------------------------")
	
	// Collect all objects and sort by waste percentage
	type wasteInfo struct {
		obj          ObjectInfo
		wastePercent float64
	}
	
	var allObjects []wasteInfo
	for _, util := range sca.utilization {
		for _, obj := range util.Objects {
			wastePercent := float64(obj.Waste) / float64(obj.AllocatedSize) * 100
			allObjects = append(allObjects, wasteInfo{obj, wastePercent})
		}
	}
	
	// Sort by waste percentage (descending)
	sort.Slice(allObjects, func(i, j int) bool {
		return allObjects[i].wastePercent > allObjects[j].wastePercent
	})
	
	// Show top 10 most wasteful
	maxShow := 10
	if len(allObjects) < maxShow {
		maxShow = len(allObjects)
	}
	
	for i := 0; i < maxShow; i++ {
		obj := allObjects[i].obj
		efficiency := float64(obj.ActualSize) / float64(obj.AllocatedSize) * 100
		
		fmt.Printf("%-25s %-12d %-15d %-10d %-10.1f%%\n",
			obj.TypeName,
			obj.ActualSize,
			obj.AllocatedSize,
			obj.Waste,
			efficiency)
	}
}

// PrintOptimizationRecommendations provides optimization suggestions
func (sca *SizeClassAnalyzer) PrintOptimizationRecommendations() {
	fmt.Println("\n=== Optimization Recommendations ===")
	
	recommendations := []string{}
	
	// Check for highly wasteful size classes
	for _, util := range sca.utilization {
		if util.TotalBytes > 0 {
			wastePercent := float64(util.WastedBytes) / float64(util.TotalBytes) * 100
			if wastePercent > 50 && util.Allocations > 1 {
				recommendations = append(recommendations, 
					fmt.Sprintf("• Size class %d has %.1f%% waste - consider struct field reordering", 
						util.SizeClass, wastePercent))
			}
		}
	}
	
	// Check for struct alignment issues
	for _, util := range sca.utilization {
		for _, obj := range util.Objects {
			if obj.Waste > obj.ActualSize/2 { // More than 50% waste
				recommendations = append(recommendations,
					fmt.Sprintf("• %s wastes %d bytes (%.1f%%) - optimize field alignment",
						obj.TypeName, obj.Waste, 
						float64(obj.Waste)/float64(obj.AllocatedSize)*100))
			}
		}
	}
	
	// General recommendations
	recommendations = append(recommendations, []string{
		"• Order struct fields from largest to smallest to minimize padding",
		"• Use byte-sized fields together to fill padding gaps",
		"• Consider using arrays instead of slices for fixed-size data",
		"• Use sync.Pool for frequently allocated objects",
		"• Consider custom allocators for specific allocation patterns",
	}...)
	
	for _, rec := range recommendations {
		fmt.Println(rec)
	}
}

// Example struct types for analysis
type SmallStruct struct {
	a int32
	b bool
}

type MediumStruct struct {
	a int64
	b int32
	c int16
	d bool
	e string
	f []byte
}

type LargeStruct struct {
	a [1024]byte
	b int64
	c string
	d map[string]int
	e []int
}

// AnalyzeSizeClassEfficiency analyzes different allocation patterns
func AnalyzeSizeClassEfficiency() {
	fmt.Println("\n=== Size Class Efficiency Analysis ===")
	
	// Test allocation efficiency for different patterns
	patterns := []struct {
		name        string
		allocations int
		fn          func() interface{}
	}{
		{"8-byte allocations", 10000, func() interface{} { return make([]byte, 8) }},
		{"16-byte allocations", 10000, func() interface{} { return make([]byte, 16) }},
		{"24-byte allocations", 10000, func() interface{} { return make([]byte, 24) }},
		{"32-byte allocations", 10000, func() interface{} { return make([]byte, 32) }},
		{"33-byte allocations (inefficient)", 10000, func() interface{} { return make([]byte, 33) }},
		{"48-byte allocations", 10000, func() interface{} { return make([]byte, 48) }},
		{"64-byte allocations", 10000, func() interface{} { return make([]byte, 64) }},
		{"65-byte allocations (inefficient)", 10000, func() interface{} { return make([]byte, 65) }},
	}
	
	fmt.Printf("%-35s %-15s %-15s %-10s\n", "Pattern", "Requested", "Allocated", "Efficiency")
	fmt.Println("-------------------------------------------------------------------------------")
	
	for _, pattern := range patterns {
		// Measure memory before
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)
		
		// Allocate objects
		var objects []interface{}
		for i := 0; i < pattern.allocations; i++ {
			obj := pattern.fn()
			objects = append(objects, obj)
		}
		
		// Measure memory after
		runtime.ReadMemStats(&m2)
		
		// Calculate metrics
		allocated := m2.TotalAlloc - m1.TotalAlloc
		mallocs := m2.Mallocs - m1.Mallocs
		
		if mallocs > 0 {
			avgAllocated := float64(allocated) / float64(mallocs)
			
			// Determine requested size from pattern name
			var requestedSize float64
			switch {
			case pattern.name == "8-byte allocations":
				requestedSize = 8
			case pattern.name == "16-byte allocations":
				requestedSize = 16
			case pattern.name == "24-byte allocations":
				requestedSize = 24
			case pattern.name == "32-byte allocations":
				requestedSize = 32
			case pattern.name == "33-byte allocations (inefficient)":
				requestedSize = 33
			case pattern.name == "48-byte allocations":
				requestedSize = 48
			case pattern.name == "64-byte allocations":
				requestedSize = 64
			case pattern.name == "65-byte allocations (inefficient)":
				requestedSize = 65
			}
			
			efficiency := requestedSize / avgAllocated * 100
			
			fmt.Printf("%-35s %-15.0f %-15.1f %-10.1f%%\n",
				pattern.name, requestedSize, avgAllocated, efficiency)
		}
		
		// Keep references to prevent GC during measurement
		_ = objects
	}
}

// AnalyzeSizeClassBoundaries shows the impact of crossing size class boundaries
func AnalyzeSizeClassBoundaries() {
	fmt.Println("\n=== Size Class Boundary Analysis ===")
	fmt.Println("This analysis shows how allocation size affects memory usage:")
	fmt.Println()
	
	fmt.Printf("%-15s %-15s %-15s %-10s\n", "Request Size", "Size Class", "Allocated", "Waste")
	fmt.Println("---------------------------------------------------------------")
	
	// Test sizes around size class boundaries
	testSizes := []int{
		8, 9, 15, 16, 17, 23, 24, 25, 31, 32, 33,
		47, 48, 49, 63, 64, 65, 79, 80, 81, 95, 96, 97,
		127, 128, 129, 255, 256, 257, 511, 512, 513,
		1023, 1024, 1025, 2047, 2048, 2049,
	}
	
	for _, size := range testSizes {
		sizeClass := GetSizeClass(uintptr(size))
		allocated := sizeClass
		if sizeClass == -1 {
			allocated = size // Large object
		}
		
		waste := allocated - size
		
		sizeClassStr := fmt.Sprintf("%d", sizeClass)
		if sizeClass == -1 {
			sizeClassStr = "Large"
		}
		
		fmt.Printf("%-15d %-15s %-15d %-10d\n", size, sizeClassStr, allocated, waste)
	}
	
	fmt.Println("\nKey Insights:")
	fmt.Println("• Allocating 33 bytes uses a 48-byte size class (31% waste)")
	fmt.Println("• Allocating 65 bytes uses an 80-byte size class (18% waste)")
	fmt.Println("• Staying within size class boundaries maximizes efficiency")
	fmt.Println("• Consider padding structs to fit size class boundaries")
}

// RunSizeClassAnalysis runs the size class analysis - called from main in allocator_analysis.go
func RunSizeClassAnalysis() {
	fmt.Println("\n=== Go Size Class Analysis ===")
	
	analyzer := NewSizeClassAnalyzer()
	
	// Analyze common types
	analyzer.AnalyzeCommonTypes()
	
	// Analyze size class efficiency
	AnalyzeSizeClassEfficiency()
	
	// Analyze size class boundaries
	AnalyzeSizeClassBoundaries()
	
	fmt.Println("\n=== Size Class Summary ===")
	fmt.Println("Understanding size classes is crucial for memory optimization:")
	fmt.Println("1. Go uses 67 size classes from 8 bytes to 32KB")
	fmt.Println("2. Objects are allocated in the smallest size class that fits")
	fmt.Println("3. Crossing size class boundaries can waste significant memory")
	fmt.Println("4. Struct field ordering affects memory efficiency")
	fmt.Println("5. Use tools like this to identify optimization opportunities")
}
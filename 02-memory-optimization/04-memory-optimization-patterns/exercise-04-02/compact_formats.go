package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"
)

// Variable-length integer encoding (similar to Protocol Buffers varint)
type VarIntEncoder struct {
	buffer []byte
}

func NewVarIntEncoder() *VarIntEncoder {
	return &VarIntEncoder{
		buffer: make([]byte, 0, 64),
	}
}

func (ve *VarIntEncoder) EncodeUint32(value uint32) []byte {
	ve.buffer = ve.buffer[:0]
	
	for value >= 0x80 {
		ve.buffer = append(ve.buffer, byte(value)|0x80)
		value >>= 7
	}
	ve.buffer = append(ve.buffer, byte(value))
	
	return ve.buffer
}

func (ve *VarIntEncoder) EncodeUint64(value uint64) []byte {
	ve.buffer = ve.buffer[:0]
	
	for value >= 0x80 {
		ve.buffer = append(ve.buffer, byte(value)|0x80)
		value >>= 7
	}
	ve.buffer = append(ve.buffer, byte(value))
	
	return ve.buffer
}

func DecodeUint32(data []byte) (uint32, int) {
	var result uint32
	var shift uint
	var bytesRead int
	
	for i, b := range data {
		if shift >= 32 {
			return 0, 0 // Overflow
		}
		
		result |= uint32(b&0x7F) << shift
		bytesRead++
		
		if b&0x80 == 0 {
			return result, bytesRead
		}
		
		shift += 7
		if i == len(data)-1 {
			return 0, 0 // Incomplete
		}
	}
	
	return result, bytesRead
}

func DecodeUint64(data []byte) (uint64, int) {
	var result uint64
	var shift uint
	var bytesRead int
	
	for i, b := range data {
		if shift >= 64 {
			return 0, 0 // Overflow
		}
		
		result |= uint64(b&0x7F) << shift
		bytesRead++
		
		if b&0x80 == 0 {
			return result, bytesRead
		}
		
		shift += 7
		if i == len(data)-1 {
			return 0, 0 // Incomplete
		}
	}
	
	return result, bytesRead
}

// Compressed sparse array using run-length encoding
type CompressedSparseArray struct {
	runs []Run
	size int
}

type Run struct {
	start  int
	length int
	value  uint32
}

func NewCompressedSparseArray(size int) *CompressedSparseArray {
	return &CompressedSparseArray{
		size: size,
	}
}

func (csa *CompressedSparseArray) Set(index int, value uint32) {
	if index < 0 || index >= csa.size {
		return
	}
	
	// Find the run that contains this index
	for i, run := range csa.runs {
		if index >= run.start && index < run.start+run.length {
			if run.value == value {
				return // Already set to this value
			}
			
			// Split the run
			csa.splitRun(i, index, value)
			return
		}
	}
	
	// Create new run
	csa.insertRun(index, 1, value)
}

func (csa *CompressedSparseArray) Get(index int) uint32 {
	if index < 0 || index >= csa.size {
		return 0
	}
	
	for _, run := range csa.runs {
		if index >= run.start && index < run.start+run.length {
			return run.value
		}
	}
	
	return 0 // Default value
}

func (csa *CompressedSparseArray) splitRun(runIndex, splitIndex int, newValue uint32) {
	run := csa.runs[runIndex]
	
	// Remove the original run
	csa.runs = append(csa.runs[:runIndex], csa.runs[runIndex+1:]...)
	
	// Add up to three new runs
	if splitIndex > run.start {
		// Left part
		csa.insertRun(run.start, splitIndex-run.start, run.value)
	}
	
	// Middle part (new value)
	csa.insertRun(splitIndex, 1, newValue)
	
	if splitIndex+1 < run.start+run.length {
		// Right part
		remainingStart := splitIndex + 1
		remainingLength := run.start + run.length - remainingStart
		csa.insertRun(remainingStart, remainingLength, run.value)
	}
}

func (csa *CompressedSparseArray) insertRun(start, length int, value uint32) {
	newRun := Run{start: start, length: length, value: value}
	
	// Find insertion point
	insertIndex := 0
	for i, run := range csa.runs {
		if start < run.start {
			insertIndex = i
			break
		}
		insertIndex = i + 1
	}
	
	// Insert the run
	csa.runs = append(csa.runs, Run{})
	copy(csa.runs[insertIndex+1:], csa.runs[insertIndex:])
	csa.runs[insertIndex] = newRun
	
	// Try to merge with adjacent runs
	csa.mergeRuns()
}

func (csa *CompressedSparseArray) mergeRuns() {
	if len(csa.runs) <= 1 {
		return
	}
	
	merged := make([]Run, 0, len(csa.runs))
	current := csa.runs[0]
	
	for i := 1; i < len(csa.runs); i++ {
		next := csa.runs[i]
		
		// Check if runs can be merged
		if current.start+current.length == next.start && current.value == next.value {
			// Merge runs
			current.length += next.length
		} else {
			// Can't merge, add current and move to next
			merged = append(merged, current)
			current = next
		}
	}
	
	merged = append(merged, current)
	csa.runs = merged
}

func (csa *CompressedSparseArray) CompressionRatio() float64 {
	if csa.size == 0 {
		return 0
	}
	
	uncompressedSize := csa.size * 4 // 4 bytes per uint32
	compressedSize := len(csa.runs) * 12 // 12 bytes per run (start, length, value)
	
	return float64(uncompressedSize) / float64(compressedSize)
}

// Bit vector for compact boolean storage
type BitVector struct {
	bits []uint64
	size int
}

func NewBitVector(size int) *BitVector {
	numWords := (size + 63) / 64
	return &BitVector{
		bits: make([]uint64, numWords),
		size: size,
	}
}

func (bv *BitVector) Set(index int, value bool) {
	if index < 0 || index >= bv.size {
		return
	}
	
	wordIndex := index / 64
	bitIndex := index % 64
	
	if value {
		bv.bits[wordIndex] |= 1 << bitIndex
	} else {
		bv.bits[wordIndex] &^= 1 << bitIndex
	}
}

func (bv *BitVector) Get(index int) bool {
	if index < 0 || index >= bv.size {
		return false
	}
	
	wordIndex := index / 64
	bitIndex := index % 64
	
	return (bv.bits[wordIndex] & (1 << bitIndex)) != 0
}

func (bv *BitVector) PopCount() int {
	count := 0
	for _, word := range bv.bits {
		count += popCount(word)
	}
	return count
}



func (bv *BitVector) And(other *BitVector) *BitVector {
	result := NewBitVector(bv.size)
	minWords := len(bv.bits)
	if len(other.bits) < minWords {
		minWords = len(other.bits)
	}
	
	for i := 0; i < minWords; i++ {
		result.bits[i] = bv.bits[i] & other.bits[i]
	}
	return result
}

func (bv *BitVector) Or(other *BitVector) *BitVector {
	result := NewBitVector(bv.size)
	maxWords := len(bv.bits)
	if len(other.bits) > maxWords {
		maxWords = len(other.bits)
		result = NewBitVector(other.size)
	}
	
	for i := 0; i < maxWords; i++ {
		var a, b uint64
		if i < len(bv.bits) {
			a = bv.bits[i]
		}
		if i < len(other.bits) {
			b = other.bits[i]
		}
		result.bits[i] = a | b
	}
	return result
}

func (bv *BitVector) Xor(other *BitVector) *BitVector {
	result := NewBitVector(bv.size)
	maxWords := len(bv.bits)
	if len(other.bits) > maxWords {
		maxWords = len(other.bits)
		result = NewBitVector(other.size)
	}
	
	for i := 0; i < maxWords; i++ {
		var a, b uint64
		if i < len(bv.bits) {
			a = bv.bits[i]
		}
		if i < len(other.bits) {
			b = other.bits[i]
		}
		result.bits[i] = a ^ b
	}
	return result
}

func popCount(x uint64) int {
	// Brian Kernighan's algorithm
	count := 0
	for x != 0 {
		x &= x - 1
		count++
	}
	return count
}

func (bv *BitVector) MemoryUsage() int {
	return len(bv.bits) * 8 // 8 bytes per uint64
}

// Delta encoding for sequences of integers
type DeltaEncoder struct {
}

func NewDeltaEncoder() *DeltaEncoder {
	return &DeltaEncoder{}
}

func (de *DeltaEncoder) Encode(sequence []uint32) []int32 {
	if len(sequence) == 0 {
		return nil
	}
	
	deltas := make([]int32, len(sequence))
	deltas[0] = int32(sequence[0]) // First value stored as-is
	
	for i := 1; i < len(sequence); i++ {
		deltas[i] = int32(sequence[i]) - int32(sequence[i-1])
	}
	
	return deltas
}

func DecodeDelta(deltas []int32) []uint32 {
	if len(deltas) == 0 {
		return nil
	}
	
	result := make([]uint32, len(deltas))
	result[0] = uint32(deltas[0])
	
	for i := 1; i < len(deltas); i++ {
		result[i] = uint32(int32(result[i-1]) + deltas[i])
	}
	
	return result
}

func (de *DeltaEncoder) CompressionRatio(original []uint32, encoded []int32) float64 {
	if len(original) == 0 {
		return 0
	}
	
	originalBytes := len(original) * 4 // 4 bytes per uint32
	
	// Calculate compressed size using variable-length encoding for deltas
	varEncoder := NewVarIntEncoder()
	compressedBytes := 0
	
	for _, delta := range encoded {
		// Encode signed delta as unsigned using zigzag encoding
		unsigned := uint32((delta << 1) ^ (delta >> 31))
		encodedDelta := varEncoder.EncodeUint32(unsigned)
		compressedBytes += len(encodedDelta)
	}
	
	return float64(originalBytes) / float64(compressedBytes)
}

// Shared string table for deduplication
type SharedStringTable struct {
	mu         sync.RWMutex
	strings    []string
	indices    map[string]uint32
	totalCalls int
}

func NewSharedStringTable() *SharedStringTable {
	return &SharedStringTable{
		indices: make(map[string]uint32),
	}
}

func (sst *SharedStringTable) Add(s string) uint32 {
	sst.mu.Lock()
	defer sst.mu.Unlock()
	
	sst.totalCalls++
	if index, exists := sst.indices[s]; exists {
		return index
	}
	
	index := uint32(len(sst.strings))
	sst.strings = append(sst.strings, s)
	sst.indices[s] = index
	return index
}

func (sst *SharedStringTable) AddString(s string) uint32 {
	return sst.Add(s)
}

func (sst *SharedStringTable) Get(index uint32) (string, bool) {
	sst.mu.RLock()
	defer sst.mu.RUnlock()
	
	if int(index) >= len(sst.strings) {
		return "", false
	}
	return sst.strings[index], true
}

func (sst *SharedStringTable) GetString(index uint32) string {
	sst.mu.RLock()
	defer sst.mu.RUnlock()
	
	if int(index) >= len(sst.strings) {
		return ""
	}
	return sst.strings[index]
}

func (sst *SharedStringTable) Stats() (unique, total, savings int) {
	sst.mu.RLock()
	defer sst.mu.RUnlock()
	
	unique = len(sst.strings)
	total = sst.totalCalls
	
	// Calculate memory savings (rough estimate)
	totalStringBytes := 0
	for _, s := range sst.strings {
		totalStringBytes += len(s)
	}
	
	// Estimate savings from deduplication
	if total > unique && unique > 0 {
		savings = (total - unique) * (totalStringBytes / unique)
	}
	
	return
}

func (sst *SharedStringTable) Size() int {
	sst.mu.RLock()
	defer sst.mu.RUnlock()
	return len(sst.strings)
}

func (sst *SharedStringTable) MemoryUsage() int {
	sst.mu.RLock()
	defer sst.mu.RUnlock()
	
	totalSize := 0
	for _, s := range sst.strings {
		totalSize += len(s)
	}
	return totalSize + len(sst.indices)*8 // Approximate map overhead
}

// Compact record format with field packing
type CompactRecord struct {
	data   []byte
	fields [3]uint32 // Support for 3 fields with different bit sizes
}

func NewCompactRecord() *CompactRecord {
	return &CompactRecord{
		data: make([]byte, 0, 64),
	}
}

// Field packing: Field 0 (8 bits), Field 1 (16 bits), Field 2 (20 bits)
func (cr *CompactRecord) SetField(index int, value uint32) {
	if index < 0 || index >= 3 {
		return
	}
	
	switch index {
	case 0:
		cr.fields[0] = value & 0xFF // 8 bits
	case 1:
		cr.fields[1] = value & 0xFFFF // 16 bits
	case 2:
		cr.fields[2] = value & 0xFFFFF // 20 bits
	}
}

func (cr *CompactRecord) GetField(index int) uint32 {
	if index < 0 || index >= 3 {
		return 0
	}
	return cr.fields[index]
}

func (cr *CompactRecord) Serialize() []byte {
	// Pack fields into bytes: 8 + 16 + 20 = 44 bits = 6 bytes
	packed := make([]byte, 6)
	
	// Field 0 (8 bits) goes into byte 0
	packed[0] = byte(cr.fields[0])
	
	// Field 1 (16 bits) goes into bytes 1-2
	packed[1] = byte(cr.fields[1])
	packed[2] = byte(cr.fields[1] >> 8)
	
	// Field 2 (20 bits) goes into bytes 3-5 (with 4 bits unused)
	packed[3] = byte(cr.fields[2])
	packed[4] = byte(cr.fields[2] >> 8)
	packed[5] = byte(cr.fields[2] >> 16)
	
	return packed
}

func DeserializeCompactRecord(data []byte) *CompactRecord {
	if len(data) < 6 {
		return NewCompactRecord()
	}
	
	cr := NewCompactRecord()
	
	// Unpack fields
	cr.fields[0] = uint32(data[0])
	cr.fields[1] = uint32(data[1]) | (uint32(data[2]) << 8)
	cr.fields[2] = uint32(data[3]) | (uint32(data[4]) << 8) | (uint32(data[5]) << 16)
	
	return cr
}

func (cr *CompactRecord) AddUint8(value uint8) {
	cr.data = append(cr.data, value)
}

func (cr *CompactRecord) AddUint16(value uint16) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, value)
	cr.data = append(cr.data, buf...)
}

func (cr *CompactRecord) AddUint32(value uint32) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, value)
	cr.data = append(cr.data, buf...)
}

func (cr *CompactRecord) AddVarUint32(value uint32) {
	encoder := NewVarIntEncoder()
	encoded := encoder.EncodeUint32(value)
	cr.data = append(cr.data, encoded...)
}

func (cr *CompactRecord) AddString(s string) {
	// Length-prefixed string
	cr.AddVarUint32(uint32(len(s)))
	cr.data = append(cr.data, []byte(s)...)
}

func (cr *CompactRecord) AddFloat32(value float32) {
	bits := math.Float32bits(value)
	cr.AddUint32(bits)
}

func (cr *CompactRecord) GetData() []byte {
	return cr.data
}

func (cr *CompactRecord) Size() int {
	return len(cr.data)
}

// Compact record reader
type CompactRecordReader struct {
	data   []byte
	offset int
}

func NewCompactRecordReader(data []byte) *CompactRecordReader {
	return &CompactRecordReader{
		data: data,
	}
}

func (crr *CompactRecordReader) ReadUint8() (uint8, bool) {
	if crr.offset >= len(crr.data) {
		return 0, false
	}
	value := crr.data[crr.offset]
	crr.offset++
	return value, true
}

func (crr *CompactRecordReader) ReadUint16() (uint16, bool) {
	if crr.offset+2 > len(crr.data) {
		return 0, false
	}
	value := binary.LittleEndian.Uint16(crr.data[crr.offset:])
	crr.offset += 2
	return value, true
}

func (crr *CompactRecordReader) ReadUint32() (uint32, bool) {
	if crr.offset+4 > len(crr.data) {
		return 0, false
	}
	value := binary.LittleEndian.Uint32(crr.data[crr.offset:])
	crr.offset += 4
	return value, true
}

func (crr *CompactRecordReader) ReadVarUint32() (uint32, bool) {
	value, bytesRead := DecodeUint32(crr.data[crr.offset:])
	if bytesRead == 0 {
		return 0, false
	}
	crr.offset += bytesRead
	return value, true
}

func (crr *CompactRecordReader) ReadString() (string, bool) {
	length, ok := crr.ReadVarUint32()
	if !ok {
		return "", false
	}
	
	if crr.offset+int(length) > len(crr.data) {
		return "", false
	}
	
	value := string(crr.data[crr.offset : crr.offset+int(length)])
	crr.offset += int(length)
	return value, true
}

func (crr *CompactRecordReader) ReadFloat32() (float32, bool) {
	bits, ok := crr.ReadUint32()
	if !ok {
		return 0, false
	}
	return math.Float32frombits(bits), true
}

// Utility functions for format analysis
func AnalyzeCompression() {
	fmt.Println("=== Compression Analysis ===")
	
	// VarInt encoding efficiency
	values := []uint32{1, 127, 128, 16383, 16384, 2097151, 2097152}
	encoder := NewVarIntEncoder()
	
	fmt.Println("VarInt Encoding:")
	for _, value := range values {
		encoded := encoder.EncodeUint32(value)
		fmt.Printf("  %d -> %d bytes (vs 4 bytes fixed)\n", value, len(encoded))
	}
	
	// Bit vector efficiency
	bv := NewBitVector(1000)
	for i := 0; i < 100; i++ {
		bv.Set(i*10, true)
	}
	fmt.Printf("\nBit Vector: %d bits in %d bytes (vs %d bytes for bool array)\n",
		1000, bv.MemoryUsage(), 1000)
	
	// Sparse array efficiency
	csa := NewCompressedSparseArray(10000)
	for i := 0; i < 100; i++ {
		csa.Set(i*100, uint32(i))
	}
	fmt.Printf("\nSparse Array: compression ratio %.2fx\n", csa.CompressionRatio())
}
package algorithms

import (
	"math"
)

const (
	// MIN_MERGE is the minimum size of a run that can be merged
	MIN_MERGE = 32
	// MIN_GALLOP is the minimum length of the initial galloping mode
	MIN_GALLOP = 7
)

// Run represents a sorted run in the array
type Run struct {
	start int
	length int
}

// TimSort implements the Tim sort algorithm
func TimSort(data []int) {
	n := len(data)
	if n <= 1 {
		return
	}
	
	// Use insertion sort for small arrays
	if n < MIN_MERGE {
		binaryInsertionSort(data, 0, n)
		return
	}
	
	ts := &timSorter{
		data: data,
		n: n,
		minGallop: MIN_GALLOP,
	}
	
	ts.sort()
}

// timSorter holds the state for Tim sort
type timSorter struct {
	data []int
	n int
	minGallop int
	runs []Run
	tmp []int
}

// sort performs the main Tim sort algorithm
func (ts *timSorter) sort() {
	minRun := computeMinRunLength(ts.n)
	
	lo := 0
	for lo < ts.n {
		// Find next run
		runLen := countRunAndMakeAscending(ts.data, lo, ts.n)
		
		// If run is too short, extend it using binary insertion sort
		if runLen < minRun {
			force := ts.n - lo
			if force > minRun {
				force = minRun
			}
			binaryInsertionSort(ts.data, lo, lo+force)
			runLen = force
		}
		
		// Push run onto pending-run stack and maybe merge
		ts.pushRun(lo, runLen)
		ts.mergeCollapse()
		
		lo += runLen
	}
	
	// Merge all remaining runs
	ts.mergeForceCollapse()
}

// computeMinRunLength computes minimum run length for Tim sort
func computeMinRunLength(n int) int {
	r := 0
	for n >= MIN_MERGE {
		r |= n & 1
		n >>= 1
	}
	return n + r
}

// countRunAndMakeAscending finds a run and ensures it's ascending
func countRunAndMakeAscending(data []int, lo, hi int) int {
	if lo == hi-1 {
		return 1
	}
	
	runHi := lo + 1
	if data[lo] > data[runHi] {
		// Descending run, find end and reverse
		for runHi < hi && data[runHi-1] > data[runHi] {
			runHi++
		}
		reverseRange(data, lo, runHi)
	} else {
		// Ascending run, find end
		for runHi < hi && data[runHi-1] <= data[runHi] {
			runHi++
		}
	}
	
	return runHi - lo
}

// reverseRange reverses elements in the range [lo, hi)
func reverseRange(data []int, lo, hi int) {
	hi--
	for lo < hi {
		data[lo], data[hi] = data[hi], data[lo]
		lo++
		hi--
	}
}

// binaryInsertionSort performs binary insertion sort on range [lo, hi)
func binaryInsertionSort(data []int, lo, hi int) {
	for start := lo + 1; start < hi; start++ {
		pivot := data[start]
		
		// Binary search for insertion point
		left, right := lo, start
		for left < right {
			mid := (left + right) >> 1
			if pivot < data[mid] {
				right = mid
			} else {
				left = mid + 1
			}
		}
		
		// Shift elements and insert
		n := start - left
		for i := 0; i < n; i++ {
			data[start-i] = data[start-i-1]
		}
		data[left] = pivot
	}
}

// pushRun pushes a run onto the stack
func (ts *timSorter) pushRun(runBase, runLen int) {
	ts.runs = append(ts.runs, Run{start: runBase, length: runLen})
}

// mergeCollapse merges runs to maintain stack invariants
func (ts *timSorter) mergeCollapse() {
	for len(ts.runs) > 1 {
		n := len(ts.runs) - 2
		
		// Check invariants and merge if necessary
		if (n > 0 && ts.runs[n-1].length <= ts.runs[n].length+ts.runs[n+1].length) ||
			(n > 1 && ts.runs[n-2].length <= ts.runs[n-1].length+ts.runs[n].length) {
			
			if ts.runs[n-1].length < ts.runs[n+1].length {
				n--
			}
			ts.mergeAt(n)
		} else if ts.runs[n].length <= ts.runs[n+1].length {
			ts.mergeAt(n)
		} else {
			break
		}
	}
}

// mergeForceCollapse merges all remaining runs
func (ts *timSorter) mergeForceCollapse() {
	for len(ts.runs) > 1 {
		n := len(ts.runs) - 2
		if n > 0 && ts.runs[n-1].length < ts.runs[n+1].length {
			n--
		}
		ts.mergeAt(n)
	}
}

// mergeAt merges runs at index i and i+1
func (ts *timSorter) mergeAt(i int) {
	base1 := ts.runs[i].start
	len1 := ts.runs[i].length
	base2 := ts.runs[i+1].start
	len2 := ts.runs[i+1].length
	
	// Update run info
	ts.runs[i].length = len1 + len2
	if i == len(ts.runs)-3 {
		ts.runs[i+1] = ts.runs[i+2]
	}
	ts.runs = ts.runs[:len(ts.runs)-1]
	
	// Find where first element of run2 goes in run1
	k := gallopRight(ts.data[base2], ts.data, base1, len1, 0)
	base1 += k
	len1 -= k
	if len1 == 0 {
		return
	}
	
	// Find where last element of run1 goes in run2
	len2 = gallopLeft(ts.data[base1+len1-1], ts.data, base2, len2, len2-1)
	if len2 == 0 {
		return
	}
	
	// Merge remaining runs
	if len1 <= len2 {
		ts.mergeLo(base1, len1, base2, len2)
	} else {
		ts.mergeHi(base1, len1, base2, len2)
	}
}

// gallopRight finds insertion point for key in sorted array using galloping search
func gallopRight(key int, data []int, base, len, hint int) int {
	lastOfs := 0
	ofs := 1
	
	if key < data[base+hint] {
		// Gallop left
		maxOfs := hint + 1
		for ofs < maxOfs && key < data[base+hint-ofs] {
			lastOfs = ofs
			ofs = (ofs << 1) + 1
		}
		if ofs > maxOfs {
			ofs = maxOfs
		}
		
		tmp := lastOfs
		lastOfs = hint - ofs
		ofs = hint - tmp
	} else {
		// Gallop right
		maxOfs := len - hint
		for ofs < maxOfs && key >= data[base+hint+ofs] {
			lastOfs = ofs
			ofs = (ofs << 1) + 1
		}
		if ofs > maxOfs {
			ofs = maxOfs
		}
		
		lastOfs += hint
		ofs += hint
	}
	
	// Binary search
	lastOfs++
	for lastOfs < ofs {
		m := lastOfs + ((ofs - lastOfs) >> 1)
		if key < data[base+m] {
			ofs = m
		} else {
			lastOfs = m + 1
		}
	}
	
	return ofs
}

// gallopLeft finds insertion point for key in sorted array using galloping search (left)
func gallopLeft(key int, data []int, base, len, hint int) int {
	lastOfs := 0
	ofs := 1
	
	if key > data[base+hint] {
		// Gallop right
		maxOfs := len - hint
		for ofs < maxOfs && key > data[base+hint+ofs] {
			lastOfs = ofs
			ofs = (ofs << 1) + 1
		}
		if ofs > maxOfs {
			ofs = maxOfs
		}
		
		lastOfs += hint
		ofs += hint
	} else {
		// Gallop left
		maxOfs := hint + 1
		for ofs < maxOfs && key <= data[base+hint-ofs] {
			lastOfs = ofs
			ofs = (ofs << 1) + 1
		}
		if ofs > maxOfs {
			ofs = maxOfs
		}
		
		tmp := lastOfs
		lastOfs = hint - ofs
		ofs = hint - tmp
	}
	
	// Binary search
	lastOfs++
	for lastOfs < ofs {
		m := lastOfs + ((ofs - lastOfs) >> 1)
		if key <= data[base+m] {
			ofs = m
		} else {
			lastOfs = m + 1
		}
	}
	
	return ofs
}

// mergeLo merges two adjacent runs with the first run copied to temp array
func (ts *timSorter) mergeLo(base1, len1, base2, len2 int) {
	// Ensure temp array is large enough
	if len(ts.tmp) < len1 {
		ts.tmp = make([]int, len1)
	}
	
	// Copy first run to temp array
	copy(ts.tmp, ts.data[base1:base1+len1])
	
	dest := base1
	cursor1 := 0
	cursor2 := base2
	
	// Move first element of second run
	ts.data[dest] = ts.data[cursor2]
	dest++
	cursor2++
	len2--
	
	if len2 == 0 {
		copy(ts.data[dest:], ts.tmp[cursor1:cursor1+len1])
		return
	}
	if len1 == 1 {
		copy(ts.data[dest:dest+len2], ts.data[cursor2:cursor2+len2])
		ts.data[dest+len2] = ts.tmp[cursor1]
		return
	}
	
	minGallop := ts.minGallop
	for {
		count1 := 0
		count2 := 0
		
		// Simple merge until galloping becomes beneficial
		for {
			if ts.tmp[cursor1] <= ts.data[cursor2] {
				ts.data[dest] = ts.tmp[cursor1]
				dest++
				cursor1++
				len1--
				count1++
				count2 = 0
				if len1 == 0 {
					goto outerLoop
				}
			} else {
				ts.data[dest] = ts.data[cursor2]
				dest++
				cursor2++
				len2--
				count2++
				count1 = 0
				if len2 == 0 {
					goto outerLoop
				}
			}
			
			if (count1 | count2) >= minGallop {
				break
			}
		}
		
		// Galloping mode
		for {
			count1 = gallopRight(ts.data[cursor2], ts.tmp, cursor1, len1, 0)
			if count1 != 0 {
				copy(ts.data[dest:], ts.tmp[cursor1:cursor1+count1])
				dest += count1
				cursor1 += count1
				len1 -= count1
				if len1 == 0 {
					goto outerLoop
				}
			}
			ts.data[dest] = ts.data[cursor2]
			dest++
			cursor2++
			len2--
			if len2 == 0 {
				goto outerLoop
			}
			
			count2 = gallopLeft(ts.tmp[cursor1], ts.data, cursor2, len2, 0)
			if count2 != 0 {
				copy(ts.data[dest:], ts.data[cursor2:cursor2+count2])
				dest += count2
				cursor2 += count2
				len2 -= count2
				if len2 == 0 {
					goto outerLoop
				}
			}
			ts.data[dest] = ts.tmp[cursor1]
			dest++
			cursor1++
			len1--
			if len1 == 0 {
				goto outerLoop
			}
			
			minGallop--
			if count1 < MIN_GALLOP && count2 < MIN_GALLOP {
				if minGallop < 0 {
					minGallop = 0
				}
				minGallop += 2
				break
			}
		}
	}
	
outerLoop:
	ts.minGallop = int(math.Max(float64(minGallop), 1))
	
	if len1 == 0 {
		copy(ts.data[dest:dest+len2], ts.data[cursor2:cursor2+len2])
	} else {
		copy(ts.data[dest:dest+len1], ts.tmp[cursor1:cursor1+len1])
	}
}

// mergeHi merges two adjacent runs with the second run copied to temp array
func (ts *timSorter) mergeHi(base1, len1, base2, len2 int) {
	// Ensure temp array is large enough
	if len(ts.tmp) < len2 {
		ts.tmp = make([]int, len2)
	}
	
	// Copy second run to temp array
	copy(ts.tmp, ts.data[base2:base2+len2])
	
	dest := base2 + len2 - 1
	cursor1 := base1 + len1 - 1
	cursor2 := len2 - 1
	
	// Move last element of first run
	ts.data[dest] = ts.data[cursor1]
	dest--
	cursor1--
	len1--
	
	if len1 == 0 {
		copy(ts.data[dest-len2+1:dest+1], ts.tmp[:len2])
		return
	}
	if len2 == 1 {
		dest -= len1
		cursor1 -= len1
		copy(ts.data[dest+1:dest+1+len1], ts.data[cursor1+1:cursor1+1+len1])
		ts.data[dest] = ts.tmp[cursor2]
		return
	}
	
	minGallop := ts.minGallop
	for {
		count1 := 0
		count2 := 0
		
		// Simple merge until galloping becomes beneficial
		for {
			if ts.tmp[cursor2] < ts.data[cursor1] {
				ts.data[dest] = ts.data[cursor1]
				dest--
				cursor1--
				len1--
				count1++
				count2 = 0
				if len1 == 0 {
					goto outerLoop
				}
			} else {
				ts.data[dest] = ts.tmp[cursor2]
				dest--
				cursor2--
				len2--
				count2++
				count1 = 0
				if len2 == 0 {
					goto outerLoop
				}
			}
			
			if (count1 | count2) >= minGallop {
				break
			}
		}
		
		// Galloping mode (similar to mergeLo but in reverse)
		for {
			count1 = len1 - gallopRight(ts.tmp[cursor2], ts.data, base1, len1, len1-1)
			if count1 != 0 {
				dest -= count1
				cursor1 -= count1
				len1 -= count1
				copy(ts.data[dest+1:dest+1+count1], ts.data[cursor1+1:cursor1+1+count1])
				if len1 == 0 {
					goto outerLoop
				}
			}
			ts.data[dest] = ts.tmp[cursor2]
			dest--
			cursor2--
			len2--
			if len2 == 0 {
				goto outerLoop
			}
			
			count2 = len2 - gallopLeft(ts.data[cursor1], ts.tmp, 0, len2, len2-1)
			if count2 != 0 {
				dest -= count2
				cursor2 -= count2
				len2 -= count2
				copy(ts.data[dest+1:dest+1+count2], ts.tmp[cursor2+1:cursor2+1+count2])
				if len2 == 0 {
					goto outerLoop
				}
			}
			ts.data[dest] = ts.data[cursor1]
			dest--
			cursor1--
			len1--
			if len1 == 0 {
				goto outerLoop
			}
			
			minGallop--
			if count1 < MIN_GALLOP && count2 < MIN_GALLOP {
				if minGallop < 0 {
					minGallop = 0
				}
				minGallop += 2
				break
			}
		}
	}
	
outerLoop:
	ts.minGallop = int(math.Max(float64(minGallop), 1))
	
	if len2 == 0 {
		dest -= len1
		cursor1 -= len1
		copy(ts.data[dest+1:dest+1+len1], ts.data[cursor1+1:cursor1+1+len1])
	} else {
		copy(ts.data[dest-len2+1:dest+1], ts.tmp[:len2])
	}
}
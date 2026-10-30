package main

import (
	_ "internal/runtime/gc/scan"
	"unsafe"
)

type mask [16]uintptr

// These entries select actual SDK-package Go bodies. The portable owner test
// does not construct or call an expander PC or use the native AX/Z1/Z2 ABI.

//go:linkname expand internal/runtime/gc/scan.ExpandAVX512
func expand(sizeClass int, packed, unpacked *mask)

//go:linkname rawScan internal/runtime/gc/scan.scanSpanPackedAVX512
func rawScan(mem unsafe.Pointer, bufp *uintptr, objects *mask, sizeClass uintptr, pointers *mask) int32

//go:linkname filter internal/runtime/gc/scan.FilterNilAVX512
func filter(bufp *uintptr, n int32) int32

var factors = [...]int{
	1, 2, 3, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32,
	36, 40, 44, 48, 52, 56, 60, 64,
}

const sentinel = uintptr(0x73928a51)

func require(ok bool) {
	if !ok {
		panic("independent gc scan oracle differs")
	}
}

func marked(object, objects, pattern int) bool {
	if object >= objects {
		return false
	}
	switch pattern {
	case 0:
		return false
	case 1:
		return object%2 == 0
	case 2:
		return object == 0 || object == objects-1 || object%64 == 0
	}
	return true
}

func checkSizeClass(sizeClass, factor, pattern int) {
	objects := 1024 / factor
	var marks, pointers mask
	var wantExpanded mask
	for object := 0; object < objects; object++ {
		if !marked(object, objects, pattern) {
			continue
		}
		marks[object/64] |= uintptr(1) << (uint(object) % 64)
		// Independent object-major expansion, unlike the word-major body.
		for within := 0; within < factor; within++ {
			word := object*factor + within
			wantExpanded[word/64] |= uintptr(1) << (uint(word) % 64)
		}
	}
	output := struct {
		before uintptr
		bits   mask
		after  uintptr
	}{before: sentinel, after: sentinel}
	for i := range output.bits {
		output.bits[i] = ^uintptr(0)
	}
	expand(sizeClass, &marks, &output.bits)
	require(output.bits == wantExpanded)
	require(output.before == sentinel && output.after == sentinel)
	alias := marks
	expand(sizeClass, &alias, &alias)
	require(alias == wantExpanded)
	var zero mask
	expand(sizeClass, &zero, &output.bits)
	require(output.bits == zero) // A reused, dirty destination must be cleared.

	var memory [1024]uintptr
	var wantRaw, wantFiltered [1024]uintptr
	rawCount, filteredCount := 0, 0
	values := [...]uintptr{0, 1, 4095, 4096, ^uintptr(0), 0x81726354}
	for word := range memory {
		memory[word] = values[word%len(values)]
		if word%5 == 0 {
			continue
		}
		pointers[word/64] |= uintptr(1) << (uint(word) % 64)
		if !marked(word/factor, objects, pattern) {
			continue
		}
		wantRaw[rawCount] = memory[word]
		rawCount++
		if memory[word] != 0 {
			wantFiltered[filteredCount] = memory[word]
			filteredCount++
		}
	}
	var buf [1026]uintptr
	buf[0], buf[1025] = sentinel, sentinel
	gotRaw := rawScan(unsafe.Pointer(&memory[0]), &buf[1], &marks, uintptr(sizeClass), &pointers)
	require(gotRaw == int32(rawCount)) // Raw scan includes selected zero values.
	for i := 0; i < rawCount; i++ {
		require(buf[i+1] == wantRaw[i])
	}
	gotFiltered := filter(&buf[1], gotRaw)
	require(gotFiltered == int32(filteredCount))
	for i := 0; i < filteredCount; i++ {
		require(buf[i+1] == wantFiltered[i]) // Stable order, retaining 1 and 4095.
	}
	require(buf[0] == sentinel && buf[1025] == sentinel)
}

func checkFilter() {
	require(filter(nil, 0) == 0)
	require(filter(nil, -1) == 0)
	// Include every scalar/vector boundary through four original SIMD groups.
	for n := 0; n <= 33; n++ {
		for pattern := 0; pattern < 3; pattern++ {
			var buf, want [35]uintptr
			buf[0], buf[n+1] = sentinel, sentinel
			count := 0
			for i := 0; i < n; i++ {
				value := uintptr(i + 1)
				if (pattern == 0) || (pattern == 1 && i%2 == 0) {
					value = 0
				}
				buf[i+1] = value
				if value != 0 {
					want[count] = value
					count++
				}
			}
			require(filter(&buf[1], int32(n)) == int32(count))
			for i := 0; i < count; i++ {
				require(buf[i+1] == want[i])
			}
			require(buf[0] == sentinel && buf[n+1] == sentinel)
		}
	}
}

func checkPartialSizeClass(sizeClass, factor int) bool {
	if 1024%factor == 0 {
		return false // This class has no partial object in a 1024-word page.
	}
	object := 1024 / factor
	var marks, want mask
	marks[object/64] = uintptr(1) << (uint(object) % 64)
	// Independent object-major expansion, clipped at the actual page boundary.
	// Deliberately mark only the final partial object, unlike all 104 original
	// full-object patterns. Its lower output bits must remain clear.
	for within := 0; within < factor; within++ {
		word := object*factor + within
		if word >= 1024 {
			break
		}
		want[word/64] |= uintptr(1) << (uint(word) % 64)
	}
	if factor == 3 {
		require(object == 341 && want[15] == uintptr(1)<<63)
	}
	output := struct {
		before uintptr
		bits   mask
		after  uintptr
	}{before: sentinel, after: sentinel}
	for i := range output.bits {
		output.bits[i] = ^uintptr(0)
	}
	input := marks
	expand(sizeClass, &marks, &output.bits)
	require(output.bits == want && marks == input)
	require(output.before == sentinel && output.after == sentinel)
	output.bits = marks
	expand(sizeClass, &output.bits, &output.bits)
	require(output.bits == want)
	require(output.before == sentinel && output.after == sentinel)
	return true
}

func rejectsSizeClass(sizeClass int) {
	rejected := false
	func() {
		defer func() {
			rejected = recover() != nil
		}()
		var input, output mask
		expand(sizeClass, &input, &output)
	}()
	require(rejected)
}

func main() {
	tails := 0
	for i, factor := range factors {
		for pattern := 0; pattern < 4; pattern++ {
			checkSizeClass(i+1, factor, pattern)
		}
		if checkPartialSizeClass(i+1, factor) {
			tails++
		}
	}
	checkFilter()
	for _, sizeClass := range []int{-1, 0, 27, 100000} {
		rejectsSizeClass(sizeClass)
	}
	println("gc scan owners", len(factors), "classes", len(factors)*4, "patterns", tails, "partial tails")
}

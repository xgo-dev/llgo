package gotest

import (
	"fmt"
	"hash/maphash"
	"math/bits"
	"reflect"
	"strings"
	"testing"
)

func TestHashAllSizesAndBitSensitivity(t *testing.T) {
	seed := maphash.MakeSeed()
	sizes := []int{
		0, 1, 2, 3, 4, 7, 8, 15, 16,
		17, 18, 24, 31, 32,
		33, 47, 48, 63, 64,
		65, 80, 96, 127, 128,
		129, 130, 192, 255, 256, 257, 384, 512, 1025, 4096,
	}

	// Zero buffers of distinct lengths must produce distinct hashes.
	zeroHashes := make(map[uint64]int, len(sizes))
	for _, n := range sizes {
		buf := make([]byte, n)
		h1 := maphash.Bytes(seed, buf)
		h2 := maphash.Bytes(seed, buf)
		if h1 != h2 {
			t.Fatalf("non-deterministic hash for len=%d: %#x != %#x", n, h1, h2)
		}
		if prev, dup := zeroHashes[h1]; dup {
			t.Fatalf("hash collision between zero buffers of len %d and %d: %#x", prev, n, h1)
		}
		zeroHashes[h1] = n
	}

	// Flipping any byte at boundary positions must change the hash with strong avalanche.
	var totalFlipped, samples int
	for _, n := range sizes {
		if n == 0 {
			continue
		}
		buf := make([]byte, n)
		for i := range buf {
			buf[i] = byte(i*37 + 11)
		}
		base := maphash.Bytes(seed, buf)
		// Check streaming maphash.Hash matches one-shot maphash.Bytes.
		var mh maphash.Hash
		mh.SetSeed(seed)
		_, _ = mh.Write(buf)
		if got := mh.Sum64(); got != base {
			t.Fatalf("maphash.Hash (%#x) != maphash.Bytes (%#x) for len=%d", got, base, n)
		}

		// Probe start, boundaries, middle, and tail byte positions.
		positions := []int{0, n / 2, n - 1}
		for _, boundary := range []int{1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129, 255, 256} {
			if boundary < n {
				positions = append(positions, boundary)
			}
		}
		seenPos := make(map[int]bool, len(positions))
		for _, pos := range positions {
			if seenPos[pos] {
				continue
			}
			seenPos[pos] = true
			buf[pos] ^= 1 << uint(pos%8)
			flipped := maphash.Bytes(seed, buf)
			buf[pos] ^= 1 << uint(pos%8)
			if flipped == base {
				t.Fatalf("single-bit flip at pos %d (len=%d) did not change hash %#x", pos, n, base)
			}
			totalFlipped += bits.OnesCount64(base ^ flipped)
			samples++
		}
	}
	avgBits := float64(totalFlipped) / float64(samples)
	if avgBits < 24 || avgBits > 40 {
		t.Fatalf("poor avalanche: average flipped bits = %.2f over %d samples (want ~32)", avgBits, samples)
	}

	// Repeating identical 128-byte blocks must not cancel out. Use
	// maphash.Comparable on full strings (not maphash.Bytes, which chunks at
	// 128 bytes) so inputs >128 B reach the multi-block memhashAES loop.
	block := make([]byte, 128)
	for i := range block {
		block[i] = byte(i*13 + 5)
	}
	rep1 := string(block)
	rep2 := rep1 + rep1
	rep3 := rep2 + rep1
	rep4 := rep3 + rep1
	h1 := maphash.Comparable(seed, rep1)
	h2 := maphash.Comparable(seed, rep2)
	h3 := maphash.Comparable(seed, rep3)
	h4 := maphash.Comparable(seed, rep4)
	seenRep := map[uint64]int{h1: 1, h2: 2, h3: 3, h4: 4}
	if len(seenRep) != 4 {
		t.Fatalf("repeating 128B blocks collided: h1=%#x h2=%#x h3=%#x h4=%#x", h1, h2, h3, h4)
	}

	// Exercise the unchunked >128 B path directly via maphash.Comparable on
	// strings and fixed arrays: verify length differentiation, multi-block
	// mixing (n >= 2), and per-byte tail diffusion into both low 16 bits and
	// the top byte.
	longSizes := []int{129, 130, 192, 255, 256, 257, 384, 512, 1025, 4096}
	longZeroHashes := make(map[uint64]int, len(longSizes))
	for _, n := range longSizes {
		s := strings.Repeat("\x00", n)
		h := maphash.Comparable(seed, s)
		if prev, dup := longZeroHashes[h]; dup {
			t.Fatalf("long zero string collision between len %d and %d: %#x", prev, n, h)
		}
		longZeroHashes[h] = n
	}

	var longTotalFlipped, longSamples, low16Changed, top8Changed int
	for _, n := range []int{257, 384, 512} {
		buf := make([]byte, n)
		for i := range buf {
			buf[i] = byte(i*41 + 7)
		}
		base := maphash.Comparable(seed, string(buf))
		// Flip every byte of the final 128-byte tail plus early/mid-block bytes.
		positions := make([]int, 0, 144)
		for _, pos := range []int{0, 1, 15, 16, 63, 64, 127, 128, 129, 191, 192, 255} {
			if pos < n-128 {
				positions = append(positions, pos)
			}
		}
		for pos := n - 128; pos < n; pos++ {
			positions = append(positions, pos)
		}
		for _, pos := range positions {
			buf[pos] ^= 1 << uint(pos%8)
			flipped := maphash.Comparable(seed, string(buf))
			buf[pos] ^= 1 << uint(pos%8)
			diff := base ^ flipped
			if diff == 0 {
				t.Fatalf("long string single-bit flip at pos %d (len=%d) did not change hash %#x", pos, n, base)
			}
			longTotalFlipped += bits.OnesCount64(diff)
			longSamples++
			if pos >= n-128 {
				if diff&0xffff != 0 {
					low16Changed++
				}
				if diff>>56 != 0 {
					top8Changed++
				}
			}
		}
	}
	longAvg := float64(longTotalFlipped) / float64(longSamples)
	if longAvg < 24 || longAvg > 40 {
		t.Fatalf("poor >128B avalanche: average flipped bits = %.2f over %d samples (want ~32)", longAvg, longSamples)
	}
	// Across 3*128 = 384 tail flips, both the low 16 bits and top byte should
	// change on nearly every flip (expected ~384 and ~382.5 respectively).
	if low16Changed < 370 {
		t.Fatalf("poor >128B tail low-16-bit mixing: %d/384 changed", low16Changed)
	}
	if top8Changed < 360 {
		t.Fatalf("poor >128B tail top-byte mixing: %d/384 changed", top8Changed)
	}

	// Fixed-size array >256 B also hits memhash directly without chunking.
	var arr [384]byte
	for i := range arr {
		arr[i] = byte(i*29 + 13)
	}
	mArr := make(map[[384]byte]int, 32)
	arrBase := maphash.Comparable(seed, arr)
	mArr[arr] = 1
	for i := 0; i < 31; i++ {
		alt := arr
		alt[i*12] ^= 1
		if h := maphash.Comparable(seed, alt); h == arrBase {
			t.Fatalf("[384]byte flip at %d did not change hash %#x", i*12, arrBase)
		}
		mArr[alt] = i + 2
	}
	if len(mArr) != 32 || mArr[arr] != 1 {
		t.Fatalf("map[[384]byte]int len=%d base=%d", len(mArr), mArr[arr])
	}
}

func TestMapFastPathMatchesReflect(t *testing.T) {
	const count = 512

	m32 := make(map[uint32]int)
	for i := uint32(0); i < count; i++ {
		m32[i*2654435761+1] = int(i) + 10
	}
	rv32 := reflect.ValueOf(m32)
	for i := uint32(0); i < count; i++ {
		k := i*2654435761 + 1
		if got := m32[k]; got != int(i)+10 {
			t.Fatalf("m32[%d] = %d, want %d", k, got, int(i)+10)
		}
		if v := rv32.MapIndex(reflect.ValueOf(k)); !v.IsValid() || int(v.Int()) != int(i)+10 {
			t.Fatalf("reflect m32[%d] = %v, want %d", k, v, int(i)+10)
		}
	}

	m64 := make(map[uint64]int)
	for i := uint64(0); i < count; i++ {
		m64[i*11400714819323198485+1] = int(i) + 20
	}
	rv64 := reflect.ValueOf(m64)
	for i := uint64(0); i < count; i++ {
		k := i*11400714819323198485 + 1
		if got := m64[k]; got != int(i)+20 {
			t.Fatalf("m64[%d] = %d, want %d", k, got, int(i)+20)
		}
		if v := rv64.MapIndex(reflect.ValueOf(k)); !v.IsValid() || int(v.Int()) != int(i)+20 {
			t.Fatalf("reflect m64[%d] = %v, want %d", k, v, int(i)+20)
		}
	}

	ptrs := make([]*int, count)
	mptr := make(map[*int]int)
	for i := range ptrs {
		v := i
		ptrs[i] = &v
		mptr[ptrs[i]] = i + 30
	}
	rvPtr := reflect.ValueOf(mptr)
	for i, p := range ptrs {
		if got := mptr[p]; got != i+30 {
			t.Fatalf("mptr[%d] = %d, want %d", i, got, i+30)
		}
		if v := rvPtr.MapIndex(reflect.ValueOf(p)); !v.IsValid() || int(v.Int()) != i+30 {
			t.Fatalf("reflect mptr[%d] = %v, want %d", i, v, i+30)
		}
	}

	mstr := make(map[string]int)
	keys := make([]string, 0, count)
	for _, prefixLen := range []int{0, 8, 16, 24, 48, 96, 160, 320, 512} {
		prefix := strings.Repeat("k", prefixLen)
		for i := 0; i < 64; i++ {
			k := fmt.Sprintf("%s-%04d", prefix, i)
			keys = append(keys, k)
			mstr[k] = len(keys)
		}
	}
	rvStr := reflect.ValueOf(mstr)
	for i, k := range keys {
		want := i + 1
		if got := mstr[k]; got != want {
			t.Fatalf("mstr[%q] = %d, want %d", k, got, want)
		}
		if v := rvStr.MapIndex(reflect.ValueOf(k)); !v.IsValid() || int(v.Int()) != want {
			t.Fatalf("reflect mstr[%q] = %v, want %d", k, v, want)
		}
	}
}

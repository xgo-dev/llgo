package gotest

import (
	"bytes"
	"encoding/binary"
	"runtime"
	"testing"
)

func TestAppendBinaryWords(t *testing.T) {
	for _, order := range []binary.AppendByteOrder{binary.LittleEndian, binary.BigEndian} {
		for _, capacity := range []int{2, 5, 32} {
			b := make([]byte, 2, capacity)
			b[0], b[1] = 0xab, 0xcd
			b = order.AppendUint16(b, 0x1234)
			b = order.AppendUint32(b, 0x56789abc)
			b = order.AppendUint64(b, 0xdef0123456789abc)
			want := []byte{0xab, 0xcd, 0x34, 0x12, 0xbc, 0x9a, 0x78, 0x56,
				0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12, 0xf0, 0xde}
			if order == binary.BigEndian {
				want = []byte{0xab, 0xcd, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc,
					0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc}
			}
			runtime.GC()
			if !bytes.Equal(b, want) {
				t.Fatalf("%s, cap %d: got %x, want %x", order, capacity, b, want)
			}
		}
	}
}

type appendBytes []byte

type appendPaddedItem struct {
	p    *int
	tag  uint64
	flag uint8
}

func appendFourBytes(dst appendBytes, src []byte) appendBytes {
	return append(dst, src[1:5]...)
}

func assertAppendOverlap[T any](t *testing.T, name string, capacity int, src, got []T, wantLen int, check func(i int, g T)) {
	t.Helper()
	if len(got) != wantLen {
		t.Fatalf("%s capacity %d: len=%d, want %d", name, capacity, len(got), wantLen)
	}
	for i, g := range got {
		check(i, g)
	}
	if (capacity == 8) != (&got[0] == &src[0]) {
		t.Fatalf("%s capacity %d: unexpected destination alias", name, capacity)
	}
}

func assertAppendOverlapEqual[T comparable](t *testing.T, name string, capacity int, src, got, want []T) {
	t.Helper()
	assertAppendOverlap(t, name, capacity, src, got, len(want), func(i int, g T) {
		if g != want[i] {
			t.Fatalf("%s capacity %d: got %v, want %v", name, capacity, got, want)
		}
	})
}

func TestAppendOverlapAndGrowth(t *testing.T) {
	for _, capacity := range []int{2, 8} {
		b := []byte{0, 1, 2, 3, 4, 5, 6, 7}
		assertAppendOverlapEqual(t, "bytes", capacity, b, []byte(appendFourBytes(b[:2:capacity], b)), []byte{0, 1, 1, 2, 3, 4})

		nums := []int{10, 20, 30, 40, 50, 60, 70, 80}
		assertAppendOverlapEqual(t, "ints", capacity, nums, append(nums[:2:capacity], nums[1:5]...), []int{10, 20, 20, 30, 40, 50})

		strs := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
		gotStrs := append(strs[:2:capacity], strs[1:5]...)
		runtime.GC()
		assertAppendOverlapEqual(t, "strings", capacity, strs, gotStrs, []string{"a", "b", "b", "c", "d", "e"})

		// 24-byte struct with a leading pointer and 7 bytes of tail padding.
		padded := make([]appendPaddedItem, 8)
		for i := range padded {
			v := new(int)
			*v = 100 + i
			padded[i] = appendPaddedItem{p: v, tag: uint64(i*7 + 1), flag: uint8(i + 1)}
		}
		gotPadded := append(padded[:2:capacity], padded[1:5]...)
		runtime.GC()
		wantIdx := []int{0, 1, 1, 2, 3, 4}
		assertAppendOverlap(t, "padded", capacity, padded, gotPadded, len(wantIdx), func(i int, g appendPaddedItem) {
			idx := wantIdx[i]
			if g.p == nil || *g.p != 100+idx || g.tag != uint64(idx*7+1) || g.flag != uint8(idx+1) {
				t.Fatalf("padded capacity %d idx %d: got %+v, want p=%d tag=%d flag=%d", capacity, i, g, 100+idx, idx*7+1, idx+1)
			}
		})
	}

	if got := append([]byte(nil), ""...); got != nil {
		t.Fatalf("append(nil, \"\"...) = %v, want nil", got)
	}
	if got := append([]byte(nil), []byte{}...); got != nil {
		t.Fatalf("append(nil, empty...) = %v, want nil", got)
	}
	if got := append(make([]byte, 0, 8), "hello"...); string(got) != "hello" {
		t.Fatalf("append string = %q, want %q", got, "hello")
	}

	defer func() {
		if recover() == nil {
			t.Fatal("short source did not panic")
		}
	}()
	appendFourBytes(make(appendBytes, 0, 8), []byte{0})
}

type appendTemporaryItem struct {
	p    *int
	a, b uint64
}

//go:noinline
func appendTemporary(s []appendTemporaryItem, v appendTemporaryItem) []appendTemporaryItem {
	return append(s, v)
}

func TestAppendTemporaryValues(t *testing.T) {
	for _, capacity := range []int{0, 128} {
		values := make([]appendTemporaryItem, 0, capacity)
		for i := 0; i < 128; i++ {
			p := new(int)
			*p = i
			values = appendTemporary(values, appendTemporaryItem{p, uint64(i), uint64(i*3 + 1)})
		}
		runtime.GC()
		for i, v := range values {
			if *v.p != i || v.a != uint64(i) || v.b != uint64(i*3+1) {
				t.Fatalf("append %d corrupted", i)
			}
		}
		// In-place compaction must copy the old value before overwriting it.
		out := values[:0]
		for i, v := range values {
			if i%2 == 0 {
				out = appendTemporary(out, v)
			}
		}
		for i, v := range out {
			if *v.p != i*2 {
				t.Fatal("compaction corrupted values")
			}
		}
	}
	a := []int{1, 2, 3, 4}
	a = append(a[:1], a[1], a[0])
	if a[0] != 1 || a[1] != 2 || a[2] != 1 {
		t.Fatal("append argument order")
	}
}

func BenchmarkAppendTemporary(b *testing.B) {
	s := make([]appendTemporaryItem, 0, 1)
	v := appendTemporaryItem{a: 1, b: 2}
	for i := 0; i < b.N; i++ {
		s = appendTemporary(s[:0], v)
	}
	runtime.KeepAlive(s)
}

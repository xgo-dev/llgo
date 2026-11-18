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

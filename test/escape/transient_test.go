package escape

import (
	"runtime"
	"strconv"
	"strings"
	"testing"
)

//go:noinline
func copiedString(b []byte) byte {
	s := string(b)
	b[0] = '!'
	runtime.GC()
	return s[0]
}

//go:noinline
func copiedBytes(s string) byte {
	b := []byte(s)
	b[0] = '!'
	runtime.GC()
	return s[0]
}

//go:noinline
func readOnlyByteSlice(s string) (int, byte) {
	b := []byte(s)
	runtime.GC()
	if len(b) == 0 {
		if b == nil {
			panic("empty string produced nil bytes")
		}
		return 0, 0
	}
	return len(b), b[0]
}

//go:noinline
func temporaryStringEquals(b []byte, s string) bool { return string(b) == s }

//go:noinline
func temporaryStringKey(m map[string]int, b []byte) int { return m[string(b)] }

//go:noinline
func mutateThenString(b []byte) string { b[0] = '!'; return "aaa" }

//go:noinline
func compareConversionOrder(b []byte) bool {
	s := string(b)
	other := mutateThenString(b)
	return s == other
}

//go:noinline
func runeRoundTrip(s string) (int, rune, rune) {
	r := []rune(s)
	runtime.GC()
	return len(r), r[0], r[len(r)-1]
}

//go:noinline
func runeString(rs []rune) (int, byte, byte) {
	s := string(rs)
	rs[0] = '!'
	runtime.GC()
	return len(s), s[0], s[len(s)-1]
}

func TestTransientStringCopiesAndFallbacks(t *testing.T) {
	for _, n := range []int{0, 16, 128, 1024} {
		s := strings.Repeat("a", n)
		b := []byte(s)
		if !temporaryStringEquals(b, s) || temporaryStringKey(map[string]int{s: 41}, b) != 41 {
			t.Fatalf("temporary string length %d", n)
		}
	}
	if !compareConversionOrder([]byte("aaa")) {
		t.Fatal("conversion borrowed across source mutation")
	}
	for _, n := range []int{0, 16, 128, 1024} {
		s := strings.Repeat("a", n)
		size, first := readOnlyByteSlice(s)
		if size != n || n > 0 && first != 'a' {
			t.Fatalf("readonly bytes length %d: %d %d", n, size, first)
		}
	}
	for _, n := range []int{1, 31, 32, 33, 128} {
		s := strings.Repeat("a", n)
		if got := copiedString([]byte(s)); got != 'a' {
			t.Fatalf("string copy length %d: %d", n, got)
		}
		if got := copiedBytes(s); got != 'a' {
			t.Fatalf("byte copy length %d: %d", n, got)
		}
	}
	for _, s := range []string{"a", "世", strings.Repeat("世", 32), strings.Repeat("世", 33), "a\xff\xfez"} {
		want := []rune(s)
		n, first, last := runeRoundTrip(s)
		if n != len(want) || first != want[0] || last != want[len(want)-1] {
			t.Fatalf("runes %q: %d %U %U", s, n, first, last)
		}
	}
	for _, n := range []int{1, 32, 33, 100} {
		r := make([]rune, n)
		for i := range r {
			r[i] = 'a'
		}
		if size, first, last := runeString(r); size != n || first != 'a' || last != 'a' {
			t.Fatalf("rune string length %d: %d %d %d", n, size, first, last)
		}
	}
	if n, first, last := runeString([]rune{-1, 0xd800, 0x110000}); n != 9 || first != 0xef || last != 0xbd {
		t.Fatalf("invalid runes: %d %x %x", n, first, last)
	}
}

//go:noinline
func localSliceChild(n int) *int {
	x := 37
	s := make([]*int, n)
	s[0] = &x
	return s[0]
}

//go:noinline
func localMap(n int) int {
	m := make(map[int]int, n)
	for i := range n {
		m[i] = i + 1
	}
	runtime.GC()
	sum := 0
	for k, v := range m {
		sum += k + v
	}
	clear(m)
	return sum
}

func TestLocalContainerStorageGrowthAndGC(t *testing.T) {
	for _, n := range []int{1, 4, 8, 9, 64, 128} {
		p := localSliceChild(n)
		runtime.GC()
		if *p != 37 {
			t.Fatalf("slice capacity %d lost retained element", n)
		}
		if got := localMap(n); got != n*n {
			t.Fatalf("map size %d: %d", n, got)
		}
	}
	var maps []map[int]int
	var slices [][]byte
	for i := range 16 {
		m := make(map[int]int)
		m[i] = i
		maps = append(maps, m)
		b := make([]byte, 8)
		b[0] = byte(i)
		slices = append(slices, b)
	}
	runtime.GC()
	for i := range 16 {
		if maps[i][i] != i || slices[i][0] != byte(i) {
			t.Fatalf("iteration %d reused escaped container", i)
		}
	}
}

//go:noinline
func sliceBounds(n, c int) { _ = make([]byte, n, c) }

//go:noinline
func sliceBounds64(n, c int64) { _ = make([]byte, n, c) }

//go:noinline
func fixedSliceLength(n int) { _ = make([]byte, n, 16) }

func TestStackSliceKeepsBoundsPanics(t *testing.T) {
	for _, sizes := range [][2]int{{-1, 4}, {3, 2}, {0, -1}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("make(%d,%d) did not panic", sizes[0], sizes[1])
				}
			}()
			sliceBounds(sizes[0], sizes[1])
		}()
	}
	for _, n := range []int{-1, 17} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("fixed-capacity make(%d,16) did not panic", n)
				}
			}()
			fixedSliceLength(n)
		}()
	}
	if strconv.IntSize == 32 {
		for _, sizes := range [][2]int64{{1 << 32, 1 << 32}, {1, 1 << 32}} {
			func() {
				defer func() {
					if recover() == nil {
						t.Errorf("wide make(%d,%d) did not panic", sizes[0], sizes[1])
					}
				}()
				sliceBounds64(sizes[0], sizes[1])
			}()
		}
	}
}

type transientReceiver struct{ p *int }

func (r transientReceiver) Read() int                { runtime.GC(); return *r.p }
func (r transientReceiver) Copy() *transientReceiver { return &r }

//go:noinline
func interfaceReceiverCopy() *transientReceiver {
	x := 71
	var r interface{ Copy() *transientReceiver } = transientReceiver{&x}
	return r.Copy()
}

func TestInterfaceReceiverStorageLifetimes(t *testing.T) {
	r := interfaceReceiverCopy()
	runtime.GC()
	if *r.p != 71 {
		t.Fatal("value receiver escaped through transient call storage")
	}
	var reader interface{ Read() int } = *r
	if reader.Read() != 71 {
		t.Fatal("interface call lost receiver through GC")
	}
	result := make(chan int, 1)
	go func() { result <- reader.Read() }()
	if <-result != 71 {
		t.Fatal("goroutine lost receiver")
	}
	defer func() {
		if reader.Read() != 71 {
			t.Error("defer lost receiver")
		}
	}()
	var nilReader interface{ Read() int }
	func() {
		defer func() {
			if recover() == nil {
				t.Error("nil interface did not panic")
			}
		}()
		nilReader.Read()
	}()
}

//go:noinline
func zeroedLoopBuffers(n int) int {
	sum := 0
	for i := range n {
		b := make([]byte, 16)
		if b[0] != 0 {
			panic("reused dirty fixed slice")
		}
		b[0] = byte(i)
		sum += int(b[0])
		c := make([]byte, n)
		if c[0] != 0 {
			panic("reused dirty dynamic slice")
		}
		c[0] = byte(i)
		sum += int(c[0])
	}
	return sum
}

func TestLoopBuffersAreZeroedAtAllocation(t *testing.T) {
	for _, n := range []int{16, 128} {
		if got := zeroedLoopBuffers(n); got != n*(n-1) {
			t.Fatalf("loop buffer sum: %d", got)
		}
	}
}

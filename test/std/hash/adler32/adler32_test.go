package adler32_test

import (
	"encoding"
	"hash"
	"hash/adler32"
	"io"
	"strings"
	"testing"
)

var golden = []struct {
	in  string
	out uint32
}{
	{"", 1},
	{"a", 6422626},
	{"ab", 19267780},
	{"abc", 38600999},
	{"message digest", 695534982},
	{"abcdefghijklmnopqrstuvwxyz", 2424703776},
	{"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789", 2329613580},
	{"12345678901234567890123456789012345678901234567890123456789012345678901234567890", 2545291369},
}

func TestGolden(t *testing.T) {
	for _, g := range golden {
		got := adler32.Checksum([]byte(g.in))
		if got != g.out {
			t.Fatalf("adler32.Checksum([]byte(%q)) = %#x, want %#x", g.in, got, g.out)
		}
	}
}

func TestChecksum(t *testing.T) {
	data := []byte("hello world")
	expect := adler32.Checksum(data)
	got := adler32.Checksum(data)
	if got != expect {
		t.Fatalf("Checksum(%q) = %#x, want %#x", data, got, expect)
	}

	// Test empty input
	empty := adler32.Checksum([]byte{})
	if empty != 1 {
		t.Fatalf("Checksum([]byte{}) = %#x, want 1", empty)
	}
}

func TestHashInterface(t *testing.T) {
	h := adler32.New()

	// Test Size method
	if h.Size() != adler32.Size {
		t.Fatalf("Size() = %d, want %d", h.Size(), adler32.Size)
	}

	// Test Size constant
	if adler32.Size != 4 {
		t.Fatalf("Size constant = %d, want 4", adler32.Size)
	}

	// Test BlockSize method (Adler-32 uses a 4-byte block)
	if h.BlockSize() != 4 {
		t.Fatalf("BlockSize() = %d, want 4", h.BlockSize())
	}

	// Test Sum32 method on empty hash
	if h.Sum32() != 1 {
		t.Fatalf("Sum32() = %#x, want 1 (initial value)", h.Sum32())
	}

	// Test Write method
	msg := "hello"
	n, err := h.Write([]byte(msg))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != len(msg) {
		t.Fatalf("Write returned %d, want %d", n, len(msg))
	}

	// Test Sum32 after writing
	expected := adler32.Checksum([]byte(msg))
	if h.Sum32() != expected {
		t.Fatalf("Sum32() = %#x, want %#x", h.Sum32(), expected)
	}

	// Test Sum method (appends to slice)
	base := []byte{0xAA, 0xBB}
	sum := h.Sum(base)
	if len(sum) != len(base)+adler32.Size {
		t.Fatalf("Sum appended %d bytes, want %d", len(sum)-len(base), adler32.Size)
	}
	// Verify base bytes are preserved
	if sum[0] != 0xAA || sum[1] != 0xBB {
		t.Fatalf("Sum modified base bytes: got %#x %#x, want 0xAA 0xBB", sum[0], sum[1])
	}

	// Test Sum with nil slice
	nilSum := h.Sum(nil)
	if len(nilSum) != adler32.Size {
		t.Fatalf("Sum(nil) returned %d bytes, want %d", len(nilSum), adler32.Size)
	}

	// Test Reset method
	h.Reset()
	if h.Sum32() != 1 {
		t.Fatalf("Reset() failed: Sum32() = %#x, want 1", h.Sum32())
	}
}

func TestLargeData(t *testing.T) {
	// Test with data larger than one block
	data := strings.Repeat("abcdefghijklmnopqrstuvwxyz", 100) // 2600 bytes

	// Compute checksum using hash interface
	h := adler32.New()
	io.WriteString(h, data)
	hashSum := h.Sum32()

	// Compute checksum using direct function
	directSum := adler32.Checksum([]byte(data))

	if hashSum != directSum {
		t.Fatalf("Hash interface Sum32() = %#x, direct Checksum() = %#x", hashSum, directSum)
	}
}

func TestIncremental(t *testing.T) {
	data := "hello world"

	// One shot
	h1 := adler32.New()
	h1.Write([]byte(data))
	sum1 := h1.Sum32()

	// Incremental
	h2 := adler32.New()
	parts := []string{"hello", " ", "world"}
	for _, part := range parts {
		h2.Write([]byte(part))
	}
	sum2 := h2.Sum32()

	if sum1 != sum2 {
		t.Fatalf("Incremental hashing failed: one shot = %#x, incremental = %#x", sum1, sum2)
	}
}

func TestBinaryMarshaling(t *testing.T) {
	for _, g := range golden {
		h, restored := adler32.New(), adler32.New()
		half := len(g.in) / 2
		if _, err := io.WriteString(h, g.in[:half]); err != nil {
			t.Fatal(err)
		}
		state, err := h.(encoding.BinaryMarshaler).MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary(%q): %v", g.in, err)
		}
		prefixed, err := h.(encoding.BinaryAppender).AppendBinary(make([]byte, 4))
		if err != nil {
			t.Fatalf("AppendBinary(%q): %v", g.in, err)
		}
		if string(prefixed[4:]) != string(state) {
			t.Fatalf("AppendBinary(%q) state differs from MarshalBinary", g.in)
		}
		if err := restored.(encoding.BinaryUnmarshaler).UnmarshalBinary(state); err != nil {
			t.Fatalf("UnmarshalBinary(%q): %v", g.in, err)
		}
		if _, err := io.WriteString(h, g.in[half:]); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(restored, g.in[half:]); err != nil {
			t.Fatal(err)
		}
		if got, restoredSum := h.Sum32(), restored.Sum32(); got != g.out || restoredSum != got {
			t.Fatalf("restored checksum(%q) = %#x, full = %#x, want %#x", g.in, restoredSum, got, g.out)
		}
	}
}

func TestHash32Interface(t *testing.T) {
	h := adler32.New()

	// Test that the returned value implements hash.Hash32
	var _ hash.Hash32 = h

	// Write some data
	h.Write([]byte("test"))

	// Test hash.Hash32 specific method
	sum32 := h.Sum32()
	if sum32 == 0 {
		t.Fatalf("Sum32() returned 0 for non-empty input")
	}

	// Verify it matches hash.Hash.Sum cast to uint32 (little-endian)
	sumBytes := h.Sum(make([]byte, 0, 4))
	sumFromBytes := uint32(sumBytes[0])<<24 | uint32(sumBytes[1])<<16 | uint32(sumBytes[2])<<8 | uint32(sumBytes[3])
	if sum32 != sumFromBytes {
		t.Fatalf("Sum32() = %#x, but Sum() bytes converted to uint32 = %#x", sum32, sumFromBytes)
	}
}

func TestResetAndReuse(t *testing.T) {
	h := adler32.New()

	// First use
	h.Write([]byte("first"))
	first := h.Sum32()

	// Reset and reuse
	h.Reset()
	if h.Sum32() != 1 {
		t.Fatalf("After reset, Sum32() = %#x, want 1", h.Sum32())
	}

	h.Write([]byte("second"))
	second := h.Sum32()

	// Values should be different
	if first == second {
		t.Fatalf("Hash produced same value for different inputs: %#x", first)
	}

	// But should match direct checksum
	if second != adler32.Checksum([]byte("second")) {
		t.Fatalf("After reset, Sum32() = %#x, direct checksum = %#x", second, adler32.Checksum([]byte("second")))
	}
}

// Benchmark to test performance characteristics
func BenchmarkChecksum(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adler32.Checksum(data)
	}
}

func BenchmarkHashInterface(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h := adler32.New()
		h.Write(data)
		h.Sum32()
	}
}

func BenchmarkWrite(b *testing.B) {
	data := []byte("hello world")
	h := adler32.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Write(data)
	}
	h.Sum32()
}

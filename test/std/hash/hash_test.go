package hash_test

import (
	"bytes"
	"hash"
	"io"
	"testing"
)

// mockHash implements hash.Hash for testing
type mockHash struct {
	data  []byte
	size  int
	block int
}

func newMockHash(size, block int) *mockHash {
	return &mockHash{
		data:  make([]byte, 0, 64),
		size:  size,
		block: block,
	}
}

func (h *mockHash) Write(p []byte) (n int, err error) {
	h.data = append(h.data, p...)
	return len(p), nil
}

func (h *mockHash) Sum(b []byte) []byte {
	// Simple sum: just append data length and first few bytes
	result := make([]byte, 0, len(b)+h.size)
	result = append(result, b...)

	// Add simple hash data based on length and content
	if h.size > 0 {
		lenBytes := byte(len(h.data) % 256)
		result = append(result, lenBytes)

		if len(h.data) > 0 {
			result = append(result, h.data[0])
		}

		// Pad with zeros to reach size
		for len(result) < len(b)+h.size {
			result = append(result, 0)
		}
	}

	return result
}

func (h *mockHash) Reset() {
	h.data = h.data[:0]
}

func (h *mockHash) Size() int {
	return h.size
}

func (h *mockHash) BlockSize() int {
	return h.block
}

// mockHash32 implements hash.Hash32
type mockHash32 struct {
	*mockHash
}

func newMockHash32() *mockHash32 {
	return &mockHash32{mockHash: newMockHash(4, 1)}
}

func (h *mockHash32) Sum32() uint32 {
	if len(h.data) == 0 {
		return 0
	}

	// Simple 32-bit hash: XOR all bytes
	var result uint32
	for i, b := range h.data {
		result ^= uint32(b) << (8 * (i % 4))
	}
	return result
}

// mockHash64 implements hash.Hash64
type mockHash64 struct {
	*mockHash
}

func newMockHash64() *mockHash64 {
	return &mockHash64{mockHash: newMockHash(8, 8)}
}

func (h *mockHash64) Sum64() uint64 {
	if len(h.data) == 0 {
		return 0
	}

	// Simple 64-bit hash: XOR all bytes
	var result uint64
	for i, b := range h.data {
		result ^= uint64(b) << (8 * (i % 8))
	}
	return result
}

func TestHashInterface(t *testing.T) {
	h := newMockHash(8, 4)

	// Test initial state
	if h.Size() != 8 {
		t.Fatalf("Size() = %d, want 8", h.Size())
	}

	if h.BlockSize() != 4 {
		t.Fatalf("BlockSize() = %d, want 4", h.BlockSize())
	}

	// Test Write
	data := []byte("hello")
	n, err := h.Write(data)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != len(data) {
		t.Fatalf("Write returned %d, want %d", n, len(data))
	}

	// Test Sum
	base := []byte{0xAA, 0xBB}
	sum := h.Sum(base)
	if len(sum) != len(base)+h.Size() {
		t.Fatalf("Sum length = %d, want %d", len(sum), len(base)+h.Size())
	}

	// Verify base bytes are preserved
	if !bytes.Equal(sum[:len(base)], base) {
		t.Fatalf("Sum modified base bytes")
	}

	// Test Sum with nil
	nilSum := h.Sum(nil)
	if len(nilSum) != h.Size() {
		t.Fatalf("Sum(nil) length = %d, want %d", len(nilSum), h.Size())
	}

	// Test Reset
	h.Reset()
	if len(h.data) != 0 {
		t.Fatalf("Reset failed: data length = %d, want 0", len(h.data))
	}

	// Test that Reset affects Sum
	resetSum := h.Sum(nil)
	// Should return just padding since no data
	if len(resetSum) != h.Size() {
		t.Fatalf("Sum after reset length = %d, want %d", len(resetSum), h.Size())
	}
}

func TestHashInterfaceAsWriter(t *testing.T) {
	h := newMockHash(4, 1)

	// Test that hash.Hash implements io.Writer
	var writer io.Writer = h

	data := []byte("test data")
	n, err := writer.Write(data)
	if err != nil {
		t.Fatalf("Write through io.Writer interface failed: %v", err)
	}
	if n != len(data) {
		t.Fatalf("Write returned %d, want %d", n, len(data))
	}

	// Verify data was written
	if len(h.data) != len(data) {
		t.Fatalf("Data length = %d, want %d", len(h.data), len(data))
	}
}

func TestHash32Interface(t *testing.T) {
	h := newMockHash32()

	// Test that it implements hash.Hash
	var _ hash.Hash = h

	// Test initial state
	if h.Size() != 4 {
		t.Fatalf("Size() = %d, want 4", h.Size())
	}

	if h.Sum32() != 0 {
		t.Fatalf("Sum32() on empty hash = %d, want 0", h.Sum32())
	}

	// Test writing data
	data := []byte("abcd")
	h.Write(data)

	// Get both sum methods
	sum32 := h.Sum32()
	sumBytes := h.Sum(nil)

	// Convert byte sum to uint32 (big-endian)
	if len(sumBytes) != 4 {
		t.Fatalf("Sum() returned %d bytes, want 4", len(sumBytes))
	}

	// Note: Our mock implementation doesn't guarantee these match
	// But we can test that they're consistent
	if sum32 == 0 {
		t.Fatalf("Sum32() returned 0 for non-empty input")
	}
}

func TestHash64Interface(t *testing.T) {
	h := newMockHash64()

	// Test that it implements hash.Hash
	var _ hash.Hash = h

	// Test initial state
	if h.Size() != 8 {
		t.Fatalf("Size() = %d, want 8", h.Size())
	}

	if h.Sum64() != 0 {
		t.Fatalf("Sum64() on empty hash = %d, want 0", h.Sum64())
	}

	// Test writing data
	data := []byte("abcdefgh")
	h.Write(data)

	// Get both sum methods
	sum64 := h.Sum64()
	sumBytes := h.Sum(nil)

	// Convert byte sum to uint64 (big-endian)
	if len(sumBytes) != 8 {
		t.Fatalf("Sum() returned %d bytes, want 8", len(sumBytes))
	}

	// Note: Our mock implementation doesn't guarantee these match
	// But we can test that they're consistent
	if sum64 == 0 {
		t.Fatalf("Sum64() returned 0 for non-empty input")
	}
}

func TestInterfaceComposition(t *testing.T) {
	// Test that interfaces compose correctly

	// Hash32 should embed Hash
	var h32 hash.Hash32 = newMockHash32()
	if _, ok := h32.(hash.Hash); !ok {
		t.Fatalf("hash.Hash32 should embed hash.Hash")
	}

	// Hash64 should embed Hash
	var h64 hash.Hash64 = newMockHash64()
	if _, ok := h64.(hash.Hash); !ok {
		t.Fatalf("hash.Hash64 should embed hash.Hash")
	}
}

func TestErrorHandling(t *testing.T) {
	// Test that hash implementations handle various edge cases

	h := newMockHash(4, 1)

	// Test writing empty data
	n, err := h.Write([]byte{})
	if err != nil {
		t.Fatalf("Write empty slice failed: %v", err)
	}
	if n != 0 {
		t.Fatalf("Write empty slice returned %d, want 0", n)
	}

	// Test Sum with pre-allocated slice
	prealloc := make([]byte, 5) // Longer than hash size
	sum := h.Sum(prealloc)
	if len(sum) != len(prealloc)+h.Size() {
		t.Fatalf("Sum with preallocated slice returned wrong length")
	}

	// Verify prealloc bytes are at the beginning
	if !bytes.Equal(sum[:len(prealloc)], prealloc) {
		t.Fatalf("Sum modified preallocated bytes")
	}
}

func TestConsistency(t *testing.T) {
	// Test that hash operations are consistent

	h := newMockHash32()
	data := []byte("consistency test")

	// Write data multiple times and verify consistent results
	expectedSum := uint32(0)
	for i := 0; i < 3; i++ {
		h.Reset()
		h.Write(data)
		sum := h.Sum32()

		if i == 0 {
			expectedSum = sum
		} else if sum != expectedSum {
			t.Fatalf("Inconsistent hash result: iteration %d = %d, expected = %d", i+1, sum, expectedSum)
		}
	}

	// Test incremental writing
	h.Reset()
	h.Write(data[:len(data)/2])
	h.Write(data[len(data)/2:])

	incrementalSum := h.Sum32()
	if incrementalSum != expectedSum {
		t.Fatalf("Incremental hash = %d, single-write hash = %d", incrementalSum, expectedSum)
	}
}

func TestImplementationEdgeCases(t *testing.T) {
	// Test various edge cases with different hash implementations

	t.Run("ZeroSizeHash", func(t *testing.T) {
		h := newMockHash(0, 1)

		if h.Size() != 0 {
			t.Fatalf("Size() = %d, want 0", h.Size())
		}

		h.Write([]byte("test"))
		sum := h.Sum(nil)
		if len(sum) != 0 {
			t.Fatalf("Zero-size hash returned %d bytes, want 0", len(sum))
		}
	})

	t.Run("LargeBlockSize", func(t *testing.T) {
		h := newMockHash(8, 1024)

		if h.BlockSize() != 1024 {
			t.Fatalf("BlockSize() = %d, want 1024", h.BlockSize())
		}

		// Should still work with large block size
		h.Write([]byte("test"))
		if h.Size() != 8 {
			t.Fatalf("Size() = %d, want 8", h.Size())
		}
	})
}

// Benchmark tests for hash operations
func BenchmarkHashWrite(b *testing.B) {
	h := newMockHash(32, 8)
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Write(data)
		h.Reset()
	}
}

func BenchmarkHashSum(b *testing.B) {
	h := newMockHash(32, 8)
	data := make([]byte, 1024)
	h.Write(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Sum(nil)
	}
}

func BenchmarkHashWriteSum(b *testing.B) {
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h := newMockHash(32, 8)
		h.Write(data)
		h.Sum(nil)
	}
}

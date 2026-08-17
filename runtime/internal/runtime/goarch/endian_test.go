package goarch_test

import (
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/runtime/goarch"
)

func TestEndianConstants(t *testing.T) {
	v := uint16(0x0102)
	actualLittleEndian := *(*byte)(unsafe.Pointer(&v)) == 0x02
	if goarch.LittleEndian != actualLittleEndian {
		t.Fatalf("LittleEndian = %v, actual memory layout is little-endian: %v", goarch.LittleEndian, actualLittleEndian)
	}
	if goarch.BigEndian == goarch.LittleEndian {
		t.Fatalf("BigEndian = %v, LittleEndian = %v; want opposite values", goarch.BigEndian, goarch.LittleEndian)
	}
}

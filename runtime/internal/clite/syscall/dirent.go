package syscall

import (
	"github.com/xgo-dev/llgo/runtime/internal/clite/byteorder"
	"github.com/xgo-dev/llgo/runtime/internal/clite/goarch"
)

// readInt returns the size-bytes unsigned integer in native byte order at offset off.
func readInt(b []byte, off, size uintptr) (u uint64, ok bool) {
	if len(b) < int(off+size) {
		return 0, false
	}
	if goarch.BigEndian {
		return readIntBE(b[off:], size), true
	}
	return readIntLE(b[off:], size), true
}

func readIntBE(b []byte, size uintptr) uint64 {
	switch size {
	case 1:
		return uint64(b[0])
	case 2:
		return uint64(byteorder.BEUint16(b))
	case 4:
		return uint64(byteorder.BEUint32(b))
	case 8:
		return uint64(byteorder.BEUint64(b))
	default:
		panic("syscall: readInt with unsupported size")
	}
}

func readIntLE(b []byte, size uintptr) uint64 {
	switch size {
	case 1:
		return uint64(b[0])
	case 2:
		return uint64(byteorder.LEUint16(b))
	case 4:
		return uint64(byteorder.LEUint32(b))
	case 8:
		return uint64(byteorder.LEUint64(b))
	default:
		panic("syscall: readInt with unsupported size")
	}
}

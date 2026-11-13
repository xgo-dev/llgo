//go:build llgo

package bits

import _ "unsafe"

// LLVM funnel shifts with equal operands are rotates. The count is reduced
// modulo the operand width, so converting k also handles negative counts.

//go:linkname llgoFshl llvm.fshl
func llgoFshl(a, b, k uint) uint

//go:linkname llgoFshl8 llvm.fshl
func llgoFshl8(a, b, k uint8) uint8

//go:linkname llgoFshl16 llvm.fshl
func llgoFshl16(a, b, k uint16) uint16

//go:linkname llgoFshl32 llvm.fshl
func llgoFshl32(a, b, k uint32) uint32

//go:linkname llgoFshl64 llvm.fshl
func llgoFshl64(a, b, k uint64) uint64

func RotateLeft(x uint, k int) uint       { return llgoFshl(x, x, uint(k)) }
func RotateLeft8(x uint8, k int) uint8    { return llgoFshl8(x, x, uint8(k)) }
func RotateLeft16(x uint16, k int) uint16 { return llgoFshl16(x, x, uint16(k)) }
func RotateLeft32(x uint32, k int) uint32 { return llgoFshl32(x, x, uint32(k)) }
func RotateLeft64(x uint64, k int) uint64 { return llgoFshl64(x, x, uint64(k)) }

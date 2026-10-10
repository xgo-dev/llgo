//go:build llgo && !wasm

package cgo

/*
// Keep real calls across the C boundary even when LTO is enabled.
static __attribute__((noinline)) int narrow_s8(signed char x) { return x; }
static __attribute__((noinline)) unsigned int narrow_u8(unsigned char x) { return x; }
static __attribute__((noinline)) int narrow_s16(short x) { return x; }
static __attribute__((noinline)) unsigned int narrow_u16(unsigned short x) { return x; }

// Eight leading integers force the narrow values onto the native stack.
static __attribute__((noinline)) int narrow_stack(
    int p0, int p1, int p2, int p3, int p4, int p5, int p6, int p7,
    signed char a, unsigned char b, short c, unsigned short d, _Bool flag) {
    if (p0 + p1 + p2 + p3 + p4 + p5 + p6 + p7 != 36) return 0;
    int result = a + 10 * b + 100 * c + 1000 * d;
    return flag ? result : -result;
}

static void *narrow_s8_address(void) { return (void *)narrow_s8; }
static void *narrow_u8_address(void) { return (void *)narrow_u8; }
static void *narrow_s16_address(void) { return (void *)narrow_s16; }
static void *narrow_u16_address(void) { return (void *)narrow_u16; }
*/
import "C"

import "unsafe"

//llgo:type C
type narrowS8 func(int8) int32

//llgo:type C
type narrowU8 func(uint8) uint32

//llgo:type C
type narrowS16 func(int16) int32

//llgo:type C
type narrowU16 func(uint16) uint32

func directNarrow(a int8, b uint8, c int16, d uint16) (int32, uint32, int32, uint32) {
	return int32(C.narrow_s8(C.schar(a))), uint32(C.narrow_u8(C.uchar(b))),
		int32(C.narrow_s16(C.short(c))), uint32(C.narrow_u16(C.ushort(d)))
}

func directNarrowConstants() (int32, uint32, int32, uint32) {
	return int32(C.narrow_s8(-8)), uint32(C.narrow_u8(250)),
		int32(C.narrow_s16(-300)), uint32(C.narrow_u16(60000))
}

func indirectNarrowConstants() (int32, uint32, int32, uint32) {
	sa, ua, sb, ub := C.narrow_s8_address(), C.narrow_u8_address(), C.narrow_s16_address(), C.narrow_u16_address()
	s8 := *(*narrowS8)(unsafe.Pointer(&sa))
	u8 := *(*narrowU8)(unsafe.Pointer(&ua))
	s16 := *(*narrowS16)(unsafe.Pointer(&sb))
	u16 := *(*narrowU16)(unsafe.Pointer(&ub))
	return s8(-8), u8(250), s16(-300), u16(60000)
}

func indirectNarrow(a int8, b uint8, c int16, d uint16) (int32, uint32, int32, uint32) {
	sa, ua, sb, ub := C.narrow_s8_address(), C.narrow_u8_address(), C.narrow_s16_address(), C.narrow_u16_address()
	s8 := *(*narrowS8)(unsafe.Pointer(&sa))
	u8 := *(*narrowU8)(unsafe.Pointer(&ua))
	s16 := *(*narrowS16)(unsafe.Pointer(&sb))
	u16 := *(*narrowU16)(unsafe.Pointer(&ub))
	return s8(a), u8(b), s16(c), u16(d)
}

func stackedNarrow(a int8, b uint8, c int16, d uint16, flag bool) int32 {
	return int32(C.narrow_stack(1, 2, 3, 4, 5, 6, 7, 8,
		C.schar(a), C.uchar(b), C.short(c), C.ushort(d), C._Bool(flag)))
}

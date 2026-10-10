//go:build llgo

package llgoext

import _ "unsafe" // for go:linkname

type narrowValues struct {
	s8  int64
	u8  uint64
	s16 int64
	u16 uint64
}

//llgo:type C
type narrowFunction func(int8, uint8, int16, uint16) narrowValues

//go:linkname linkedNarrow C.narrow_promote
func linkedNarrow(int8, uint8, int16, uint16) narrowValues

//go:linkname linkedNarrowAddress C.narrow_address
func linkedNarrowAddress() narrowFunction

type narrowSigned int8
type narrowUnsigned = uint16

//llgo:type C
type narrowStackFunction func(int32, int32, int32, int32, int32, int32, int32, int32,
	narrowSigned, uint8, int16, narrowUnsigned, bool) narrowValues

//go:linkname linkedNarrowStack C.narrow_stack
func linkedNarrowStack(int32, int32, int32, int32, int32, int32, int32, int32,
	narrowSigned, uint8, int16, narrowUnsigned, bool) narrowValues

//go:linkname linkedNarrowStackAddress C.narrow_stack_address
func linkedNarrowStackAddress() narrowStackFunction

//llgo:type C
type narrowArgumentsCallback func(int8, uint8, int16, uint16, bool) int32

//go:linkname linkedNarrowInvoke C.narrow_invoke
func linkedNarrowInvoke(narrowArgumentsCallback, bool) int32

//llgo:type C
type narrowVariadicFunction func(a int8, b int16, flag bool, marker int32, __llgo_va_list ...any) int32

//go:linkname linkedNarrowVariadic C.narrow_variadic
func linkedNarrowVariadic(a int8, b int16, flag bool, marker int32, __llgo_va_list ...any) int32

//go:linkname linkedNarrowVariadicAddress C.narrow_variadic_address
func linkedNarrowVariadicAddress() narrowVariadicFunction

//llgo:type C
type narrowS8Callback func(int32) int8

//llgo:type C
type narrowU8Callback func(int32) uint8

//llgo:type C
type narrowS16Callback func(int32) int16

//llgo:type C
type narrowU16Callback func(int32) uint16

//go:linkname linkedNarrowReturnS8 C.narrow_return_s8
func linkedNarrowReturnS8(int32) int8

//go:linkname linkedNarrowReturnU8 C.narrow_return_u8
func linkedNarrowReturnU8(int32) uint8

//go:linkname linkedNarrowReturnS16 C.narrow_return_s16
func linkedNarrowReturnS16(int32) int16

//go:linkname linkedNarrowReturnU16 C.narrow_return_u16
func linkedNarrowReturnU16(int32) uint16

//go:linkname linkedNarrowS8Roundtrip C.narrow_s8_roundtrip
func linkedNarrowS8Roundtrip(narrowS8Callback) narrowS8Callback

//go:linkname linkedNarrowU8Roundtrip C.narrow_u8_roundtrip
func linkedNarrowU8Roundtrip(narrowU8Callback) narrowU8Callback

//go:linkname linkedNarrowS16Roundtrip C.narrow_s16_roundtrip
func linkedNarrowS16Roundtrip(narrowS16Callback) narrowS16Callback

//go:linkname linkedNarrowU16Roundtrip C.narrow_u16_roundtrip
func linkedNarrowU16Roundtrip(narrowU16Callback) narrowU16Callback

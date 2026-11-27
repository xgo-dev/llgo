package main

import "unsafe"

type Alias = uint32

type NamedInt int64

type Recursive struct {
	Value NamedInt
	Next  *Recursive
}

type Pair[T any] struct {
	First  T
	Second T
}

type Number interface {
	Number() NamedInt
}

type Sample struct {
	Bool     bool
	I8       int8
	I16      int16
	I32      int32
	I64      int64
	U8       uint8
	U16      uint16
	U32      Alias
	U64      uint64
	F32      float32
	F64      float64
	C64      complex64
	C128     complex128
	Text     string
	Values   []NamedInt
	Fixed    [3]uint16
	Lookup   map[string]NamedInt
	Queue    chan Pair[NamedInt]
	Callback func(NamedInt) (NamedInt, error)
	Any      any
	Iface    Number
	Pointer  *Recursive
	Unsafe   unsafe.Pointer
	Pair     Pair[NamedInt]
}

var GlobalInt NamedInt = 41
var GlobalPtr = &Recursive{Value: 1}
var GlobalSample Sample
var GlobalMap map[string]NamedInt
var GlobalChan chan Pair[NamedInt]
var GlobalFunc func(NamedInt) (NamedInt, error)
var GlobalInterface Number
var GlobalUnsafe unsafe.Pointer

func (s *Sample) Number() NamedInt {
	return NamedInt(s.I64)
}

func identity[T any](value T) T {
	return value
}

//go:noinline
func inspect(input NamedInt, pair Pair[NamedInt], values ...int) (result NamedInt) {
	local := helper(input) + pair.First + pair.Second
	for index, value := range values {
		local += NamedInt(index + value)
	}
	if local > 0 {
		shadow := local + 1
		result = shadow // DWARF_SCOPE_MARKER
	}
	closure := func(delta NamedInt) NamedInt {
		return local + delta
	}
	boxed := any(result)
	switch typed := boxed.(type) {
	case NamedInt:
		result += typed
	}
	result = closure(result)
	return identity(result)
}

func main() {
	GlobalMap = map[string]NamedInt{"answer": GlobalInt}
	GlobalChan = make(chan Pair[NamedInt], 1)
	GlobalFunc = func(value NamedInt) (NamedInt, error) { return value + 1, nil }
	GlobalSample.I64 = int64(GlobalInt)
	GlobalInterface = &GlobalSample
	GlobalUnsafe = unsafe.Pointer(GlobalPtr)
	result := inspect(2, Pair[NamedInt]{First: 3, Second: 4}, 1, 2)
	if result <= 0 || GlobalInterface.Number() != GlobalInt {
		panic("bad DWARF fixture result")
	}
	println("dwarf-ok")
}

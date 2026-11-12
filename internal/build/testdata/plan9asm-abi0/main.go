package main

type Uint128 struct {
	Lo, Hi uint64
}

func wrap(t *byte, a, b Uint128, result uintptr) int32
func roundTrip(value uint64) uint64

//go:noinline
func impl(t *byte, a, b Uint128, result uintptr) int32 {
	if t != nil {
		panic("lost pointer argument")
	}
	return int32(a.Lo + 10*a.Hi + 100*b.Lo + 1000*b.Hi + 10000*uint64(result))
}

func main() {
	if got := wrap(nil, Uint128{2, 3}, Uint128{5, 7}, 11); got != 117532 {
		println("wrong ABI0 result", got)
		panic("lost ABI0 aggregate arguments or result")
	}
	const bits = uint64(0xfedcba9876543210)
	if roundTrip(bits) != bits {
		panic("VMOVQ lost scalar bits")
	}
	println("ok")
}

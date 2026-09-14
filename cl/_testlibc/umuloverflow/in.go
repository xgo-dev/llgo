// LITTEST
// Scope: common
package main

type unsigned interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uint | ~uintptr
}

//llgo:link multiply llgo.umulOverflow
func multiply[T unsigned](a, b T) (T, bool) {
	return a * b, a != 0 && b > ^T(0)/a
}

func check[T unsigned](a, b T) {
	product, overflow := multiply(a, b)
	// Use division as an independent oracle for the overflow flag.
	want := a != 0 && b > ^T(0)/a
	if product != a*b || overflow != want {
		panic("unsigned multiply overflow mismatch")
	}
}

func checkWidth[T unsigned]() {
	max := ^T(0)
	edges := []T{0, 1, 2, 3, max, max - 1, max / 2, max/2 + 1}
	// Include both sides of every power-of-two boundary and its exact
	// largest non-overflowing cofactor.
	for p := T(1); p != 0; p <<= 1 {
		edges = append(edges, p, p-1, p+1)
		limit := max / p
		check(p, limit)
		if limit < max {
			check(p, limit+1)
		}
	}
	for _, a := range edges {
		for _, b := range edges {
			check(a, b)
		}
	}
	seed := uint64(0x123456789abcdef)
	next := func() T {
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		return T(seed)
	}
	for i := 0; i < 1024; i++ {
		check(next(), next())
	}
}

type word uint64

var multiplyValue = multiply[uintptr]

// CHECK-LABEL: define void @main.main()
// CHECK: call { i[[WIDTH:32|64]], i1 } @llvm.umul.with.overflow.i[[WIDTH]](
// CHECK: call {{.*}} %{{[^ (]+}}(
func main() {
	checkWidth[uint8]()
	checkWidth[uint16]()
	checkWidth[uint32]()
	checkWidth[uint64]()
	checkWidth[uint]()
	checkWidth[uintptr]()
	checkWidth[word]()
	max := ^uintptr(0)
	// Calls must evaluate their operands once, from left to right.
	order := 0
	arg := func(digit int) uintptr { order = order*10 + digit; return uintptr(order) }
	product, overflow := multiply(arg(1), arg(2))
	if order != 12 || product != 12 || overflow {
		panic("multiply argument evaluation mismatch")
	}
	// Taking an intrinsic's address must produce a callable wrapper.
	product, overflow = multiplyValue(max, 2)
	if product != max-1 || !overflow {
		panic("multiply function value mismatch")
	}
	func() {
		order = 0
		defer multiply(arg(1), arg(2))
		if order != 12 {
			panic("deferred multiply argument evaluation mismatch")
		}
	}()
	checkGoCall()
	println("unsigned multiply overflow ok")
}

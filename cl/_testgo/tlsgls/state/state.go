package state

import "runtime"

var sequence int

func next(base int) int {
	sequence++
	return base + sequence
}

func nextPointer(base int) *int {
	value := next(base)
	return &value
}

//llgo:tls
var TLS = next(10)

//llgo:gls
var GLS = next(20)

//llgo:gls
var Pointer = nextPointer(30)

var poisonAttempts int

func poisonOtherThreads() int {
	poisonAttempts++
	if poisonAttempts > 1 {
		panic("poison TLS initializer")
	}
	return 40 + poisonAttempts
}

//llgo:tls
var Poison = poisonOtherThreads()

type recursiveInitializer interface {
	value() int
}

type recursiveValue struct{}

func (recursiveValue) value() int {
	return Recursive + 1
}

var recursiveSource recursiveInitializer = recursiveValue{}

//llgo:gls
var Recursive = recursiveSource.value()

var zLateAttempts int

func nextLate() int {
	zLateAttempts++
	return 100 + zLateAttempts
}

// zLate sorts after the synthetic package init function in SSA member order.
//
//llgo:tls
var zLate = nextLate()

func Values() (tls, gls, pointer, count int) {
	tls = TLS
	gls = GLS
	if Pointer == nil {
		panic("nil GLS pointer")
	}
	runtime.GC()
	pointer = *Pointer
	count = sequence
	return
}

func readPoison() int {
	return Poison
}

func PoisonValues() (value, attempts int) {
	func() {
		defer func() {
			_ = recover()
		}()
		value = readPoison()
	}()
	value = readPoison()
	attempts = poisonAttempts
	return
}

func RecursiveValue() int {
	return Recursive
}

func LateValues() (value, attempts int) {
	return zLate, zLateAttempts
}

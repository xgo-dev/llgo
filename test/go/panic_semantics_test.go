package gotest

import (
	"testing"
	"unsafe"
)

type panicSemanticsBlock [1 << 19]byte
type panicSemanticsChanBlock [1<<16 - 1]byte

var (
	panicSemanticsSliceSink []panicSemanticsBlock
	panicSemanticsChanSink  chan panicSemanticsChanBlock
)

//go:noinline
func panicSemanticsFive() int {
	return 5
}

//go:noinline
func panicSemanticsBig() int64 {
	return 10 | 1<<46
}

func TestMakeSliceAndChanPanics(t *testing.T) {
	minusOne := -1
	five := panicSemanticsFive()
	big := panicSemanticsBig()

	expectPanicContaining(t, "len out of range", func() {
		panicSemanticsSliceSink = make([]panicSemanticsBlock, minusOne)
	})
	expectPanicContaining(t, "len out of range", func() {
		panicSemanticsSliceSink = make([]panicSemanticsBlock, big)
	})
	expectPanicContaining(t, "cap out of range", func() {
		panicSemanticsSliceSink = make([]panicSemanticsBlock, 10, minusOne)
	})
	expectPanicContaining(t, "cap out of range", func() {
		panicSemanticsSliceSink = make([]panicSemanticsBlock, 10, five)
	})
	expectPanicContaining(t, "cap out of range", func() {
		panicSemanticsSliceSink = make([]panicSemanticsBlock, 10, big)
	})

	expectPanicContaining(t, "makechan: size out of range", func() {
		panicSemanticsChanSink = make(chan panicSemanticsChanBlock, minusOne)
	})
	expectPanicContaining(t, "makechan: size out of range", func() {
		panicSemanticsChanSink = make(chan panicSemanticsChanBlock, big)
	})
	expectPanicContaining(t, "makechan: size out of range", func() {
		const ptrSize = unsafe.Sizeof(uintptr(0))
		panicSemanticsChanSink = make(chan panicSemanticsChanBlock, 1<<(30*(ptrSize/4)))
	})
}

type nilAddressCalcStruct struct {
	a, b int
}

//go:noinline
func nilAddressCalc(x *nilAddressCalcStruct, p *bool, n int) {
	*p = n != 0
	useNilAddressCalcStack(64)
	sinkNilAddressCalc(&x.b)
}

//go:noinline
func sinkNilAddressCalc(*int) {
}

func useNilAddressCalcStack(n int) {
	if n == 0 {
		return
	}
	useNilAddressCalcStack(n - 1)
}

func TestNilAddressCalculationPanicsAfterPriorSideEffects(t *testing.T) {
	var b bool
	expectPanicContaining(t, "nil pointer", func() {
		nilAddressCalc(nil, &b, 3)
	})
	if !b {
		t.Fatal("side effect before nil field address calculation was not preserved")
	}
}

func TestGenericUnsafeSizeofIndexPanics(t *testing.T) {
	expectPanicContaining(t, "index out of range", func() {
		genericSizeofIndex(byte(0))
	})
	expectPanicContaining(t, "index out of range", func() {
		genericSizeofIndexPlusZero(byte(0))
	})
}

func genericSizeofIndex[T byte](t T) {
	const str = "a"
	_ = str[unsafe.Sizeof(t)]
}

func genericSizeofIndexPlusZero[T byte](t T) {
	const str = "a"
	_ = str[unsafe.Sizeof(t)+0]
}

func TestIndexBoundsPanicTextIncludesLength(t *testing.T) {
	expectPanicEqual(t, "runtime error: index out of range [1] with length 1", func() {
		s := []uint64{0}
		for n := range len(s) {
			_ = n
			_ = s[1]
		}
	})
}

func expectPanicEqual(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		err := recover()
		if err == nil {
			t.Fatalf("expected panic %q", want)
		}
		if got := panicString(err); got != want {
			t.Fatalf("panic = %q, want %q", got, want)
		}
	}()
	f()
}

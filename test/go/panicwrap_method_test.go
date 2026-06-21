package gotest

import "testing"

type panicWrapT int

func (panicWrapT) PanicWrapValueMethod() {}

type panicWrapI interface {
	PanicWrapValueMethod()
}

var (
	panicWrapPtr *panicWrapT
	panicWrapItf panicWrapI = panicWrapPtr
)

func TestValueMethodWrapperNilPointerPanic(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected value method wrapper panic")
		}
		err, ok := r.(error)
		if !ok {
			t.Fatalf("panic type = %T, want error", r)
		}
		const want = "value method github.com/goplus/llgo/test/go.panicWrapT.PanicWrapValueMethod called using nil *panicWrapT pointer"
		if got := err.Error(); got != want {
			t.Fatalf("panic text = %q, want %q", got, want)
		}
	}()

	panicWrapItf.PanicWrapValueMethod()
}

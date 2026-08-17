package gotest

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/test/go/recoverpkg"
)

func TestRecoverCrossPackageDeferredFunction(t *testing.T) {
	var recovered any
	func() {
		defer recoverpkg.Store(&recovered)
		panic("cross-package")
	}()
	if recovered != "cross-package" {
		t.Fatalf("cross-package recover = %v, want cross-package", recovered)
	}
}

type fixedbug4066Panic struct{}

func fixedbug4066NamedReturn() (val int) {
	val = 0
	defer func() {
		if x := recover(); x != nil {
			_ = x.(fixedbug4066Panic)
		}
	}()
	for {
		val = 2
		fixedbug4066Throw()
	}
}

func fixedbug4066Throw() {
	panic(fixedbug4066Panic{})
}

func TestRecoverFixedbug4066NamedReturn(t *testing.T) {
	if got := fixedbug4066NamedReturn(); got != 2 {
		t.Fatalf("named return after recover = %d, want 2", got)
	}
}

func TestRecoverFixedbugDirectDeferredFuncValue(t *testing.T) {
	recovered := false
	func() {
		f := func() {
			if recover() != nil {
				recovered = true
			}
		}
		defer f()
		panic("direct deferred func value")
	}()
	if !recovered {
		t.Fatal("direct deferred func value did not recover")
	}
}

func TestRecoverFixedbugNestedDeferInDeferredFuncDoesNotRecover(t *testing.T) {
	nested := any("unset")
	func() {
		defer func() {
			if r := recover(); r != "outer" {
				t.Fatalf("outer recover = %v, want outer", r)
			}
		}()
		defer func() {
			defer func() {
				nested = recover()
			}()
		}()
		panic("outer")
	}()
	if nested != nil {
		t.Fatalf("nested recover = %v, want nil", nested)
	}
}

func fixedbugCaptureRecover(dst *any) {
	*dst = recover()
}

func TestRecoverNestedDeferredBuiltinRecoversOuterPanic(t *testing.T) {
	outer := any("unset")
	func() {
		defer fixedbugCaptureRecover(&outer)
		defer func() {
			defer recover()
		}()
		panic("outer")
	}()
	if outer != nil {
		t.Fatalf("outer recover = %v, want nil after nested deferred recover", outer)
	}
}

func TestRecoverNestedDeferredBuiltinAfterInnerRecover(t *testing.T) {
	outer := any("unset")
	inner := any("unset")
	func() {
		defer fixedbugCaptureRecover(&outer)
		defer func() {
			defer recover()
			defer fixedbugCaptureRecover(&inner)
			panic("inner")
		}()
		panic("outer")
	}()
	if inner != "inner" {
		t.Fatalf("inner recover = %v, want inner", inner)
	}
	if outer != nil {
		t.Fatalf("outer recover = %v, want nil after nested deferred recover", outer)
	}
}

func TestRecoverNestedDeferredBuiltinBeforeInnerRecover(t *testing.T) {
	outer := any("unset")
	inner := any("unset")
	func() {
		defer fixedbugCaptureRecover(&outer)
		defer func() {
			defer fixedbugCaptureRecover(&inner)
			defer recover()
			panic("inner")
		}()
		panic("outer")
	}()
	if inner != "inner" {
		t.Fatalf("inner recover = %v, want inner", inner)
	}
	if outer != "outer" {
		t.Fatalf("outer recover = %v, want outer", outer)
	}
}

type fixedbugNestedPanicInterface interface {
	panicInner()
}

type fixedbugNestedPanicValue struct{}

func (fixedbugNestedPanicValue) panicInner() {
	panic("inner")
}

var fixedbugNestedPanicRecover any

func fixedbugRecoverNestedPanic() {
	defer func() {
		fixedbugNestedPanicRecover = recover()
	}()
	var v fixedbugNestedPanicInterface = fixedbugNestedPanicValue{}
	v.panicInner()
}

type fixedbugOuterRecoverInterface interface {
	recoverOuter()
}

type fixedbugOuterRecoverValue struct{}

var fixedbugOuterPanicRecover any

func (fixedbugOuterRecoverValue) recoverOuter() {
	fixedbugRecoverNestedPanic()
	fixedbugOuterPanicRecover = recover()
}

func TestRecoverOuterPanicAfterNestedPanicIsRecovered(t *testing.T) {
	fixedbugNestedPanicRecover = nil
	fixedbugOuterPanicRecover = nil
	func() {
		var v fixedbugOuterRecoverInterface = fixedbugOuterRecoverValue{}
		defer v.recoverOuter()
		panic("outer")
	}()
	if fixedbugNestedPanicRecover != "inner" {
		t.Fatalf("nested recover = %v, want inner", fixedbugNestedPanicRecover)
	}
	if fixedbugOuterPanicRecover != "outer" {
		t.Fatalf("outer recover = %v, want outer", fixedbugOuterPanicRecover)
	}
}

type fixedbugReflectRecoverValue struct{}

var fixedbugReflectMethodRecover any

func (*fixedbugReflectRecoverValue) RecoverValue() {
	fixedbugReflectMethodRecover = recover()
}

func TestRecoverDeferredReflectMethodWrappers(t *testing.T) {
	tests := []struct {
		name string
		run  func(*fixedbugReflectRecoverValue)
	}{
		{
			name: "method value",
			run: func(v *fixedbugReflectRecoverValue) {
				f := reflect.ValueOf(v).Method(0).Interface().(func())
				defer f()
				panic("method value")
			},
		},
		{
			name: "method expression",
			run: func(v *fixedbugReflectRecoverValue) {
				f := reflect.TypeOf(v).Method(0).Func.Interface().(func(*fixedbugReflectRecoverValue))
				defer f(v)
				panic("method expression")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixedbugReflectMethodRecover = nil
			test.run(&fixedbugReflectRecoverValue{})
			if fixedbugReflectMethodRecover != test.name {
				t.Fatalf("reflect recover = %v, want %q", fixedbugReflectMethodRecover, test.name)
			}
		})
	}
}

func TestRecoverDeferredReflectMakeFunc(t *testing.T) {
	got := any("unset")
	f := reflect.MakeFunc(reflect.TypeOf((func())(nil)), func([]reflect.Value) []reflect.Value {
		got = recover()
		return nil
	}).Interface().(func())
	func() {
		defer f()
		panic("make func")
	}()
	if got != "make func" {
		t.Fatalf("reflect MakeFunc recover = %v, want make func", got)
	}
}

func fixedbugReflectIndirectRecover() {
	fixedbugReflectMethodRecover = recover()
}

func TestRecoverReflectCallRemainsIndirect(t *testing.T) {
	fixedbugReflectMethodRecover = any("unset")
	func() {
		defer fixedbugCaptureRecover(&fixedbugReflectMethodRecover)
		defer func() {
			reflect.ValueOf(fixedbugReflectIndirectRecover).Call(nil)
		}()
		panic("outer")
	}()
	if fixedbugReflectMethodRecover != "outer" {
		t.Fatalf("outer recover = %v, want outer", fixedbugReflectMethodRecover)
	}
}

func TestRecoverGoexitSupersedesSuspendedPanic(t *testing.T) {
	result := make(chan any, 2)
	go func() {
		defer func() {
			result <- recover()
		}()
		defer func() {
			runtime.Goexit()
			result <- "runtime.Goexit returned"
		}()
		panic("outer")
	}()

	select {
	case got := <-result:
		if got != nil {
			t.Fatalf("recover during Goexit = %v, want nil", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Goexit did not run the remaining defer")
	}
}

var fixedbugReturnedRecover any

func fixedbugReturnedRecoverFunc() func() {
	return func() {
		fixedbugReturnedRecover = recover()
	}
}

func TestRecoverFixedbugReturnedDeferredFuncValue(t *testing.T) {
	fixedbugReturnedRecover = nil
	func() {
		defer fixedbugReturnedRecoverFunc()()
		panic("returned deferred func value")
	}()
	if fixedbugReturnedRecover != "returned deferred func value" {
		t.Fatalf("returned deferred recover = %v, want panic value", fixedbugReturnedRecover)
	}
}

type fixedbugRecoverMethod int

var fixedbugMethodRecover any

func (fixedbugRecoverMethod) recoverValue() {
	fixedbugMethodRecover = recover()
}

func TestRecoverDirectDeferredMethod(t *testing.T) {
	fixedbugMethodRecover = nil
	func() {
		defer fixedbugRecoverMethod(0).recoverValue()
		panic("direct deferred method")
	}()
	if fixedbugMethodRecover != "direct deferred method" {
		t.Fatalf("direct deferred method recover = %v, want panic value", fixedbugMethodRecover)
	}
}

type fixedbugRecoverInterface interface {
	recoverValue()
}

func TestRecoverDirectDeferredInterfaceMethod(t *testing.T) {
	fixedbugMethodRecover = nil
	func() {
		var v fixedbugRecoverInterface = fixedbugRecoverMethod(0)
		defer v.recoverValue()
		panic("direct deferred interface method")
	}()
	if fixedbugMethodRecover != "direct deferred interface method" {
		t.Fatalf("direct deferred interface method recover = %v, want panic value", fixedbugMethodRecover)
	}
}

func TestRecoverDirectDeferredMethodValue(t *testing.T) {
	fixedbugMethodRecover = nil
	func() {
		var v fixedbugRecoverInterface = fixedbugRecoverMethod(0)
		f := v.recoverValue
		defer f()
		panic("direct deferred method value")
	}()
	if fixedbugMethodRecover != "direct deferred method value" {
		t.Fatalf("direct deferred method value recover = %v, want panic value", fixedbugMethodRecover)
	}
}

var fixedbugRecursiveRecover any

func fixedbugRecursiveDeferredRecover(depth int) {
	if depth > 0 {
		fixedbugRecursiveDeferredRecover(depth - 1)
		return
	}
	fixedbugRecursiveRecover = recover()
}

func TestRecoverRecursiveDeferredActivationDoesNotRecover(t *testing.T) {
	fixedbugRecursiveRecover = "unset"
	var outer any
	func() {
		defer func() {
			outer = recover()
		}()
		defer fixedbugRecursiveDeferredRecover(1)
		panic("recursive deferred activation")
	}()
	if fixedbugRecursiveRecover != nil {
		t.Fatalf("recursive recover = %v, want nil", fixedbugRecursiveRecover)
	}
	if outer != "recursive deferred activation" {
		t.Fatalf("outer recover = %v, want panic value", outer)
	}
}

func fixedbug8047bNilDeferredCall() {
	var fn func()
	defer fn()
	panic(1)
}

func TestRecoverFixedbug8047bNilDeferredCallReplacesPanic(t *testing.T) {
	var got any
	func() {
		defer func() {
			got = recover()
		}()
		fixedbug8047bNilDeferredCall()
	}()
	if got == nil {
		t.Fatal("nil deferred call did not panic")
	}
	if got == 1 {
		t.Fatal("recover observed the original panic after the nil deferred call replaced it")
	}
}

var fixedbug73916Recovered bool

func fixedbug73916CallRecover() {
	if recover() != nil {
		fixedbug73916Recovered = true
	}
}

func fixedbug73916Deferred(int) {
	fixedbug73916CallRecover()
}

func fixedbug73916MustPanic(t *testing.T, fn func()) any {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("deferred indirect recover swallowed panic")
		}
	}()
	fn()
	return nil
}

func TestRecoverFixedbug73916IndirectRecoverDoesNotRecover(t *testing.T) {
	skipBeforeGo126(t)
	fixedbug73916Recovered = false
	fixedbug73916MustPanic(t, func() {
		defer fixedbug73916Deferred(1)
		panic("fixedbug73916")
	})
	if fixedbug73916Recovered {
		t.Fatal("indirect recover returned non-nil")
	}
}

var fixedbug73916bRecovered bool

func fixedbug73916bCallRecover() {
	func() {
		if recover() != nil {
			fixedbug73916bRecovered = true
		}
	}()
}

func fixedbug73916bDeferred() int {
	fixedbug73916bCallRecover()
	return 0
}

func TestRecoverFixedbug73916NestedRecoverDoesNotRecover(t *testing.T) {
	skipBeforeGo126(t)
	fixedbug73916bRecovered = false
	fixedbug73916MustPanic(t, func() {
		defer fixedbug73916bDeferred()
		panic("fixedbug73916b")
	})
	if fixedbug73916bRecovered {
		t.Fatal("nested recover returned non-nil")
	}
}

func skipBeforeGo126(t *testing.T) {
	t.Helper()
	version := runtime.Version()
	if strings.HasPrefix(version, "go1.26") || strings.HasPrefix(version, "devel") {
		return
	}
	t.Skip("requires Go 1.26 recover semantics")
}

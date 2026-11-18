package main

import (
	"reflect"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	_ "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

const LLGoFiles = "_wrap/ffi.c"

//go:linkname callOnForeignThread C.llgo_windows_call_foreign_thread
func callOnForeignThread(fn, arg uintptr, result *uintptr) int32

//go:linkname callOnForeignThreadStdcall C.llgo_windows_call_foreign_thread_stdcall
func callOnForeignThreadStdcall(fn, arg uintptr, result *uintptr) int32

//go:linkname callOnForeignThreadCDecl C.llgo_windows_call_foreign_thread_cdecl
func callOnForeignThreadCDecl(fn, arg uintptr, result *uintptr) int32

//go:linkname repeatOnForeignThreadCDecl C.llgo_windows_repeat_foreign_thread_cdecl
func repeatOnForeignThreadCDecl(fn, arg uintptr, repeats uint32, result *uintptr) int32

type callbackPair struct {
	Low  uint32
	High uint32
}

//go:linkname callPairCallback C.llgo_windows_call_pair_callback
func callPairCallback(fn uintptr, value callbackPair) uintptr

//go:linkname callNoArgCallback C.llgo_windows_call_no_arg_callback
func callNoArgCallback(fn uintptr) uintptr

type pair struct {
	Integer int64
	Float   float64
}

type mixedFunc func(int64, float64, complex128, pair) (int64, float64, complex128, pair)

type foreignGCProbe struct {
	value uintptr
}

func checkMixed(label string, got []reflect.Value) {
	if len(got) != 4 || got[0].Int() != 47 || got[1].Float() != 5.25 ||
		got[2].Complex() != complex(4.5, -1.25) ||
		got[3].Interface().(pair) != (pair{Integer: 16, Float: 5.5}) {
		panic("Windows reflect FFI corrupted " + label + " arguments or results")
	}
}

func makeForeignCallback(finalized chan uintptr, deferred, recovered *bool) func(uintptr) uintptr {
	probe := &foreignGCProbe{value: 100}
	runtime.SetFinalizer(probe, func(value *foreignGCProbe) {
		finalized <- value.value
	})
	return reflect.MakeFunc(reflect.TypeOf((func(uintptr) uintptr)(nil)), func(args []reflect.Value) []reflect.Value {
		defer func() { *deferred = true }()
		func() {
			defer func() {
				if value := recover(); value == "Windows foreign callback panic" {
					*recovered = true
				}
			}()
			panic("Windows foreign callback panic")
		}()
		stackProbe := &foreignGCProbe{value: 7}
		runtime.SetFinalizer(stackProbe, func(value *foreignGCProbe) {
			finalized <- value.value
		})
		runtime.GC()
		select {
		case <-finalized:
			panic("Windows foreign-thread callback lost a live GC root")
		default:
		}
		if stackProbe.value != 7 {
			panic("Windows foreign-thread callback corrupted a stack root")
		}
		runtime.KeepAlive(stackProbe)
		return []reflect.Value{reflect.ValueOf(probe.value + uintptr(args[0].Uint()))}
	}).Interface().(func(uintptr) uintptr)
}

//go:noinline
func checkForeignCallback(finalized chan uintptr) {
	deferred := false
	recovered := false
	foreign := makeForeignCallback(finalized, &deferred, &recovered)
	var result uintptr
	if errno := callOnForeignThread(reflect.ValueOf(foreign).Pointer(), 23, &result); errno != 0 || result != 123 {
		panic("Windows foreign-thread callback failed")
	}
	runtime.KeepAlive(foreign)
	if !deferred || !recovered {
		panic("Windows foreign-thread callback lost defer or panic/recover state")
	}
}

func expectCallbackPanic(want string, call func()) {
	defer func() {
		if got := recover(); got != want {
			panic("syscall callback validation returned the wrong panic")
		}
	}()
	call()
	panic("syscall callback validation did not panic")
}

func callbackWith64Args(
	a00, a01, a02, a03, a04, a05, a06, a07 uintptr,
	a08, a09, a10, a11, a12, a13, a14, a15 uintptr,
	a16, a17, a18, a19, a20, a21, a22, a23 uintptr,
	a24, a25, a26, a27, a28, a29, a30, a31 uintptr,
	a32, a33, a34, a35, a36, a37, a38, a39 uintptr,
	a40, a41, a42, a43, a44, a45, a46, a47 uintptr,
	a48, a49, a50, a51, a52, a53, a54, a55 uintptr,
	a56, a57, a58, a59, a60, a61, a62, a63 uintptr,
) uintptr {
	return a00
}

func testCallbackValidation() {
	expectCallbackPanic("compileCallback: expected function with one uintptr-sized result", func() {
		syscall.NewCallback(42)
	})
	expectCallbackPanic("compileCallback: expected function with one uintptr-sized result", func() {
		syscall.NewCallback(func() {})
	})
	expectCallbackPanic("compileCallback: expected function with one uintptr-sized result", func() {
		syscall.NewCallback(func() uint8 { return 0 })
	})
	expectCallbackPanic("compileCallback: argument size is larger than uintptr", func() {
		syscall.NewCallback(func([2]uintptr) uintptr { return 0 })
	})
	expectCallbackPanic("compileCallback: type chan int is currently not supported for use in system callbacks", func() {
		syscall.NewCallback(func(chan int) uintptr { return 0 })
	})
	if runtime.GOARCH != "386" {
		expectCallbackPanic("compileCallback: float arguments not supported", func() {
			syscall.NewCallback(func(float32) uintptr { return 0 })
		})
	} else {
		expectCallbackPanic("compileCallback: function argument frame too large", func() {
			syscall.NewCallback(callbackWith64Args)
		})
	}
}

func registerStdcallCallback(finalized chan uintptr, deferred, recovered *bool) uintptr {
	probe := &foreignGCProbe{value: 200}
	runtime.SetFinalizer(probe, func(value *foreignGCProbe) {
		finalized <- value.value
	})
	callback := func(argument uintptr) uintptr {
		defer func() { *deferred = true }()
		func() {
			defer func() {
				if value := recover(); value == "Windows syscall callback panic" {
					*recovered = true
				}
			}()
			panic("Windows syscall callback panic")
		}()
		stackProbe := &foreignGCProbe{value: 11}
		runtime.SetFinalizer(stackProbe, func(value *foreignGCProbe) {
			finalized <- value.value
		})
		runtime.GC()
		select {
		case <-finalized:
			panic("Windows syscall callback lost a live GC root")
		default:
		}
		if stackProbe.value != 11 {
			panic("Windows syscall callback corrupted a stack root")
		}
		runtime.KeepAlive(stackProbe)
		return probe.value + argument
	}
	code := syscall.NewCallback(callback)
	if code == 0 || syscall.NewCallback(callback) != code {
		panic("syscall.NewCallback did not cache the callback")
	}
	codes := make(chan uintptr, 8)
	for i := 0; i < 8; i++ {
		go func() { codes <- syscall.NewCallback(callback) }()
	}
	for i := 0; i < 8; i++ {
		if <-codes != code {
			panic("concurrent syscall.NewCallback returned a different callback")
		}
	}
	if runtime.GOARCH != "386" && syscall.NewCallbackCDecl(callback) != code {
		panic("Windows 64-bit callbacks unexpectedly distinguished cdecl")
	}
	return code
}

//go:noinline
func makeSyscallCallback(base uintptr) func(uintptr) uintptr {
	return func(argument uintptr) uintptr { return base + argument }
}

func testDistinctClosureCallbacks() {
	first := makeSyscallCallback(600)
	second := makeSyscallCallback(700)
	if reflect.ValueOf(first).Pointer() != reflect.ValueOf(second).Pointer() {
		panic("callback closure test did not share one code entry")
	}

	firstCode := syscall.NewCallbackCDecl(first)
	secondCode := syscall.NewCallbackCDecl(second)
	if firstCode == secondCode {
		panic("callbacks with distinct closure environments shared a trampoline")
	}
	first = nil
	second = nil
	runtime.GC()

	var result uintptr
	if errno := callOnForeignThreadCDecl(firstCode, 1, &result); errno != 0 || result != 601 {
		panic("first closure callback lost its environment")
	}
	if errno := callOnForeignThreadCDecl(secondCode, 2, &result); errno != 0 || result != 702 {
		panic("second closure callback lost its environment")
	}
}

func testRepeatedForeignThreadCallback() {
	var calls uintptr
	callback := syscall.NewCallbackCDecl(func(argument uintptr) uintptr {
		calls++
		if calls%8 == 0 {
			runtime.GC()
		}
		return 800 + argument
	})
	var result uintptr
	if errno := repeatOnForeignThreadCDecl(callback, 10, 64, &result); errno != 0 || result != 873 {
		panic("repeated foreign-thread callback failed")
	}
	if calls != 64 {
		panic("repeated foreign-thread callback count is wrong")
	}
	// The foreign thread has exited, so its FLS lifecycle must have released
	// both the Go context root and the retained collector registration.
	runtime.GC()
}

func testForeignThreadCallbackGoexit() {
	deferred := false
	callback := syscall.NewCallbackCDecl(func(uintptr) uintptr {
		defer func() { deferred = true }()
		runtime.Goexit()
		return 1
	})
	var result uintptr
	if errno := callOnForeignThreadCDecl(callback, 0, &result); errno != 0 {
		panic("foreign-thread callback Goexit failed")
	}
	if !deferred {
		panic("foreign-thread callback Goexit skipped defer")
	}
	runtime.GC()
}

func testSyscallCallbacks() {
	testCallbackValidation()
	testNativeCallbacks()
	testNativeFinalizers()

	finalized := make(chan uintptr, 2)
	deferred := false
	recovered := false
	stdcall := registerStdcallCallback(finalized, &deferred, &recovered)
	runtime.GC()
	var result uintptr
	if errno := callOnForeignThreadStdcall(stdcall, 23, &result); errno != 0 || result != 223 {
		panic("syscall.NewCallback failed on a foreign thread")
	}
	if !deferred || !recovered {
		panic("syscall.NewCallback lost defer or panic/recover state")
	}

	base := uintptr(300)
	cdeclFn := func(argument uintptr) uintptr { return base + argument }
	cdecl := syscall.NewCallbackCDecl(cdeclFn)
	if cdecl == 0 || syscall.NewCallbackCDecl(cdeclFn) != cdecl {
		panic("syscall.NewCallbackCDecl did not cache the callback")
	}
	stdcallForCDeclFn := syscall.NewCallback(cdeclFn)
	if runtime.GOARCH == "386" {
		if stdcallForCDeclFn == cdecl {
			panic("Windows 386 callbacks did not distinguish stdcall and cdecl")
		}
	} else if stdcallForCDeclFn != cdecl {
		panic("Windows 64-bit callbacks unexpectedly distinguished stdcall")
	}
	cdeclFn = nil
	runtime.GC()
	result = 0
	if errno := callOnForeignThreadCDecl(cdecl, 21, &result); errno != 0 || result != 321 {
		panic("syscall.NewCallbackCDecl failed on a foreign thread")
	}

	if runtime.GOARCH != "386" {
		pairBase := uintptr(400)
		pairCallback := syscall.NewCallbackCDecl(func(value callbackPair) uintptr {
			return pairBase + uintptr(value.Low) + uintptr(value.High)
		})
		if got := callPairCallback(pairCallback, callbackPair{Low: 5, High: 7}); got != 412 {
			panic("Windows callback corrupted a pointer-sized aggregate argument")
		}
	}

	type emptyArg struct{}
	emptyCallback := syscall.NewCallbackCDecl(func(emptyArg) uintptr { return 515 })
	if got := callNoArgCallback(emptyCallback); got != 515 {
		panic("Windows callback corrupted a zero-sized argument")
	}
	testDistinctClosureCallbacks()
	testRepeatedForeignThreadCallback()
	testForeignThreadCallbackGoexit()
}

//llgo:cdecl
type nativeCDeclCallback func(uintptr) uintptr

//llgo:stdcall
type nativeStdcallCallback func(uintptr) uintptr

//llgo:cdecl
type nativeInvalidCallback func(float64) uintptr

func nativeCallbackResult(value uintptr) uintptr { return value + 600 }

func testNativeCallbacks() {
	for _, fn := range []any{nativeCDeclCallback(nativeCallbackResult), nativeStdcallCallback(nativeCallbackResult)} {
		cdecl := syscall.NewCallbackCDecl(fn)
		stdcall := syscall.NewCallback(fn)
		if cdecl == 0 || syscall.NewCallbackCDecl(fn) != cdecl || syscall.NewCallback(fn) != stdcall {
			panic("native callback was not cached")
		}
		if (cdecl == stdcall) != (runtime.GOARCH != "386") {
			panic("native callback entry has the wrong calling convention")
		}
		var result uintptr
		if errno := callOnForeignThreadStdcall(stdcall, 23, &result); errno != 0 || result != 623 {
			panic("native callback through stdcall entry failed")
		}
		// Repetition on one thread exposes 386 stack cleanup errors in both
		// the generated callback entry and the call to the original function.
		const repeats = 4096
		if errno := repeatOnForeignThreadCDecl(cdecl, 23, repeats, &result); errno != 0 || result != 623+repeats-1 {
			panic("native callback through cdecl entry failed")
		}
	}
	for _, fn := range []any{nativeCDeclCallback(nil), nativeStdcallCallback(nil)} {
		expectCallbackPanic("compileCallback: expected function with one uintptr-sized result", func() {
			syscall.NewCallback(fn)
		})
	}
	if runtime.GOARCH != "386" {
		expectCallbackPanic("compileCallback: float arguments not supported", func() {
			syscall.NewCallback(nativeInvalidCallback(func(float64) uintptr { return 0 }))
		})
	}
}

type nativeFinalizerObject struct {
	done  chan<- uintptr
	value uintptr
}

//llgo:cdecl
type nativeCDeclFinalizer func(*nativeFinalizerObject)

//llgo:stdcall
type nativeStdcallFinalizer func(*nativeFinalizerObject)

//llgo:cdecl
type collectorFinalizer func(unsafe.Pointer, unsafe.Pointer)

//go:linkname registerCollectorFinalizer C.GC_register_finalizer
func registerCollectorFinalizer(unsafe.Pointer, collectorFinalizer, unsafe.Pointer, *collectorFinalizer, *unsafe.Pointer)

func nativeFinalize(p *nativeFinalizerObject) { p.done <- p.value }

func testNativeFinalizers() {
	for _, fn := range []any{nativeCDeclFinalizer(nativeFinalize), nativeStdcallFinalizer(nativeFinalize)} {
		done := make(chan uintptr, 1)
		p := &nativeFinalizerObject{done: done, value: 42}
		runtime.SetFinalizer(p, fn)
		// Invoke the collector's registered callback deterministically; the
		// regular runtime tests separately exercise discovery by the GC.
		var callback collectorFinalizer
		var data unsafe.Pointer
		registerCollectorFinalizer(unsafe.Pointer(p), nil, nil, &callback, &data)
		if callback == nil {
			panic("native finalizer was not registered")
		}
		callback(unsafe.Pointer(p), data)
		select {
		case value := <-done:
			if value != 42 {
				panic("native finalizer lost its argument")
			}
		case <-time.After(3 * time.Second):
			panic("native finalizer did not run")
		}
		runtime.KeepAlive(p)
	}
}

func main() {
	base := int64(40)
	integer := func(value int64) int64 { return base + value }
	if got := reflect.ValueOf(integer).Call([]reflect.Value{reflect.ValueOf(int64(2))})[0].Int(); got != 42 {
		panic("Windows reflect FFI corrupted an integer call")
	}
	floating := func(value float64) float64 { return value + 1.5 }
	if got := reflect.ValueOf(floating).Call([]reflect.Value{reflect.ValueOf(2.25)})[0].Float(); got != 3.75 {
		panic("Windows reflect FFI corrupted a floating-point call")
	}
	aggregate := func(value pair) pair {
		return pair{Integer: value.Integer + 1, Float: value.Float + 2}
	}
	if got := reflect.ValueOf(aggregate).Call([]reflect.Value{reflect.ValueOf(pair{3, 4})})[0].Interface().(pair); got != (pair{4, 6}) {
		panic("Windows reflect FFI corrupted an aggregate call")
	}
	complexValue := func(value complex128) complex128 { return value + complex(1, -2) }
	if got := reflect.ValueOf(complexValue).Call([]reflect.Value{reflect.ValueOf(complex(3.5, 0.75))})[0].Complex(); got != complex(4.5, -1.25) {
		panic("Windows reflect FFI corrupted a complex call")
	}
	dynamic := mixedFunc(func(integer int64, floating float64, value complex128, aggregate pair) (int64, float64, complex128, pair) {
		return base + integer, floating + 2, value + complex(1, -2), pair{
			Integer: aggregate.Integer + 5,
			Float:   aggregate.Float + 1.5,
		}
	})
	args := []reflect.Value{
		reflect.ValueOf(int64(7)),
		reflect.ValueOf(3.25),
		reflect.ValueOf(complex(3.5, 0.75)),
		reflect.ValueOf(pair{Integer: 11, Float: 4}),
	}
	checkMixed("dynamic", reflect.ValueOf(dynamic).Call(args))

	typ := reflect.TypeOf(mixedFunc(nil))
	made := reflect.MakeFunc(typ, func(args []reflect.Value) []reflect.Value {
		return reflect.ValueOf(dynamic).Call(args)
	}).Interface().(mixedFunc)
	gotInteger, gotFloat, gotComplex, gotPair := made(7, 3.25, complex(3.5, 0.75), pair{Integer: 11, Float: 4})
	if gotInteger != 47 || gotFloat != 5.25 || gotComplex != complex(4.5, -1.25) ||
		gotPair != (pair{Integer: 16, Float: 5.5}) {
		panic("Windows libffi closure corrupted a direct call")
	}
	checkMixed("MakeFunc", reflect.ValueOf(made).Call(args))

	for attempt := 0; attempt < 4; attempt++ {
		finalized := make(chan uintptr, 2)
		checkForeignCallback(finalized)
	}
	testSyscallCallbacks()

	println("windows FFI smoke: ok")
}

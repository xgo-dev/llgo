package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestImmediateClosureStorage(t *testing.T) {
	const src = `package capturealloc
import "unsafe"
func pointer(p unsafe.Pointer) unsafe.Pointer { return func() unsafe.Pointer { return p }() }
func mutate(x int) int { func(){x++}(); return x }
func loop(x int, n int) int { for i:=0;i<n;i++ { func(){x+=n}() }; return x }
func zero(x struct{}) { func(){ _ = x }() }
func escape(x int) func() int { return func() int { return x } }
func address(x int) *int { return func() *int { return &x }() }
func deferred(x int) (r int) { defer func(){r=x}(); return }
func consume(int)
func async(x int) { go func(){consume(x)}() }
func unknown(func() int)
func argument(x int) { unknown(func() int { return x }) }
`
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "386"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			for _, mode := range []ssa.BuilderMode{0, ssa.GlobalDebug} {
				ir := compileWithRewritesModeTarget(t, src, nil, mode, &target)
				for _, name := range []string{"pointer", "mutate", "loop", "zero"} {
					if body := llvmFunction(t, ir, "capturealloc."+name); strings.Contains(body, "runtime.Alloc") {
						t.Errorf("%s retains heap capture storage:\n%s", name, body)
					}
				}
				for _, name := range []string{"escape", "address", "deferred", "async", "argument"} {
					body := llvmFunction(t, ir, "capturealloc."+name)
					// Captured variables use zeroed storage; fully initialized
					// environments use AllocU. Check the two lifetimes separately.
					if !strings.Contains(body, "runtime.AllocZ") {
						t.Errorf("%s lost heap capture storage:\n%s", name, body)
					}
					wantHeapEnv := name != "address"
					if got := strings.Contains(body, "runtime.AllocU"); got != wantHeapEnv {
						t.Errorf("%s heap environment = %t, want %t:\n%s", name, got, wantHeapEnv, body)
					}
				}
			}
		})
	}
}

func TestCgoWrapperElidesUnusedFrame(t *testing.T) {
	const src = `package cgoframe
import "unsafe"
var _cgo_scalar, _cgo_pair unsafe.Pointer
func _cgo_runtime_cgocall(unsafe.Pointer,uintptr) int32
type pair struct { a,b uint64 }
func _Cfunc_scalar(p0 uint64)(r1 uint64) {
	_cgo_runtime_cgocall(_cgo_scalar,uintptr(unsafe.Pointer(&p0)))
	return
}

func _Cfunc_pair(p0 uint64)(r1 pair) {
	_cgo_runtime_cgocall(_cgo_pair,uintptr(unsafe.Pointer(&p0)))
	return
}
func _Cfunc_noargs()(r1 uint64) {
	_cgo_runtime_cgocall(_cgo_scalar,uintptr(unsafe.Pointer(&r1)))
	return
}
`
	ir := compileWithRewrites(t, src, nil)
	for _, name := range []string{"scalar", "pair", "noargs"} {
		body := llvmFunction(t, ir, "cgoframe._Cfunc_"+name)
		if strings.Contains(body, "runtime.Alloc") {
			t.Fatalf("%s retains unused argument/result frame:\n%s", name, body)
		}
	}
}

func TestImmediateClosureRejectsAdditionalUses(t *testing.T) {
	for _, scenario := range []string{"no calls", "also passed as argument"} {
		t.Run(scenario, func(t *testing.T) {
			pkg, _, _ := buildGoSSAPkg(t, `package captureuses
func f(x int) int { return func(y int) int { return x+y }(1) }
`)
			var closure *ssa.MakeClosure
			var call *ssa.Call
			for _, block := range pkg.Func("f").Blocks {
				for _, instr := range block.Instrs {
					switch instr := instr.(type) {
					case *ssa.MakeClosure:
						closure = instr
					case *ssa.Call:
						call = instr
					}
				}
			}
			if closure == nil || call == nil {
				t.Fatal("missing closure graph")
			}
			ctx := &context{prog: newLLSSAProg(t)}
			if !ctx.canStackAllocateClosure(closure) {
				t.Fatal("control closure was not eligible")
			}
			if scenario == "no calls" {
				*closure.Referrers() = nil
			} else {
				// A use-def rewrite must not hide an additional argument use
				// behind the same instruction's synchronous callee use.
				copy := *call
				copy.Call.Args = append([]ssa.Value(nil), call.Call.Args...)
				copy.Call.Args = append(copy.Call.Args, closure)
				*closure.Referrers() = []ssa.Instruction{&copy}
			}
			if ctx.canStackAllocateClosure(closure) {
				t.Fatal("additional use was accepted")
			}
		})
	}
}

func TestImmediateClosureCaptureBoundaries(t *testing.T) {
	if (&context{}).canStackAllocateCapture(&ssa.Alloc{Heap: true}) {
		t.Fatal("untyped capture was accepted")
	}
	for _, scenario := range []string{"zero-sized", "unknown callee", "no users"} {
		t.Run(scenario, func(t *testing.T) {
			captureType := "int"
			if scenario == "zero-sized" {
				captureType = "struct{}"
			}
			pkg, _, _ := buildGoSSAPkg(t, "package captureboundaries\nfunc f(x "+captureType+") "+captureType+" { return func() "+captureType+" { return x }() }")
			var alloc *ssa.Alloc
			var closure *ssa.MakeClosure
			var call *ssa.Call
			for _, block := range pkg.Func("f").Blocks {
				for _, instr := range block.Instrs {
					switch instr := instr.(type) {
					case *ssa.Alloc:
						if instr.Comment == "x" {
							alloc = instr
						}
					case *ssa.MakeClosure:
						closure = instr
					case *ssa.Call:
						call = instr
					}
				}
			}
			if alloc == nil || closure == nil || call == nil || !alloc.Heap {
				t.Fatal("missing heap capture graph")
			}
			prog := newLLSSAProg(t)
			t.Cleanup(prog.Dispose)
			ctx := &context{prog: prog}
			if scenario == "zero-sized" {
				if ctx.canStackAllocateCapture(alloc) {
					t.Fatal("zero-sized capture lost the shared sentinel fallback")
				}
				return
			}
			if !ctx.canStackAllocateCapture(alloc) {
				t.Fatal("control capture was not eligible")
			}
			copy := *alloc
			if scenario == "no users" {
				*copy.Referrers() = nil
			} else {
				closureCopy, callCopy := *closure, *call
				closureCopy.Fn = ssa.NewConst(nil, closure.Fn.Type())
				closureCopy.Bindings = []ssa.Value{&copy}
				callCopy.Call.Value = &closureCopy
				*closureCopy.Referrers() = []ssa.Instruction{&callCopy}
				*copy.Referrers() = []ssa.Instruction{&closureCopy}
			}
			if ctx.canStackAllocateCapture(&copy) {
				t.Fatal("capture without a known borrowed lifetime was accepted")
			}
		})
	}
}

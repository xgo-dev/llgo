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
				for _, name := range []string{"pointer", "mutate", "loop"} {
					if body := llvmFunction(t, ir, "capturealloc."+name); strings.Contains(body, "runtime.Alloc") {
						t.Errorf("%s retains heap capture storage:\n%s", name, body)
					}
				}
				for _, name := range []string{"escape", "address", "deferred", "async", "argument"} {
					if body := llvmFunction(t, ir, "capturealloc."+name); !strings.Contains(body, "runtime.Alloc") {
						t.Errorf("%s lost escaping storage:\n%s", name, body)
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

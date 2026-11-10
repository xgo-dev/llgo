package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestNativePointerConversionRejectsDifferentStorage(t *testing.T) {
	const declarations = `package repr
//llgo:cdecl
type Native struct { Fn func(); Tail uint64 }
type Copy Native
type Plain struct { Fn func(); Tail uint64 }
`
	for _, target := range []*llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-emscripten", WasmProfile: "j32"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			for _, tc := range []struct {
				source string
				valid  bool
			}{
				{`func convert(p *Native) *Copy { return (*Copy)(p) }`, true},
				{`func convert(p *Native) *Plain { return (*Plain)(p) }`, false},
				{`func convert(p *Plain) *Native { return (*Native)(p) }`, false},
			} {
				pkg, files := buildCallerFrameSSAPackage(t, "repr", declarations+tc.source)
				prog := newLLSSAProgForTarget(t, target)
				_, _, err := NewPackageEx(prog, nil, nil, pkg, files)
				prog.Dispose()
				if tc.valid {
					if err != nil {
						t.Fatal(err)
					}
				} else if err == nil || !strings.Contains(err.Error(), "caller_frame_compile.go:") || !strings.Contains(err.Error(), "incompatible native storage") {
					t.Fatalf("conversion error = %v", err)
				}
			}
		})
	}
}

func TestNativeCallbackConversionChecksInheritedABI(t *testing.T) {
	const source = `package repr
//llgo:cdecl
type C func() int32
//llgo:stdcall
type S func() int32
type Inherited S
func convert(fn Inherited) C { return C(fn) }
`
	for _, arch := range []string{"386", "amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			pkg, files := buildCallerFrameSSAPackage(t, "repr", source)
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "windows", GOARCH: arch})
			defer prog.Dispose()
			_, _, err := NewPackageEx(prog, nil, nil, pkg, files)
			if arch == "386" {
				if err == nil || !strings.Contains(err.Error(), "incompatible native callback ABI") {
					t.Fatalf("inherited stdcall conversion error = %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNativeArrayConversionChecksElementStorage(t *testing.T) {
	for _, target := range []*llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-emscripten", WasmProfile: "j32"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			pkg, files := buildCallerFrameSSAPackage(t, "repr", `package repr
//llgo:cdecl
type Native [3]uintptr
type Copy Native
func inherited(a Native) Copy { return Copy(a) }
func ordinary(a Native) [3]uintptr { return [3]uintptr(a) }
`)
			prog := newLLSSAProgForTarget(t, target)
			defer prog.Dispose()
			_, _, err := NewPackageEx(prog, nil, nil, pkg, files)
			if target.GOARCH == "wasm" {
				if err == nil || !strings.Contains(err.Error(), "incompatible native array storage") {
					t.Fatalf("array conversion error = %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

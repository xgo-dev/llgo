package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestUnsafeViewStorageTargets(t *testing.T) {
	const source = `package unsafeviews
import "unsafe"
func local(value byte) byte {
 data:=[2]byte{value,value+1}
 view:=unsafe.Slice(unsafe.SliceData(data[:]),len(data))
 p:=(*byte)(unsafe.Add(unsafe.Pointer(&view[0]),1))
 return *unsafe.StringData(unsafe.String(p,1))
}
func retained(value byte) string {
 data:=[2]byte{value,value+1}
 return unsafe.String(unsafe.SliceData(data[:]),len(data))
}
`
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			ir := compileInterfaceBoxIR(t, source, ssa.InstantiateGenerics, &target, target.GOARCH == "wasm")
			if body := llvmFunction(t, ir, "unsafeviews.local"); strings.Contains(body, "runtime.Alloc") {
				t.Errorf("local unsafe view retains heap storage:\n%s", body)
			}
			if body := llvmFunction(t, ir, "unsafeviews.retained"); !strings.Contains(body, "runtime.Alloc") {
				t.Errorf("returned unsafe view lost backing storage:\n%s", body)
			}
		})
	}
}

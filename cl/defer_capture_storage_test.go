package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestRecoverVisibleInvocationStorage(t *testing.T) {
	const source = `package recoverstorage
var Sink int
var Escaped *bool
func local(value int, callback func()) {
 returned := false
 defer func(){ if !returned { Sink = -value } else { Sink = value } }()
 callback()
 returned = true
}
func named(value int, callback func()) (result int) {
 defer func(){ result += value }()
 callback()
 return value
}
func escaping(callback func()) {
 returned := false
 defer func(){ returned = !returned }()
 Escaped = &returned
 callback()
}
func asynchronous(callback func()) {
 returned := false
 defer func(){ if returned { Sink++ } }()
 go func(){ returned = true; callback() }()
}
func loop(n int) {
 for i:=0; i<n; i++ { value := i; defer func(){ Sink += value }() }
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
			for _, name := range []string{"local", "named"} {
				body := llvmFunction(t, ir, "recoverstorage."+name)
				if strings.Contains(body, "runtime.Alloc") {
					t.Errorf("%s retains invocation-owned capture storage:\n%s", name, body)
				}
				if !strings.Contains(body, "store volatile") || (name == "named" && !strings.Contains(body, "load volatile")) {
					t.Errorf("%s lost recover-visible accesses:\n%s", name, body)
				}
			}
			for _, name := range []string{"escaping", "asynchronous", "loop"} {
				if body := llvmFunction(t, ir, "recoverstorage."+name); !strings.Contains(body, "runtime.Alloc") {
					t.Errorf("%s lost retained storage:\n%s", name, body)
				}
			}
		})
	}
}

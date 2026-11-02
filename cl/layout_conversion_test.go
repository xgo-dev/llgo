package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestInstantiatedPointerLayoutConversion(t *testing.T) {
	for _, tc := range []struct {
		name, field, conversion string
		allowed                 bool
	}{
		{"different", "uint64", "(*Plain[T])(p)", false},
		{"same", "byte", "(*Plain[T])(p)", true},
		{"unsafe", "uint64", "(*Plain[T])(unsafe.Pointer(p))", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := `package main
import "unsafe"
var _ unsafe.Pointer
//llgo:packed
type Packed[T any] struct { B byte; X T }
type Plain[T any] struct { B byte; X T }
func convert[T any](p *Packed[T]) *Plain[T] { return ` + tc.conversion + ` }
func main() { p := Packed[` + tc.field + `]{X: 99}; println(convert(&p).X) }
`
			ssapkg, _, files := buildGoSSAPkg(t, source)
			prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			pkg, _, err := NewPackageEx(prog, nil, nil, ssapkg, files)
			if tc.allowed {
				if err != nil {
					t.Fatal(err)
				}
				if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "foo.go:7:") || !strings.Contains(err.Error(), "different LLGo layouts") {
				t.Fatalf("expected positioned conversion error, got %v", err)
			}
		})
	}
}

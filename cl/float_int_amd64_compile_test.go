//go:build !llgo

package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestAMD64FloatToIntegerConversionIR(t *testing.T) {
	const src = `package floatconvert
func I8(x float64) int8 { return int8(x) }
func I32(x float64) int32 { return int32(x) }
func I64(x float64) int64 { return int64(x) }
func U8(x float64) uint8 { return uint8(x) }
func U32(x float64) uint32 { return uint32(x) }
func U64(x float64) uint64 { return uint64(x) }
`
	ssaPkg, _, files := buildGoSSAPkg(t, src)
	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	pkg, err := NewPackage(prog, ssaPkg, files)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		wants []string
	}{
		{name: "I8", wants: []string{"fptosi double", "to i32", "trunc i32", "to i8"}},
		{name: "I32", wants: []string{"fcmp olt double", "fcmp oge double", "fptosi double", "i32 -2147483648"}},
		{name: "I64", wants: []string{"fptosi double", "to i64", "i64 -9223372036854775808"}},
		{name: "U8", wants: []string{"fptosi double", "to i32", "trunc i32", "to i8"}},
		{name: "U32", wants: []string{"fptosi double", "to i64", "trunc i64", "to i32"}},
		{name: "U64", wants: []string{"fcmp oge double", "fsub double", "fptosi double", "or i64"}},
	}
	for _, tt := range tests {
		ir := mustNamedFunction(t, pkg.Module(), "floatconvert."+tt.name).String()
		for _, want := range tt.wants {
			if !strings.Contains(ir, want) {
				t.Errorf("%s conversion IR missing %q:\n%s", tt.name, want, ir)
			}
		}
	}
}

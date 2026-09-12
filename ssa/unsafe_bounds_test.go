//go:build !llgo

package ssa

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestUnsafeBoundsAddressWidth(t *testing.T) {
	for _, tc := range []struct {
		arch   string
		layout string
		width  int
	}{
		{"amd64", "e-p:64:64", 64},
		{"386", "e-p:32:32", 32},
		{"amd64", "e-p:64:64:64:32", 32},
	} {
		t.Run(tc.layout, func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: "linux", GOARCH: tc.arch})
			defer prog.Dispose()
			setTestRuntime(t, prog)
			pkg := prog.NewPackage("bounds", "bounds")
			pkg.Module().SetDataLayout(tc.layout)
			for _, builtin := range []string{"String", "Slice"} {
				elem := types.Typ[types.Byte]
				if builtin == "Slice" {
					elem = types.Typ[types.Uint64]
				}
				params := types.NewTuple(
					types.NewParam(token.NoPos, nil, "p", types.NewPointer(elem)),
					types.NewParam(token.NoPos, nil, "n", types.Typ[types.Int]),
				)
				fn := pkg.NewFunc(builtin, types.NewSignatureType(nil, nil, nil, params, nil, false), InC)
				fn.impl.Param(0).SetName("p")
				fn.impl.Param(1).SetName("n")
				b := fn.MakeBody(1)
				b.BuiltinCall(builtin, fn.Param(0), fn.Param(1))
				b.Return()
				b.EndBuild()
				ir := fn.impl.String()
				if !strings.Contains(ir, fmt.Sprintf("ptrtoaddr ptr %%p to i%d", tc.width)) || strings.Contains(ir, "ptrtoint") {
					t.Fatalf("%s should observe the address without pointer provenance:\n%s", builtin, ir)
				}
				if !strings.Contains(ir, fmt.Sprintf("add i%d", tc.width)) || !strings.Contains(ir, fmt.Sprintf("icmp ult i%d", tc.width)) {
					t.Fatalf("%s wraparound must use the address width:\n%s", builtin, ir)
				}
				if tc.arch == "amd64" && tc.width == 32 {
					maxLen := uint64(1<<32 - 1)
					if builtin == "Slice" {
						maxLen /= 8
					}
					if !strings.Contains(ir, fmt.Sprintf("icmp ugt i64 %%n, %d", maxLen)) || !strings.Contains(ir, "trunc i64") {
						t.Fatalf("%s must reject oversized lengths before truncating the byte offset:\n%s", builtin, ir)
					}
				}
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

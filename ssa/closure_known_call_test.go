package ssa

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestKnownClosureEntryRequiresGoRole(t *testing.T) {
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			setTestRuntime(t, prog)
			pkg := prog.NewPackage("known", "example.com/known")
			params := types.NewTuple(types.NewParam(token.NoPos, nil, "n", types.Typ[types.Int64]))
			sig := types.NewSignatureType(nil, nil, nil, params, nil, false)
			entry := pkg.NewFunc("goEntry", sig, GoDecl)
			caller := pkg.NewFunc("caller", NoArgsNoRet, GoDecl)
			body := caller.MakeBody(1)
			value := body.MakeClosure(entry.Expr, nil)
			argument := prog.IntVal(42, prog.Int64())
			body.CallKnownClosure(value, entry, argument)
			other := NewProgram(target)
			defer other.Dispose()
			foreign := other.NewPackage("other", "example.com/other").NewFunc("otherEntry", sig, GoDecl)
			tests := []struct {
				name  string
				value Expr
				entry Function
			}{
				{"CDecl", value, pkg.NewFunc("cEntry", sig, CDecl)},
				{"nil entry", value, nil},
				{"different program", value, foreign},
				{"different signature", value, pkg.NewFunc("wrongSignature", NoArgsNoRet, GoDecl)},
				{"not a closure", entry.Expr, entry},
			}
			if target.GOOS == "windows" {
				tests = append(tests, struct {
					name  string
					value Expr
					entry Function
				}{"Stdcall", value, pkg.NewFunc("stdcallEntry", sig, Stdcall)})
			}
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					defer func() {
						if recover() == nil {
							t.Fatal("accepted an entry without the matching Go closure contract")
						}
					}()
					body.CallKnownClosure(test.value, test.entry, argument)
				})
			}
			body.Return()
			body.EndBuild()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

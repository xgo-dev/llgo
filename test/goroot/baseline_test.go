package goroot

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"runtime"
	"sort"
	"strings"
)

// Go's reference cgo toolchain uses GNU/UCRT even when LLGo is being tested
// against MSVC. Host LLVM build flags must not enter that standalone program.
func baselineCgoEnv(env []string) []string {
	cc := caseEnvValue(env, "LLGO_GOROOT_BASELINE_CC")
	if cc == "" {
		return env
	}
	out := append([]string{}, env...)
	out = upsertEnv(out, "CC="+cc)
	if cxx := caseEnvValue(env, "LLGO_GOROOT_BASELINE_CXX"); cxx != "" {
		out = upsertEnv(out, "CXX="+cxx)
	}
	for _, key := range []string{"CGO_CFLAGS", "CGO_CPPFLAGS", "CGO_CXXFLAGS", "CGO_FFLAGS", "CGO_LDFLAGS"} {
		out = upsertEnv(out, key+"=")
	}
	return out
}

func caseEnvValue(env []string, name string) string {
	for _, item := range env {
		key, value, _ := strings.Cut(item, "=")
		if key == name || runtime.GOOS == "windows" && strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}

func targetGoArch(env []string) string {
	if *flagWasmProfile != "" {
		return "wasm"
	}
	if arch := caseEnvValue(env, "GOARCH"); arch != "" {
		return arch
	}
	return runtime.GOARCH
}

// A few upstream run cases pass constants outside int32 to print/println.
// Give just those arguments an explicit 64-bit type in the staged source used
// by both compilers. Keep the expressions, assertions and line numbers intact.
func normalizePrintConstants(filename, arch string) (int, error) {
	sizes := types.SizesFor("gc", arch)
	if sizes == nil || sizes.Sizeof(types.Typ[types.Int]) >= 8 {
		return 0, nil
	}
	src, err := os.ReadFile(filename)
	if err != nil {
		return 0, err
	}
	if !bytes.Contains(src, []byte("print")) {
		return 0, nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		return 0, nil // Keep the compiler's original source diagnostic.
	}
	info := &types.Info{
		Types:  make(map[ast.Expr]types.TypeAndValue),
		Uses:   make(map[*ast.Ident]types.Object),
		Scopes: make(map[ast.Node]*types.Scope),
	}
	// Resolve local constant expressions without loading imported export data.
	// The real compiler retains responsibility for the complete program.
	config := types.Config{Sizes: sizes, Error: func(error) {}}
	pkg, _ := config.Check(file.Name.Name, fset, []*ast.File{file}, info)
	bits := uint(sizes.Sizeof(types.Typ[types.Int]) * 8)
	minimum := constant.MakeInt64(-1 << (bits - 1))
	maximum := constant.MakeInt64(1<<(bits-1) - 1)
	type edit struct {
		start, end int
		typeName   string
	}
	var edits []edit
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name, ok := call.Fun.(*ast.Ident)
		if !ok || (name.Name != "print" && name.Name != "println") || info.Uses[name] != types.Universe.Lookup(name.Name) {
			return true
		}
		for _, arg := range call.Args {
			value := info.Types[arg]
			if value.Type != types.Typ[types.UntypedInt] || value.Value == nil {
				continue
			}
			if !constant.Compare(value.Value, token.LSS, minimum) && !constant.Compare(value.Value, token.GTR, maximum) {
				continue
			}
			typeName := "int64"
			if _, exact := constant.Int64Val(value.Value); !exact {
				if _, exact := constant.Uint64Val(value.Value); !exact {
					continue
				}
				typeName = "uint64"
			}
			if pkg != nil {
				if scope := pkg.Scope().Innermost(arg.Pos()); scope != nil {
					if _, obj := scope.LookupParent(typeName, arg.Pos()); obj != types.Universe.Lookup(typeName) {
						continue
					}
				}
			}
			edits = append(edits, edit{fset.Position(arg.Pos()).Offset, fset.Position(arg.End()).Offset, typeName})
		}
		return true
	})
	if len(edits) == 0 {
		return 0, nil
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	for _, edit := range edits {
		replacement := edit.typeName + "(" + string(src[edit.start:edit.end]) + ")"
		src = append(src[:edit.start], append([]byte(replacement), src[edit.end:]...)...)
	}
	if err := os.WriteFile(filename, src, 0o644); err != nil {
		return 0, fmt.Errorf("stage explicit print constant types: %w", err)
	}
	return len(edits), nil
}

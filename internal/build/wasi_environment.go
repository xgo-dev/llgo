package build

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"

	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
)

const wasiEnvironmentGuard = "syscall.llgoMutableEnvironment"
const wasiEnvironmentPatch = "getenv_wasip1.go"

func wasiEnvironmentPackage(prog *ssa.Program) *ssa.Package {
	p := prog.ImportedPackage("syscall")
	if p == nil || p.Members["llgoMutableEnvironment"] == nil {
		return nil
	}
	return p
}

func configureWASIEnvironment(ctx *context, p *ssa.Package, result *rta.Result, isSymbolRoot func(string) bool) {
	for _, name := range []string{wasiEnvironmentGuard, "syscall.envs", "syscall.copyenv", "syscall.env", "syscall.envLock", "syscall.envOnce"} {
		if isSymbolRoot(name) {
			return
		}
	}
	if _, reads := result.Reachable[p.Func("Getenv")]; !reads {
		return
	}
	// Environ can trigger the standard library's first-time duplicate removal.
	// Leave both the reader lock and enumeration intact when it is reachable.
	for _, name := range []string{"Environ", "Clearenv", "Setenv", "Unsetenv"} {
		if _, mutable := result.Reachable[p.Func(name)]; mutable {
			return
		}
	}
	ctx.initializerGuardsOff = append(ctx.initializerGuardsOff, wasiEnvironmentGuard)
	if ctx.buildConf.Verbose || ctx.buildConf.PrintCommands {
		fmt.Fprintln(os.Stderr, "llgo: specialize WASI read-only environment")
	}
}

func envIdent(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == name
}

// Select only the Unix environment table used by the supported WASI Go
// toolchains. The lock continues to protect the ordinary stdlib lookup.
func rewriteWASIGetenv(source []byte) []byte {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "env_unix.go", source, parser.SkipObjectResolution)
	if err != nil || file.Name.Name != "syscall" {
		return nil
	}
	var hasIndex, hasEntries bool
	for _, decl := range file.Decls {
		vars, ok := decl.(*ast.GenDecl)
		if !ok || vars.Tok != token.VAR {
			continue
		}
		for _, spec := range vars.Specs {
			v := spec.(*ast.ValueSpec)
			if len(v.Names) != 1 {
				continue
			}
			switch v.Names[0].Name {
			case "env":
				index, ok := v.Type.(*ast.MapType)
				hasIndex = ok && envIdent(index.Key, "string") && envIdent(index.Value, "int")
			case "envs":
				entries, ok := v.Type.(*ast.ArrayType)
				hasEntries = ok && entries.Len == nil && envIdent(entries.Elt, "string")
			}
		}
	}
	if !hasIndex || !hasEntries {
		return nil
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != "Getenv" || fn.Body == nil || len(fn.Type.Params.List) != 1 {
			continue
		}
		param := fn.Type.Params.List[0]
		results := fn.Type.Results
		if len(param.Names) != 1 || !envIdent(param.Type, "string") || results == nil || len(results.List) != 2 ||
			!envIdent(results.List[0].Type, "string") || !envIdent(results.List[1].Type, "bool") {
			return nil
		}
		for i := 0; i+1 < len(fn.Body.List); i++ {
			stmt, ok := fn.Body.List[i].(*ast.ExprStmt)
			if !ok {
				continue
			}
			call, ok := stmt.X.(*ast.CallExpr)
			deferStmt, isDefer := fn.Body.List[i+1].(*ast.DeferStmt)
			if !ok || !isDefer || !envLockCall(call, "RLock") || !envLockCall(deferStmt.Call, "RUnlock") {
				continue
			}
			pos := fset.Position(stmt.Pos()).Offset
			resultText := source[fset.Position(results.Pos()).Offset:fset.Position(results.End()).Offset]
			key := param.Names[0].Name
			branch := fmt.Sprintf("if !llgoMutableEnvironment { return llgoGetenvWithoutLock(%[1]s) }; return llgoGetenvWithLock(%[1]s)\n}\n\nfunc llgoGetenvWithLock(%[1]s string) %[2]s {\n\t", key, resultText)
			return slices.Concat(source[:pos], []byte(branch), source[pos:])
		}
	}
	return nil
}

func envLockCall(call *ast.CallExpr, name string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && len(call.Args) == 0 && envIdent(sel.X, "envLock") && sel.Sel.Name == name
}

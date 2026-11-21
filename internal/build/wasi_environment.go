package build

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"os"
	"slices"

	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
)

const mutableEnvironmentGuard = "syscall.llgoMutableEnvironment"
const wasiEnvironmentNeededGuard = "syscall.llgoEnvironmentNeeded"
const environmentSizePatch = "getenv_size.go"

func environmentSyscallPackage(prog *ssa.Program) *ssa.Package {
	p := prog.ImportedPackage("syscall")
	if p == nil || p.Members["llgoMutableEnvironment"] == nil && p.Members[wasiPathStateMember] == nil {
		return nil
	}
	return p
}

func configureReadOnlyEnvironment(ctx *context, p *ssa.Package, result *rta.Result, isSymbolRoot func(string) bool) {
	if p.Members["llgoMutableEnvironment"] == nil {
		return
	}
	for _, name := range []string{mutableEnvironmentGuard, "syscall.envs", "syscall.copyenv", "syscall.env", "syscall.envLock", "syscall.envOnce",
		"syscall.llgoEnvironmentCopier", "syscall.llgoCopyEnvironmentIndex", "syscall.llgoEnvironmentSlots",
		"syscall.llgoLoadIndexedEnvironment", "syscall.llgoIndexEnvironment", "syscall.llgoGetenvIndexed",
		"syscall.llgoGetenvWithLock", "syscall.Environ", "syscall.Clearenv", "syscall.Setenv", "syscall.Unsetenv"} {
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
	ctx.initializerGuardsOff = append(ctx.initializerGuardsOff, mutableEnvironmentGuard)
	if ctx.buildConf.Verbose || ctx.buildConf.PrintCommands {
		target := "native"
		if ctx.buildConf.Goos == "wasip1" {
			target = "WASI"
		}
		fmt.Fprintf(os.Stderr, "llgo: specialize %s read-only environment\n", target)
		if _, indexed := result.Reachable[p.Func("llgoLoadIndexedEnvironment")]; indexed {
			fmt.Fprintf(os.Stderr, "llgo: index the %s environment without a Go map\n", target)
		}
	}
}

// A Go environment snapshot is only needed if code can observe its state.
// libc's process environment does not depend on this copy.
func configureWASIEnvironmentSnapshot(ctx *context, p *ssa.Package, roots []*ssa.Function, result *rta.Result, pathInit *ssa.Function, isSymbolRoot func(string) bool) {
	load := p.Func("llgoLoadEnvironment")
	indexedLoad := p.Func("llgoLoadIndexedEnvironment")
	snapshot, _ := p.Members["envs"].(*ssa.Global)
	if p.Members["llgoEnvironmentNeeded"] == nil || load == nil || snapshot == nil {
		return
	}
	for _, name := range []string{wasiEnvironmentNeededGuard, mutableEnvironmentGuard, "syscall.llgoLoadEnvironment",
		"syscall.llgoEnvironmentCopier", "syscall.llgoCopyEnvironmentIndex", "syscall.llgoGetenvWithoutLock",
		"syscall.llgoLoadIndexedEnvironment", "syscall.llgoIndexEnvironment", "syscall.llgoGetenvIndexed",
		"syscall.llgoGetenvWithLock", "syscall.init", "syscall.Getenv", "syscall.Setenv", "syscall.Unsetenv",
		"syscall.Environ", "syscall.Clearenv"} {
		if isSymbolRoot(name) {
			return
		}
	}
	var state []*ssa.Global
	for _, name := range []string{"envs", "env", "envLock", "envOnce", "copyenv", "llgoEnvironmentSlots"} {
		if isSymbolRoot("syscall." + name) {
			return
		}
		if global, ok := p.Members[name].(*ssa.Global); ok {
			state = append(state, global)
		}
	}
	ignored := wasiPathEnvironmentCall(p, pathInit)
	copier := p.Func("llgoEnvironmentCopier")
	copyenv, _ := p.Members["copyenv"].(*ssa.Global)
	fillIndex := p.Func("llgoCopyEnvironmentIndex")
	slots, _ := p.Members["llgoEnvironmentSlots"].(*ssa.Global)
	init := p.Func("init")
	// The snapshot store and its eager index builder only initialize the
	// table. Any other reachable reference can observe or modify it.
	for fn := range result.Reachable {
		if fn == fillIndex {
			continue
		}
		for _, block := range fn.Blocks {
			for _, inst := range block.Instrs {
				if _, debug := inst.(*ssa.DebugRef); debug {
					continue
				}
				if store, ok := inst.(*ssa.Store); ok {
					if fn == indexedLoad && store.Addr == slots {
						continue
					}
					if call, ok := store.Val.(*ssa.Call); ok && fn == init {
						callee := call.Common().StaticCallee()
						if (store.Addr == snapshot && (callee == load || indexedLoad != nil && callee == indexedLoad)) ||
							(store.Addr == copyenv && copier != nil && callee == copier) {
							continue
						}
					}
				}
				for _, operand := range inst.Operands(nil) {
					if global, ok := (*operand).(*ssa.Global); ok && slices.Contains(state, global) &&
						hasCallPath(result.CallGraph, roots, fn, ignored) {
						return
					}
				}
			}
		}
	}
	ctx.initializerGuardsOff = append(ctx.initializerGuardsOff, wasiEnvironmentNeededGuard)
	if ctx.buildConf.Verbose || ctx.buildConf.PrintCommands {
		fmt.Fprintln(os.Stderr, "llgo: omit unobserved WASI environment snapshot")
	}
}

// Ignore the PWD lookup only when the linker disables the path-state guard
// and the guard's true branch dominates the call.
func wasiPathEnvironmentCall(p *ssa.Package, init *ssa.Function) ssa.CallInstruction {
	if init == nil {
		return nil
	}
	guard, _ := p.Members[wasiPathStateMember].(*ssa.Global)
	getenv := p.Func("Getenv")
	if guard == nil || getenv == nil {
		return nil
	}
	for _, block := range init.Blocks {
		if len(block.Instrs) == 0 {
			continue
		}
		branch, ok := block.Instrs[len(block.Instrs)-1].(*ssa.If)
		if !ok {
			continue
		}
		load, ok := branch.Cond.(*ssa.UnOp)
		if !ok || load.Op != token.MUL || load.X != guard {
			continue
		}
		for _, candidate := range init.Blocks {
			if !block.Succs[0].Dominates(candidate) {
				continue
			}
			for _, inst := range candidate.Instrs {
				call, ok := inst.(*ssa.Call)
				if !ok || call.Call.StaticCallee() != getenv || len(call.Call.Args) != 1 {
					continue
				}
				if key, ok := call.Call.Args[0].(*ssa.Const); ok && key.Value != nil &&
					key.Value.Kind() == constant.String && constant.StringVal(key.Value) == "PWD" {
					return call
				}
			}
		}
	}
	return nil
}

func envIdent(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == name
}

// Select only the Unix environment table used by the supported Go
// toolchains. The lock continues to protect the ordinary stdlib lookup.
func rewriteEnvironmentLookup(source []byte) []byte {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "env_unix.go", source, parser.SkipObjectResolution)
	if err != nil || file.Name.Name != "syscall" {
		return nil
	}
	var hasIndex bool
	var snapshot *ast.CallExpr
	var copyEnv *ast.CallExpr
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
			case "copyenv":
				if !vars.Lparen.IsValid() && len(v.Values) == 1 {
					copyEnv = wasiOnceEnvironmentCopy(v.Values[0])
				}
			case "env":
				index, ok := v.Type.(*ast.MapType)
				hasIndex = ok && envIdent(index.Key, "string") && envIdent(index.Value, "int")
			case "envs":
				entries, ok := v.Type.(*ast.ArrayType)
				if ok && entries.Len == nil && envIdent(entries.Elt, "string") && len(v.Values) == 1 {
					if call, ok := v.Values[0].(*ast.CallExpr); ok && envIdent(call.Fun, "runtime_envs") && len(call.Args) == 0 {
						snapshot = call
					}
				}
			}
		}
	}
	if !hasIndex || snapshot == nil {
		return nil
	}
	snapshotStart := fset.Position(snapshot.Pos()).Offset
	snapshotEnd := fset.Position(snapshot.End()).Offset
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
			if snapshotEnd > pos || copyEnv != nil && snapshotEnd > fset.Position(copyEnv.Pos()).Offset {
				return nil
			}
			prefix := source[:pos]
			resultText := source[fset.Position(results.Pos()).Offset:fset.Position(results.End()).Offset]
			key := param.Names[0].Name
			getter, loader := "llgoGetenvWithoutLock", "llgoLoadEnvironment"
			onceFunc := copyEnv != nil && copyEnv.End() < fn.Pos() && directEnvironmentCopy(fn.Body.List[0])
			if onceFunc || copyEnv == nil && onceEnvironmentCopy(fn.Body.List[0]) {
				getter, loader = "llgoGetenvIndexed", "llgoLoadIndexedEnvironment"
				pos = fset.Position(fn.Body.List[0].Pos()).Offset
				prefix = source[:pos]
			}
			if onceFunc {
				body := copyEnv.Args[0].(*ast.FuncLit).Body
				prefix = slices.Concat(source[:fset.Position(copyEnv.Pos()).Offset], []byte(wasiEnvironmentCopier),
					source[fset.Position(body.Pos()).Offset:fset.Position(body.End()).Offset],
					source[fset.Position(copyEnv.End()).Offset:pos])
			}
			branch := fmt.Sprintf("if !llgoMutableEnvironment { return %[3]s(%[1]s) }; return llgoGetenvWithLock(%[1]s)\n}\n\nfunc llgoGetenvWithLock(%[1]s string) %[2]s {\n\t", key, resultText, getter)
			return slices.Concat(prefix[:snapshotStart], []byte(loader+"()"), prefix[snapshotEnd:],
				[]byte(branch), source[pos:])
		}
	}
	return nil
}

// The indexed loader prepares read-only lookups during package initialization.
// Mutable links keep the standard library's lazy OnceFunc path.
const wasiEnvironmentCopier = `llgoEnvironmentCopier()

func llgoEnvironmentCopier() func() {
	if !llgoEnvironmentNeeded || !llgoMutableEnvironment {
		return nil
	}
	return sync.OnceFunc(llgoCopyEnvironmentIndex)
}

func llgoCopyEnvironmentIndex() `

func wasiOnceEnvironmentCopy(expr ast.Expr) *ast.CallExpr {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return nil
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !envIdent(selector.X, "sync") || selector.Sel.Name != "OnceFunc" {
		return nil
	}
	if fn, ok := call.Args[0].(*ast.FuncLit); ok && len(fn.Type.Params.List) == 0 && fn.Type.Results == nil {
		return call
	}
	return nil
}

func directEnvironmentCopy(stmt ast.Stmt) bool {
	expr, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := expr.X.(*ast.CallExpr)
	return ok && envIdent(call.Fun, "copyenv") && len(call.Args) == 0
}

func onceEnvironmentCopy(stmt ast.Stmt) bool {
	expr, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || !envIdent(call.Args[0], "copyenv") {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && envIdent(sel.X, "envOnce") && sel.Sel.Name == "Do"
}

func envLockCall(call *ast.CallExpr, name string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && len(call.Args) == 0 && envIdent(sel.X, "envLock") && sel.Sel.Name == name
}

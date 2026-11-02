package build

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/optlevel"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// A closed program with no Go allocation, panic, callbacks or scheduler use
// needs neither logical frames nor runtime initialization. Prove this before
// lowering: frame bookkeeping itself would otherwise introduce allocations.
func configureWasmRuntimeFree(ctx *context) {
	conf := ctx.buildConf
	if len(ctx.initial) != 1 || ctx.initial[0].Types == nil || ctx.initial[0].Types.Name() != "main" ||
		conf.Goarch != "wasm" || conf.BuildMode != BuildModeExe || ctx.mode == ModeGen || conf.Coverage != nil ||
		conf.OptLevel == optlevel.O0 || ctx.prog.Target().WasmProvider != "wasi" ||
		conf.Target != "" && conf.Target != "wasi" || len(ctx.crossCompile.ExtraFiles) != 0 ||
		ctx.frontendOptions.DebuggerFrames || needsWasmRuntimeScheduler(ctx) ||
		os.Getenv("LDFLAGS") != "" || ctx.commands.lookup("LDFLAGS") != "" || conf.LinkOptions.ExternalLinkerFlags != "" {
		return
	}
	linkname := func(fn *ssa.Function) string {
		if fn.Pkg == nil {
			return ""
		}
		name, _ := ctx.prog.Linkname(llssa.FullName(fn.Pkg.Pkg, fn.Name()))
		return name
	}
	globalSafe := func(global *ssa.Global) bool {
		name := llssa.FullName(global.Pkg.Pkg, global.Name())
		if _, linked := ctx.prog.Linkname(name); linked {
			return false
		}
		_, locality, ok, err := ctx.prog.ResolveLocalityFor(global.Pkg.Pkg, name)
		return err == nil && (!ok || locality.Locality == llssa.LocalityNone)
	}
	// Check reachable globals even inside the trusted runtime tree. Most
	// programs also fail this proof without needing any filesystem checks.
	if !wasmRuntimeFreeFunctions(wasmReflectRoots(ctx), linkname, globalSafe) {
		return
	}
	// Foreign objects and exported entries can hide calls back into Go.
	runtimeDir, err := filepath.EvalSymlinks(env.LLGoRuntimeDir())
	if err != nil {
		return
	}
	resolvedDirs := make(map[string]string)
	for _, pkg := range ctx.pkgByID {
		dir, resolved := resolvedDirs[pkg.Dir]
		if !resolved {
			dir, err = filepath.EvalSymlinks(pkg.Dir)
			if err != nil {
				return
			}
			resolvedDirs[pkg.Dir] = dir
		}
		if rel, err := filepath.Rel(runtimeDir, dir); err == nil && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if len(pkg.OtherFiles) != 0 || len(pkg.EmbedFiles) != 0 {
			return
		}
		for _, file := range pkg.Syntax {
			for _, imp := range file.Imports {
				if imp.Path.Value == `"C"` || imp.Path.Value == "`C`" {
					return
				}
			}
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
					if symbol, linked := ctx.prog.Linkname(llssa.FullName(pkg.Types, fn.Name.Name)); linked && !strings.HasPrefix(symbol, "llgo.") {
						return
					}
				}
			}
			for _, group := range file.Comments {
				for _, comment := range group.List {
					fields := strings.Fields(strings.TrimPrefix(comment.Text, "//"))
					if len(fields) > 0 && (fields[0] == "export" || fields[0] == "go:wasmexport" || fields[0] == "llgo:export") {
						return
					}
				}
			}
		}
		if flags, imports := collectGoCgoPragmas(pkg.Syntax); len(flags) != 0 || len(imports) != 0 {
			return
		}
		for _, name := range []string{"LLGoFiles", "LLGoPackage"} {
			if value, ok := pkg.Types.Scope().Lookup(name).(*types.Const); ok && value.Val().Kind() == constant.String {
				text := constant.StringVal(value.Val())
				if name == "LLGoFiles" && text != "" || name == "LLGoPackage" && strings.HasPrefix(text, "link") {
					return
				}
			}
		}
		for _, name := range pkg.Types.Scope().Names() {
			if _, variable := pkg.Types.Scope().Lookup(name).(*types.Var); !variable {
				continue
			}
			if _, locality, ok, err := ctx.prog.ResolveLocalityFor(pkg.Types, llssa.FullName(pkg.Types, name)); err != nil || ok && locality.Locality != llssa.LocalityNone {
				return
			}
		}
		if _, err := os.Stat(filepath.Join(pkg.Dir, "_wrap")); err == nil || !os.IsNotExist(err) {
			return
		}
	}
	ctx.wasmRuntimeFree = true
	ctx.frontendOptions.ShadowStack = false
	ctx.prog.EnableGCRoots(false)
	ctx.prog.EnableCooperativeSafepoints(false)
	if conf.Verbose || conf.PrintCommands {
		fmt.Fprintln(os.Stderr, "llgo: runtime startup omitted: proven allocation-free WASI program")
	}
}

func wasmRuntimeFreeFunctions(roots []*ssa.Function, linkname func(*ssa.Function) string, globalSafe func(*ssa.Global) bool) bool {
	if len(roots) == 0 {
		return false
	}
	state := make(map[*ssa.Function]uint8)
	var visit func(*ssa.Function) bool
	visit = func(fn *ssa.Function) bool {
		if fn == nil || state[fn] == 1 {
			return false
		}
		if state[fn] == 2 {
			return true
		}
		if len(fn.Blocks) == 0 || len(fn.FreeVars) != 0 {
			return false
		}
		state[fn] = 1
		for _, block := range fn.Blocks {
			for _, instruction := range block.Instrs {
				switch instr := instruction.(type) {
				case *ssa.DebugRef, *ssa.Return, *ssa.Jump, *ssa.If, *ssa.Phi:
				case *ssa.UnOp:
					if instr.Op == token.MUL {
						if global, ok := instr.X.(*ssa.Global); !ok || !globalSafe(global) || !runtimeFreeScalar(instr.Type()) {
							return false
						}
					} else if instr.Op != token.NOT && instr.Op != token.SUB && instr.Op != token.XOR {
						return false
					}
				case *ssa.Store:
					if global, ok := instr.Addr.(*ssa.Global); !ok || !globalSafe(global) || !runtimeFreeScalar(instr.Val.Type()) {
						return false
					}
				case *ssa.BinOp:
					basic, ok := instr.X.Type().Underlying().(*types.Basic)
					if !ok || basic.Info()&(types.IsInteger|types.IsBoolean) == 0 {
						return false
					}
					switch instr.Op {
					case token.QUO, token.REM, token.SHL, token.SHR:
						return false
					}
				case *ssa.ChangeType:
					if !runtimeFreeScalar(instr.Type()) || !runtimeFreeScalar(instr.X.Type()) {
						return false
					}
				case *ssa.Call:
					call := instr.Common()
					if builtin, ok := call.Value.(*ssa.Builtin); ok {
						if builtin.Name() != "print" && builtin.Name() != "println" {
							return false
						}
						for _, arg := range call.Args {
							if !runtimeFreeScalar(arg.Type()) {
								return false
							}
						}
						continue
					}
					callee := call.StaticCallee()
					if callee == nil {
						return false
					}
					if symbol := linkname(callee); symbol != "" {
						if len(callee.Blocks) != 0 {
							return false
						}
						switch symbol {
						case "llgo.cstr":
							if len(call.Args) != 1 {
								return false
							}
							arg, ok := call.Args[0].(*ssa.Const)
							if !ok || arg.Value == nil || arg.Value.Kind() != constant.String {
								return false
							}
						case "C.printf":
							if len(call.Args) != 2 {
								return false
							}
							args, ok := call.Args[1].(*ssa.Const)
							if !ok || !args.IsNil() {
								return false
							}
						default:
							return false
						}
					} else if !visit(callee) {
						return false
					}
				default:
					return false
				}
			}
		}
		state[fn] = 2
		return true
	}
	for _, root := range roots {
		if !visit(root) {
			return false
		}
	}
	return true
}

func runtimeFreeScalar(typ types.Type) bool {
	basic, ok := typ.Underlying().(*types.Basic)
	return ok && basic.Info()&(types.IsInteger|types.IsBoolean|types.IsString) != 0
}

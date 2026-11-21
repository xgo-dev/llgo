package build

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"

	"github.com/xgo-dev/llgo/cl"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
)

const wasiPathStateMember = "llgoWASIPathStateNeeded"
const wasiPathStateGuard = "syscall." + wasiPathStateMember

func configureWASISyscallState(ctx *context, p *ssa.Package, roots []*ssa.Function, omitted []cl.OptionalInitializer, isSymbolRoot func(string) bool) {
	// Recheck after stdout replacement and initializer pruning. Methods kept
	// for possible reflection do not necessarily have a call from a root.
	restore := omitInitializerEdges(omitted)
	defer restore()
	mayReflect := ctx.prog.Target().WasmReflectBridges
	result := rta.Analyze(roots, !mayReflect)
	pathInit := configureWASIPathState(ctx, p, roots, result, mayReflect, isSymbolRoot)
	if !mayReflect {
		configureWASIEnvironmentSnapshot(ctx, p, roots, result, pathInit, isSymbolRoot)
	}
}

func configureWASIPathState(ctx *context, p *ssa.Package, roots []*ssa.Function, result *rta.Result, mayReflect bool, isSymbolRoot func(string) bool) *ssa.Function {
	guard, _ := p.Members[wasiPathStateMember].(*ssa.Global)
	cwd, _ := p.Members["cwd"].(*ssa.Global)
	preopens, _ := p.Members["preopens"].(*ssa.Global)
	if guard == nil || cwd == nil || preopens == nil {
		return nil
	}
	init := wasiPathStateInitializer(p, guard)
	if init == nil {
		return nil
	}
	for _, name := range []string{wasiPathStateGuard, "syscall." + init.Name(), "syscall.cwd", "syscall.preopens"} {
		if isSymbolRoot(name) {
			return nil
		}
	}
	for fn := range result.Reachable {
		if fn == init {
			continue
		}
		for _, block := range fn.Blocks {
			for _, inst := range block.Instrs {
				if _, debug := inst.(*ssa.DebugRef); debug {
					continue
				}
				for _, operand := range inst.Operands(nil) {
					if (*operand == cwd || *operand == preopens) &&
						(mayReflect || hasCallPath(result.CallGraph, roots, fn, nil)) {
						return nil
					}
				}
			}
		}
	}
	ctx.initializerGuardsOff = append(ctx.initializerGuardsOff, wasiPathStateGuard)
	if ctx.buildConf.Verbose || ctx.buildConf.PrintCommands {
		fmt.Fprintln(os.Stderr, "llgo: omit unobserved WASI path state")
	}
	return init
}

func wasiPathStateInitializer(p *ssa.Package, guard *ssa.Global) *ssa.Function {
	for _, member := range p.Members {
		fn, ok := member.(*ssa.Function)
		if !ok || !strings.HasPrefix(fn.Name(), "init#") {
			continue
		}
		for _, block := range fn.Blocks {
			for _, inst := range block.Instrs {
				if load, ok := inst.(*ssa.UnOp); ok && load.Op == token.MUL && load.X == guard {
					return fn
				}
			}
		}
	}
	return nil
}

// Match the two state updates in the standard-library WASI directory scan.
// The original discovery calls and their error handling stay outside the guard.
func rewriteWASIPathState(source []byte) []byte {
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "fs_wasip1.go", source, parser.SkipObjectResolution)
	if err != nil || file.Name.Name != "syscall" {
		return nil
	}
	offset := files.File(file.Pos()).Offset
	matches := func(stmt ast.Stmt, expected string) bool {
		return strings.Join(strings.Fields(string(source[offset(stmt.Pos()):offset(stmt.End())])), " ") ==
			strings.Join(strings.Fields(expected), " ")
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "init" || fn.Recv != nil || fn.Body == nil || len(fn.Body.List) != 3 {
			continue
		}
		scan, ok := fn.Body.List[1].(*ast.ForStmt)
		if !ok || len(scan.Body.List) == 0 {
			continue
		}
		saveMount := scan.Body.List[len(scan.Body.List)-1]
		saveCwd := fn.Body.List[2]
		if !matches(saveMount, wasiSaveMount) || !matches(saveCwd, wasiSaveCwd) {
			continue
		}
		// Only the guarded statements may access the state from this function.
		otherAccess := false
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if node == saveMount || node == saveCwd {
				return false
			}
			if id, ok := node.(*ast.Ident); ok && (id.Name == "cwd" || id.Name == "preopens") {
				otherAccess = true
			}
			return !otherAccess
		})
		if otherAccess {
			return nil
		}
		mountStart, mountEnd := offset(saveMount.Pos()), offset(saveMount.End())
		cwdStart, cwdEnd := offset(saveCwd.Pos()), offset(saveCwd.End())
		// Leave the syscall and panic on their original source lines.
		const begin = "if " + wasiPathStateMember + " { "
		const end = " }"
		const declaration = "\nvar " + wasiPathStateMember + " bool\n"
		return slices.Concat(source[:mountStart], []byte(begin), source[mountStart:mountEnd], []byte(end),
			source[mountEnd:cwdStart], []byte(begin), source[cwdStart:cwdEnd], []byte(end),
			source[cwdEnd:], []byte(declaration))
	}
	return nil
}

const wasiSaveMount = `preopens = append(preopens, opendir{
	fd: preopenFd,
	name: string(dirNameBuf[:prestat.dir.prNameLen]),
})`

const wasiSaveCwd = `if cwd, _ = Getenv("PWD"); cwd != "" {
	cwd = joinPath("/", cwd)
} else if len(preopens) > 0 {
	cwd = preopens[0].name
}`

package build

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"

	"github.com/xgo-dev/llgo/internal/packages"
	asm "github.com/xgo-dev/plan9asm"
)

// These SDK declarations are code-address placeholders for libc calls, not
// ordinary Go ABI0 callees. FuncPCABI0 lowering already resolves their native
// entry, and compileCgoImportDynamicAliases supplies the import aliases.
// Check every body and its selected Go declaration before omitting assembly;
// changed source must still reach the translator and its normal diagnostics.
func darwinStdlibImportTrampolines(ctx *context, pkg *packages.Package, path string, source []byte) bool {
	if ctx.buildConf.Goos != "darwin" || pkg.PkgPath != "internal/syscall/unix" ||
		filepath.Base(path) != "asm_darwin.s" || pkg.Types == nil {
		return false
	}
	_, imports := collectGoCgoPragmas(pkg.Syntax)
	dynamic := make(map[string]bool, len(imports))
	for _, imp := range imports {
		dynamic[imp.local] = imp.local != "" && imp.alias != ""
	}
	bodyless := make(map[string]bool)
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body == nil && fn.Recv == nil {
				bodyless[fn.Name.Name] = true
			}
		}
	}
	file, err := asm.Parse(asm.Arch(ctx.buildConf.Goarch), string(source))
	if err != nil || len(file.Funcs) == 0 || len(file.Data) != 0 || len(file.Globl) != 0 || len(file.UnlinkedPrelude) != 0 {
		return false
	}
	for _, fn := range file.Funcs {
		name, local := strings.TrimPrefix(fn.Sym, "·"), ""
		if fn.Sym != "·"+name || !strings.HasSuffix(name, "_trampoline") {
			return false
		}
		local = strings.TrimSuffix(name, "_trampoline")
		decl, ok := pkg.Types.Scope().Lookup(name).(*types.Func)
		if !ok || !bodyless[name] || !dynamic[local] {
			return false
		}
		sig := decl.Type().(*types.Signature)
		if sig.Recv() != nil || sig.Params().Len() != 0 || sig.Results().Len() != 0 || sig.Variadic() ||
			fn.FrameSize != 0 || fn.ArgSize != 0 || len(fn.Instrs) != 2 {
			return false
		}
		flags := strings.Split(fn.Instrs[0].Raw, ",")
		if len(flags) != 3 || (strings.TrimSpace(flags[1]) != "NOSPLIT" && strings.TrimSpace(flags[1]) != "4") {
			return false
		}
		jump := fn.Instrs[1]
		if jump.Op != "JMP" || len(jump.Args) != 1 || jump.Args[0].String() != local+"(SB)" {
			return false
		}
	}
	return true
}

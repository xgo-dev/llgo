package build

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/packages"
	"github.com/xgo-dev/llgo/ssa/abi"
	llvm "github.com/xgo-dev/llvm"
	extplan9asm "github.com/xgo-dev/plan9asm"
)

// translateForeignNativeAsm returns an owned standalone module for the shared
// Plan 9 .ll compilation path. Native entries retain their register-level ABI.
func translateForeignNativeAsm(ctx *context, aPkg *aPackage, pkg *packages.Package, src []byte) (llvm.Module, bool, error) {
	opts, native, err := nativeAsmOptions(ctx, pkg, src)
	if !native || err != nil {
		return llvm.Module{}, native, err
	}
	goMod := aPkg.LPkg.Module()
	mod, err := extplan9asm.TranslateNativeModule(goMod.Context(), src, opts)
	if err != nil {
		return llvm.Module{}, true, err
	}
	mod.SetTarget(ctx.prog.Target().Spec().Triple)
	mod.SetDataLayout(ctx.prog.DataLayout())
	return mod, true, nil
}

// foreignNativeImports is shared by compilation and signature discovery. Its
// input has already been preprocessed under the selected package/target.
func foreignNativeImports(ctx *context, pkg *packages.Package, src []byte) (map[string]string, bool, error) {
	if !extplan9asm.SupportsNativeTarget(ctx.buildConf.Goos, ctx.buildConf.Goarch) {
		return nil, false, nil
	}
	_, decls := collectGoCgoPragmas(pkg.Syntax)
	if len(decls) == 0 {
		return nil, false, nil
	}
	funcs := extplan9asm.ForeignNativeFunctions(src, ctx.buildConf.Goarch)
	if len(funcs) == 0 {
		return nil, false, nil
	}
	imports := make(map[string]string)
	for _, d := range decls {
		if prev, ok := imports[d.local]; ok && prev != d.alias {
			return nil, true, fmt.Errorf("conflicting dynamic import %s", d.local)
		}
		imports[d.local] = d.alias
	}
	return imports, true, nil
}

// nativeAsmOptions separates source-owned address entries from callable Go
// declarations. Native instruction/frame validation remains in the shared
// backend; neither DATA nor this routing decision supplies a C prototype.
func nativeAsmOptions(ctx *context, pkg *packages.Package, src []byte) (extplan9asm.NativeOptions, bool, error) {
	opts := extplan9asm.NativeOptions{
		GOOS: ctx.buildConf.Goos, GOARCH: ctx.buildConf.Goarch,
		PackagePath: abi.PathOf(pkg.Types),
	}
	imports, native, err := foreignNativeImports(ctx, pkg, src)
	if native || err != nil {
		opts.Imports = imports
		return opts, native, err
	}
	if !extplan9asm.SupportsNativeTarget(opts.GOOS, opts.GOARCH) {
		return opts, false, nil
	}
	opts.CodeEntries = nativeAddressEntries(pkg, src, opts.GOARCH)
	if len(opts.CodeEntries) == 0 {
		return opts, false, nil
	}
	_, _, err = extplan9asm.TranslateNativeSource(src, opts)
	return opts, true, err
}

// nativeAddressEntries recognizes only whole-file DATA address ownership. It
// consumes the selected, preprocessed source and actual package types, never a
// directory glob, function-name convention, or guessed missing declaration.
func nativeAddressEntries(pkg *packages.Package, src []byte, arch string) map[string]string {
	if pkg.Types == nil || !strings.Contains(string(src), "DATA") {
		return nil
	}
	f, err := extplan9asm.Parse(extplan9asm.Arch(arch), string(src))
	if err != nil || len(f.Funcs) == 0 || len(f.Globl) == 0 || len(f.Data) != len(f.Globl) {
		return nil
	}
	entries := make(map[string]string, len(f.Funcs))
	objects := make(map[string]bool, len(f.Funcs))
	for _, fn := range f.Funcs {
		if !token.IsIdentifier(fn.Sym) || fn.Sym == "_" || pkg.Types.Scope().Lookup(fn.Sym) != nil {
			return nil
		}
		identity := sha256.Sum256([]byte(pkg.Types.Path() + "\x00" + fn.Sym))
		object := fmt.Sprintf("__llgo_native_address_%x", identity)
		entries[fn.Sym], objects[object] = object, true
	}
	owners := make(map[string]bool, len(f.Globl))
	for _, global := range f.Globl {
		name := strings.TrimPrefix(global.Sym, "·")
		variable, ok := pkg.Types.Scope().Lookup(name).(*types.Var)
		if !strings.HasPrefix(global.Sym, "·") || !ok || !token.IsIdentifier(name) || global.Size != 8 || owners[name] {
			return nil
		}
		basic, ok := variable.Type().Underlying().(*types.Basic)
		if !ok || basic.Kind() != types.Uintptr {
			return nil
		}
		owners[name] = true
	}
	uninitialized := make(map[string]bool, len(owners))
	for _, syntax := range pkg.Syntax {
		for _, declaration := range syntax.Decls {
			group, ok := declaration.(*ast.GenDecl)
			if !ok || group.Tok != token.VAR {
				continue
			}
			for _, spec := range group.Specs {
				value := spec.(*ast.ValueSpec)
				if len(value.Values) != 0 {
					continue
				}
				for _, name := range value.Names {
					variable, ok := pkg.Types.Scope().Lookup(name.Name).(*types.Var)
					if ok && owners[name.Name] && variable.Pos() == name.Pos() {
						uninitialized[name.Name] = true
					}
				}
			}
		}
		for _, group := range syntax.Comments {
			for _, item := range directive.ParseGroup(group) {
				if item.Name != "go:linkname" && item.Name != "llgo:link" && item.Name != "go:cgo_import_dynamic" {
					continue
				}
				fields := strings.Fields(item.Args)
				if len(fields) > 0 && owners[fields[0]] || len(fields) > 1 && objects[fields[1]] {
					return nil
				}
			}
		}
	}
	if len(uninitialized) != len(owners) {
		return nil
	}
	used := make(map[string]bool, len(entries))
	for _, data := range f.Data {
		name := strings.TrimPrefix(data.Sym, "·")
		entry := strings.TrimSuffix(data.Addr, "(SB)")
		if !strings.HasPrefix(data.Sym, "·") || !owners[name] || data.Off != 0 || data.Width != 8 {
			return nil
		}
		if data.Addr != entry+"(SB)" || entries[entry] == "" {
			return nil
		}
		delete(owners, name)
		used[entry] = true
	}
	if len(owners) != 0 || len(used) != len(entries) {
		return nil
	}
	return entries
}

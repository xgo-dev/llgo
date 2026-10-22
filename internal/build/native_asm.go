package build

import (
	"fmt"

	"github.com/xgo-dev/llgo/internal/packages"
	"github.com/xgo-dev/llgo/ssa/abi"
	llvm "github.com/xgo-dev/llvm"
	extplan9asm "github.com/xgo-dev/plan9asm"
)

// translateForeignNativeAsm returns an owned standalone module for the shared
// Plan 9 .ll compilation path. Native entries retain their register-level ABI.
func translateForeignNativeAsm(ctx *context, aPkg *aPackage, pkg *packages.Package, src []byte) (llvm.Module, bool, error) {
	imports, native, err := foreignNativeImports(ctx, pkg, src)
	if !native || err != nil {
		return llvm.Module{}, native, err
	}
	pkgPath := abi.PathOf(pkg.Types)
	goMod := aPkg.LPkg.Module()
	mod, err := extplan9asm.TranslateNativeModule(goMod.Context(), src, extplan9asm.NativeOptions{GOOS: ctx.buildConf.Goos, GOARCH: ctx.buildConf.Goarch, PackagePath: pkgPath, Imports: imports})
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

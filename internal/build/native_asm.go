package build

import (
	"fmt"

	"github.com/xgo-dev/llgo/internal/llvmgen/abi"
	"github.com/xgo-dev/llgo/internal/packages"
	llvm "github.com/xgo-dev/llvm"
	extplan9asm "github.com/xgo-dev/plan9asm"
)

// translateForeignNativeAsm returns an owned standalone module for the shared
// Plan 9 .ll compilation path. Native entries retain their register-level ABI.
func translateForeignNativeAsm(ctx *context, aPkg *aPackage, pkg *packages.Package, src []byte) (llvm.Module, bool, error) {
	if !extplan9asm.SupportsNativeTarget(ctx.buildConf.Goos, ctx.buildConf.Goarch) {
		return llvm.Module{}, false, nil
	}
	_, decls := collectGoCgoPragmas(pkg.Syntax)
	if len(decls) == 0 {
		return llvm.Module{}, false, nil
	}
	funcs := extplan9asm.ForeignNativeFunctions(src, ctx.buildConf.Goarch)
	if len(funcs) == 0 {
		return llvm.Module{}, false, nil
	}
	imports := make(map[string]string)
	for _, d := range decls {
		if prev, ok := imports[d.local]; ok && prev != d.alias {
			return llvm.Module{}, true, fmt.Errorf("conflicting dynamic import %s", d.local)
		}
		imports[d.local] = d.alias
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

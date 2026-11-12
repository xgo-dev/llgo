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

// nativeAddressEntries is the single-source view used by source-role tests.
// Compilation and signatures supply the complete selected source component.
func nativeAddressEntries(pkg *packages.Package, src []byte, arch string) map[string]string {
	ctx := &context{buildConf: &Config{Goos: "linux", Goarch: arch}}
	plan, err := planNativeAsmComponents(ctx, pkg, []plan9AsmSource{{source: src}})
	if err != nil || plan == nil {
		return nil
	}
	return plan.files[""].CodeEntries
}

// A source-proven byte alias observes the actual code symbol, not a byte-sized
// DATA definition. Keep the Go reference but let the native object own storage.
func externalizeNativeCodeAliases(goMod, nativeMod llvm.Module, plan *nativeAsmPlan) error {
	for _, object := range plan.aliases {
		if nativeMod.NamedFunction(object).IsNil() {
			continue
		}
		global := goMod.NamedGlobal(object)
		if global.IsNil() || global.IsDeclaration() {
			continue
		}
		if global.IsThreadLocal() || global.GlobalValueType().TypeKind() != llvm.IntegerTypeKind || global.GlobalValueType().IntTypeWidth() != 8 || !global.Initializer().IsNull() {
			return fmt.Errorf("native code alias %s conflicts with Go storage", object)
		}
		global.SetInitializer(llvm.Value{})
		global.SetLinkage(llvm.ExternalLinkage)
		global.SetGlobalConstant(false)
	}
	return nil
}

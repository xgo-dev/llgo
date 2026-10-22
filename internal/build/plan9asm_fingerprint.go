package build

import (
	"fmt"
	"slices"

	"github.com/xgo-dev/llgo/internal/packages"
	llplan9asm "github.com/xgo-dev/llgo/internal/plan9asm"
	llruntime "github.com/xgo-dev/llgo/runtime"
)

// The expanded source binds headers (including SDK and generated go_asm.h),
// overlay contents and active conditionals to the same input used to compile.
// File size/mtime alone cannot identify those transitive inputs.
type plan9AsmDigest struct {
	Path       string `yaml:"path"`
	SourceHash string `yaml:"source_hash"`
}

func digestPackagePlan9Asm(ctx *context, pkg *aPackage, files []string) ([]plan9AsmDigest, error) {
	var digests []plan9AsmDigest
	var err error
	// Match compilePackageModule: a replacing alt package does not compile
	// the original package's assembly; an additive one compiles both.
	if pkg.AltPkg == nil || llruntime.HasAdditiveAltPkg(pkg.PkgPath) {
		digests, err = digestPlan9AsmFiles(ctx, pkg.Package, files)
		if err != nil {
			return nil, err
		}
	}
	if pkg.AltPkg != nil {
		alt := pkg.AltPkg.Package
		altFiles, err := pkgSFiles(ctx, alt)
		if err != nil {
			return nil, err
		}
		altDigests, err := digestPlan9AsmFiles(ctx, alt, altFiles)
		if err != nil {
			return nil, err
		}
		digests = append(digests, altDigests...)
	}
	return digests, nil
}

func digestPlan9AsmFiles(ctx *context, pkg *packages.Package, files []string) ([]plan9AsmDigest, error) {
	if len(files) == 0 {
		return nil, nil
	}
	if len(ctx.patchFiles[pkg.PkgPath]) != 0 && llruntime.SourcePatchReplacesAsmForGOARCH(pkg.PkgPath, ctx.buildConf.Goarch) {
		return nil, nil
	}
	if !ctx.plan9asmEnabled(pkg.PkgPath) && llruntime.HasAltPkg(pkg.PkgPath) {
		return nil, nil
	}
	files = slices.Clone(files)
	slices.Sort(files)
	files = slices.Compact(files)
	digests := make([]plan9AsmDigest, 0, len(files))
	opt := plan9asmTranslateOptions(ctx)
	for _, file := range files {
		src, err := llplan9asm.PreprocessFileForPkg(pkg, file, ctx.buildConf.Goos, ctx.buildConf.Goarch, opt)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		digests = append(digests, plan9AsmDigest{Path: file, SourceHash: digestBytes(src)})
	}
	return digests, nil
}

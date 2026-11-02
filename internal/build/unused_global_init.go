/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package build

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	llplan9asm "github.com/xgo-dev/llgo/internal/plan9asm"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/xtool/safesplit"
	gllvm "github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
)

func configureUnusedGlobalInits(ctx *context) {
	ctx.wasmClosedProgram = false
	ctx.frontendOptions.UnusedGlobalInits = nil
	if ctx.buildConf.BuildMode != BuildModeExe || ctx.mode == ModeGen || ctx.buildConf.Goarch != "wasm" {
		return
	}
	if ctx.buildConf.OptLevel == optlevel.O0 {
		return
	}
	exported := make(map[string]bool)
	if pkg := opaqueGlobalInitPackage(ctx, exported); pkg != "" {
		if ctx.buildConf.Verbose || ctx.buildConf.PrintCommands {
			fmt.Fprintf(os.Stderr, "llgo: retain global initializers: native inputs in %s\n", pkg)
		}
		return
	}
	// Both pruning passes require this proof, even when no initializer plan
	// is produced. Its representation is independent of the initializer plan.
	ctx.wasmClosedProgram = true
	// A one-name go:linkname is an export annotation, with no alias entry in
	// Program. Preserve those declarations as well as two-name link aliases.
	inspect := func(pkg *types.Package, files []*ast.File) {
		for _, file := range files {
			for _, group := range file.Comments {
				for _, comment := range group.List {
					fields := strings.Fields(comment.Text)
					if len(fields) >= 2 && fields[0] == "//go:linkname" {
						exported[llssa.FullName(pkg, fields[1])] = true
					}
				}
			}
		}
	}
	for _, pkg := range ctx.pkgByID {
		inspect(pkg.Types, pkg.Syntax)
		if pkg.AltPkg != nil {
			inspect(pkg.AltPkg.Types, pkg.AltPkg.Syntax)
		}
		for name := range pkg.rewriteVars {
			exported[llssa.FullName(pkg.Types, name)] = true
		}
	}

	ctx.frontendOptions.UnusedGlobalInits = cl.FindUnusedGlobalInits(ctx.progSSA, func(global *ssa.Global) bool {
		name := llssa.FullName(global.Pkg.Pkg, global.Name())
		if exported[name] || global.Pkg.Pkg.Name() == "C" {
			return true
		}
		if _, alias := ctx.prog.Linkname(name); alias || ctx.prog.HasLinknameTarget(name) {
			return true
		}
		if _, locality, ok, err := ctx.prog.ResolveLocalityFor(global.Pkg.Pkg, name); err != nil || ok && locality.Locality != llssa.LocalityNone {
			return true
		}
		return false
	})
	if ctx.buildConf.Verbose || ctx.buildConf.PrintCommands {
		var count int
		for _, pkg := range ctx.progSSA.AllPackages() {
			count += len(ctx.frontendOptions.UnusedGlobalInits.Names(pkg))
		}
		fmt.Fprintf(os.Stderr, "llgo: prune %d unused global initializers\n", count)
	}
}

// Native objects can reference Go globals without a Go SSA operand. Runtime
// providers use declared linknames; user-supplied objects require a future
// linker-summary join, so retain all initializers in their presence.
func opaqueGlobalInitPackage(ctx *context, exported map[string]bool) string {
	// Arbitrary linker arguments can add objects or force exported symbols.
	// Permit output-only diagnostics and backend caches used by LTO/size jobs.
	for _, flags := range []string{os.Getenv("LDFLAGS"), ctx.buildConf.LinkOptions.ExternalLinkerFlags} {
		for _, flag := range safesplit.SplitPkgConfigFlags(flags) {
			// Clang splits -Wl on every comma; a diagnostic option can be
			// followed by an export, forced root or native object.
			if strings.Count(flag, ",") != 1 || flag != "-Wl,--save-temps" && flag != "-Wl,--print-map" &&
				!strings.HasPrefix(flag, "-Wl,--lto-obj-path=") && !strings.HasPrefix(flag, "-Wl,--thinlto-cache-dir=") &&
				!strings.HasPrefix(flag, "-Wl,-Map=") {
				return "linker flags"
			}
		}
	}
	root := env.LLGoROOT()
	if real, err := filepath.EvalSymlinks(root); err == nil {
		root = real
	}
	trusted := func(pkg *packages.Package) bool {
		dirPath := pkg.Dir
		if real, err := filepath.EvalSymlinks(dirPath); err == nil {
			dirPath = real
		}
		for _, dir := range []string{"runtime", "lib"} {
			rel, err := filepath.Rel(filepath.Join(root, dir), dirPath)
			if err == nil && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return true
			}
		}
		return false
	}
	for _, entry := range ctx.pkgByID {
		pkg := entry.Package
		if trusted(pkg) {
			continue
		}
		for _, path := range pkg.OtherFiles {
			// Headers are consumed by the selected assembly/native inputs;
			// they are not independently linked objects.
			if ext := filepath.Ext(path); ext != ".s" && ext != ".S" && ext != ".h" {
				return pkg.PkgPath
			}
		}
		if !collectAsmGlobalInitReferences(ctx, pkg, exported) {
			return pkg.PkgPath
		}
		if pkg.Types != nil {
			for _, name := range []string{"LLGoFiles", "LLGoPackage"} {
				if value, ok := pkg.Types.Scope().Lookup(name).(*types.Const); ok && value.Val().Kind() == constant.String {
					text := constant.StringVal(value.Val())
					if name == "LLGoFiles" && text != "" || name == "LLGoPackage" && strings.HasPrefix(text, "link") {
						return pkg.PkgPath
					}
				}
			}
		}
		for _, file := range pkg.Syntax {
			for _, imp := range file.Imports {
				if imp.Path.Value == `"C"` || imp.Path.Value == "`C`" {
					return pkg.PkgPath
				}
			}
		}
		if flags, imports := collectGoCgoPragmas(pkg.Syntax); len(flags) != 0 || len(imports) != 0 {
			return pkg.PkgPath
		}
	}
	return ""
}

// Translated assembly exposes its global references in LLVM IR, including
// cross-package references. Unsupported or opaque inputs retain all Go inits.
func collectAsmGlobalInitReferences(ctx *context, pkg *packages.Package, exported map[string]bool) bool {
	files, err := compiledPkgSFiles(ctx, pkg)
	if err != nil {
		return false
	}
	for _, path := range files {
		src, err := llplan9asm.ReadFileWithOverlay(ctx.buildConf.Overlay, path)
		if err != nil {
			return false
		}
		tr, err := llplan9asm.TranslateSourceModuleForPkgWithOptions(pkg, path, src, ctx.buildConf.Goos, ctx.buildConf.Goarch, plan9asmTranslateOptions(ctx.buildConf))
		if err != nil {
			if strings.Contains(err.Error(), "no TEXT directive found") {
				continue
			}
			return false
		}
		for global := tr.Module.FirstGlobal(); !global.IsNil(); global = gllvm.NextGlobal(global) {
			exported[global.Name()] = true
		}
		tr.Module.Dispose()
	}
	return true
}

// canPruneWasmTypeMetadata shares the closed-program boundary with initializer
// pruning while retaining the metadata pass's executable/target restrictions.
func canPruneWasmTypeMetadata(ctx *context) bool {
	return ctx.buildConf.Goarch == "wasm" && ctx.buildConf.BuildMode == BuildModeExe && ctx.wasmClosedProgram
}

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
	"os"

	"github.com/xgo-dev/llgo/analysis/inline"
	"github.com/xgo-dev/llgo/cmd/cl"
	"github.com/xgo-dev/llgo/exportdata"
	"github.com/xgo-dev/llgo/target"
	lltypes "github.com/xgo-dev/llgo/types"
)

const (
	llgoDevIncremental = "LLGO_DEV_INCREMENTAL"
	cacheTypesExt      = ".llgotypes"
)

func devIncrementalModeEnabled() bool {
	return isEnvOn(llgoDevIncremental, false)
}

func (c *context) targetSpec() target.Spec {
	var spec target.Spec
	if c.buildConf != nil {
		spec = target.Spec{
			GOOS:         c.buildConf.Goos,
			GOARCH:       c.buildConf.Goarch,
			GO386:        c.buildConf.GO386,
			GOAMD64:      c.buildConf.GOAMD64,
			GOARM:        c.buildConf.GOARM,
			GOARM64:      c.buildConf.GOARM64,
			Name:         c.buildConf.Target,
			LLVMTarget:   c.crossCompile.LLVMTarget,
			WasmProfile:  string(c.crossCompile.WasmProfile),
			WasmProvider: string(c.crossCompile.WasmProvider),
			TargetABI:    c.crossCompile.TargetABI,
		}
	}
	return target.ResolveSpec(spec)
}

// writeExportMetaSidecar lowers pkg.SSA to pure-Go llgo/ssa IR, runs pure-Go
// callgraph/escape/inline analysis using any already-loaded dependency
// summaries, and persists the resulting compact binary exportdata.PackageMeta
// alongside the cached .a archive.
func (c *context) writeExportMetaSidecar(path string, pkg *aPackage) error {
	if pkg == nil || pkg.SSA == nil || pkg.Package == nil {
		return nil
	}
	meta := pkg.exportMeta
	if meta == nil {
		importedEsc := make(map[string]*exportdata.EscapeSummary)
		for _, dep := range pkg.Imports {
			if dep == nil {
				continue
			}
			if depPkg := c.pkgByID[dep.ID]; depPkg != nil && depPkg.exportMeta != nil {
				importedEsc[depPkg.PkgPath] = &depPkg.exportMeta.Escape
			}
		}
		llPkg, err := cl.LowerPackage(c.progSSA, pkg.SSA, c.targetSpec(), nil)
		if err != nil {
			return err
		}
		meta = cl.BuildPackageMeta(llPkg, importedEsc, inline.DefaultBudget)
		pkg.exportMeta = meta
	}
	if meta == nil {
		return nil
	}
	raw, err := exportdata.MarshalMeta(meta)
	if err != nil {
		return err
	}
	return writeFileAtomicBytes(path, raw)
}

func (c *context) readExportMetaSidecar(path string) (*exportdata.PackageMeta, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return exportdata.UnmarshalMeta(raw)
}

// writePackageTypesSidecar encodes a package's effective llgo/types.Package
// into compact binary exportdata — explicitly supporting stdlib packages
// (pkg.Module == nil) and runtime-patched effective scopes.
func (c *context) writePackageTypesSidecar(path string, pkg *aPackage) error {
	if pkg == nil || pkg.Package == nil || pkg.Types == nil {
		return fmt.Errorf("missing types for package export sidecar")
	}
	tctx := lltypes.NewContextWithConfig(lltypes.ContextConfig{Spec: c.targetSpec()})
	llPkg := cl.LowerGoPackageTypes(pkg.Types, tctx)
	raw, err := exportdata.MarshalPackage(llPkg)
	if err != nil {
		return err
	}
	return writeFileAtomicBytes(path, raw)
}

func (c *context) readPackageTypesSidecar(path string, imports map[string]*lltypes.Package) (*lltypes.Package, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return exportdata.UnmarshalPackage(raw, imports)
}

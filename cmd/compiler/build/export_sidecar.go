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
	"bytes"
	"fmt"
	"go/token"
	"os"

	"github.com/xgo-dev/llgo/cmd/compiler/pkgmeta"
	"github.com/xgo-dev/llgo/exportdata"
	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llgo/types"
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

func (c *context) writeExportMetaSidecar(path string, pkg *aPackage) error {
	if pkg == nil || pkg.SSA == nil || pkg.Package == nil {
		return nil
	}
	meta := pkg.getExportMeta()
	if meta == nil {
		importedEsc := make(map[string]*pkgmeta.EscapeSummary)
		for _, dep := range pkg.Imports {
			if dep == nil {
				continue
			}
			depPkg := c.pkgByID[dep.ID]
			if depPkg == nil {
				continue
			}
			em := depPkg.getExportMeta()
			if em == nil && depPkg.CacheHit && depPkg.Fingerprint != "" {
				depPaths := c.ensureCacheManager().PackagePaths(c.targetTriple(), depPkg.PkgPath, depPkg.Fingerprint)
				if loaded, err := c.readExportMetaSidecar(depPaths.Export); err == nil {
					em = loaded
					depPkg.setExportMeta(loaded)
				}
			}
			if em != nil {
				importedEsc[depPkg.PkgPath] = &em.Escape
			}
		}
		meta = pkgmeta.BuildPackageMeta(pkg.SSA, nil, importedEsc, pkgmeta.DefaultBudget)
		pkg.setExportMeta(meta)
	}
	if meta == nil {
		return nil
	}
	raw, err := pkgmeta.Marshal(meta)
	if err != nil {
		return err
	}
	return writeFileAtomicBytes(path, raw)
}

func (c *context) readExportMetaSidecar(path string) (*pkgmeta.PackageMeta, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return pkgmeta.Unmarshal(raw)
}

// writePackageTypesSidecar encodes a package's effective llgo/types.Package
// via exportdata.Write — explicitly supporting stdlib packages (pkg.Module == nil)
// and runtime-patched effective scopes.
func (c *context) writePackageTypesSidecar(path string, pkg *aPackage) error {
	if pkg == nil || pkg.Package == nil || pkg.Types == nil {
		return fmt.Errorf("missing types for package export sidecar")
	}
	var fset *token.FileSet
	if c.progSSA != nil {
		fset = c.progSSA.Fset
	}
	var buf bytes.Buffer
	if err := exportdata.Write(&buf, fset, pkg.Types); err != nil {
		return err
	}
	return writeFileAtomicBytes(path, buf.Bytes())
}

func (c *context) readPackageTypesSidecar(path, pkgPath string, imports map[string]*types.Package) (*types.Package, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fset *token.FileSet
	if c.progSSA != nil {
		fset = c.progSSA.Fset
	}
	return exportdata.Read(bytes.NewReader(raw), fset, imports, pkgPath)
}

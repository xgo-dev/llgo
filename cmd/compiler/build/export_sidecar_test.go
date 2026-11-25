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
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xgo-dev/llgo/analysis/deadcode"
	"github.com/xgo-dev/llgo/exportdata"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	lltypes "github.com/xgo-dev/llgo/types"
	"golang.org/x/tools/go/ssa"
)

func TestExportSidecarMetaAndStdlibTypesRoundTrip(t *testing.T) {
	fset := token.NewFileSet()
	src := `package stdsim

type Box[T any] struct {
	Value T
}

func (b *Box[T]) Get() T { return b.Value }

func Keep(x int) int { return x + 1 }

func Unused() int { return 42 }
`
	f, err := parser.ParseFile(fset, "stdsim.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	info := &gotypes.Info{
		Types:      make(map[ast.Expr]gotypes.TypeAndValue),
		Defs:       make(map[*ast.Ident]gotypes.Object),
		Uses:       make(map[*ast.Ident]gotypes.Object),
		Implicits:  make(map[ast.Node]gotypes.Object),
		Selections: make(map[*ast.SelectorExpr]*gotypes.Selection),
		Scopes:     make(map[ast.Node]*gotypes.Scope),
		Instances:  make(map[*ast.Ident]gotypes.Instance),
	}
	conf := gotypes.Config{Importer: importer.Default()}
	goPkg, err := conf.Check("stdsim", fset, []*ast.File{f}, info)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	prog := ssa.NewProgram(fset, ssa.InstantiateGenerics)
	ssaPkg := prog.CreatePackage(goPkg, []*ast.File{f}, info, true)
	prog.Build()

	// Explicitly verify Module == nil (stdlib / runtime-patched effective scope)
	// works cleanly with both binary PackageMeta and binary llgo/types.Package sidecars.
	pkg := &aPackage{
		Package: &packages.Package{
			ID:      "stdsim",
			PkgPath: "stdsim",
			Name:    "stdsim",
			Module:  nil,
			Types:   goPkg,
		},
		SSA: ssaPkg,
	}
	ctx := &context{
		buildConf: &Config{Goos: "linux", Goarch: "amd64"},
		progSSA:   prog,
		pkgByID:   map[string]Package{"stdsim": pkg},
	}

	dir := t.TempDir()
	metaPath := filepath.Join(dir, "stdsim"+cacheExportExt)
	if err := ctx.writeExportMetaSidecar(metaPath, pkg); err != nil {
		t.Fatalf("writeExportMetaSidecar: %v", err)
	}
	loadedMeta, err := ctx.readExportMetaSidecar(metaPath)
	if err != nil {
		t.Fatalf("readExportMetaSidecar: %v", err)
	}
	if loadedMeta.Summary.Path != "stdsim" {
		t.Fatalf("loaded meta path = %q, want stdsim", loadedMeta.Summary.Path)
	}
	dc := deadcode.Analyze(nil, []*exportdata.PackageMeta{loadedMeta}, nil, []string{"stdsim.Keep"})
	if !slices.Contains(dc.ReachableFuncs, "stdsim.Keep") || !slices.Contains(dc.DeadFuncs, "stdsim.Unused") {
		t.Fatalf("unexpected deadcode result from cached sidecar: reachable=%v dead=%v", dc.ReachableFuncs, dc.DeadFuncs)
	}

	typesPath := filepath.Join(dir, "stdsim"+cacheTypesExt)
	if err := ctx.writePackageTypesSidecar(typesPath, pkg); err != nil {
		t.Fatalf("writePackageTypesSidecar: %v", err)
	}
	decodedPkg, err := ctx.readPackageTypesSidecar(typesPath, make(map[string]*lltypes.Package))
	if err != nil {
		t.Fatalf("readPackageTypesSidecar: %v", err)
	}
	if decodedPkg.Path() != "stdsim" || decodedPkg.Scope().Lookup("Box") == nil || decodedPkg.Scope().Lookup("Keep") == nil {
		t.Fatalf("decoded stdlib-like package missing expected symbols: %v", decodedPkg)
	}
}

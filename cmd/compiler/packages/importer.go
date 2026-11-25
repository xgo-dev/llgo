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

package packages

import (
	"go/ast"
	goimporter "go/importer"
	"go/token"
	gotypes "go/types"
	"sync"

	gogenpkgs "github.com/goplus/gogen/packages"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

// WrapGoImporter adapts a stdlib go/types.Importer into an llgo/types.Importer
// backed by a shared *types.Context, preserving package and object identity
// across repeated imports and downstream types.Config.Check calls.
func WrapGoImporter(imp gotypes.Importer, ctx ...*types.Context) types.Importer {
	var c *types.Context
	if len(ctx) > 0 && ctx[0] != nil {
		c = ctx[0]
	} else {
		c = types.NewContext()
	}
	return &goImporterBridge{
		imp:  imp,
		ctx:  c,
		pkgs: make(map[*gotypes.Package]*types.Package),
	}
}

// DefaultImporter returns an llgo/types.Importer backed by go/importer.Default().
func DefaultImporter() types.Importer {
	return WrapGoImporter(goimporter.Default())
}

// SourceImporter returns an llgo/types.Importer backed by go/importer.For("source", nil).
func SourceImporter() types.Importer {
	return WrapGoImporter(goimporter.For("source", nil))
}

// NewImporter returns an llgo/types.Importer backed by gogen/packages.NewImporter.
func NewImporter(fset *token.FileSet, workDir ...string) types.Importer {
	return WrapGoImporter(gogenpkgs.NewImporter(fset, workDir...))
}

type goImporterBridge struct {
	mu   sync.Mutex
	imp  gotypes.Importer
	ctx  *types.Context
	pkgs map[*gotypes.Package]*types.Package
}

func (b *goImporterBridge) Import(path string) (*types.Package, error) {
	if path == "unsafe" {
		return types.Unsafe, nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	gp, err := b.imp.Import(path)
	if err != nil || gp == nil {
		return nil, err
	}
	return b.convertPackageLocked(gp), nil
}

func (b *goImporterBridge) convertPackageLocked(gp *gotypes.Package) *types.Package {
	if gp == nil {
		return nil
	}
	if gp == gotypes.Unsafe {
		return types.Unsafe
	}
	if existing, ok := b.pkgs[gp]; ok {
		return existing
	}

	var lp *types.Package
	// Materialize at least one object via b.ctx.FromGoObject so b.ctx creates
	// and registers the canonical *types.Package bound to gp.
	if gs := gp.Scope(); gs != nil {
		for _, name := range gs.Names() {
			if obj := b.ctx.FromGoObject(gs.Lookup(name)); obj != nil && obj.Pkg() != nil {
				lp = obj.Pkg()
				break
			}
		}
	}
	if lp == nil {
		lp = types.NewPackage(gp.Path(), gp.Name())
		if gp.Complete() {
			lp.MarkComplete()
		}
	}
	b.pkgs[gp] = lp

	if gs := gp.Scope(); gs != nil && lp.Scope() != nil {
		for _, name := range gs.Names() {
			if obj := b.ctx.FromGoObject(gs.Lookup(name)); obj != nil {
				lp.Scope().Insert(obj)
			}
		}
	}
	if imps := gp.Imports(); len(imps) > 0 {
		llImps := make([]*types.Package, len(imps))
		for i, imp := range imps {
			llImps[i] = b.convertPackageLocked(imp)
		}
		lp.SetImports(llImps)
	}
	return lp
}

// Packages creates an LLGo SSA program for the initial packages (mirroring
// golang.org/x/tools/go/ssa/ssautil.Packages).
func Packages(initial []*Package, mode ssa.BuilderMode) (*ssa.Program, []*ssa.Package) {
	return doSSAPackages(initial, mode, false)
}

// AllPackages creates an LLGo SSA program for initial packages plus all
// dependencies (mirroring golang.org/x/tools/go/ssa/ssautil.AllPackages).
func AllPackages(initial []*Package, mode ssa.BuilderMode) (*ssa.Program, []*ssa.Package) {
	return doSSAPackages(initial, mode, true)
}

func doSSAPackages(initial []*Package, mode ssa.BuilderMode, deps bool) (*ssa.Program, []*ssa.Package) {
	var fset *token.FileSet
	if len(initial) > 0 {
		fset = initial[0].Fset
	}
	prog := ssa.NewProgram(fset, mode)

	isInitial := make(map[*Package]bool, len(initial))
	for _, p := range initial {
		isInitial[p] = true
	}

	ssamap := make(map[*Package]*ssa.Package)
	Visit(initial, nil, func(p *Package) {
		if p.Types != nil && !p.IllTyped {
			var files []*ast.File
			var info *types.Info
			if deps || isInitial[p] {
				files = p.Syntax
				info = p.TypesInfo
			}
			ssamap[p] = prog.CreatePackage(p.Types, files, info, true)
		}
	})

	ssapkgs := make([]*ssa.Package, len(initial))
	for i, p := range initial {
		ssapkgs[i] = ssamap[p]
	}
	return prog, ssapkgs
}

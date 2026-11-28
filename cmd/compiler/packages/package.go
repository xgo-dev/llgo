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
	"fmt"
	"go/ast"
	"go/token"
	gotypes "go/types"
	"os"
	"strings"

	"github.com/xgo-dev/llgo/types"
	gopackages "golang.org/x/tools/go/packages"
)

// Module mirrors golang.org/x/tools/go/packages.Module.
type Module = gopackages.Module

// Convenience LoadMode aliases mirroring golang.org/x/tools/go/packages.
const (
	LoadSyntax    = gopackages.LoadSyntax
	LoadAllSyntax = gopackages.LoadAllSyntax
)

// Load loads packages using LLGo's native type checker (mirroring
// golang.org/x/tools/go/packages.Load).
func Load(cfg *Config, patterns ...string) ([]*Package, error) {
	return LoadEx(NewDeduper(), nil, cfg, patterns...)
}

// PrintErrors prints to os.Stderr the accumulated errors of all packages in
// the import graph rooted at pkgs, returning the number of errors printed.
func PrintErrors(pkgs []*Package) int {
	var n int
	Visit(pkgs, nil, func(pkg *Package) {
		for _, err := range pkg.Errors {
			fmt.Fprintln(os.Stderr, err)
			n++
		}
	})
	return n
}

// Package describes a loaded Go package speaking LLGo's native types.Package
// and types.Info while delegating Go workspace metadata loading to
// golang.org/x/tools/go/packages.
type Package struct {
	ID              string
	Name            string
	PkgPath         string
	Dir             string
	Errors          []Error
	TypeErrors      []types.Error
	GoFiles         []string
	CompiledGoFiles []string
	OtherFiles      []string
	EmbedFiles      []string
	EmbedPatterns   []string
	IgnoredFiles    []string
	ExportFile      string
	Imports         map[string]*Package
	Types           *types.Package
	Fset            *token.FileSet
	IllTyped        bool
	Syntax          []*ast.File
	TypesInfo       *types.Info
	TypesSizes      types.Sizes
	Module          *Module
	Target          string
	ForTest         string
}

func (p *Package) String() string {
	return p.ID
}

// Visit visits all packages in the import graph whose roots are pkgs, calling
// the optional pre function in preorder and post in postorder.
func Visit(pkgs []*Package, pre func(*Package) bool, post func(*Package)) {
	seen := make(map[*Package]bool)
	var visit func(*Package)
	visit = func(pkg *Package) {
		if pkg == nil || seen[pkg] {
			return
		}
		seen[pkg] = true
		if pre == nil || pre(pkg) {
			paths := make([]string, 0, len(pkg.Imports))
			for path := range pkg.Imports {
				paths = append(paths, path)
			}
			// Deterministic traversal matching golang.org/x/tools/go/packages.Visit.
			sortStrings(paths)
			for _, path := range paths {
				visit(pkg.Imports[path])
			}
		}
		if post != nil {
			post(pkg)
		}
	}
	for _, pkg := range pkgs {
		visit(pkg)
	}
}

func sortStrings(a []string) {
	for i := 1; i < len(a); i++ {
		v := a[i]
		j := i - 1
		for j >= 0 && a[j] > v {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = v
	}
}

func convertGoPackages(initial []*gopackages.Package) []*Package {
	seen := make(map[*gopackages.Package]*Package)
	var convert func(*gopackages.Package) *Package
	convert = func(gp *gopackages.Package) *Package {
		if gp == nil {
			return nil
		}
		if p, ok := seen[gp]; ok {
			return p
		}
		p := &Package{
			ID:              gp.ID,
			Name:            gp.Name,
			PkgPath:         gp.PkgPath,
			Dir:             gp.Dir,
			Errors:          gp.Errors,
			GoFiles:         gp.GoFiles,
			CompiledGoFiles: gp.CompiledGoFiles,
			OtherFiles:      gp.OtherFiles,
			EmbedFiles:      gp.EmbedFiles,
			EmbedPatterns:   gp.EmbedPatterns,
			IgnoredFiles:    gp.IgnoredFiles,
			ExportFile:      gp.ExportFile,
			Fset:            gp.Fset,
			IllTyped:        gp.IllTyped,
			Syntax:          gp.Syntax,
			TypesSizes:      convertGoSizes(gp.TypesSizes),
			Module:          gp.Module,
			Target:          gp.Target,
			ForTest:         gp.ForTest,
		}
		seen[gp] = p
		if gp.Imports != nil {
			p.Imports = make(map[string]*Package, len(gp.Imports))
			for path, dep := range gp.Imports {
				p.Imports[path] = convert(dep)
			}
		}
		return p
	}
	out := make([]*Package, len(initial))
	for i, gp := range initial {
		out[i] = convert(gp)
	}
	return out
}

func convertGoSizes(sizes gotypes.Sizes) types.Sizes {
	if sizes == nil {
		return nil
	}
	if std, ok := sizes.(*gotypes.StdSizes); ok {
		return &types.StdSizes{WordSize: std.WordSize, MaxAlign: std.MaxAlign}
	}
	for _, arch := range strings.Fields("386 amd64 amd64p32 arm arm64 loong64 mips mipsle mips64 mips64le ppc64 ppc64le riscv64 s390x sparc64 wasm") {
		if sizes == gotypes.SizesFor("gc", arch) {
			return types.SizesFor("gc", arch)
		}
	}
	return &types.StdSizes{
		WordSize: sizes.Sizeof(gotypes.Typ[gotypes.Int]),
		MaxAlign: sizes.Alignof(gotypes.Typ[gotypes.Complex128]),
	}
}

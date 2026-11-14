// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/frontcache"
	"github.com/xgo-dev/llgo/internal/goescape"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
)

var errFrontendSourceRequired = errors.New("frontend cache requires source bodies")

type frontendTypeExport struct {
	Version                  int
	Input, Frontend, Archive string
	Package                  *frontcache.PackageExport
}

type frontendTypeLoader struct {
	ctx       *context
	namespace string
	keys      map[string]string
	exports   map[string]*frontendTypeExport
	selected  map[string]bool
}

func newFrontendTypeLoader(ctx *context) *frontendTypeLoader {
	if ctx.buildConf.frontendSourceOnly || !cacheEnabled() || !isEnvOn(llgoEscapeCache, false) || ctx.buildConf.ForceRebuild || ctx.buildConf.ModuleHook != nil || ctx.buildConf.Coverage != nil || ctx.mode == ModeGen || ctx.buildConf.BuildMode != BuildModeExe || ctx.buildConf.deadcodeDropEnabled() || ctx.buildConf.DependencyEscapeDiagnostics > 0 {
		return nil
	}
	m := newManifestBuilder()
	ctx.collectEnvInputs(m)
	ctx.collectCommonInputs(m)
	m.common.WasmReflectBridges, m.common.WasmFuncInfoEntries, m.common.WasmRuntimeFree = false, false, false
	return &frontendTypeLoader{ctx: ctx, namespace: m.Fingerprint() + "|lang=" + ctx.buildConf.GoVersion, keys: make(map[string]string), exports: make(map[string]*frontendTypeExport), selected: make(map[string]bool)}
}

// Source keys are available before type checking, source transforms or SSA.
// The archive certificate below additionally covers the resulting transforms.
func (l *frontendTypeLoader) key(pkg *packages.Package, active map[string]bool) string {
	if key, ok := l.keys[pkg.ID]; ok {
		return key
	}
	if active[pkg.ID] {
		return ""
	}
	active[pkg.ID] = true
	defer delete(active, pkg.ID)
	var inputs []string
	inputs = append(inputs, l.namespace, pkg.ID, pkg.PkgPath, pkg.Name, moduleVersion(pkg.Module))
	if pkg.Module != nil {
		inputs = append(inputs, "language="+pkg.Module.GoVersion)
	}
	files := append(append(append([]string{}, pkg.CompiledGoFiles...), pkg.OtherFiles...), pkg.EmbedFiles...)
	for _, filename := range files {
		data, ok := l.ctx.buildConf.Overlay[filename]
		if !ok {
			var err error
			data, err = os.ReadFile(filename)
			if err != nil {
				return ""
			}
		}
		inputs = append(inputs, filename+":"+frontendDigest(data))
	}
	for name, dep := range pkg.Imports {
		key := l.key(dep, active)
		if key == "" {
			return ""
		}
		inputs = append(inputs, name+":"+key)
	}
	sort.Strings(inputs)
	data, _ := json.Marshal(inputs)
	key := frontendDigest(data)
	l.keys[pkg.ID] = key
	return key
}

func (l *frontendTypeLoader) path(key string) string {
	return filepath.Join(l.ctx.ensureCacheManager().root, "frontend", "types-"+key+".json")
}

func (l *frontendTypeLoader) load(pkg *packages.Package, fset *token.FileSet) bool {
	// Source bodies of generic/inline candidates remain available on demand.
	// Test variants and runtime patches keep their independently checked scopes.
	if pkg.Name == "main" || pkg.Module == nil || pkg.ID != pkg.PkgPath || pkg.ForTest != "" || strings.HasPrefix(pkg.PkgPath, "github.com/xgo-dev/llgo/runtime") {
		return false
	}
	key := l.key(pkg, make(map[string]bool))
	if key == "" {
		return false
	}
	filename := l.path(key)
	info, err := os.Stat(filename)
	if err != nil || info.Size() > goescape.MaxCacheBytes {
		return false
	}
	data, err := os.ReadFile(filename)
	var envelope frontendEnvelope
	if err != nil || json.Unmarshal(data, &envelope) != nil || frontendDigest(envelope.Payload) != envelope.Digest {
		return false
	}
	var record frontendTypeExport
	if json.Unmarshal(envelope.Payload, &record) != nil || record.Version != frontendExportVersion || record.Input != key || record.Package == nil || record.Package.ID != pkg.ID || record.Package.Path != pkg.PkgPath || record.Archive == "" || record.Frontend == "" {
		return false
	}
	// Missing archives/companions need source. Their full input fingerprints are
	// checked later, after whole-program layout/feature decisions are fixed.
	paths := l.ctx.ensureCacheManager().PackagePaths(l.ctx.targetTriple(), pkg.PkgPath, record.Archive)
	if _, err := os.Stat(paths.Archive); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(l.ctx.ensureCacheManager().root, "frontend", record.Frontend+".json")); err != nil {
		return false
	}
	packagesByID := map[string]*types.Package{pkg.ID: types.NewPackage(pkg.PkgPath, pkg.Name)}
	ids := make(map[*types.Package]string)
	packages.Visit([]*packages.Package{pkg}, nil, func(dep *packages.Package) {
		if dep != pkg && dep.Types != nil {
			packagesByID[dep.ID] = dep.Types
			ids[dep.Types] = dep.ID
		}
	})
	index := frontcache.NewTypes(ids, fset)
	for _, dep := range packagesByID {
		index.AddPackage(dep, nil)
	}
	// Positions are mapped to the parsed source FileSet, including private types.
	position := frontendPosition(fset)
	importer := &frontcache.TypeImporter{Packages: packagesByID, Existing: index, Position: position}
	checked, err := frontcache.ImportPackage(record.Package, importer)
	if err != nil {
		return false
	}
	pkg.Types = checked
	pkg.TypesInfo = &types.Info{Defs: make(map[*ast.Ident]types.Object), Uses: make(map[*ast.Ident]types.Object), Types: make(map[ast.Expr]types.TypeAndValue), Instances: make(map[*ast.Ident]types.Instance)}
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						pkg.TypesInfo.Defs[s.Name] = checked.Scope().Lookup(s.Name.Name)
					case *ast.ValueSpec:
						for _, id := range s.Names {
							pkg.TypesInfo.Defs[id] = checked.Scope().Lookup(id.Name)
						}
					}
				}
			case *ast.FuncDecl:
				if d.Recv == nil {
					pkg.TypesInfo.Defs[d.Name] = checked.Scope().Lookup(d.Name.Name)
				}
			}
		}
	}
	l.selected[pkg.ID] = true
	return true
}

func (l *frontendTypeLoader) capture(pkg Package, record *frontendExport, index *frontcache.Types) {
	if pkg.Module == nil || pkg.Name == "main" || pkg.ID != pkg.PkgPath || pkg.AltPkg != nil || l.selected[pkg.ID] {
		return
	}
	// Never discard bodies that can be used by the current cross-package inliner
	// or generic instantiator. A demand outside these contracts restarts loading
	// from source before any backend work is published.
	budget, _, _ := cl.FrontendInlinePolicy()
	for _, fn := range record.Functions {
		if fn.Facts.InlineCost <= budget {
			return
		}
	}
	for _, name := range pkg.Types.Scope().Names() {
		obj := pkg.Types.Scope().Lookup(name)
		typ := obj.Type()
		// Type exports currently retain the underlying type, but not its
		// declaration RHS. Keep native definitions source-checked until that
		// provenance has an export contract. Archive and SSA caching remain
		// available; only the opaque type-checking shortcut is ineligible.
		if _, ok := obj.(*types.TypeName); ok && l.ctx.prog.RepresentationOf(typ).Storage == llssa.NativeRepresentation {
			return
		}
		switch v := typ.(type) {
		case *types.Signature:
			if v.TypeParams().Len() != 0 {
				return
			}
		case *types.Named:
			if v.TypeParams().Len() != 0 {
				return
			}
		case *types.Alias:
			if v.TypeParams().Len() != 0 {
				return
			}
		}
	}
	if info := pkg.TypesInfo; info != nil {
		for _, obj := range info.Defs {
			if name, ok := obj.(*types.TypeName); ok && l.ctx.prog.RepresentationOf(name.Type()).Storage == llssa.NativeRepresentation {
				// Local defined types can escape through exported signatures
				// or runtime type metadata without appearing in package scope.
				return
			}
		}
	}
	for _, file := range pkg.Syntax {
		for _, group := range file.Comments {
			for _, c := range group.List {
				if strings.Contains(c.Text, "llgo:env") {
					return
				}
			}
		}
	}
	key := l.key(pkg.Package, make(map[string]bool))
	if key == "" {
		return
	}
	l.exports[pkg.ID] = &frontendTypeExport{Version: frontendExportVersion, Input: key, Frontend: record.Input, Package: frontcache.ExportPackage(pkg.Types, index, l.ctx.progSSA.Fset)}
}

func (l *frontendTypeLoader) store(pkg Package) error {
	record := l.exports[pkg.ID]
	if record == nil {
		return nil
	}
	copy := *record
	copy.Archive = pkg.Fingerprint
	data, err := json.Marshal(copy)
	if err != nil {
		return err
	}
	wrapped, err := json.Marshal(frontendEnvelope{Digest: frontendDigest(data), Payload: data})
	if err != nil {
		return err
	}
	if len(wrapped) > goescape.MaxCacheBytes {
		return fmt.Errorf("type export exceeds cache limit")
	}
	return writeManifest(l.path(copy.Input), string(wrapped))
}

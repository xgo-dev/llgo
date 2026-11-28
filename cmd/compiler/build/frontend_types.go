// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xgo-dev/llgo/types"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/xgo-dev/llgo/cmd/compiler/frontcache"
	"github.com/xgo-dev/llgo/cmd/compiler/goescape"
	"github.com/xgo-dev/llgo/cmd/compiler/packages"
	"github.com/xgo-dev/llgo/directive"
)

var errFrontendSourceRequired = errors.New("frontend cache requires source bodies")

type frontendSourceRequiredError struct {
	pkgIDs []string
}

func (e *frontendSourceRequiredError) Error() string { return errFrontendSourceRequired.Error() }
func (e *frontendSourceRequiredError) Unwrap() error { return errFrontendSourceRequired }

const frontendTypesFormatExportData = "exportdata"

type frontendTypeExport struct {
	Version                  int
	Input, Frontend, Archive string
	PkgID, PkgPath, PkgName  string                    `json:",omitempty"`
	TypesFormat              string                    `json:",omitempty"`
	Package                  *frontcache.PackageExport `json:",omitempty"`
}

type frontendInlineHint struct {
	Version  int
	Packages []string
}

type frontendTypeLoader struct {
	mu             sync.RWMutex
	ctx            *context
	cache          *cacheManager
	namespace      string
	inlineHintKey  string
	forceSource    map[string]bool
	keys           map[string]string
	exports        map[string]*frontendTypeExport
	selected       map[string]bool
	exportDataHits map[string]bool
}

func newFrontendTypeLoader(ctx *context, patterns []string) *frontendTypeLoader {
	if ctx.buildConf.frontendSourceOnly || !cacheEnabled() || !isEnvOn(llgoEscapeCache, true) || ctx.buildConf.ForceRebuild || ctx.buildConf.ModuleHook != nil || ctx.buildConf.Coverage != nil || ctx.mode == ModeGen || ctx.buildConf.BuildMode != BuildModeExe || ctx.buildConf.deadcodeDropEnabled() || ctx.buildConf.DependencyEscapeDiagnostics > 0 {
		return nil
	}
	m := newManifestBuilder()
	ctx.collectEnvInputs(m)
	ctx.collectCommonInputs(m)
	m.common.WasmReflectBridges, m.common.WasmFuncInfoEntries, m.common.WasmRuntimeFree = false, false, false
	ns := m.Fingerprint() + "|lang=" + ctx.buildConf.GoVersion
	cm := ctx.ensureCacheManager()
	l := &frontendTypeLoader{
		ctx:            ctx,
		cache:          cm,
		namespace:      ns,
		keys:           make(map[string]string),
		exports:        make(map[string]*frontendTypeExport),
		selected:       make(map[string]bool),
		exportDataHits: make(map[string]bool),
	}
	if len(ctx.buildConf.frontendSourcePkgs) > 0 {
		l.forceSource = make(map[string]bool, len(ctx.buildConf.frontendSourcePkgs))
		for id, forced := range ctx.buildConf.frontendSourcePkgs {
			if forced {
				l.forceSource[id] = true
			}
		}
	}
	dir := ""
	if ctx.conf != nil {
		dir = ctx.conf.Dir
	}
	if dir == "" {
		dir, _ = os.Getwd()
	} else if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	sortedPatterns := append([]string(nil), patterns...)
	sort.Strings(sortedPatterns)
	hintInput := ns + "|dir=" + dir + "|patterns=" + strings.Join(sortedPatterns, ",")
	l.inlineHintKey = frontendDigest([]byte(hintInput))
	for _, id := range l.loadInlineHint() {
		if l.forceSource == nil {
			l.forceSource = make(map[string]bool)
		}
		l.forceSource[id] = true
	}
	return l
}

func (l *frontendTypeLoader) inlineHintPath() string {
	return filepath.Join(l.cacheManager().root, "frontend", "inline-"+l.inlineHintKey+".json")
}

func (l *frontendTypeLoader) loadInlineHint() []string {
	if l.inlineHintKey == "" {
		return nil
	}
	data, err := os.ReadFile(l.inlineHintPath())
	if err != nil {
		return nil
	}
	var hint frontendInlineHint
	if json.Unmarshal(data, &hint) != nil || hint.Version != frontendExportVersion {
		return nil
	}
	return hint.Packages
}

func (l *frontendTypeLoader) storeInlineHint(pkgIDs []string) {
	if l.inlineHintKey == "" {
		return
	}
	sorted := append([]string(nil), pkgIDs...)
	sort.Strings(sorted)
	data, err := json.Marshal(frontendInlineHint{
		Version:  frontendExportVersion,
		Packages: sorted,
	})
	if err != nil {
		return
	}
	_ = writeManifest(l.inlineHintPath(), string(data))
}

func (l *frontendTypeLoader) eligible(pkg *packages.Package) bool {
	if pkg == nil || pkg.Name == "main" || pkg.PkgPath == "unsafe" || pkg.ID != pkg.PkgPath || pkg.ForTest != "" {
		return false
	}
	if pkg.PkgPath == "runtime" || strings.HasPrefix(pkg.PkgPath, "runtime/") ||
		strings.HasPrefix(pkg.PkgPath, "github.com/xgo-dev/llgo/runtime") ||
		strings.HasPrefix(pkg.PkgPath, altPkgPathPrefix) {
		return false
	}
	if l.ctx != nil && l.ctx.buildConf != nil {
		if hasAltPkgForTarget(l.ctx.buildConf, pkg.PkgPath) {
			return false
		}
		if len(l.ctx.buildConf.GlobalRewrites[pkg.PkgPath]) > 0 {
			return false
		}
	}
	return true
}

// Source keys are available before type checking, source transforms or SSA.
// Non-overlaid files use fast stat (size + mtime) keys here because certify()
// subsequently verifies full SHA-256 archive fingerprints before accepting any
// opaque package.
func (l *frontendTypeLoader) key(pkg *packages.Package, active map[string]bool) string {
	l.mu.RLock()
	key, ok := l.keys[pkg.ID]
	l.mu.RUnlock()
	if ok {
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
		if data, ok := l.ctx.buildConf.Overlay[filename]; ok {
			inputs = append(inputs, filename+":ov:"+frontendDigest(data))
			continue
		}
		info, err := os.Stat(filename)
		if err != nil {
			return ""
		}
		inputs = append(inputs, fmt.Sprintf("%s:st:%d:%d", filename, info.Size(), info.ModTime().UnixNano()))
	}
	for name, dep := range pkg.Imports {
		depKey := l.key(dep, active)
		if depKey == "" {
			return ""
		}
		inputs = append(inputs, name+":"+depKey)
	}
	sort.Strings(inputs)
	data, _ := json.Marshal(inputs)
	key = frontendDigest(data)
	l.mu.Lock()
	l.keys[pkg.ID] = key
	l.mu.Unlock()
	return key
}

func (l *frontendTypeLoader) cacheManager() *cacheManager {
	if l.cache != nil {
		return l.cache
	}
	return l.ctx.ensureCacheManager()
}

func (l *frontendTypeLoader) path(key string) string {
	return filepath.Join(l.cacheManager().root, "frontend", "types-"+key+".json")
}

func (l *frontendTypeLoader) load(pkg *packages.Package, fset *token.FileSet) bool {
	// Source bodies of generic/inline candidates remain available on demand.
	// Test variants and runtime patches keep their independently checked scopes.
	if !l.eligible(pkg) || l.forceSource[pkg.ID] {
		return false
	}
	for _, dep := range pkg.Imports {
		if dep != nil && dep.ID != dep.PkgPath {
			return false
		}
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
	if json.Unmarshal(envelope.Payload, &record) != nil || record.Version != frontendExportVersion || record.Input != key || record.Archive == "" || record.Frontend == "" {
		return false
	}
	// Missing archives/companions need source. Their full input fingerprints are
	// checked later, after whole-program layout/feature decisions are fixed.
	cm := l.cacheManager()
	paths := cm.PackagePaths(l.ctx.targetTriple(), pkg.PkgPath, record.Archive)
	if _, err := os.Stat(paths.Archive); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(cm.root, "frontend", record.Frontend+".llgoex")); err != nil {
		return false
	}

	var checked *types.Package
	isExportDataHit := false
	if record.TypesFormat == frontendTypesFormatExportData {
		if record.PkgID != pkg.ID || record.PkgPath != pkg.PkgPath {
			return false
		}
		imports := map[string]*types.Package{"unsafe": types.Unsafe}
		packages.Visit([]*packages.Package{pkg}, nil, func(dep *packages.Package) {
			if dep != pkg && dep.Types != nil {
				imports[dep.PkgPath] = dep.Types
			}
		})
		checked, err = l.ctx.readPackageTypesSidecar(paths.Types, pkg.PkgPath, imports)
		if err != nil || checked == nil || !checked.Complete() {
			return false
		}
		isExportDataHit = true
	} else {
		if record.Package == nil || record.Package.ID != pkg.ID || record.Package.Path != pkg.PkgPath {
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
		position := l.ctx.frontendPosition(fset)
		importer := &frontcache.TypeImporter{Packages: packagesByID, Existing: index, Position: position}
		checked, err = frontcache.ImportPackage(record.Package, importer)
		if err != nil {
			return false
		}
	}

	pkg.Types = checked
	pkg.TypesInfo = &types.Info{
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Instances:  make(map[*ast.Ident]types.Instance),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
	}
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
	l.mu.Lock()
	if isExportDataHit && l.exportDataHits != nil {
		l.exportDataHits[pkg.ID] = true
	}
	l.selected[pkg.ID] = true
	l.mu.Unlock()
	return true
}

func (l *frontendTypeLoader) capture(pkg Package, record *frontendExport, index *frontcache.Types) {
	if pkg == nil || !l.eligible(pkg.Package) || pkg.AltPkg != nil || l.selected[pkg.ID] || pkg.Types == nil {
		return
	}
	// Packages containing generic declarations or cross-package inline
	// candidates are still exported via .llgotypes (exportdata v4 preserves
	// full type parameters and constraints). Only the small set actually
	// instantiated or inlined by the current root's callers is parsed from
	// source via persisted source hints / targeted retry.
	for _, name := range pkg.Types.Scope().Names() {
		obj := pkg.Types.Scope().Lookup(name)
		if v, ok := obj.(*types.Var); ok && v.Locality() != directive.LocalityNone {
			return
		}
	}
	for _, file := range pkg.Syntax {
		for _, group := range file.Comments {
			if directive.HasClosureEnv(group) {
				return
			}
		}
	}
	key := l.key(pkg.Package, make(map[string]bool))
	if key == "" {
		return
	}
	l.exports[pkg.ID] = &frontendTypeExport{
		Version:     frontendExportVersion,
		Input:       key,
		Frontend:    record.Input,
		PkgID:       pkg.ID,
		PkgPath:     pkg.PkgPath,
		PkgName:     pkg.Name,
		TypesFormat: frontendTypesFormatExportData,
	}
}

func (l *frontendTypeLoader) store(pkg Package) error {
	record := l.exports[pkg.ID]
	if record == nil {
		return nil
	}
	copy := *record
	copy.Archive = pkg.Fingerprint
	if copy.TypesFormat == frontendTypesFormatExportData {
		paths := l.ctx.ensureCacheManager().PackagePaths(l.ctx.targetTriple(), pkg.PkgPath, copy.Archive)
		if _, err := os.Stat(paths.Types); err != nil {
			if err := l.ctx.writePackageTypesSidecar(paths.Types, pkg); err != nil {
				return err
			}
		}
	}
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

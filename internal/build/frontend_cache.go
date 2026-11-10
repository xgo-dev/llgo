// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package build

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"sort"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/frontcache"
	"github.com/xgo-dev/llgo/internal/goescape"
	"golang.org/x/tools/go/ssa"
)

const frontendExportVersion = 4

type frontendFunction struct {
	ID       string
	Facts    cl.FrontendFacts
	Escape   *goescape.Summary `json:",omitempty"`
	CacheKey string            `json:",omitempty"`
}

// frontendExport certifies both the source-derived features and the particular
// archive whose placement/caller contracts it describes. Source features may
// be read before whole-program feature selection; archive contracts may only
// be used after the final archive fingerprint matches Archive.
type frontendExport struct {
	Version      int
	Input        string
	Archive      string
	Functions    []frontendFunction
	Types        frontcache.TypeGraph
	RuntimeTypes []int
	Link         *frontendLinkExport `json:",omitempty"`
}

type frontendEnvelope struct {
	Digest  string
	Payload json.RawMessage
}

type frontendCache struct {
	ctx          *context
	keys         map[string]string
	locals       map[string]string
	records      map[string]*frontendExport
	owners       map[*ssa.Package]Package
	packages     map[string]*types.Package
	ids          map[*types.Package]string
	types        *frontcache.Types
	entries      []ssaBuildEntry
	built        map[*ssa.Package]bool
	methods      []*ssa.Function
	functions    map[string]*ssa.Function
	imports      map[*ssa.Function]goescape.Summary
	importKeys   map[*ssa.Function]string
	ambiguous    map[string]bool
	uncacheable  map[Package]bool
	loader       *frontendTypeLoader
	err          error
	programInput string
	program      *frontendProgramExport
}

func (c *context) newFrontendCache(entries []ssaBuildEntry, mode ssa.BuilderMode) *frontendCache {
	if !cacheEnabled() || !isEnvOn(llgoEscapeCache, false) || c.buildConf.ForceRebuild || c.buildConf.ModuleHook != nil ||
		c.mode == ModeGen || c.buildConf.BuildMode != BuildModeExe || c.buildConf.deadcodeDropEnabled() || c.buildConf.DependencyEscapeDiagnostics > 0 {
		return nil
	}
	f := &frontendCache{ctx: c, entries: entries, keys: make(map[string]string), locals: make(map[string]string),
		records: make(map[string]*frontendExport), owners: make(map[*ssa.Package]Package),
		packages: make(map[string]*types.Package), ids: make(map[*types.Package]string), built: make(map[*ssa.Package]bool)}
	for _, entry := range entries {
		if entry.pkg != nil && entry.pkg.Pkg != nil {
			f.packages[entry.id], f.ids[entry.pkg.Pkg] = entry.pkg.Pkg, entry.id
		}
	}
	for _, pkg := range c.pkgByID {
		f.owners[pkg.SSA] = pkg
		if pkg.AltPkg != nil {
			f.owners[c.progSSA.Package(pkg.AltPkg.Types)] = pkg
		}
	}
	f.types = frontcache.NewTypes(f.ids, c.progSSA.Fset)
	if c.buildConf.Goarch == "wasm" {
		f.programInput = f.programKey(mode)
		f.program = f.loadProgram()
	}
	return f
}

// inputKey is deliberately separate from the final archive key. Late Wasm
// feature decisions and initializer-removal plans are not known yet. Reusing
// a provisional final fingerprint would silently freeze those inputs to zero.
func (f *frontendCache) inputKey(pkg Package) (string, error) {
	if key, ok := f.keys[pkg.ID]; ok {
		return key, nil
	}
	// Hash a sorted dependency closure. Runtime source replacements can close
	// package cycles even though the original Go import graph is acyclic.
	// A closure identity is stable for cycles and also covers source overlays in
	// versioned modules, without treating an unresolved back-edge as a cache hit.
	visited := make(map[string]bool)
	var inputs []string
	var visit func(Package) error
	visit = func(current Package) error {
		if visited[current.ID] {
			return nil
		}
		visited[current.ID] = true
		local := f.locals[current.ID]
		if local == "" {
			m := newManifestBuilder()
			f.ctx.collectEnvInputs(m)
			f.ctx.collectCommonInputs(m)
			m.common.WasmReflectBridges, m.common.WasmFuncInfoEntries, m.common.WasmRuntimeFree = false, false, false
			if err := f.ctx.collectPackageInputs(m, current); err != nil {
				return err
			}
			m.pkg.DeadInits, m.pkg.FrontendInputs = nil, ""
			local = m.Fingerprint()
			f.locals[current.ID] = local
		}
		inputs = append(inputs, current.ID+":"+local)
		for _, dep := range effectiveDependencies(current) {
			inputs = append(inputs, current.ID+" -> "+dep.ID+"@"+moduleVersion(dep.Module))
			loaded := f.ctx.pkgByID[dep.ID]
			if loaded == nil {
				loaded = &aPackage{Package: dep}
			}
			if err := visit(loaded); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(pkg); err != nil {
		return "", err
	}
	sort.Strings(inputs)
	data, _ := json.Marshal(inputs)
	key := frontendDigest(append([]byte(fmt.Sprintf("frontend:%d:%s\n", frontendExportVersion, pkg.ID)), data...))
	f.keys[pkg.ID] = key
	return key, nil
}

func frontendDigest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func (f *frontendCache) path(key string) string {
	return filepath.Join(f.ctx.ensureCacheManager().root, "frontend", key+".json")
}

func (f *frontendCache) load(pkg Package) *frontendExport {
	key, err := f.inputKey(pkg)
	if err != nil || key == "" {
		return nil
	}
	path := f.path(key)
	info, err := os.Stat(path)
	if err != nil || info.Size() > goescape.MaxCacheBytes {
		return nil
	}
	data, err := os.ReadFile(path)
	var envelope frontendEnvelope
	if err != nil || json.Unmarshal(data, &envelope) != nil || frontendDigest(envelope.Payload) != envelope.Digest {
		return nil
	}
	var record frontendExport
	if json.Unmarshal(envelope.Payload, &record) != nil || record.Version != frontendExportVersion ||
		record.Input != key || record.Archive == "" || record.Link == nil || len(record.Functions) == 0 {
		return nil
	}
	return &record
}

func (f *frontendCache) store(pkg Package) error {
	record := f.records[pkg.ID]
	if record == nil || record.Input == "" || pkg.frontendOpaque != nil || pkg.linkSnapshot == nil || pkg.Name == "main" || f.ctx.packageCacheDisabled(pkg.ID) {
		return nil
	}
	// The pre-backend feature record is immutable. Each package worker owns
	// its own copy and type encoder when adding the archive's link snapshot.
	copy := *record
	copy.Archive = pkg.Fingerprint
	link, err := f.exportLink(pkg.linkSnapshot)
	if err != nil {
		return err
	}
	copy.Link = link
	data, err := json.Marshal(copy)
	if err != nil {
		return err
	}
	wrapped, err := json.Marshal(frontendEnvelope{Digest: frontendDigest(data), Payload: data})
	if err != nil || len(wrapped) > goescape.MaxCacheBytes {
		return err
	}
	if err := writeManifest(f.path(record.Input), string(wrapped)); err != nil {
		return err
	}
	if f.loader != nil {
		return f.loader.store(pkg)
	}
	return nil
}

func (f *frontendCache) orderedPackages() []Package {
	var pkgs []Package
	for _, pkg := range f.ctx.pkgByID {
		pkgs = append(pkgs, pkg)
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].ID < pkgs[j].ID })
	return pkgs
}

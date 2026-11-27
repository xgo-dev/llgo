// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package build

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/xgo-dev/llgo/types"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"github.com/xgo-dev/llgo/cmd/compiler/cl"
	"github.com/xgo-dev/llgo/cmd/compiler/frontcache"
	"github.com/xgo-dev/llgo/cmd/compiler/goescape"
	"github.com/xgo-dev/llgo/ssa"
)

const frontendExportVersion = 5

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
	ctx            *context
	keys           map[string]string
	locals         map[string]string
	records        map[string]*frontendExport
	owners         map[*ssa.Package]Package
	packages       map[string]*types.Package
	ids            map[*types.Package]string
	types          *frontcache.Types
	entries        []ssaBuildEntry
	built          map[*ssa.Package]bool
	methods        []*ssa.Function
	functions      map[string]*ssa.Function
	instances      map[*ssa.Function]bool
	bound          map[*ssa.Function]bool
	fnIDs          map[*ssa.Function]string
	imports        map[*ssa.Function]goescape.Summary
	importKeys     map[*ssa.Function]string
	ambiguous      map[string]bool
	uncacheable    map[Package]bool
	fingerprintErr map[Package]bool
	sourceReason   map[string]string
	loader         *frontendTypeLoader
	err            error
	programInput   string
	program        *frontendProgramExport
}

func (c *context) newFrontendCache(entries []ssaBuildEntry, mode ssa.BuilderMode) *frontendCache {
	if !cacheEnabled() || !isEnvOn(llgoEscapeCache, true) || c.buildConf.ForceRebuild || c.buildConf.ModuleHook != nil ||
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

func (f *frontendCache) manifestTemplate() *manifestBuilder {
	base := newManifestBuilder()
	f.ctx.collectEnvInputs(base)
	f.ctx.collectCommonInputs(base)
	base.common.WasmReflectBridges, base.common.WasmFuncInfoEntries, base.common.WasmRuntimeFree = false, false, false
	if len(base.common.BuildTags) > 1 {
		base.common.BuildTags = append([]string(nil), base.common.BuildTags...)
		sort.Strings(base.common.BuildTags)
	}
	return base
}

func (f *frontendCache) computeLocalFingerprint(base *manifestBuilder, current Package) (string, error) {
	m := newManifestBuilder()
	m.env = base.env
	m.common = base.common
	if err := f.ctx.collectPackageInputs(m, current); err != nil {
		return "", err
	}
	m.pkg.DeadInits, m.pkg.FrontendInputs = nil, ""
	return m.Fingerprint(), nil
}

// precomputeLocals populates f.locals for pkgs and all reachable effective
// dependencies in parallel, warming non-thread-safe metadata caches once up front.
func (f *frontendCache) precomputeLocals(pkgs []Package) {
	seen := make(map[string]bool, len(pkgs))
	var todo []Package
	var collect func(Package)
	collect = func(current Package) {
		if current == nil || seen[current.ID] {
			return
		}
		seen[current.ID] = true
		if f.locals[current.ID] == "" {
			todo = append(todo, current)
		}
		for _, dep := range effectiveDependencies(current) {
			loaded := f.ctx.pkgByID[dep.ID]
			if loaded == nil {
				loaded = &aPackage{Package: dep}
			}
			collect(loaded)
		}
	}
	for _, pkg := range pkgs {
		collect(pkg)
	}
	if len(todo) == 0 {
		return
	}

	// Warm shared caches serially before fanning out file digesting across workers.
	_ = f.ctx.ensureCacheManager()
	_ = f.ctx.getLLVMVersion()
	base := f.manifestTemplate()
	for _, pkg := range todo {
		_, _ = pkgSFiles(f.ctx, pkg.Package)
		if f.ctx.mode != ModeGen {
			_, _ = llgoPkgFileInputs(f.ctx, pkg.Package)
			if pkg.AltPkg != nil {
				_, _ = llgoPkgFileInputs(f.ctx, pkg.AltPkg.Package)
			}
		}
	}

	results := make([]string, len(todo))
	workers := runtime.GOMAXPROCS(0)
	if workers > len(todo) {
		workers = len(todo)
	}
	if workers <= 1 {
		for i, pkg := range todo {
			if fp, err := f.computeLocalFingerprint(base, pkg); err == nil {
				results[i] = fp
			}
		}
	} else {
		jobs := make(chan int, len(todo))
		for i := range todo {
			jobs <- i
		}
		close(jobs)
		var wg sync.WaitGroup
		wg.Add(workers)
		for w := 0; w < workers; w++ {
			go func() {
				defer wg.Done()
				for i := range jobs {
					if fp, err := f.computeLocalFingerprint(base, todo[i]); err == nil {
						results[i] = fp
					}
				}
			}()
		}
		wg.Wait()
	}
	for i, pkg := range todo {
		if results[i] != "" {
			f.locals[pkg.ID] = results[i]
		}
	}
}

func (f *frontendCache) computeInputKeyReadOnly(pkg Package) (string, bool) {
	visited := make(map[string]bool)
	var inputs []string
	var visit func(Package) bool
	visit = func(current Package) bool {
		if visited[current.ID] {
			return true
		}
		visited[current.ID] = true
		local := f.locals[current.ID]
		if local == "" {
			return false
		}
		inputs = append(inputs, current.ID+":"+local)
		for _, dep := range effectiveDependencies(current) {
			inputs = append(inputs, current.ID+" -> "+dep.ID+"@"+moduleVersion(dep.Module))
			loaded := f.ctx.pkgByID[dep.ID]
			if loaded == nil {
				loaded = &aPackage{Package: dep}
			}
			if !visit(loaded) {
				return false
			}
		}
		return true
	}
	if !visit(pkg) {
		return "", false
	}
	sort.Strings(inputs)
	data, _ := json.Marshal(inputs)
	return frontendDigest(append([]byte(fmt.Sprintf("frontend:%d:%s\n", frontendExportVersion, pkg.ID)), data...)), true
}

func (f *frontendCache) loadByKey(cacheRoot, key string) *frontendExport {
	if key == "" {
		return nil
	}
	path := filepath.Join(cacheRoot, "frontend", key+".json")
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

type frontendLoadResult struct {
	key    string
	record *frontendExport
}

func (f *frontendCache) loadAll(pkgs []Package) []frontendLoadResult {
	f.precomputeLocals(pkgs)
	cacheRoot := f.ctx.ensureCacheManager().root
	results := make([]frontendLoadResult, len(pkgs))
	workers := runtime.GOMAXPROCS(0)
	if workers > len(pkgs) {
		workers = len(pkgs)
	}
	runOne := func(i int) {
		pkg := pkgs[i]
		key, ok := f.computeInputKeyReadOnly(pkg)
		if !ok || key == "" {
			return
		}
		results[i].key = key
		if pkg.Name != "main" && !f.ctx.packageCacheDisabled(pkg.ID) {
			results[i].record = f.loadByKey(cacheRoot, key)
		}
	}
	if workers <= 1 {
		for i := range pkgs {
			runOne(i)
		}
	} else {
		jobs := make(chan int, len(pkgs))
		for i := range pkgs {
			jobs <- i
		}
		close(jobs)
		var wg sync.WaitGroup
		wg.Add(workers)
		for w := 0; w < workers; w++ {
			go func() {
				defer wg.Done()
				for i := range jobs {
					runOne(i)
				}
			}()
		}
		wg.Wait()
	}
	for i, pkg := range pkgs {
		if results[i].key != "" {
			f.keys[pkg.ID] = results[i].key
		}
	}
	return results
}

func (f *frontendCache) load(pkg Package) *frontendExport {
	key, err := f.inputKey(pkg)
	if err != nil || key == "" {
		return nil
	}
	return f.loadByKey(f.ctx.ensureCacheManager().root, key)
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

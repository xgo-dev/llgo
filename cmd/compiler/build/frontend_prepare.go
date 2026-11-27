// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package build

import (
	"fmt"
	"github.com/xgo-dev/llgo/types"
	"os"
	"sort"
	"strings"

	"github.com/xgo-dev/llgo/cmd/compiler/cl"
	"github.com/xgo-dev/llgo/cmd/compiler/frontcache"
	"github.com/xgo-dev/llgo/cmd/compiler/goescape"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssautil"
)

func (f *frontendCache) build(selected map[*ssa.Package]bool) {
	var entries []ssaBuildEntry
	var missing []string
	for _, entry := range f.entries {
		if !f.built[entry.pkg] && selected[entry.pkg] {
			if f.ctx.dedup.IsExported(entry.id) {
				if f.ctx.buildConf.Verbose {
					fmt.Fprintf(os.Stderr, "FRONTEND SOURCE REQUIRED: %s (reason=%s)\n", entry.id, f.sourceReason[entry.id])
				}
				missing = append(missing, entry.id)
				continue
			}
			entries = append(entries, entry)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		f.err = &frontendSourceRequiredError{pkgIDs: missing}
		return
	}
	buildSSAPkgs(f.ctx, entries)
	for _, entry := range entries {
		f.built[entry.pkg] = true
	}
}

func (f *frontendCache) buildAll() {
	selected := make(map[*ssa.Package]bool)
	for _, entry := range f.entries {
		selected[entry.pkg] = true
	}
	f.build(selected)
}

func (f *frontendCache) source(pkg Package, selected map[*ssa.Package]bool, reason string) {
	selected[pkg.SSA] = true
	if pkg.AltPkg != nil {
		selected[f.ctx.progSSA.Package(pkg.AltPkg.Types)] = true
	}
	pkg.frontendOpaque = nil
	if f.sourceReason == nil {
		f.sourceReason = make(map[string]string)
	}
	if _, exists := f.sourceReason[pkg.ID]; !exists {
		f.sourceReason[pkg.ID] = reason
	}
}

func (f *frontendCache) indexTypes() {
	for _, entry := range f.entries {
		if cached := f.ctx.dedup.Check(entry.id); cached != nil {
			f.types.AddPackage(cached.Types, cached.TypesInfo)
		} else if entry.pkg != nil {
			f.types.AddPackage(entry.pkg.Pkg, nil)
		}
	}
}

// prepare consults source exports before Package.Build, then certifies opaque
// imports against the final archive key. A missing body contract always selects
// the ordinary source path; an archive hit alone never proves frontend facts.
func (f *frontendCache) prepare() error {
	span := f.ctx.buildTrace.startCoordinator("load frontend exports", nil)
	defer span.done()
	f.indexTypes()
	selected := make(map[*ssa.Package]bool)
	ordered := f.orderedPackages()
	loaded := f.loadAll(ordered)
	for i, pkg := range ordered {
		record := loaded[i].record
		if record != nil && pkg.Name != "main" && !f.ctx.packageCacheDisabled(pkg.ID) {
			f.records[pkg.ID] = record
		} else {
			f.source(pkg, selected, "prepare-miss")
		}
	}
	for _, root := range f.ctx.initial {
		if pkg := f.ctx.pkgs[root]; pkg != nil {
			f.source(pkg, selected, "initial-root")
		}
	}
	for _, entry := range f.entries {
		if f.owners[entry.pkg] == nil {
			selected[entry.pkg] = true
		}
	}
	// The Wasm initializer plan is a whole-program fixed point. Until its exact
	// program-context export is available, compute it from complete source SSA.
	if f.ctx.buildConf.Goarch == "wasm" && f.program == nil {
		f.buildAll()
	} else {
		f.build(selected)
	}
	if f.err != nil {
		return f.err
	}
	return f.materialize()
}

func (f *frontendCache) materialize() error {
	var runtimeTypes []types.Type
	for _, pkg := range f.orderedPackages() {
		record := f.records[pkg.ID]
		if record == nil || f.built[pkg.SSA] {
			continue
		}
		importer := &frontcache.TypeImporter{Graph: &record.Types, Packages: f.packages, Existing: f.types, Position: f.position}
		valid := true
		for _, ref := range record.RuntimeTypes {
			typ, err := importer.Import(ref)
			if err != nil {
				valid = false
				break
			}
			runtimeTypes = append(runtimeTypes, typ)
		}
		if !valid {
			selected := make(map[*ssa.Package]bool)
			f.source(pkg, selected, "materialize-runtime-type")
			f.build(selected)
		}
	}
	methods := f.ctx.callerTracking.MaterializeFrontendMethods(f.ctx.progSSA.AllPackages(), runtimeTypes)
	f.methods = nil
	for _, values := range methods {
		f.methods = append(f.methods, values...)
	}
	if f.err != nil {
		return f.err
	}
	f.bindFunctions()
	return nil
}

func (f *frontendCache) functionPackage(fn *ssa.Function) *ssa.Package {
	if fn.Pkg != nil {
		return fn.Pkg
	}
	return f.ctx.progSSA.Package(frontcache.Owner(fn))
}

func (f *frontendCache) functionID(fn *ssa.Function) string {
	if id, ok := f.fnIDs[fn]; ok {
		return id
	}
	id := frontcache.FunctionID(fn, f.types)
	if f.fnIDs == nil {
		f.fnIDs = make(map[*ssa.Function]string)
	}
	f.fnIDs[fn] = id
	return id
}

func (f *frontendCache) bindFunctions() {
	if f.functions == nil {
		f.functions = make(map[string]*ssa.Function)
	}
	if f.instances == nil {
		f.instances = make(map[*ssa.Function]bool)
	}
	if f.ambiguous == nil {
		f.ambiguous = make(map[string]bool)
	}
	if f.uncacheable == nil {
		f.uncacheable = make(map[Package]bool)
	}
	if f.bound == nil {
		f.bound = make(map[*ssa.Function]bool)
	}
	add := func(fn *ssa.Function) {
		if fn == nil || f.bound[fn] {
			return
		}
		f.bound[fn] = true
		if frontendCallerSynthetic(fn) {
			return
		}
		if frontendInstance(fn) {
			f.instances[fn] = true
			return
		}
		id := f.functionID(fn)
		if f.ambiguous[id] {
			f.uncacheable[f.owners[f.functionPackage(fn)]] = true
			return
		}
		if previous := f.functions[id]; previous != nil && previous != fn {
			f.ambiguous[id] = true
			f.uncacheable[f.owners[f.functionPackage(fn)]] = true
			f.uncacheable[f.owners[f.functionPackage(previous)]] = true
			delete(f.functions, id)
			return
		}
		f.functions[id] = fn
	}
	for fn := range ssautil.AllFunctions(f.ctx.progSSA) {
		add(fn)
	}
	for _, fn := range f.methods {
		add(fn)
	}
}

// importBodies preserves the ordinary inliner's decisions. Source packages
// which actually lower code may import an eligible dependency body; cached
// packages that only contribute declarations do not request optimization bodies.
func (f *frontendCache) importBodies() error {
	if f.ctx.frontendOptions.DisableInlineBodyImports || devIncrementalModeEnabled() ||
		(f.ctx.buildConf != nil && f.ctx.buildConf.OptLevel == optlevel.O0) {
		return nil
	}
	bodyBudget, _, depth := cl.FrontendInlinePolicy()
	// Resolving a target depends on declarations, not materialized bodies.
	// Keep these results across rounds without caching depth-dependent visits.
	targets := make(map[*ssa.Function]*ssa.Function)
	costs := make(map[string]int)
	for _, record := range f.records {
		for _, fn := range record.Functions {
			costs[fn.ID] = fn.Facts.InlineCost
		}
	}
	active := make(map[*ssa.Function]bool)
	var visit func(*ssa.Function, int, map[*ssa.Package]bool)
	visit = func(fn *ssa.Function, level int, needed map[*ssa.Package]bool) {
		if fn == nil || active[fn] || level > depth {
			return
		}
		active[fn] = true
		defer delete(active, fn)
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				call, ok := instr.(ssa.CallInstruction)
				if !ok {
					continue
				}
				callee := call.Common().StaticCallee()
				target, resolved := targets[callee]
				if !resolved {
					target = cl.ResolveInlineTarget(callee, f.ctx.patches)
					targets[callee] = target
				}
				if target == nil {
					continue
				}
				if cost, ok := costs[f.functionID(target)]; ok && cost <= bodyBudget {
					if owner := f.owners[target.Pkg]; owner != nil && !f.built[target.Pkg] {
						f.source(owner, needed, "inline-body")
					}
					visit(target, level+1, needed)
				}
			}
		}
	}
	for round := 0; round <= depth; round++ {
		selected := make(map[*ssa.Package]bool)
		for _, fn := range f.functions {
			owner := f.owners[f.functionPackage(fn)]
			if owner != nil && (!owner.CacheHit || owner.Name == "main") && f.built[f.functionPackage(fn)] {
				visit(fn, 0, selected)
			}
		}
		for fn := range f.instances {
			if len(fn.Blocks) == 0 {
				continue
			}
			owner := f.owners[f.functionPackage(fn)]
			if owner != nil && (!owner.CacheHit || owner.Name == "main") {
				visit(fn, 0, selected)
			}
		}
		if len(selected) == 0 {
			return nil
		}
		f.build(selected)
		if err := f.materialize(); err != nil {
			return err
		}
	}
	return nil
}

func (f *frontendCache) markUncacheable(pkg Package) {
	if pkg == nil {
		return
	}
	if f.uncacheable == nil {
		f.uncacheable = make(map[Package]bool)
	}
	f.uncacheable[pkg] = true
}

// certify runs only after all layout/feature/initializer inputs are fixed.
func (f *frontendCache) certify() error {
	span := f.ctx.buildTrace.startCoordinator("certify frontend exports", nil)
	defer span.done()
	selected := make(map[*ssa.Package]bool)
	for _, pkg := range f.orderedPackages() {
		record := f.records[pkg.ID]
		if record == nil || f.built[pkg.SSA] {
			continue
		}
		if f.uncacheable[pkg] {
			f.source(pkg, selected, "uncacheable")
			continue
		}
		if err := f.ctx.collectFingerprint(pkg); err != nil {
			f.markUncacheable(pkg)
			if f.fingerprintErr == nil {
				f.fingerprintErr = make(map[Package]bool)
			}
			f.fingerprintErr[pkg] = true
			f.source(pkg, selected, "fingerprint-error")
			continue
		}
		if record.Archive != pkg.Fingerprint || !f.ctx.tryLoadFromCacheEx(pkg, false) {
			f.source(pkg, selected, "archive-mismatch")
			continue
		}
		snapshot, err := f.importLink(record.Link)
		if err != nil {
			f.ctx.loadLinkSnapshotSidecar(pkg)
			f.source(pkg, selected, "import-link:"+err.Error())
			continue
		}
		pkg.frontendOpaque, pkg.linkSnapshot = record, snapshot
	}
	if len(selected) > 0 {
		f.build(selected)
		if f.err != nil {
			return f.err
		}
		if err := f.materialize(); err != nil {
			return err
		}
	}
	// Inline preflight also needs archive state for the source-built packages.
	// Skip packages that are decl-only/empty-link or fail speculative
	// fingerprinting (e.g. unused cross-build runtime packages); real build
	// errors are surfaced by prePackageBuild only if the package is needed.
	for _, pkg := range f.orderedPackages() {
		if f.fingerprintErr[pkg] {
			continue
		}
		task := newPackageBuildTask(pkg)
		if task.isDeclOnly() || (task.isLinkOnly() && !task.hasSource()) {
			continue
		}
		if pkg.Fingerprint == "" {
			if err := f.ctx.collectFingerprint(pkg); err != nil {
				f.markUncacheable(pkg)
				if f.fingerprintErr == nil {
					f.fingerprintErr = make(map[Package]bool)
				}
				f.fingerprintErr[pkg] = true
				continue
			}
		}
		f.ctx.tryLoadFromCache(pkg)
	}
	if err := f.importBodies(); err != nil {
		return err
	}
	// A new generic wrapper or runtime type can require a function absent from a
	// dependency's inventory. Build its source instead of fabricating false facts.
	for {
		selected = make(map[*ssa.Package]bool)
		inventories := make(map[*ssa.Package]map[string]frontendFunction)
		for ssaPkg, owner := range f.owners {
			if owner.frontendOpaque == nil {
				continue
			}
			inventory := make(map[string]frontendFunction)
			for _, fn := range owner.frontendOpaque.Functions {
				inventory[fn.ID] = fn
			}
			inventories[ssaPkg] = inventory
		}
		for id, fn := range f.functions {
			owner := f.owners[f.functionPackage(fn)]
			if owner == nil || owner.frontendOpaque == nil {
				continue
			}
			if _, found := inventories[f.functionPackage(fn)][id]; !found || f.uncacheable[owner] {
				f.source(owner, selected, "missing-inventory:"+id)
			}
		}
		for fn := range f.instances {
			owner := f.owners[f.functionPackage(fn)]
			if owner == nil || owner.frontendOpaque == nil {
				continue
			}
			f.source(owner, selected, "generic-instance:"+fn.String())
		}
		if len(selected) == 0 {
			break
		}
		f.build(selected)
		if err := f.materialize(); err != nil {
			return err
		}
	}
	f.imports, f.importKeys = make(map[*ssa.Function]goescape.Summary), make(map[*ssa.Function]string)
	for ssaPkg, owner := range f.owners {
		if owner.frontendOpaque == nil {
			continue
		}
		facts := make(map[*ssa.Function]cl.FrontendFacts)
		for _, item := range owner.frontendOpaque.Functions {
			fn := f.functions[item.ID]
			if fn == nil || f.functionPackage(fn) != ssaPkg {
				continue
			}
			facts[fn] = item.Facts
			if item.Escape != nil && goescape.ValidSummary(fn, *item.Escape) && len(fn.Blocks) == 0 {
				f.imports[fn], f.importKeys[fn] = *item.Escape, item.CacheKey
			}
		}
		f.ctx.callerTracking.ImportFrontendFacts(ssaPkg, facts)
	}
	hits := 0
	for _, pkg := range f.orderedPackages() {
		if pkg.frontendOpaque != nil {
			hits++
		}
	}
	if f.ctx.buildConf.Verbose {
		fmt.Fprintf(os.Stderr, "FRONTEND CACHE packages=%d skipped_ssa=%d\n", len(f.ctx.pkgByID), hits)
	}
	span.setArg("skipped_ssa", hits)
	if target := f.ctx.prog.Target(); target != nil && target.GOARCH == "wasm" {
		span.setArg("wasm_reflect_bridges", target.WasmReflectBridges)
		span.setArg("wasm_func_info_entries", target.WasmFuncInfoEntries)
		span.setArg("wasm_runtime_free", f.ctx.wasmRuntimeFree)
	}
	if f.loader != nil {
		span.setArg("type_exports", len(f.loader.selected))
		span.setArg("types_exportdata", len(f.loader.exportDataHits))
	}
	if f.ctx.dedup != nil {
		span.setArg("skipped_parse", f.ctx.dedup.SkippedParseCount())
	}
	if f.program != nil {
		if err := f.restoreInitializers(); err != nil {
			return err
		}
	}
	return f.err
}

// capture freezes package contracts before LLVM lowering mutates patch scopes.
func (f *frontendCache) capture(result *goescape.Result) {
	f.captureProgram()
	for _, pkg := range f.orderedPackages() {
		if pkg.frontendOpaque != nil {
			if f.loader != nil && !f.loader.selected[pkg.ID] {
				f.loader.capture(pkg, pkg.frontendOpaque, f.types)
				_ = f.loader.store(pkg)
			}
			continue
		}
		if f.uncacheable[pkg] {
			delete(f.records, pkg.ID)
			continue
		}
		graph := frontcache.NewTypeGraph(f.types, f.ctx.progSSA.Fset)
		record := &frontendExport{Version: frontendExportVersion, Input: f.keys[pkg.ID]}
		for _, fn := range f.functions {
			owner := f.owners[f.functionPackage(fn)]
			if owner != pkg || frontendInstance(fn) {
				continue
			}
			item := frontendFunction{ID: f.functionID(fn), Facts: f.ctx.callerTracking.ExportFrontendFacts(f.functionPackage(fn), fn), CacheKey: result.ComponentKey(fn)}
			if summary, ok := result.Summary(fn); ok && goescape.ValidSummary(fn, summary) {
				item.Escape = &summary
			}
			record.Functions = append(record.Functions, item)
		}
		sort.Slice(record.Functions, func(i, j int) bool { return record.Functions[i].ID < record.Functions[j].ID })
		for ssaPkg, owner := range f.owners {
			if owner == pkg {
				for _, typ := range ssaPkg.Prog.RuntimeTypes() {
					// RuntimeTypes is global. Only retain types owned by this archive;
					// caller-created instances remain owned by the referring package.
					if named, ok := derefFrontendType(typ).(*types.Named); ok && named.Obj().Pkg() == ssaPkg.Pkg {
						record.RuntimeTypes = append(record.RuntimeTypes, graph.Add(typ))
					}
				}
			}
		}
		record.Types = *graph
		f.records[pkg.ID] = record
		if f.loader != nil {
			f.loader.capture(pkg, record, f.types)
		}
	}
	if f.loader != nil && f.loader.inlineHintKey != "" {
		f.loader.storeInlineHint(f.rootInlineSourcePackages())
	}
}

// rootInlineSourcePackages identifies the small set of dependency packages
// whose inlineable bodies are transitively called from the current invocation's
// initial root packages. Persisting this hint lets subsequent -Os incremental
// builds skip parsing unrelated dependencies via .llgotypes without retrying.
func (f *frontendCache) rootInlineSourcePackages() []string {
	bodyBudget, _, depth := cl.FrontendInlinePolicy()
	costs := make(map[string]int)
	for _, record := range f.records {
		for _, fn := range record.Functions {
			costs[fn.ID] = fn.Facts.InlineCost
		}
	}
	roots := make(map[Package]bool, len(f.ctx.initial))
	for _, initPkg := range f.ctx.initial {
		if pkg := f.ctx.pkgs[initPkg]; pkg != nil {
			roots[pkg] = true
		}
	}
	targets := make(map[*ssa.Function]*ssa.Function)
	active := make(map[*ssa.Function]bool)
	needed := make(map[string]bool)
	var visit func(*ssa.Function, int)
	visit = func(fn *ssa.Function, level int) {
		if fn == nil || active[fn] || level > depth {
			return
		}
		active[fn] = true
		defer delete(active, fn)
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				call, ok := instr.(ssa.CallInstruction)
				if !ok {
					continue
				}
				callee := call.Common().StaticCallee()
				target, resolved := targets[callee]
				if !resolved {
					target = cl.ResolveInlineTarget(callee, f.ctx.patches)
					targets[callee] = target
				}
				if target == nil {
					continue
				}
				if cost, ok := costs[f.functionID(target)]; ok && cost <= bodyBudget {
					if owner := f.owners[target.Pkg]; owner != nil && !roots[owner] && owner.Name != "main" {
						needed[owner.ID] = true
					}
					visit(target, level+1)
				}
			}
		}
	}
	for _, fn := range f.functions {
		owner := f.owners[f.functionPackage(fn)]
		if owner != nil && roots[owner] && len(fn.Blocks) > 0 {
			visit(fn, 0)
		}
	}
	for fn := range f.instances {
		if len(fn.Blocks) == 0 {
			continue
		}
		owner := f.owners[f.functionPackage(fn)]
		if owner != nil && roots[owner] {
			visit(fn, 0)
		}
	}
	out := make([]string, 0, len(needed))
	for id := range needed {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func derefFrontendType(typ types.Type) types.Type {
	if pointer, ok := typ.(*types.Pointer); ok {
		return types.Unalias(pointer.Elem())
	}
	return types.Unalias(typ)
}

// frontendCallerSynthetic reports whether fn is a per-use-site synthetic
// wrapper ($bound method value or $thunk method expression) generated on demand
// with a body by ssa.Builder. Multiple call sites legitimately create distinct
// *ssa.Function instances with identical identities and never rely on opaque
// package contracts.
func frontendCallerSynthetic(fn *ssa.Function) bool {
	if fn == nil || fn.Pkg != nil || fn.Parent() != nil {
		return false
	}
	return strings.HasPrefix(fn.Synthetic, "bound method wrapper for ") ||
		strings.HasPrefix(fn.Synthetic, "thunk for ")
}

// Instances and nested bodies can depend on caller-defined types outside the
// defining library's source closure. They remain source/body imports rather
// than being certified by that library's generic archive contract.
func frontendInstance(fn *ssa.Function) bool {
	for fn != nil {
		if fn.Origin() != nil || len(fn.TypeArgs()) != 0 {
			return true
		}
		fn = fn.Parent()
	}
	return false
}

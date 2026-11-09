// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package cl

import (
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// FrontendFacts are the body-independent contracts a caller may need from an
// archive-cached function. They must be exported from the same analysis and
// policy that compiled the owning archive, never inferred from an empty body.
type FrontendFacts struct {
	CallerBase, CallerFrame, RecoverPanicSite bool
	UsesRecover, RecoverScope                 bool
	InlineCost                                int
}

// FrontendInlinePolicy exposes the single import policy to the coordinator,
// which materializes eligible dependency bodies before backend workers start.
func FrontendInlinePolicy() (bodyBudget, moduleBudget, depth int) {
	return inlineBodyBudget, inlineModuleBudget, inlineImportDepth
}

// MaterializeFrontendMethods restores runtime method requirements before
// workers start. It includes private/local types absent from the public API.
func (c *CallerTracking) MaterializeFrontendMethods(pkgs []*ssa.Package, extra []types.Type) map[*ssa.Package][]*ssa.Function {
	c.exportedRuntimeTypes = extra
	runtimeTypes := callerTrackingRuntimeTypes(pkgs, extra...)
	return callerTrackingMethods(pkgs, runtimeTypes)
}

func (c *CallerTracking) ExportFrontendFacts(pkg *ssa.Package, fn *ssa.Function) FrontendFacts {
	sets := callerTrackingFuncSetsForPackage(c, pkg)
	r := c.recoverAnalysis()
	facts := FrontendFacts{
		CallerBase: runtimeCallerBaseSet(c, pkg)[fn], CallerFrame: sets.frames[fn], RecoverPanicSite: sets.recoverPanicSites[fn],
		UsesRecover: r.functionUsesRecover(fn), RecoverScope: r.needsRecoverScope(fn),
		InlineCost: inlineBodyBudget + 1,
	}
	if !facts.CallerBase && !facts.UsesRecover && !needsRuntimeStackNoInline(pkg.Pkg, fn) {
		facts.InlineCost = c.inlines.info(fn).cost
	}
	return facts
}

// ImportFrontendFacts installs a complete validated package contract. The
// caller must materialize all required method wrappers and reject incomplete
// function inventories before calling this method. Source-built packages use
// the ordinary Precompute path, including dependencies' imported base sets.
func (c *CallerTracking) ImportFrontendFacts(pkg *ssa.Package, functions map[*ssa.Function]FrontendFacts) {
	if c.precomputed {
		panic("frontend facts imported after caller precomputation")
	}
	base, frames, panicSites := make(map[*ssa.Function]bool), make(map[*ssa.Function]bool), make(map[*ssa.Function]bool)
	r := c.recoverAnalysis()
	r.mu.Lock()
	defer r.mu.Unlock()
	for fn, facts := range functions {
		if facts.CallerBase {
			base[fn] = true
		}
		if facts.CallerFrame {
			frames[fn] = true
		}
		if facts.RecoverPanicSite {
			panicSites[fn] = true
		}
		r.uses[fn], r.scopes[fn] = facts.UsesRecover, facts.RecoverScope
	}
	c.base[pkg] = base
	c.extended[pkg] = callerTrackingFuncSets{frames: frames, recoverPanicSites: panicSites}
	if c.imported == nil {
		c.imported = make(map[*ssa.Package]bool)
	}
	c.imported[pkg] = true
}

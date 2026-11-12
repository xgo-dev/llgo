package ssa

import "strconv"

// recordReflectMethodSite assigns an unresolved Value.MethodByName lookup an
// identity shared by its package-metadata demand and LLVM call-site attribute.
// caller is the full LLVM function symbol, including its package identity, so
// packages can record sites independently without a program-wide counter.
// Each lookup has its own demand: proving names at one site must not erase an
// unrelated dynamic lookup in the same function.
func (p Package) recordReflectMethodSite(caller string) string {
	if !p.Prog.DeadcodeDropEnabled() || p.metaBuilder == nil {
		p.MarkReflectMethod(caller)
		return ""
	}
	// Count only recorded sites, separately for each function. Re-emitting the
	// same generic body in another package intentionally gives its corresponding
	// sites the same IDs, independent of unrelated functions' emission order.
	// These are compilation identities, not stable IDs across source edits.
	if p.reflectSiteCounts == nil {
		p.reflectSiteCounts = make(map[string]uint64)
	}
	p.reflectSiteCounts[caller]++
	site := caller + "$reflectsite$" + strconv.FormatUint(p.reflectSiteCounts[caller], 10)
	p.metaBuilder.AddReflectMethodSite(p.metaBuilder.Sym(caller), p.metaBuilder.Sym(site))
	return site
}

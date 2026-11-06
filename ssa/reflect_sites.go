package ssa

import "strconv"

// Each unresolved lookup remains a separate demand. Proving a finite name set
// at one site must not erase an unrelated dynamic lookup in the same function.
func (p Package) recordReflectMethodSite(caller string) string {
	if !p.Prog.DeadcodeDropEnabled() || p.metaBuilder == nil {
		p.MarkReflectMethod(caller)
		return ""
	}
	// Function-local numbering is stable when the same generic body is
	// emitted in several packages with different package compilation orders.
	if p.reflectSiteCounts == nil {
		p.reflectSiteCounts = make(map[string]uint64)
	}
	p.reflectSiteCounts[caller]++
	site := caller + "$reflectsite$" + strconv.FormatUint(p.reflectSiteCounts[caller], 10)
	p.metaBuilder.AddReflectMethodSite(p.metaBuilder.Sym(caller), p.metaBuilder.Sym(site))
	return site
}

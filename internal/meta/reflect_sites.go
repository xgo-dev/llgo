package meta

import (
	"slices"
	"strings"
)

// ReflectMethodSites returns the distinct source lookup identities that could
// be refined from this package's optimized IR, in sorted order. These are the
// stable string keys used by NewGlobalSummaryWithReflectNames, not local IDs.
// This inventory includes unreachable functions' sites; demand activation is
// left to whole-program reachability, not this package-level query.
// Both the slice and its strings are owned by the caller and remain valid after
// pm.Close.
func (pm *PackageMeta) ReflectMethodSites() []string {
	if pm == nil || pm.nsyms == 0 {
		return nil
	}
	_, total := pm.csrRange(pm.demandOff, Symbol(pm.nsyms-1))
	if total == 0 {
		return nil
	}
	var seen map[string]struct{}
	var out []string
	for sym := Symbol(0); sym < Symbol(pm.nsyms); sym++ {
		for _, demand := range pm.funcDemands(sym) {
			if demand.Kind != DemandReflectMethodSite {
				continue
			}
			name := pm.symbolName(Symbol(demand.Target))
			if _, ok := seen[name]; ok {
				continue
			}
			if seen == nil {
				seen = make(map[string]struct{})
			}
			seen[name] = struct{}{}
			out = append(out, strings.Clone(name))
		}
	}
	slices.Sort(out)
	return out
}

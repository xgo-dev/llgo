package meta

import (
	"fmt"
	"sort"
	"strings"
)

// String returns a human-readable representation for testing and debugging.
func (pm *PackageMeta) String() string {
	if pm == nil {
		return "<nil>"
	}
	var sb strings.Builder
	formatMeta(&sb, pm)
	return sb.String()
}

// formatMeta writes a human-readable representation of pm to w,
// grouped by section (TypeChildren, OrdinaryEdges, UseIface, etc.)
// to match the original metadata format used by golden file tests.
func formatMeta(w *strings.Builder, pm *PackageMeta) {
	n := pm.nsyms
	symName := func(sym Symbol) string { return pm.symbolName(sym) }

	// collect per-sym edge lists by kind
	type kindMap = map[string][]string // src → []dst (sorted)
	ordinary := make(map[string][]string)
	useIface := make(map[string][]string)
	useIfaceMethod := make(map[string][]string) // src → ["iface[idx]", ...]
	useNamed := make(map[string][]string)

	for i := Symbol(0); i < Symbol(n); i++ {
		src := symName(i)
		for _, dst := range pm.ordinaryEdges(i) {
			ordinary[src] = append(ordinary[src], symName(dst))
		}
		for _, d := range pm.funcDemands(i) {
			switch d.Kind {
			case DemandUseIface:
				useIface[src] = append(useIface[src], symName(Symbol(d.Target)))
			case DemandIfaceMethod:
				ifaceSym := Symbol(d.Target)
				iface := symName(ifaceSym)
				sigs := pm.ifaceMethods(ifaceSym)
				s := sigs[d.Extra]
				useIfaceMethod[src] = append(useIfaceMethod[src],
					fmt.Sprintf("%s %s %s", iface, pm.nameString(s.Name), symName(s.MType)))
			case DemandNamedMethod:
				name := pm.nameString(nameRef{Off: d.Target, Len: d.Extra})
				useNamed[src] = append(useNamed[src], name)
			}
		}
	}

	// collect TypeChildren
	typeChildren := make(map[string][]string)
	for i := Symbol(0); i < Symbol(n); i++ {
		parent := symName(i)
		for _, c := range pm.typeChildren(i) {
			typeChildren[parent] = append(typeChildren[parent], symName(c))
		}
	}

	// collect MethodInfo
	type slotInfo struct{ name, mtype, ifn, tfn string }
	methodInfo := make(map[string][]slotInfo)
	for i := Symbol(0); i < Symbol(n); i++ {
		typ := symName(i)
		for _, s := range pm.methodSlots(i) {
			methodInfo[typ] = append(methodInfo[typ], slotInfo{
				name:  pm.nameString(s.Name),
				mtype: symName(s.MType),
				ifn:   symName(s.IFn),
				tfn:   symName(s.TFn),
			})
		}
	}

	// collect InterfaceInfo
	type sigInfo struct{ name, mtype string }
	ifaceInfo := make(map[string][]sigInfo)
	for i := Symbol(0); i < Symbol(n); i++ {
		iface := symName(i)
		for _, s := range pm.ifaceMethods(i) {
			ifaceInfo[iface] = append(ifaceInfo[iface], sigInfo{
				name:  pm.nameString(s.Name),
				mtype: symName(s.MType),
			})
		}
	}

	// collect Reflect
	var reflectSyms []string
	for i := Symbol(0); i < Symbol(n); i++ {
		for _, d := range pm.funcDemands(i) {
			if d.Kind == DemandReflectMethod {
				reflectSyms = append(reflectSyms, symName(i))
				break
			}
		}
	}

	printSection := func(title string, m map[string][]string) {
		if len(m) == 0 {
			return
		}
		fmt.Fprintf(w, "[%s]\n", title)
		keys := sortedKeys(m)
		for _, k := range keys {
			vals := m[k]
			sort.Strings(vals)
			fmt.Fprintf(w, "%s:\n", k)
			for _, v := range vals {
				fmt.Fprintf(w, "    %s\n", v)
			}
		}
		fmt.Fprintln(w)
	}

	printSection("TypeChildren", typeChildren)
	printSection("OrdinaryEdges", ordinary)
	printSection("UseIface", useIface)
	printSection("UseIfaceMethod", useIfaceMethod)
	printSection("UseNamedMethod", useNamed)

	if len(methodInfo) > 0 {
		fmt.Fprintln(w, "[MethodInfo]")
		keys := make([]string, 0, len(methodInfo))
		for k := range methodInfo {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, typ := range keys {
			fmt.Fprintf(w, "%s:\n", typ)
			for idx, s := range methodInfo[typ] {
				fmt.Fprintf(w, "    %d %s %s %s %s\n", idx, s.name, s.mtype, s.ifn, s.tfn)
			}
		}
		fmt.Fprintln(w)
	}

	if len(ifaceInfo) > 0 {
		fmt.Fprintln(w, "[InterfaceInfo]")
		keys := make([]string, 0, len(ifaceInfo))
		for k := range ifaceInfo {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, iface := range keys {
			fmt.Fprintf(w, "%s:\n", iface)
			for _, s := range ifaceInfo[iface] {
				fmt.Fprintf(w, "    %s %s\n", s.name, s.mtype)
			}
		}
		fmt.Fprintln(w)
	}

	if len(reflectSyms) > 0 {
		sort.Strings(reflectSyms)
		fmt.Fprintln(w, "[Reflect]")
		for _, r := range reflectSyms {
			fmt.Fprintf(w, "    %s\n", r)
		}
		fmt.Fprintln(w)
	}
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

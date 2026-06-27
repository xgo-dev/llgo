package meta

import (
	"fmt"
	"sort"
	"strings"
)

// MetaString formats pm as a human-readable string for testing and debugging.
func MetaString(pm *PackageMeta) string {
	if pm == nil {
		return "<nil>"
	}
	var sb strings.Builder
	FormatMeta(&sb, pm)
	return sb.String()
}

// FormatMeta writes a human-readable representation of pm to w,
// grouped by section (TypeChildren, OrdinaryEdges, UseIface, etc.)
// to match the original metadata format used by golden file tests.
func FormatMeta(w *strings.Builder, pm *PackageMeta) {
	n := pm.NSyms()
	symName := func(sym LocalSymbol) string { return pm.SymbolName(sym) }

	// collect per-sym edge lists by kind
	type kindMap = map[string][]string // src → []dst (sorted)
	ordinary := make(map[string][]string)
	useIface := make(map[string][]string)
	useIfaceMethod := make(map[string][]string) // src → ["iface[idx]", ...]
	useNamed := make(map[string][]string)

	for i := LocalSymbol(0); i < LocalSymbol(n); i++ {
		src := symName(i)
		for _, e := range pm.Edges(i) {
			switch e.Kind {
			case EdgeOrdinary:
				ordinary[src] = append(ordinary[src], symName(LocalSymbol(e.Target)))
			case EdgeUseIface:
				useIface[src] = append(useIface[src], symName(LocalSymbol(e.Target)))
			case EdgeUseIfaceMethod:
				ifaceSym := LocalSymbol(e.Target)
				iface := symName(ifaceSym)
				sigs := pm.IfaceMethods(ifaceSym)
				if int(e.Extra) < len(sigs) {
					s := sigs[e.Extra]
					useIfaceMethod[src] = append(useIfaceMethod[src],
						fmt.Sprintf("%s %s %s", iface, pm.NameString(s.Name), symName(s.MType)))
				} else {
					useIfaceMethod[src] = append(useIfaceMethod[src],
						fmt.Sprintf("%s[%d]", iface, e.Extra))
				}
			case EdgeUseNamedMethod:
				name := pm.NameString(NameRef{Off: e.Target, Len: e.Extra})
				useNamed[src] = append(useNamed[src], name)
			}
		}
	}

	// collect TypeChildren
	typeChildren := make(map[string][]string)
	for i := LocalSymbol(0); i < LocalSymbol(n); i++ {
		parent := symName(i)
		for _, c := range pm.TypeChildren(i) {
			typeChildren[parent] = append(typeChildren[parent], symName(c))
		}
	}

	// collect MethodInfo
	type slotInfo struct{ name, mtype, ifn, tfn string }
	methodInfo := make(map[string][]slotInfo)
	for i := LocalSymbol(0); i < LocalSymbol(n); i++ {
		typ := symName(i)
		for _, s := range pm.MethodSlots(i) {
			methodInfo[typ] = append(methodInfo[typ], slotInfo{
				name:  pm.NameString(s.Name),
				mtype: symName(s.MType),
				ifn:   symName(s.IFn),
				tfn:   symName(s.TFn),
			})
		}
	}

	// collect InterfaceInfo
	type sigInfo struct{ name, mtype string }
	ifaceInfo := make(map[string][]sigInfo)
	for i := LocalSymbol(0); i < LocalSymbol(n); i++ {
		iface := symName(i)
		for _, s := range pm.IfaceMethods(i) {
			ifaceInfo[iface] = append(ifaceInfo[iface], sigInfo{
				name:  pm.NameString(s.Name),
				mtype: symName(s.MType),
			})
		}
	}

	// collect Reflect
	var reflectSyms []string
	for i := LocalSymbol(0); i < LocalSymbol(n); i++ {
		if pm.HasReflect(i) {
			reflectSyms = append(reflectSyms, symName(i))
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
